"""Optional real Chromium regression: python tests/browser_refactor.py.

Requires Playwright and Chrome (CHROME_BIN can override its location).
Builds in a temporary directory; never opens the user's overlay.db.
"""

import os
from pathlib import Path
import socket
import shutil
import json
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

from playwright.sync_api import expect, sync_playwright


ROOT = Path(__file__).resolve().parents[1]


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def run():
    with tempfile.TemporaryDirectory(prefix="obs-refactor-") as directory:
        work = Path(directory)
        app = work / "app"
        subprocess.run(["go", "build", "-o", str(app), "."], cwd=ROOT, check=True)
        admin = f"http://127.0.0.1:{free_port()}"
        public = f"http://127.0.0.1:{free_port()}"
        while public == admin:
            public = f"http://127.0.0.1:{free_port()}"
        shutil.copytree(ROOT / "template", work / "template")
        (work / "configuration.yaml").write_text(
            json.dumps({"addr": admin[7:], "public_addr": public[7:],
                        "public_origin": public, "admin_origins": [admin],
                        "db_path": str(work / "test.db")}), encoding="utf-8")
        env = dict(os.environ, GIN_MODE="release")
        with (work / "server.log").open("w+") as log:
            proc = subprocess.Popen([str(app)], cwd=work, env=env, stdout=log, stderr=log)
            try:
                for _ in range(100):
                    try:
                        with urllib.request.urlopen(admin, timeout=1):
                            break
                    except (OSError, urllib.error.URLError):
                        if proc.poll() is not None:
                            log.seek(0)
                            raise RuntimeError(log.read())
                        time.sleep(0.05)
                else:
                    raise RuntimeError("Server startup timed out")
                check_browser(admin)
            finally:
                proc.terminate()
                try:
                    proc.wait(timeout=8)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait()


