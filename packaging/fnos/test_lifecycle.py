"""Exercise actual fnOS script ordering without touching a NAS."""
import json,os,shutil,subprocess,tempfile,unittest
from pathlib import Path
ROOT=Path(__file__).resolve().parent/'songlibtvinstaller'
class Lifecycle(unittest.TestCase):
 def run_script(self,name,env):
  return subprocess.run(['sh',str(ROOT/'cmd'/name)],env=env,capture_output=True,text=True)
 def test_fresh_install_before_data_exists_and_upgrade(self):
  with tempfile.TemporaryDirectory() as tmp:
   temp=Path(tmp);var=temp/'not-created-yet';dest=temp/'app';shutil.copytree(ROOT/'app',dest)
   password='音乐小岛 #$!123'
   env=dict(os.environ,TRIM_PKGVAR=str(var),TRIM_APPDEST=str(dest),wizard_port='16533',wizard_password=password)
   # fnOS validates compose before assigning/creating the application data directory.
   r=subprocess.run(['docker','compose','-f',str(dest/'docker/docker-compose.yaml'),'config','--quiet'],env=env,capture_output=True,text=True)
   self.assertEqual(r.returncode,0,r.stderr);self.assertFalse(var.exists())
   self.assertEqual(self.run_script('validate-install',env).returncode,0)
   self.assertFalse(var.exists(),'install_init must not write private data')
   r=self.run_script('prepare',env);self.assertEqual(r.returncode,0,r.stderr)
   initial=var/'data/keys/initial-password';self.assertEqual(initial.read_text(),password)
   self.assertEqual(initial.stat().st_mode&0o777,0o600)
   compose=(dest/'docker/docker-compose.yaml').read_text();self.assertNotIn('env_file',compose);self.assertNotIn(password,compose)
   self.assertIn('16533:5533',compose);self.assertIn(f'{os.getuid()}:{os.getgid()}',compose)
   self.assertEqual(json.loads((dest/'ui/config').read_text())['.url']['songlibtvinstaller.Application']['port'],'16533')
   r=subprocess.run(['docker','compose','-f',str(dest/'docker/docker-compose.yaml'),'config','--quiet'],env=env,capture_output=True,text=True);self.assertEqual(r.returncode,0,r.stderr)
   initial.unlink();saved=var/'data/keys/management-password.hash';saved.write_bytes(b'preserve-existing-hash')
   marker=var/'data/pairing-record';marker.write_bytes(b'preserve-pairing')
   shutil.rmtree(dest);shutil.copytree(ROOT/'app',dest)
   upgrade={k:v for k,v in env.items() if not k.startswith('wizard_')}
   r=self.run_script('prepare',upgrade);self.assertEqual(r.returncode,0,r.stderr)
   self.assertEqual(saved.read_bytes(),b'preserve-existing-hash');self.assertEqual(marker.read_bytes(),b'preserve-pairing');self.assertFalse(initial.exists())
   self.assertIn('16533:5533',(dest/'docker/docker-compose.yaml').read_text())
 def test_password_validation_accepts_simple_eight_and_punctuation(self):
  for value in ['12345678','pass#$!8','eight 88']:
   r=self.run_script('validate-install',dict(os.environ,wizard_password=value,wizard_port='16533'))
   self.assertEqual(r.returncode,0,r.stderr)
  r=self.run_script('validate-install',dict(os.environ,wizard_password='1234567',wizard_port='16533'))
  self.assertEqual(r.returncode,1)
if __name__=='__main__':unittest.main()
