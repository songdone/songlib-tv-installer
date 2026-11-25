#!/usr/bin/env python3
"""Build one architecture-neutral FPK referencing a published multi-platform image."""
import argparse
import pathlib
import re
import shutil
import subprocess
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
        if a.output.stat().st_size > 2 * 1024 * 1024:
            raise RuntimeError("Unexpected payload: FPK must not contain image layers or native helpers")
        print(a.output.resolve())


if __name__ == "__main__":
    main()