def check_browser(admin):
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            executable_path=os.environ.get("CHROME_BIN", "/opt/google/chrome/chrome"),
            headless=True, args=["--no-sandbox"])
        try:
            context = browser.new_context()
            bootstrap = context.request.post(admin + "/access/local", data={},
                headers={"Origin": admin, "X-Overlay-Request": "1"})
            assert bootstrap.status == 200, f"Local bootstrap: {bootstrap.status}"
            errors = []
            context.on("page", lambda page: page.on("pageerror", lambda error: errors.append(str(error))))
            panel = context.new_page()
            panel.on("dialog", lambda dialog: dialog.accept())
            panel.goto(admin + "/yeni")
            panel.locator("#record-name").fill("Refactor acceptance")
            original = b"<!doctype html><html><body><p>{{guest}}</p><p>{{guest}}</p><div>{{counter}}</div></body></html>"
            panel.locator("#html-file").set_input_files({"name": "original.html", "mimeType": "text/html", "buffer": original})
            panel.locator("#create-button").click()
            panel.wait_for_url("**/duzelt/*")
            edit_url = panel.url
            for name, value, enabled in [("guest", "Saved guest", True), ("counter", "300", False)]:
                panel.locator("#add-field").click()
                row = panel.locator(".field-row").last
                row.locator("input[type=text]").fill(name)
                row.locator("textarea").fill(value)
                row.locator("input[type=checkbox]").set_checked(enabled)
            panel.locator("#apply-button").click()
            expect(panel.locator("#edit-form")).to_have_attribute("data-revision", "2")
            viewer = context.new_page()
            viewer.goto(panel.locator("#obs-url").input_value())
            expect(viewer.locator('[data-overlay-field="guest"]')).to_have_text(["Saved guest", "Saved guest"])
            viewer.evaluate("window.documentMarker = 'unchanged'; document.querySelector('[data-overlay-field=counter]').textContent = 'JS running'")
            guest = panel.locator(".field-row").first.locator("textarea")
            guest.fill("Live guest")
            # Hold a real PATCH so both programmatic submits run while it is pending.
            panel.locator("#html-file").set_input_files({"name": "selected.html", "mimeType": "text/html", "buffer": original})
            held_patch = []
            mutation_requests = []
            panel.on("request", lambda request: mutation_requests.append((request.method, request.url))
                     if request.method in ("PATCH", "POST") else None)
            panel.route("**/api/pages/*", lambda route: held_patch.append(route)
                        if route.request.method == "PATCH" else route.continue_())
            panel.locator("#apply-button").click()
            expect(panel.locator("#publish-button")).to_be_disabled()
            panel.locator("#template-form").dispatch_event("submit")
            panel.locator("#edit-form").dispatch_event("submit")
            panel.evaluate("() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))")
            assert len(held_patch) == 1
            assert len(mutation_requests) == 1, mutation_requests
            held_patch[0].continue_()
            panel.unroute("**/api/pages/*")
            expect(panel.locator("#edit-form")).to_have_attribute("data-revision", "3")
            expect(viewer.locator('[data-overlay-field="guest"]')).to_have_text(["Live guest", "Live guest"])
            expect(viewer.locator('[data-overlay-field="counter"]')).to_have_text("JS running")
            assert viewer.evaluate("window.documentMarker") == "unchanged"
            print("PASS create, PATCH, repeated fields, SSE-off DOM preserved without reload")

            expect(panel.locator("#html-file")).to_be_enabled()
            expect(panel.locator("#publish-button")).to_be_enabled()
            guest.fill("Unpublished draft")
            replacement = b'<!doctype html><html><body><h1 id="new-design">New design</h1><p>{{guest}}</p><div>{{counter}}</div></body></html>'
            upload = panel.locator("#html-file")
            upload.set_input_files({"name": "replacement.html", "mimeType": "text/html", "buffer": replacement})
            held_publish = []
            mutation_requests.clear()
            panel.route("**/api/pages/*/template", lambda route: held_publish.append(route))
            panel.locator("#publish-button").click()
            expect(panel.locator("#apply-button")).to_be_disabled()
            panel.locator("#edit-form").dispatch_event("submit")
            panel.locator("#template-form").dispatch_event("submit")
            panel.evaluate("() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))")
            assert len(held_publish) == 1
            assert len(mutation_requests) == 1, mutation_requests
            held_publish[0].continue_()
            panel.unroute("**/api/pages/*/template")
            expect(panel.locator("#edit-form")).to_have_attribute("data-revision", "4")
            expect(viewer.locator("#new-design")).to_have_text("New design")
            expect(viewer.locator('[data-overlay-field="guest"]')).to_have_text("Live guest")
            expect(viewer.locator('[data-overlay-field="counter"]')).to_have_text("300")
            expect(guest).to_have_value("Unpublished draft")
            assert upload.evaluate("element => element.files.length") == 0
            print("PASS template publish reloads viewer; does not publish unsaved field draft")
            print("PASS PATCH/publish shared lock blocks duplicate and cross-form submits in both directions")

            upload.set_input_files({"name": "invalid.html", "mimeType": "text/html", "buffer": b"<div>fragment</div>"})
            with panel.expect_response(lambda response: response.url.endswith("/template") and response.request.method == "POST") as result:
                panel.locator("#publish-button").click()
            assert result.value.status == 400
            expect(panel.locator("#publish-button")).to_be_enabled()
            expect(panel.locator("#edit-form")).to_have_attribute("data-revision", "4")
            expect(guest).to_have_value("Unpublished draft")
            assert upload.evaluate("element => element.files[0].name") == "invalid.html"

            other = context.new_page()
            other.goto(edit_url)
            other.locator("#record-name").fill("Other tab update")
            other.locator("#apply-button").click()
            expect(other.locator("#edit-form")).to_have_attribute("data-revision", "5")
            upload.set_input_files({"name": "retry.html", "mimeType": "text/html", "buffer": replacement})
            with panel.expect_response(lambda response: response.url.endswith("/template") and response.request.method == "POST") as conflict:
                panel.locator("#publish-button").click()
            assert conflict.value.status == 409
            expect(panel.locator("#reload-record")).to_be_visible()
            expect(guest).to_have_value("Unpublished draft")
            assert upload.evaluate("element => element.files[0].name") == "retry.html"
            expect(panel.locator("#apply-button")).to_be_enabled()
            expect(panel.locator("#publish-button")).to_be_enabled()
            print("PASS invalid HTML and stale revision preserve draft, file and usable controls")

            panel.route("**/api/pages/*/template", lambda route: route.abort("failed"))
            panel.locator("#publish-button").click()
            expect(panel.locator("#publish-button")).to_be_enabled()
            expect(panel.locator("#apply-button")).to_be_enabled()
            expect(guest).to_have_value("Unpublished draft")
            assert upload.evaluate("element => element.files[0].name") == "retry.html"
            panel.unroute("**/api/pages/*/template")
            print("PASS intentionally aborted publish request preserves draft and retry controls")

            nojs = browser.new_context(java_script_enabled=False, storage_state=context.storage_state())
            disabled_panel = nojs.new_page()
            disabled_panel.goto(edit_url)
            expect(disabled_panel.locator("#apply-button")).to_be_disabled()
            expect(disabled_panel.locator("#publish-button")).to_be_disabled()
            expect(disabled_panel.locator("#html-file")).to_be_disabled()
            nojs.close()
            panel.goto(admin + "/")
            panel.locator(".delete-record").click()
            expect(panel.locator(".records-table tbody tr")).to_have_count(0)
            expect(panel.locator("#empty-records")).to_be_visible()
            panel.reload()
            expect(panel.locator(".records-table tbody tr")).to_have_count(0)
            # Chrome may reject navigation to an empty 404 document. Verify the
            # server's exact response rather than treating that as an app error.
            response = context.request.get(viewer.url)
            assert response.status == 404
            print("PASS delete persists after reload and invalidates public page")
            assert not errors, errors
            print("PASS no-JS controls disabled; no browser JavaScript errors. OBS not tested.")
        finally:
            browser.close()


if __name__ == "__main__":
    run()
