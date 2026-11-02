#!/usr/bin/env python3
"""Validate source metadata and brand assets, not a substitute for fnOS device acceptance."""
from pathlib import Path
import json
from PIL import Image
root=Path(__file__).parent/'songlibtvinstaller'
for name,size in [('ICON.PNG',256),('ICON_256.PNG',256),('ICON_512.PNG',512),('app/ui/images/icon_64.png',64),('app/ui/images/icon_256.png',256),('../../../marketing/icon_512.png',512),('../../../branding/ICON.PNG',256)]:
 im=Image.open(root/name);assert im.format=='PNG' and im.size==(size,size),(name,im.size)
 assert im.mode=='RGBA' and im.getpixel((0,0))[3]==0 and im.getpixel((size//2,size//2))[3]==255,(name,'rounded alpha')
 assert im.info.get('icc_profile'),(name,'sRGB profile')
for name in ['config/privilege','config/resource','app/ui/config','wizard/install','wizard/uninstall']:json.loads((root/name).read_text())
s=(root/'app/docker/docker-compose.yaml').read_text();assert 'privileged:' not in s and 'seccomp:unconfined' not in s and 'docker.sock' not in s
assert '音屿-Apple TV音乐播放器安装助手' in (root/'manifest').read_text()
print('PASS: icons, metadata, JSON, minimum container privileges')

policy=json.loads((Path(__file__).parent/'release-policy.json').read_text())
manifest=dict(line.split('=',1) for line in (root/'manifest').read_text().splitlines() if '=' in line)
manifest={k.strip():v.strip() for k,v in manifest.items()}
for key in ['maintainer','distributor','distributor_url','platform']:assert manifest[key]==policy[key],key
assert 'image: '+policy['image_repository']+':' in s
assert '127.0.0.1:15933' not in s and 'pull_policy: never' not in s
print('PASS: 6uos publisher, platform=all, DockerHub image, no bundled image source')
