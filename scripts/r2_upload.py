#!/usr/bin/env python3
"""Upload a locally signed SongLib web hot-update snapshot to Cloudflare R2.

Guarantees:
1. Pre-flight verification of manifest.json against local files/ (size + SHA-256).
2. Uploads all files/* objects FIRST (in parallel with retries), then uploads
   manifest.json LAST so clients never observe a manifest referencing missing files.
3. Supports least-privilege bucket-scoped R2 S3 API tokens (recommended) via
   zero-dependency AWS SigV4, or Cloudflare API Bearer tokens via R2 REST API.
4. Never prints credentials or environment secret values.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import datetime
import hashlib
import hmac
import json
import mimetypes
import os
import re
import ssl
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Tuple


CONTENT_TYPES: Dict[str, str] = {
    ".html": "text/html; charset=utf-8",
    ".htm": "text/html; charset=utf-8",
    ".js": "application/javascript; charset=utf-8",
    ".mjs": "application/javascript; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".json": "application/json",
    ".svg": "image/svg+xml",
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".webp": "image/webp",
    ".gif": "image/gif",
    ".ico": "image/x-icon",
    ".woff2": "font/woff2",
    ".woff": "font/woff",
    ".ttf": "font/ttf",
    ".otf": "font/otf",
    ".txt": "text/plain; charset=utf-8",
    ".xml": "application/xml; charset=utf-8",
}

SAFE_PREFIX_RE = re.compile(r"^[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*$")


@dataclass(frozen=True)
class UploadItem:
    local_path: Path
    object_key: str
    content_type: str
    size: int


def detect_content_type(path: Path) -> str:
    ext = path.suffix.lower()
    if ext in CONTENT_TYPES:
        return CONTENT_TYPES[ext]
    guessed, _ = mimetypes.guess_type(str(path))
    return guessed or "application/octet-stream"


def normalize_prefix(raw: str) -> str:
    prefix = raw.strip().strip("/")
    if not prefix:
        raise ValueError("object key prefix must not be empty")
    if ".." in prefix.split("/") or not SAFE_PREFIX_RE.match(prefix):
        raise ValueError(f"unsafe object key prefix: {raw!r}")
    return prefix


def verify_and_collect_release(release_dir: Path, prefix: str) -> Tuple[List[UploadItem], UploadItem, dict]:
    if not release_dir.is_dir():
        raise ValueError(f"release directory does not exist: {release_dir}")

    manifest_path = release_dir / "manifest.json"
    files_root = release_dir / "files"
    if not manifest_path.is_file() or manifest_path.is_symlink():
        raise ValueError(f"missing regular manifest.json in {release_dir}")
    if not files_root.is_dir() or files_root.is_symlink():
        raise ValueError(f"missing files/ directory in {release_dir}")

    envelope = json.loads(manifest_path.read_text(encoding="utf-8"))
    if not isinstance(envelope, dict) or "payload" not in envelope or "signature" not in envelope:
        raise ValueError("manifest.json is not a signed envelope")

    payload = envelope["payload"]
    if not isinstance(payload, dict) or not isinstance(payload.get("files"), list):
        raise ValueError("manifest.json payload is missing files list")

    expected_rel_paths = set()
    items: List[UploadItem] = []
    for entry in payload["files"]:
        rel = entry["path"]
        if not isinstance(rel, str) or rel.startswith("/") or ".." in rel.split("/"):
            raise ValueError(f"unsafe manifest file path: {rel!r}")
        expected_rel_paths.add(rel)

        local_file = files_root / rel
        if not local_file.is_file() or local_file.is_symlink():
            raise ValueError(f"manifest file missing on disk: {local_file}")

        data = local_file.read_bytes()
        if len(data) != int(entry["size"]):
            raise ValueError(
                f"size mismatch for {rel}: disk={len(data)} manifest={entry['size']}"
            )
        digest = hashlib.sha256(data).hexdigest()
        if digest != str(entry["sha256"]).lower():
            raise ValueError(
                f"sha256 mismatch for {rel}: disk={digest} manifest={entry['sha256']}"
            )

        items.append(
            UploadItem(
                local_path=local_file,
                object_key=f"{prefix}/files/{rel}",
                content_type=detect_content_type(local_file),
                size=len(data),
            )
        )

    actual_rel_paths = set()
    for path in files_root.rglob("*"):
        if path.is_symlink():
            raise ValueError(f"symlink not allowed in release files: {path}")
        if path.is_file():
            actual_rel_paths.add(path.relative_to(files_root).as_posix())

    extra = sorted(actual_rel_paths - expected_rel_paths)
    if extra:
        raise ValueError(f"unexpected files not listed in manifest.json: {extra[:5]}")

    items.sort(key=lambda x: x.object_key)
    manifest_item = UploadItem(
        local_path=manifest_path,
        object_key=f"{prefix}/manifest.json",
        content_type="application/json",
        size=manifest_path.stat().st_size,
    )
    meta = {
        "key_id": envelope.get("key_id", ""),
        "version": payload.get("version", ""),
        "sequence": payload.get("sequence", 0),
        "min_backend_version": payload.get("min_backend_version", ""),
        "backend_id": payload.get("backend_id", ""),
        "file_count": len(items),
        "total_bytes": sum(i.size for i in items) + manifest_item.size,
    }
    return items, manifest_item, meta


def _hmac_sha256(key: bytes, msg: str) -> bytes:
    return hmac.new(key, msg.encode("utf-8"), hashlib.sha256).digest()


def build_sigv4_put_request(
    *,
    endpoint: str,
    bucket: str,
    object_key: str,
    body: bytes,
    content_type: str,
    access_key_id: str,
    secret_access_key: str,
    region: str = "auto",
    service: str = "s3",
    now: Optional[datetime.datetime] = None,
) -> urllib.request.Request:
    parsed = urllib.parse.urlparse(endpoint.rstrip("/"))
    if not parsed.scheme or not parsed.netloc:
        raise ValueError(f"invalid S3 endpoint: {endpoint!r}")

    encoded_bucket = urllib.parse.quote(bucket, safe="-_.~")
    encoded_key = urllib.parse.quote(object_key, safe="/-_.~")
    base_path = parsed.path.rstrip("/")
    canonical_uri = f"{base_path}/{encoded_bucket}/{encoded_key}"
    url = f"{parsed.scheme}://{parsed.netloc}{canonical_uri}"

    ts = now or datetime.datetime.now(datetime.timezone.utc)
    amz_date = ts.strftime("%Y%m%dT%H%M%SZ")
    date_stamp = ts.strftime("%Y%m%d")
    payload_hash = hashlib.sha256(body).hexdigest()

    headers_to_sign = {
        "content-type": content_type,
        "host": parsed.netloc,
        "x-amz-content-sha256": payload_hash,
        "x-amz-date": amz_date,
    }
    signed_header_names = sorted(headers_to_sign.keys())
    canonical_headers = "".join(
        f"{name}:{headers_to_sign[name]}\n" for name in signed_header_names
    )
    signed_headers = ";".join(signed_header_names)

    canonical_request = "\n".join(
        [
            "PUT",
            canonical_uri,
            "",
            canonical_headers,
            signed_headers,
            payload_hash,
        ]
    )
    credential_scope = f"{date_stamp}/{region}/{service}/aws4_request"
    string_to_sign = "\n".join(
        [
            "AWS4-HMAC-SHA256",
            amz_date,
            credential_scope,
            hashlib.sha256(canonical_request.encode("utf-8")).hexdigest(),
        ]
    )

    k_date = _hmac_sha256(("AWS4" + secret_access_key).encode("utf-8"), date_stamp)
    k_region = _hmac_sha256(k_date, region)
    k_service = _hmac_sha256(k_region, service)
    k_signing = _hmac_sha256(k_service, "aws4_request")
    signature = hmac.new(
        k_signing, string_to_sign.encode("utf-8"), hashlib.sha256
    ).hexdigest()

    authorization = (
        f"AWS4-HMAC-SHA256 Credential={access_key_id}/{credential_scope}, "
        f"SignedHeaders={signed_headers}, Signature={signature}"
    )

    req = urllib.request.Request(url=url, data=body, method="PUT")
    req.add_header("Content-Type", content_type)
    req.add_header("Content-Length", str(len(body)))
    req.add_header("x-amz-content-sha256", payload_hash)
    req.add_header("x-amz-date", amz_date)
    req.add_header("Authorization", authorization)
    return req


def build_cf_rest_put_request(
    *,
    api_base: str,
    account_id: str,
    bucket: str,
    object_key: str,
    body: bytes,
    content_type: str,
    api_token: str,
) -> urllib.request.Request:
    encoded_bucket = urllib.parse.quote(bucket, safe="-_.~")
    encoded_key = urllib.parse.quote(object_key, safe="/-_.~")
    url = (
        f"{api_base.rstrip('/')}/accounts/{urllib.parse.quote(account_id, safe='')}"
        f"/r2/buckets/{encoded_bucket}/objects/{encoded_key}"
    )
    req = urllib.request.Request(url=url, data=body, method="PUT")
    req.add_header("Authorization", f"Bearer {api_token}")
    req.add_header("Content-Type", content_type)
    req.add_header("Content-Length", str(len(body)))
    return req


class R2Uploader:
    def __init__(
        self,
        *,
        mode: str,
        bucket: str,
        account_id: str,
        access_key_id: str = "",
        secret_access_key: str = "",
        s3_endpoint: str = "",
        api_token: str = "",
        api_base: str = "https://api.cloudflare.com/client/v4",
        timeout: int = 30,
        retries: int = 3,
    ) -> None:
        self.mode = mode
        self.bucket = bucket
        self.account_id = account_id
        self.access_key_id = access_key_id
        self.secret_access_key = secret_access_key
        self.s3_endpoint = s3_endpoint or f"https://{account_id}.r2.cloudflarestorage.com"
        self.api_token = api_token
        self.api_base = api_base
        self.timeout = timeout
        self.retries = retries
        self.ssl_context = ssl.create_default_context()

    def upload_item(self, item: UploadItem) -> None:
        body = item.local_path.read_bytes()
        last_err: Optional[Exception] = None
        for attempt in range(1, self.retries + 1):
            try:
                if self.mode == "s3":
                    req = build_sigv4_put_request(
                        endpoint=self.s3_endpoint,
                        bucket=self.bucket,
                        object_key=item.object_key,
                        body=body,
                        content_type=item.content_type,
                        access_key_id=self.access_key_id,
                        secret_access_key=self.secret_access_key,
                    )
                else:
                    req = build_cf_rest_put_request(
                        api_base=self.api_base,
                        account_id=self.account_id,
                        bucket=self.bucket,
                        object_key=item.object_key,
                        body=body,
                        content_type=item.content_type,
                        api_token=self.api_token,
                    )
                ctx = self.ssl_context if req.full_url.startswith("https://") else None
                with urllib.request.urlopen(req, timeout=self.timeout, context=ctx) as resp:
                    status = resp.status
                    raw = resp.read()
                    if status not in (200, 201, 204):
                        raise RuntimeError(f"unexpected HTTP {status} for {item.object_key}")
                    if self.mode == "rest":
                        parsed = json.loads(raw.decode("utf-8"))
                        if not parsed.get("success"):
                            raise RuntimeError(
                                f"Cloudflare API error for {item.object_key}: {parsed.get('errors')}"
                            )
                return
            except urllib.error.HTTPError as exc:
                err_body = exc.read().decode("utf-8", errors="replace")[:300]
                last_err = RuntimeError(
                    f"HTTP {exc.code} uploading {item.object_key}: {err_body}"
                )
                if 400 <= exc.code < 500 and exc.code not in (408, 429):
                    raise last_err from exc
            except Exception as exc:  # noqa: BLE001
                last_err = exc

            if attempt < self.retries:
                time.sleep(0.5 * attempt)

        assert last_err is not None
        raise last_err


def resolve_uploader(args: argparse.Namespace) -> R2Uploader:
    account_id = (args.account_id or os.environ.get("CLOUDFLARE_ACCOUNT_ID", "")).strip()
    access_key_id = os.environ.get("R2_ACCESS_KEY_ID", "").strip()
    secret_access_key = os.environ.get("R2_SECRET_ACCESS_KEY", "").strip()
    s3_endpoint = os.environ.get("R2_S3_ENDPOINT", "").strip()
    api_token = os.environ.get("CLOUDFLARE_API_TOKEN", "").strip()
    api_base = os.environ.get("CLOUDFLARE_API_BASE", "https://api.cloudflare.com/client/v4").strip()

    if access_key_id and secret_access_key:
        if not s3_endpoint and not account_id:
            raise ValueError("CLOUDFLARE_ACCOUNT_ID or R2_S3_ENDPOINT is required for S3 mode")
        return R2Uploader(
            mode="s3",
            bucket=args.bucket,
            account_id=account_id,
            access_key_id=access_key_id,
            secret_access_key=secret_access_key,
            s3_endpoint=s3_endpoint,
        )

    if api_token:
        if not account_id:
            raise ValueError("CLOUDFLARE_ACCOUNT_ID is required for Cloudflare API token mode")
        return R2Uploader(
            mode="rest",
            bucket=args.bucket,
            account_id=account_id,
            api_token=api_token,
            api_base=api_base,
        )

    raise ValueError(
        "Missing R2 upload credentials. Set R2_ACCESS_KEY_ID + R2_SECRET_ACCESS_KEY "
        "(recommended, bucket-scoped R2 S3 API token) or CLOUDFLARE_API_TOKEN "
        "in ~/.config/songlib-r2.env (chmod 600)."
    )


def main(argv: Optional[List[str]] = None) -> int:
    parser = argparse.ArgumentParser(
        description="Verify and upload a signed SongLib web release snapshot to Cloudflare R2."
    )
    parser.add_argument("--release-dir", required=True, help="Signed release directory containing manifest.json and files/")
    parser.add_argument("--bucket", required=True, help="Target R2 bucket name")
    parser.add_argument("--prefix", required=True, help="Object key prefix, e.g. tv-installer/v1.2.0")
    parser.add_argument("--account-id", default="", help="Cloudflare Account ID (or env CLOUDFLARE_ACCOUNT_ID)")
    parser.add_argument("--concurrency", type=int, default=8, help="Parallel uploads for files/* (default: 8)")
    parser.add_argument("--dry-run", action="store_true", help="Verify snapshot and print planned uploads without network calls")
    args = parser.parse_args(argv)

    prefix = normalize_prefix(args.prefix)
    items, manifest_item, meta = verify_and_collect_release(Path(args.release_dir), prefix)

    print(
        f"[verify] OK version={meta['version']} seq={meta['sequence']} "
        f"key_id={meta['key_id']} backend_id={meta['backend_id'][:12]}... "
        f"files={meta['file_count']} total={meta['total_bytes']} bytes"
    )

    if args.dry_run:
        for item in items:
            print(f"[dry-run] PUT {args.bucket}/{item.object_key} ({item.content_type}, {item.size} B)")
        print(
            f"[dry-run] PUT (LAST) {args.bucket}/{manifest_item.object_key} "
            f"({manifest_item.content_type}, {manifest_item.size} B)"
        )
        return 0

    uploader = resolve_uploader(args)
    started = time.monotonic()
    print(
        f"[upload] Starting files/* upload ({len(items)} objects, mode={uploader.mode}, "
        f"concurrency={args.concurrency}) -> bucket={args.bucket} prefix={prefix}"
    )

    with concurrent.futures.ThreadPoolExecutor(max_workers=max(1, args.concurrency)) as pool:
        future_map = {pool.submit(uploader.upload_item, item): item for item in items}
        done_count = 0
        for fut in concurrent.futures.as_completed(future_map):
            item = future_map[fut]
            fut.result()
            done_count += 1
            if done_count % 20 == 0 or done_count == len(items):
                print(f"[upload] files/* progress: {done_count}/{len(items)}")

    print(f"[upload] All files/* uploaded. Uploading manifest.json LAST -> {manifest_item.object_key}")
    uploader.upload_item(manifest_item)
    elapsed = time.monotonic() - started
    print(f"[upload] Done in {elapsed:.2f}s. Published: {manifest_item.object_key}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(f"r2_upload: ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
