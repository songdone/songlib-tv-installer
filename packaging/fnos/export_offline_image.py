"""Convert a Docker save archive to a single-image read-only distribution store."""
import gzip,hashlib,json,pathlib,shutil,tarfile,tempfile

def sha_file(path):
 h=hashlib.sha256()
 with path.open('rb') as src:
  for chunk in iter(lambda:src.read(1024*1024),b''):h.update(chunk)
 return h.hexdigest()

def export_image(archive,dest,expected_arch,tag):
 dest=pathlib.Path(dest);blobs=dest/'blobs/sha256';blobs.mkdir(parents=True,exist_ok=True)
 with tarfile.open(archive,'r:*') as saved:
  entries=json.load(saved.extractfile('manifest.json'))
  if len(entries)!=1:raise ValueError('Archive must contain exactly one image')
  entry=entries[0];config=saved.extractfile(entry['Config']).read();details=json.loads(config)
  if details.get('architecture')!=expected_arch or details.get('os')!='linux':raise ValueError('Image architecture/OS mismatch')
  config_hash=hashlib.sha256(config).hexdigest();(blobs/config_hash).write_bytes(config)
  if len(details.get('rootfs',{}).get('diff_ids',[]))!=len(entry['Layers']):raise ValueError('Layer count mismatch')
  layers=[]
  with tempfile.TemporaryDirectory(dir=dest) as td:
   for n,name in enumerate(entry['Layers']):
    target=pathlib.Path(td)/str(n)
    with saved.extractfile(name) as src,target.open('wb') as dst,gzip.GzipFile(filename='',fileobj=dst,mode='wb',mtime=0,compresslevel=6) as gz:
     raw_hash=hashlib.sha256()
     for chunk in iter(lambda:src.read(1024*1024),b''):
      raw_hash.update(chunk);gz.write(chunk)
     if 'sha256:'+raw_hash.hexdigest()!=details['rootfs']['diff_ids'][n]:raise ValueError('Layer content digest mismatch')
    digest=sha_file(target);size=target.stat().st_size;target.replace(blobs/digest)
    layers.append({'mediaType':'application/vnd.docker.image.rootfs.diff.tar.gzip','digest':'sha256:'+digest,'size':size})
 manifest={'schemaVersion':2,'mediaType':'application/vnd.docker.distribution.manifest.v2+json','config':{'mediaType':'application/vnd.docker.container.image.v1+json','digest':'sha256:'+config_hash,'size':len(config)},'layers':layers}
 raw=json.dumps(manifest,separators=(',',':')).encode();(dest/'manifest.json').write_bytes(raw)
 descriptor={'repository':'songlib-tv-installer','tag':tag,'manifest_digest':'sha256:'+hashlib.sha256(raw).hexdigest(),'architecture':expected_arch}
 port=19000+int(descriptor['manifest_digest'][7:11],16)%40000
 descriptor['listen_port']=port
 (dest/'listen-port').write_text(str(port)+'\n')
 (dest/'architecture').write_text(expected_arch+'\n')
 (dest/'image-ref').write_text('127.0.0.1:'+str(port)+'/songlib-tv-installer@'+descriptor['manifest_digest']+'\n')
 (dest/'descriptor.json').write_text(json.dumps(descriptor,indent=2));(dest/'manifest-digest').write_text(descriptor['manifest_digest']+'\n')
 return descriptor

if __name__=='__main__':
 import argparse
 p=argparse.ArgumentParser();p.add_argument('archive');p.add_argument('destination');p.add_argument('--arch',required=True);p.add_argument('--tag',required=True);a=p.parse_args()
 print(json.dumps(export_image(a.archive,a.destination,a.arch,a.tag),indent=2))
