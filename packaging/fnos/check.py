#!/usr/bin/env python3
"""Validate source metadata and brand assets, not a substitute for fnOS device acceptance."""
from pathlib import Path
import json
from PIL import Image
root=Path(__file__).parent/'songlibtvinstaller'
for name,size in [('ICON.PNG',64),('ICON_256.PNG',256),('app/ui/images/icon_64.png',64),('app/ui/images/icon_256.png',256),('../../../marketing/icon_512.png',512),('../../../branding/ICON.PNG',256)]:
 im=Image.open(root/name);assert im.format=='PNG' and im.size==(size,size),(name,im.size)
for name in ['config/privilege','config/resource','app/ui/config','wizard/install','wizard/uninstall']:json.loads((root/name).read_text())
s=(root/'app/docker/docker-compose.yaml').read_text();assert 'privileged:' not in s and 'seccomp:unconfined' not in s and 'docker.sock' not in s
assert '音屿-Apple TV音乐播放器安装助手' in (root/'manifest').read_text()
print('PASS: icons, metadata, JSON, minimum container privileges')
