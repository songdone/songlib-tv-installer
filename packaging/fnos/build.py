#!/usr/bin/env python3
"""Build one architecture-neutral FPK referencing a published multi-platform image."""
import argparse,pathlib,subprocess,shutil,tempfile,re
from PIL import Image
p=argparse.ArgumentParser();p.add_argument('--fnpack',required=True);p.add_argument('--image',required=True);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
if not re.fullmatch(r'666uos/songlib-tv-installer:[0-9][a-zA-Z0-9.-]*',a.image):p.error('Use the published versioned multi-platform DockerHub image')
with tempfile.TemporaryDirectory(prefix='songlib-fpk-') as td:
 stage=pathlib.Path(td)/'songlibtvinstaller';shutil.copytree(pathlib.Path(__file__).parent/'songlibtvinstaller',stage,ignore=shutil.ignore_patterns('.DS_Store','*.fpk','*.new','offline-image','offline-registry','offline-source'))
 # Icons: fnOS's stock ICON.PNG is 64x64 and blurs on the detail page. The
 # product spec is 256x256 for the app-center grid and 512x512 for the detail
 # page; regenerate the whole set from the rounded 512 master in-package.
 im=Image.open(stage/'ICON_512.PNG').convert('RGBA')
 im.save(stage/'ICON.PNG')
 im.resize((256,256),Image.Resampling.LANCZOS).save(stage/'ICON_256.PNG')
 im.save(stage/'ICON_512.PNG')
 for s in (64,256,512):
  m=im if s==512 else im.resize((s,s),Image.Resampling.LANCZOS)
  m.save(stage/'app/ui/images'/f'icon_{s}.png')
 for name in ['docker-compose.yaml','docker-compose.yaml.template']:
  c=stage/'app/docker'/name;c.write_text(re.sub(r'^    image:.*$','    image: '+a.image,c.read_text(),flags=re.M))
 subprocess.run([str(pathlib.Path(a.fnpack).resolve()),'build'],cwd=stage,check=True)
 a.output.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(stage/'songlibtvinstaller.fpk',a.output)
 if a.output.stat().st_size>2*1024*1024:raise RuntimeError('Unexpected payload: FPK must not contain image layers or native helpers')
 print(a.output.resolve())
