"""Exercise actual fnOS script ordering without touching a NAS."""
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent / "songlibtvinstaller"


class Lifecycle(unittest.TestCase):
    def run_script(self, name, env):
        return subprocess.run(["sh", str(ROOT / "cmd" / name)], env=env, capture_output=True, text=True)

    def test_fresh_install_before_data_exists_and_upgrade(self):
        with tempfile.TemporaryDirectory() as tmp:
            temp = Path(tmp)
            var = temp / "not-created-yet"
            dest = temp / "app"
            shutil.copytree(ROOT / "app", dest)
            password = "音乐小岛 #$!123"
            env = dict(
                os.environ,
                TRIM_PKGVAR=str(var),
                TRIM_APPDEST=str(dest),
                TRIM_UID=str(os.getuid()),
                TRIM_GID=str(os.getgid()),
                wizard_port="16533",
                wizard_password=password,
            )
            # fnOS validates compose before assigning/creating the application data directory.
            r = subprocess.run(
                ["docker", "compose", "-f", str(dest / "docker/docker-compose.yaml"), "config", "--quiet"],
                env=env,
                capture_output=True,
                text=True,
            )
            self.assertEqual(r.returncode, 0, r.stderr)
            self.assertFalse(var.exists())
            self.assertEqual(self.run_script("validate-install", env).returncode, 0)
            self.assertFalse(var.exists(), "install_init must not write private data")

            r = self.run_script("prepare", env)
            self.assertEqual(r.returncode, 0, r.stderr)
            initial = var / "data/keys/initial-password"
            self.assertEqual(initial.read_text(), password)
            self.assertEqual(initial.stat().st_mode & 0o777, 0o600)

            compose = (dest / "docker/docker-compose.yaml").read_text()
            self.assertNotIn("env_file", compose)
            self.assertNotIn(password, compose)
            self.assertNotIn("<hosts>", compose)
            self.assertIn('"16533:80"', compose)
            self.assertIn('SONGLIB_DEPLOY_MODE: "fnos"', compose)
            # app/ui/config keeps literal ${wizard_port} so fnOS substitutes it at registration/update time.
            self.assertEqual(
                json.loads((dest / "ui/config").read_text())[".url"]["songlibtvinstaller.Application"]["port"],
                "${wizard_port}",
            )

            r = subprocess.run(
                ["docker", "compose", "-f", str(dest / "docker/docker-compose.yaml"), "config", "--quiet"],
                env=env,
                capture_output=True,
                text=True,
            )
            self.assertEqual(r.returncode, 0, r.stderr)

            # Simulate upgrade where wizard_* env vars are absent and existing password hash + pairing data exist.
            initial.unlink()
            saved = var / "data/keys/management-password.hash"
            saved.write_bytes(b"preserve-existing-hash")
            marker = var / "data/pairing-record"
            marker.write_bytes(b"preserve-pairing")
            shutil.rmtree(dest)
            shutil.copytree(ROOT / "app", dest)
            upgrade = {k: v for k, v in env.items() if not k.startswith("wizard_")}
            r = self.run_script("prepare", upgrade)
            self.assertEqual(r.returncode, 0, r.stderr)
            self.assertEqual(saved.read_bytes(), b"preserve-existing-hash")
            self.assertEqual(marker.read_bytes(), b"preserve-pairing")
            self.assertFalse(initial.exists())
            self.assertIn('"16533:80"', (dest / "docker/docker-compose.yaml").read_text())

    def test_missing_wizard_values_fall_back_to_15533_and_default_password(self):
        with tempfile.TemporaryDirectory() as tmp:
            temp = Path(tmp)
            var = temp / "var"
            dest = temp / "app"
            shutil.copytree(ROOT / "app", dest)
            env = {k: v for k, v in os.environ.items() if not k.startswith("wizard_")}
            env.update(TRIM_PKGVAR=str(var), TRIM_APPDEST=str(dest))

            r = self.run_script("prepare", env)
            self.assertEqual(r.returncode, 0, r.stderr)
            self.assertEqual((var / "port").read_text().strip(), "15533")
            initial = var / "data/keys/initial-password"
            self.assertEqual(initial.read_text(), "songlib123")
            self.assertEqual(initial.stat().st_mode & 0o777, 0o600)
            compose = (dest / "docker/docker-compose.yaml").read_text()
            self.assertIn('"15533:80"', compose)
            self.assertIn('SONGLIB_DEPLOY_MODE: "fnos"', compose)

    def test_config_callback_updates_port_and_password_and_preserves_on_empty(self):
        with tempfile.TemporaryDirectory() as tmp:
            temp = Path(tmp)
            var = temp / "var"
            dest = temp / "app"
            shutil.copytree(ROOT / "app", dest)
            base_env = {k: v for k, v in os.environ.items() if not k.startswith("wizard_")}
            base_env.update(TRIM_PKGVAR=str(var), TRIM_APPDEST=str(dest))

            r = self.run_script("prepare", dict(base_env, wizard_port="16533", wizard_password="initpass1"))
            self.assertEqual(r.returncode, 0, r.stderr)
            (var / "data/keys/initial-password").unlink()
            saved = var / "data/keys/management-password.hash"
            saved.write_bytes(b"existing-hash")

            # Empty wizard values keep current port and existing hash untouched.
            r = self.run_script("config_callback", dict(base_env, wizard_port="", wizard_password=""))
            self.assertEqual(r.returncode, 0, r.stderr)
            self.assertEqual(saved.read_bytes(), b"existing-hash")
            self.assertEqual((var / "port").read_text().strip(), "16533")

            # Changing port and UTF-8 password updates compose, writes new initial-password, and removes stale hash.
            new_pw = "音屿新密码123"
            r = self.run_script("config_callback", dict(base_env, wizard_port="16544", wizard_password=new_pw))
            self.assertEqual(r.returncode, 0, r.stderr)
            self.assertEqual((var / "port").read_text().strip(), "16544")
            self.assertFalse(saved.exists())
            initial = var / "data/keys/initial-password"
            self.assertEqual(initial.read_text(), new_pw)
            self.assertEqual(initial.stat().st_mode & 0o777, 0o600)
            self.assertIn('"16544:80"', (dest / "docker/docker-compose.yaml").read_text())

    def test_password_validation_accepts_simple_eight_and_punctuation(self):
        for value in ["12345678", "pass#$!8", "eight 88", "音乐小岛1234"]:
            r = self.run_script("validate-install", dict(os.environ, wizard_password=value, wizard_port="16533"))
            self.assertEqual(r.returncode, 0, r.stderr)
        # Missing wizard_port defaults to 15533 (not 5533).
        env_no_port = {k: v for k, v in os.environ.items() if not k.startswith("wizard_")}
        env_no_port["wizard_password"] = "12345678"
        r = self.run_script("validate-install", env_no_port)
        self.assertEqual(r.returncode, 0, r.stderr)
        r = self.run_script("validate-install", dict(os.environ, wizard_password="1234567", wizard_port="16533"))
        self.assertEqual(r.returncode, 1)


if __name__ == "__main__":
    unittest.main()
