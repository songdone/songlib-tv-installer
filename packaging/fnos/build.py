#!/usr/bin/env python3
"""Build a self-contained offline FPK with the official fnpack tool."""
import argparse,pathlib,subprocess,shutil,tempfile,re
from PIL import Image
from export_offline_image import export_image
p=argparse.ArgumentParser();p.add_argument('--fnpack',required=True);p.add_argument('--arch',choices=['x86','arm'],default='x86');p.add_argument('--image',required=True);p.add_argument('--image-archive',type=pathlib.Path);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
if not re.fullmatch(r'[a-z0-9][a-zA-Z0-9./_:@-]+',a.image) or a.image.endswith(':latest'):p.error('Use a fixed source image tag/digest')
repo=pathlib.Path(__file__).resolve().parents[2];arch={'x86':'amd64','arm':'arm64'}[a.arch]
helper=repo/'build'/('offline-registry-linux-'+arch)
if not helper.is_file():p.error('Build the corresponding offline-registry helper first')
with tempfile.TemporaryDirectory(prefix='songlib-offline-fpk-') as td:
 stage=pathlib.Path(td)/'songlibtvinstaller';shutil.copytree(pathlib.Path(__file__).parent/'songlibtvinstaller',stage,ignore=shutil.ignore_patterns('.DS_Store','*.fpk','*.new','offline-image','offline-registry'))
 im=Image.open(stage/'ICON.PNG');im.resize((64,64),Image.Resampling.LANCZOS).save(stage/'ICON.PNG',icc_profile=im.info.get('icc_profile'))
 m=stage/'manifest';m.write_text(m.read_text().replace('platform = x86','platform = '+a.arch))
 archive=a.image_archive
 if archive is None:
  archive=pathlib.Path(td)/'image.tar'
  with archive.open('wb') as out:subprocess.run(['docker','save',a.image],stdout=out,check=True)
 descriptor=export_image(archive,stage/'cmd/offline-image',arch,'0.1.2')
 shutil.copy2(helper,stage/'cmd/offline-registry');(stage/'cmd/offline-registry').chmod(0o755)
 ref=(stage/'cmd/offline-image/image-ref').read_text().strip()
 for name in ['docker-compose.yaml','docker-compose.yaml.template']:
  c=stage/'app/docker'/name;c.write_text(re.sub(r'^    image:.*$','    image: '+ref,c.read_text(),flags=re.M))
 subprocess.run([str(pathlib.Path(a.fnpack).resolve()),'build'],cwd=stage,check=True)
 a.output.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(stage/'songlibtvinstaller.fpk',a.output)
 a.output.with_suffix('.image.json').write_text(__import__('json').dumps(descriptor,indent=2))
 print(a.output.resolve())
