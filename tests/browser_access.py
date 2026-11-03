"""Real Chrome access acceptance; isolated loopback listeners and SQLite database.

overlay.test is mapped to loopback: this simulates distinct hosts, not a LAN.
"""
import base64
import os
from pathlib import Path
import re
import socket
import shutil
import json
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
from urllib.parse import urlsplit

from playwright.sync_api import expect, sync_playwright

ROOT = Path(__file__).resolve().parents[1]


def port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def run():
    with tempfile.TemporaryDirectory(prefix="obs-access-") as tmp:
        work = Path(tmp)
        binary = work / "app"
        subprocess.run(["go", "build", "-o", str(binary), "."], cwd=ROOT, check=True)
        admin_port, public_port = port(), port()
        while public_port == admin_port:
            public_port = port()
        local = f"http://127.0.0.1:{admin_port}"
        remote = f"http://overlay.test:{admin_port}"
        public = f"http://127.0.0.1:{public_port}"
        shutil.copytree(ROOT / "template", work / "template")
        (work / "configuration.yaml").write_text(
            json.dumps({"addr": local[7:], "public_addr": public[7:],
                        "public_origin": public, "admin_origins": [local, remote],
                        "db_path": str(work / "test.db")}), encoding="utf-8")
        env = dict(os.environ, GIN_MODE="release")
        proc = None
        with (work / "server.log").open("w+") as log:
            def start():
                nonlocal proc
                proc = subprocess.Popen([str(binary)], cwd=work, env=env, stdout=log, stderr=log)
                for _ in range(100):
                    try:
                        with urllib.request.urlopen(local + "/access", timeout=1):
                            return
                    except (OSError, urllib.error.URLError):
                        if proc.poll() is not None:
                            log.flush()
                            log.seek(0)
                            raise RuntimeError(f"Server exited during startup: {log.read()}")
                        time.sleep(.05)
                raise RuntimeError("Server startup timed out")

            def stop():
                if proc and proc.poll() is None:
                    proc.terminate()
                    try:
                        proc.wait(timeout=8)
                    except subprocess.TimeoutExpired:
                        proc.kill()
                        proc.wait()

            try:
                start()
                with sync_playwright() as p:
                    browser = p.chromium.launch(executable_path=os.environ.get("CHROME_BIN", "/opt/google/chrome/chrome"),
                        headless=True, args=["--no-sandbox", "--host-resolver-rules=MAP overlay.test 127.0.0.1", "--no-proxy-server"])
                    try:
                        check(browser, local, remote, public, stop, start)
                    finally:
                        browser.close()
            finally:
                stop()


