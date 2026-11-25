#!/usr/bin/env python3
"""Build one architecture-neutral FPK referencing a published multi-platform image."""
import argparse
import copy
import gzip
import io
import os
import pathlib
import re
import shutil
import stat
import subprocess
import tarfile
import tempfile
from PIL import Image

REQUIRED_ICONS = [
    ("ICON.PNG", 256),
    ("ICON_256.PNG", 256),
    ("ICON_512.PNG", 512),
    ("app/ui/images/icon_64.png", 64),
    ("app/ui/images/icon_256.png", 256),
    ("app/ui/images/icon_512.png", 512),
]


def verify_icons(root: pathlib.Path) -> None:
    """Enforce the 256/256/512 icon gate without mutating source bytes so FPK icon hashes match source."""
    for rel, expected in REQUIRED_ICONS:
        im = Image.open(root / rel)
        if im.format != "PNG" or im.size != (expected, expected):
            raise RuntimeError(f"{rel} must be {expected}x{expected} PNG, got {im.format} {im.size}")
        if im.mode != "RGBA" or im.getpixel((0, 0))[3] != 0 or im.getpixel((expected // 2, expected // 2))[3] != 255:
            raise RuntimeError(f"{rel} must be RGBA with rounded transparent corner and opaque center")
        if not im.info.get("icc_profile"):
            raise RuntimeError(f"{rel} is missing embedded sRGB ICC profile")


def ensure_detail_icon(package: pathlib.Path, icon_path: pathlib.Path) -> None:
    """Ensure fnpack's outer tar archive contains the exact staged ICON_512.PNG without modifying app.tgz."""
    original_mode = stat.S_IMODE(package.stat().st_mode)
    source = gzip.decompress(package.read_bytes())
    with tarfile.open(fileobj=io.BytesIO(source), mode="r:") as archive:
        members = archive.getmembers()
        names = [m.name.rstrip("/") for m in members]
        if len(names) != len(set(names)):
            raise RuntimeError("FPK contains duplicate member paths")
        existing = {m.name.rstrip("/"): m for m in members}
        if "manifest" not in existing or "app.tgz" not in existing:
            raise RuntimeError("FPK is missing required outer members")
        icon_bytes = icon_path.read_bytes()
        contents = {}
        for name, member in existing.items():
            if name.startswith("/") or ".." in pathlib.Path(name).parts:
                raise RuntimeError(f"unsafe FPK member path: {name}")
            if not (member.isfile() or member.isdir()):
                raise RuntimeError(f"unsupported FPK member type: {name}")
            if member.isfile():
                contents[name] = archive.extractfile(member).read()
        current_icon = existing.get("ICON_512.PNG")
        if current_icon is not None:
            if not current_icon.isfile() or contents["ICON_512.PNG"] != icon_bytes:
                raise RuntimeError("FPK ICON_512.PNG exists but does not match staged asset")
            return

        output_tar = io.BytesIO()
        with tarfile.open(fileobj=output_tar, mode="w:", format=tarfile.USTAR_FORMAT) as rebuilt:
            for name, member in existing.items():
                rebuilt.addfile(copy.copy(member), io.BytesIO(contents[name]) if member.isfile() else None)
            icon = tarfile.TarInfo("ICON_512.PNG")
            icon.mode = stat.S_IMODE(icon_path.stat().st_mode)
            icon.size = len(icon_bytes)
            icon.mtime = 0
            rebuilt.addfile(icon, io.BytesIO(icon_bytes))

    rebuilt_bytes = gzip.compress(output_tar.getvalue(), compresslevel=9, mtime=0)
    with tarfile.open(fileobj=io.BytesIO(gzip.decompress(rebuilt_bytes)), mode="r:") as check:
        entries = {m.name.rstrip("/"): m for m in check.getmembers()}
        if "ICON_512.PNG" not in entries or check.extractfile(entries["ICON_512.PNG"]).read() != icon_bytes:
            raise RuntimeError("rebuilt FPK failed ICON_512.PNG verification")
        for name, member in existing.items():
            match = entries.get(name)
            if match is None or member.type != match.type or member.mode != match.mode:
                raise RuntimeError(f"rebuilt FPK changed existing member header: {name}")
            if member.isfile() and contents[name] != check.extractfile(match).read():
                raise RuntimeError(f"rebuilt FPK changed existing member: {name}")

    fd, temporary = tempfile.mkstemp(prefix=package.name + ".", suffix=".tmp", dir=package.parent)
    try:
        with os.fdopen(fd, "wb") as handle:
            handle.write(rebuilt_bytes)
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(temporary, original_mode)
        os.replace(temporary, package)
    except BaseException:
        try:
            os.unlink(temporary)
        except FileNotFoundError:
            pass
        raise


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--fnpack", required=True)
    p.add_argument("--image", required=True)
    p.add_argument("--output", type=pathlib.Path, required=True)
    a = p.parse_args()

    if not re.fullmatch(r"666uos/songlib-tv-installer:[0-9][a-zA-Z0-9.-]*", a.image):
        p.error("Use the published versioned multi-platform DockerHub image")

    src_root = pathlib.Path(__file__).parent / "songlibtvinstaller"
    verify_icons(src_root)

    with tempfile.TemporaryDirectory(prefix="songlib-fpk-") as td:
        stage = pathlib.Path(td) / "songlibtvinstaller"
        shutil.copytree(
            src_root,
            stage,
            ignore=shutil.ignore_patterns(
                ".DS_Store",
                "*.fpk",
                "*.new",
                "offline-image",
                "offline-registry",
                "offline-source",
            ),
        )
        for name in ("docker-compose.yaml", "docker-compose.yaml.template"):
            c = stage / "app/docker" / name
            txt = re.sub(r"^    image:.*$", "    image: " + a.image, c.read_text(), flags=re.M)
            c.write_text(txt)
            if ':80" # songlib:port' not in txt:
                raise RuntimeError(name + " drifted: container port must be :80")
            if "${wizard_port:-15533}" not in txt:
                raise RuntimeError(
                    name + " drifted: default host port must be 15533 (5533 is taken by the store ATVLoadly)"
                )
            if 'SONGLIB_DEPLOY_MODE: "fnos"' not in txt:
                raise RuntimeError(name + ' drifted: must declare SONGLIB_DEPLOY_MODE: "fnos"')
            if "<hosts>" in txt:
                raise RuntimeError(name + " drifted: contains unresolved <hosts> placeholder")

        subprocess.run([str(pathlib.Path(a.fnpack).resolve()), "build"], cwd=stage, check=True)
        a.output.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(stage / "songlibtvinstaller.fpk", a.output)
        ensure_detail_icon(a.output, stage / "ICON_512.PNG")
        if a.output.stat().st_size > 2 * 1024 * 1024:
            raise RuntimeError("Unexpected payload: FPK must not contain image layers or native helpers")
        print(a.output.resolve())


if __name__ == "__main__":
    main()
