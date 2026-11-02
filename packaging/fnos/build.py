#!/usr/bin/env python3
"""Official fnpack build. Local acceptance package needs preloaded image; never publishes."""
import argparse,pathlib,subprocess,shutil,tempfile,re
from PIL import Image
p=argparse.ArgumentParser();p.add_argument('--fnpack',required=True);p.add_argument('--arch',choices=['x86','arm'],default='x86');p.add_argument('--image',required=True);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
if not re.fullmatch(r'[a-z0-9][a-zA-Z0-9./_:@-]+',a.image) or a.image.endswith(':latest'):p.error('Use a fixed image tag/digest')
with tempfile.TemporaryDirectory() as td:
 stage=pathlib.Path(td)/'songlibtvinstaller';shutil.copytree(pathlib.Path(__file__).parent/'songlibtvinstaller',stage,ignore=shutil.ignore_patterns('.DS_Store','*.fpk','*.new'))
 # Official ICON.PNG compatibility thumbnail; 256/512 masters remain in source and package assets.
 im=Image.open(stage/'ICON.PNG');im.resize((64,64),Image.Resampling.LANCZOS).save(stage/'ICON.PNG',icc_profile=im.info.get('icc_profile'))
 m=stage/'manifest';m.write_text(m.read_text().replace('platform = x86','platform = '+a.arch))
 c=stage/'app/docker/docker-compose.yaml';c.write_text(c.read_text().replace('songlib-tv-installer:0.1.1',a.image))
 (stage/'app/docker/docker-compose.yaml.template').write_text(c.read_text())
 subprocess.run([str(pathlib.Path(a.fnpack).resolve()),'build'],cwd=stage,check=True)
 a.output.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(stage/'songlibtvinstaller.fpk',a.output)
 print(a.output.resolve())