def check(browser, local, remote, public, stop, start):
    def remote_fetch(page, path, method="GET", body=None, headers=None):
        return page.evaluate("""async ({path, method, body, headers}) => {
          const response = await fetch(path, {method, headers, credentials: 'same-origin',
            ...(body === null ? {} : {body: JSON.stringify(body)})});
          return {status: response.status};
        }""", {"path": path, "method": method, "body": body,
                 "headers": {**({"Content-Type": "application/json"} if body is not None else {}), **(headers or {})}})

    marker = {"X-Overlay-Request": "1"}
    local_ctx = browser.new_context(viewport={"width": 390, "height": 844})
    anon = browser.new_context()
    assert anon.request.get(local + "/api/pages").status == 401
    assert anon.request.post(local + "/api/pages", data={}, headers={"Origin": local}).status == 401
    redirect = anon.request.get(local + "/yeni", max_redirects=0)
    assert redirect.status in (302, 303, 307, 308) and "/access" in redirect.headers["location"]
    anon_page = anon.new_page()
    anon_page.goto(remote + "/access")
    assert remote_fetch(anon_page, "/access/local", "POST", {}, marker)["status"] == 403

    page = local_ctx.new_page()
    page.goto(local + "/access")
    page.get_by_role("button", name="Bu bilgisayarda paneli aç").click()
    expect(page).not_to_have_url(re.compile(r"/access(?:$|[?#])"))
    session = local_ctx.request.get(local + "/access/session").json()
    assert session["local"] and remote in session["admin_origins"]
    csrf = {"Origin": local, "X-CSRF-Token": session["csrf_token"]}
    page.goto(local + "/cihazlar")
    expect(page.locator('#pair-origin option[value="' + remote + '"]')).to_have_count(1)
    page.locator("#pair-origin").select_option(remote)

    # Capture the actual response produced by the UI, rather than synthesizing a challenge.
    def create_challenge():
        with page.expect_response(lambda r: r.url.endswith("/access/pairing") and r.request.method == "POST") as result:
            page.locator("#create-pairing").click()
        response = result.value
        assert response.status == 200
        data = response.json()
        assert re.fullmatch(r"\d{6}", data["code"])
        assert data["expires_at"] and data["qr_token"]
        assert data["qr_image"].startswith("data:image/png;base64,")
        assert base64.b64decode(data["qr_image"].split(",", 1)[1], validate=True).startswith(b"\x89PNG\r\n\x1a\n")
        assert urlsplit(data["qr_url"]).fragment and data["qr_url"].startswith(remote + "/access#")
        return data

    challenge = create_challenge()
    device = browser.new_context()
    secondary = device.new_page()
    secondary.goto(remote + "/access")
    assert challenge["code"] not in secondary.content() and challenge["qr_token"] not in secondary.content()
    secondary.locator('input[name="device_name"], #device-name').first.fill("Browser code device")
    secondary.locator('input[name="code"], #pair-code, #code').first.fill(challenge["code"])
    secondary.get_by_role("button", name=re.compile("eşleştir|bağlan|giriş", re.I)).first.click()
    expect(secondary).not_to_have_url(re.compile(r"/access(?:$|[?#])"))
    cookies = device.cookies(remote)
    assert any(c["httpOnly"] and c["expires"] > time.time() and c["domain"] == "overlay.test" for c in cookies)
    state = device.storage_state()
    reopened = browser.new_context(storage_state=state)
    reopened_page = reopened.new_page()
    reopened_page.goto(remote + "/")
    assert remote_fetch(reopened_page, "/access/session")["status"] == 200
    stop()
    start()
    assert remote_fetch(reopened_page, "/access/session")["status"] == 200
    reuse = remote_fetch(anon_page, "/access/pair", "POST", {"code": challenge["code"], "device_name": "reuse"}, marker)
    assert reuse["status"] == 403

    challenge = create_challenge()
    qrctx = browser.new_context()
    qrpage = qrctx.new_page()
    requests = []
    qr_cdp = qrctx.new_cdp_session(qrpage)
    qr_cdp.send("Network.enable")
    qr_cdp.on("Network.requestWillBeSent", lambda event: requests.append(event["request"]["url"]))
    qrpage.goto(challenge["qr_url"])
    expect(qrpage).not_to_have_url(re.compile(r"#"))
    expect(qrpage).not_to_have_url(re.compile(r"/access(?:$|[?#])"))
    assert remote_fetch(qrpage, "/access/session")["status"] == 200
    assert requests and all(not urlsplit(url).fragment and challenge["code"] not in url
                            and challenge["qr_token"] not in url for url in requests)
    print("PASS code and QR URL pairing, persistence and one-use challenge; PNG signature checked (not QR decoded)")

    # Upload a real hostile public document and observe the actual browser request.
    page.goto(local + "/yeni")
    page.locator("#record-name").fill("Access isolation")
    html = ("<!doctype html><html><body><p>{{guest}}</p><script>"
            f"fetch({(local + '/api/pages')!r},{{method:'POST',mode:'no-cors',credentials:'include',body:'attack'}}).catch(()=>{{}})"
            "</script></body></html>").encode()
    page.locator("#html-file").set_input_files({"name": "test.html", "mimeType": "text/html", "buffer": html})
    page.locator("#create-button").click()
    page.wait_for_url("**/duzelt/*")
    viewer = local_ctx.new_page()
    cdp = local_ctx.new_cdp_session(viewer)
    cdp.send("Network.enable")
    extra = []
    network = []
    cdp.on("Network.requestWillBeSentExtraInfo", lambda event: extra.append(event))
    cdp.on("Network.requestWillBeSent", lambda event: network.append(event))
    with viewer.expect_response(lambda r: r.url == local + "/api/pages" and r.request.method == "POST") as attack:
        viewer.goto(page.locator("#obs-url").input_value())
    assert attack.value.status == 403
    attack_ids = {event["requestId"] for event in network
                  if event["request"]["url"] == local + "/api/pages" and event["request"]["method"] == "POST"}
    assert any(event["requestId"] in attack_ids and
               any(key.lower() == "cookie" and value for key, value in event["headers"].items())
               for event in extra), "Public-origin attack must carry the admin host cookie on the wire"
    api = local + "/api/pages/" + page.url.rsplit("/", 1)[-1]
    payload = {"name": "Updated", "type": "screen_overlay", "expected_revision": 1, "fields": []}
    assert local_ctx.request.patch(api, data=payload, headers={"Origin": local}).status == 403
    assert local_ctx.request.patch(api, data=payload, headers={"Origin": local, "X-CSRF-Token": "wrong"}).status == 403
    # The panel uses the real CSRF helper and performs the mutation.
    page.locator("#record-name").fill("Updated")
    page.locator("#add-field").click()
    row = page.locator(".field-row").last
    row.locator("input[type=text]").fill("guest")
    row.locator("textarea").fill("Access test live")
    page.locator("#apply-button").click()
    expect(page.locator("#edit-form")).to_have_attribute("data-revision", "2")
    expect(viewer.locator('[data-overlay-field="guest"]')).to_have_text("Access test live")

    devices = local_ctx.request.get(local + "/access/devices").json()["devices"]
    target = next(d for d in devices if d["device_name"] == "Browser code device")
    assert local_ctx.request.post(local + "/access/devices/revoke", data={"id": target["id"]}, headers=csrf).status == 200
    assert remote_fetch(reopened_page, "/access/session")["status"] == 401
    assert local_ctx.request.post(local + "/access/logout", data={}, headers=csrf).status == 200
    assert local_ctx.request.get(local + "/access/session").status == 401
    print("PASS public-origin rejection, CSRF, panel mutation, OBS view, revoke and logout")
    print("Host simulation only; no physical LAN, QR scanning or OBS Browser Source acceptance performed.")


if __name__ == "__main__":
    run()
