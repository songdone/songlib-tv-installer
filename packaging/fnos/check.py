#!/usr/bin/env python3
"""Validate source metadata and brand assets, not a substitute for fnOS device acceptance."""
from pathlib import Path
import json
from PIL import Image

root = Path(__file__).parent / "songlibtvinstaller"
for name, size in [
    ("ICON.PNG", 256),
    ("ICON_256.PNG", 256),
    ("ICON_512.PNG", 512),
    ("app/ui/images/icon_64.png", 64),
    ("app/ui/images/icon_256.png", 256),
    ("app/ui/images/icon_512.png", 512),
    ("../../../marketing/icon_512.png", 512),
    ("../../../branding/ICON.PNG", 256),
]:
    im = Image.open(root / name)
    assert im.format == "PNG" and im.size == (size, size), (name, im.size)
    assert im.mode == "RGBA" and im.getpixel((0, 0))[3] == 0 and im.getpixel((size // 2, size // 2))[3] == 255, (
        name,
        "rounded alpha",
    )
    assert im.info.get("icc_profile"), (name, "sRGB profile")

for name in [
    "config/privilege",
    "config/resource",
    "app/ui/config",
    "wizard/install",
    "wizard/config",
    "wizard/uninstall",
]:
    json.loads((root / name).read_text())

compose = (root / "app/docker/docker-compose.yaml").read_text()
compose_tpl = (root / "app/docker/docker-compose.yaml.template").read_text()
for label, body in [("docker-compose.yaml", compose), ("docker-compose.yaml.template", compose_tpl)]:
    assert "privileged:" not in body and "seccomp:unconfined" not in body and "docker.sock" not in body, label
    assert 'SONGLIB_DEPLOY_MODE: "fnos"' in body, label
    assert "${wizard_port:-15533}:80" in body, label
    assert "<hosts>" not in body, label

assert "音屿-Apple TV音乐播放器安装助手" in (root / "manifest").read_text()
print("PASS: icons, metadata, JSON, minimum container privileges")

policy = json.loads((Path(__file__).parent / "release-policy.json").read_text())
manifest = dict(line.split("=", 1) for line in (root / "manifest").read_text().splitlines() if "=" in line)
manifest = {k.strip(): v.strip() for k, v in manifest.items()}
for key in ["maintainer", "distributor", "distributor_url", "platform", "version"]:
    assert manifest[key] == policy[key], (key, manifest.get(key), policy.get(key))
assert manifest["service_port"] == "15533", manifest.get("service_port")
assert "image: " + policy["image_repository"] + ":" in compose
assert "image: " + policy["image_repository"] + ":" in compose_tpl
assert "127.0.0.1:15933" not in compose and "pull_policy: never" not in compose
print("PASS: 6uos publisher, platform=all, version parity, DockerHub image, no bundled image source")
