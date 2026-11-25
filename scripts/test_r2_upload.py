#!/usr/bin/env python3
"""Unit tests for scripts/r2_upload.py using stdlib mock HTTP server."""

from __future__ import annotations

import hashlib
import http.server
import json
import os
import tempfile
import threading
import unittest
from pathlib import Path
from typing import List, Tuple

import r2_upload


class RecordingHandler(http.server.BaseHTTPRequestHandler):
    recorded: List[Tuple[str, str, dict, bytes]] = []
    mode: str = "s3"

    def do_PUT(self) -> None:  # noqa: N802
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        self.recorded.append((self.command, self.path, dict(self.headers), body))
        if self.mode == "rest":
            resp = json.dumps({"success": True, "errors": [], "result": {"key": self.path}}).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(resp)))
            self.end_headers()
            self.wfile.write(resp)
        else:
            self.send_response(200)
            self.send_header("Content-Length", "0")
            self.end_headers()

    def log_message(self, format: str, *args: object) -> None:  # noqa: A002
        return


def create_sample_release(root: Path) -> Path:
    files_dir = root / "files"
    (files_dir / "assets").mkdir(parents=True)
    index_bytes = b"<!doctype html><html><body>ok</body></html>"
    js_bytes = b"console.log('hi');\n"
    (files_dir / "index.html").write_bytes(index_bytes)
    (files_dir / "assets" / "app.1234.js").write_bytes(js_bytes)

    payload = {
        "schema": 1,
        "version": "1.2.0-ui.1",
        "sequence": 1,
        "min_backend_version": "1.2.0",
        "backend_id": "a" * 64,
        "files": [
            {
                "path": "assets/app.1234.js",
                "size": len(js_bytes),
                "sha256": hashlib.sha256(js_bytes).hexdigest(),
            },
            {
                "path": "index.html",
                "size": len(index_bytes),
                "sha256": hashlib.sha256(index_bytes).hexdigest(),
            },
        ],
    }
    envelope = {
        "key_id": "songlib-web-2026",
        "payload": payload,
        "signature": "dGVzdA==",
    }
    (root / "manifest.json").write_text(json.dumps(envelope), encoding="utf-8")
    return root


class R2UploadTests(unittest.TestCase):
    def test_verify_rejects_corrupted_file(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            rel = create_sample_release(Path(tmp))
            (rel / "files" / "index.html").write_bytes(b"tampered")
            with self.assertRaises(ValueError):
                r2_upload.verify_and_collect_release(rel, "tv-installer/v1.2.0")

    def test_s3_upload_preserves_slashes_and_uploads_manifest_last(self) -> None:
        RecordingHandler.recorded = []
        RecordingHandler.mode = "s3"
        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), RecordingHandler)
        port = server.server_address[1]
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory() as tmp:
                rel = create_sample_release(Path(tmp))
                old_env = dict(os.environ)
                try:
                    os.environ["R2_ACCESS_KEY_ID"] = "ak_test"
                    os.environ["R2_SECRET_ACCESS_KEY"] = "sk_test"
                    os.environ["R2_S3_ENDPOINT"] = f"http://127.0.0.1:{port}"
                    os.environ["CLOUDFLARE_ACCOUNT_ID"] = "acct123"
                    rc = r2_upload.main(
                        [
                            "--release-dir",
                            str(rel),
                            "--bucket",
                            "songlib-update",
                            "--prefix",
                            "tv-installer/v1.2.0",
                            "--concurrency",
                            "2",
                        ]
                    )
                    self.assertEqual(rc, 0)
                finally:
                    os.environ.clear()
                    os.environ.update(old_env)
        finally:
            server.shutdown()
            server.server_close()

        paths = [entry[1] for entry in RecordingHandler.recorded]
        self.assertEqual(len(paths), 3)
        # Slashes must be literal, never %2F
        self.assertTrue(all("%2F" not in p and "%2f" not in p for p in paths))
        # manifest.json MUST be uploaded strictly last
        self.assertEqual(paths[-1], "/songlib-update/tv-installer/v1.2.0/manifest.json")
        self.assertEqual(
            set(paths[:-1]),
            {
                "/songlib-update/tv-installer/v1.2.0/files/index.html",
                "/songlib-update/tv-installer/v1.2.0/files/assets/app.1234.js",
            },
        )
        # Verify SigV4 headers & Content-Type
        by_path = {entry[1]: entry[2] for entry in RecordingHandler.recorded}
        self.assertIn("AWS4-HMAC-SHA256", by_path[paths[-1]].get("Authorization", ""))
        self.assertEqual(by_path[paths[-1]].get("Content-Type"), "application/json")
        self.assertEqual(
            by_path["/songlib-update/tv-installer/v1.2.0/files/assets/app.1234.js"].get("Content-Type"),
            "application/javascript; charset=utf-8",
        )

    def test_rest_upload_preserves_slashes_and_manifest_last(self) -> None:
        RecordingHandler.recorded = []
        RecordingHandler.mode = "rest"
        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), RecordingHandler)
        port = server.server_address[1]
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory() as tmp:
                rel = create_sample_release(Path(tmp))
                old_env = dict(os.environ)
                try:
                    os.environ.pop("R2_ACCESS_KEY_ID", None)
                    os.environ.pop("R2_SECRET_ACCESS_KEY", None)
                    os.environ["CLOUDFLARE_API_TOKEN"] = "tok_test"
                    os.environ["CLOUDFLARE_ACCOUNT_ID"] = "acct123"
                    os.environ["CLOUDFLARE_API_BASE"] = f"http://127.0.0.1:{port}"
                    rc = r2_upload.main(
                        [
                            "--release-dir",
                            str(rel),
                            "--bucket",
                            "songlib-update",
                            "--prefix",
                            "tv-installer/v1.2.0",
                            "--concurrency",
                            "2",
                        ]
                    )
                    self.assertEqual(rc, 0)
                finally:
                    os.environ.clear()
                    os.environ.update(old_env)
        finally:
            server.shutdown()
            server.server_close()

        paths = [entry[1] for entry in RecordingHandler.recorded]
        self.assertEqual(
            paths[-1],
            "/accounts/acct123/r2/buckets/songlib-update/objects/tv-installer/v1.2.0/manifest.json",
        )
        self.assertTrue(all("%2F" not in p for p in paths))


if __name__ == "__main__":
    unittest.main()
