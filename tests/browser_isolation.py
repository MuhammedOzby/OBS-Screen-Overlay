"""Exercise browser test server setup without Playwright: python tests/browser_isolation.py.

This starts real local servers; browser interactions are deliberately stubbed.
"""
import importlib.util, json, os, sys, types, urllib.request
from pathlib import Path
fake = types.ModuleType('playwright.sync_api')
fake.expect = lambda *a: None
class Browser:
 def close(self): pass
class Manager:
 def __enter__(self):
  return types.SimpleNamespace(chromium=types.SimpleNamespace(launch=lambda **kw: Browser()))
 def __exit__(self, *args): pass
fake.sync_playwright = Manager
sys.modules['playwright'] = types.ModuleType('playwright')
sys.modules['playwright.sync_api'] = fake
root = Path(__file__).resolve().parents[1]
os.environ.update(ADDR='127.0.0.1:1', PUBLIC_ADDR='127.0.0.1:2', PUBLIC_ORIGIN='http://127.0.0.1:2', DB_PATH='/invalid/live.db')
for name in ('browser_access', 'browser_refactor'):
 spec = importlib.util.spec_from_file_location(name, root / 'tests' / (name + '.py'))
 mod = importlib.util.module_from_spec(spec); spec.loader.exec_module(mod)
 original_popen = mod.subprocess.Popen
 def checked_popen(*args, **kwargs):
  if args[0][0] == "go":
   return original_popen(*args, **kwargs)
  work = Path(kwargs["cwd"])
  assert work != root and work.is_dir()
  config = json.loads((work / "configuration.yaml").read_text())
  assert Path(config["db_path"]) == work / "test.db"
  assert config["addr"].startswith("127.0.0.1:") and config["public_addr"].startswith("127.0.0.1:")
  assert config["addr"] != config["public_addr"]
  assert (work / "template" / "access.html.tmpl").is_file()
  return original_popen(*args, **kwargs)
 mod.subprocess.Popen = checked_popen
 if name == 'browser_access':
  def check(browser, local, remote, public, stop, start):
   assert urllib.request.urlopen(local + '/access').status == 200
   stop(); start()
   assert urllib.request.urlopen(local + '/access').status == 200
  mod.check = check
 else:
  def check_browser(admin):
   assert urllib.request.urlopen(admin + '/access').status == 200
  mod.check_browser = check_browser
 try:
  mod.run()
 finally:
  mod.subprocess.Popen = original_popen
 print('PASS', name, 'isolated startup' + (' and restart' if name == 'browser_access' else ''))
