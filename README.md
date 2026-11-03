# OBS Screen Overlay

> **AI-assisted development:** This project was developed with the help of AI tools.

**English** | [Türkçe](README-TR.md)

A Go and SQLite application for managing single-file HTML overlays and widgets for OBS Browser Source. Upload an HTML document, edit its text fields in the Turkish-language management panel, and apply live changes using Server-Sent Events (SSE).

Each overlay or widget has its own read-only OBS URL. There is no account/password flow: devices are authorized through local access or QR/code pairing. This is a separate application from OBS LAN Controller and does not require its live preview plugin.

## Features and limits

- Upload one complete UTF-8 `.html` file, up to 2 MiB. CSS, JavaScript, and SVG may be embedded in that file.
- Use placeholders such as `{{guest_name}}` in supported HTML text nodes. Edit values and per-field live-update settings in the panel.
- **Apply to broadcast (Yayına uygula)** saves values and live-update settings together. SSE-enabled fields update without reloading the OBS page.
- With SSE disabled, a field keeps its current DOM content, including changes made by your own JavaScript. Saved values take effect on a full page reload; enabling SSE applies the current saved value.
- **Publish design (Tasarımı yayınla)** publishes replacement HTML separately and refreshes open overlay pages.
- Store HTML, field settings, and device sessions in SQLite.

There is no built-in timer engine, drag-and-drop designer, asset manager, ZIP upload, or separate image/font upload flow. Upload only HTML/JavaScript you trust. This release targets local/trusted LAN use; physical LAN, phone-camera QR scanning, and real OBS Browser Source acceptance remain open in the recorded test plan.

## Requirements

- **Go 1.27.3 or later**, as declared in `go.mod`.
- **CGO enabled and GCC available** for `github.com/mattn/go-sqlite3`. Check with `go env CGO_ENABLED`; it should print `1`.
- Run from the project root. `configuration.yaml`, `template/`, and its assets are read from the working directory. The compiled executable alone is not a complete distribution.
- Node.js is optional for JavaScript tests. Python, Playwright, and Chrome are optional for browser tests, not application runtime dependencies.

## Download and run

```sh
git clone https://github.com/MuhammedOzby/OBS-Screen-Overlay.git
cd OBS-Screen-Overlay
go run .
```

Or build and run from the same directory:

```sh
go build -o obs-screen-overlay .
./obs-screen-overlay
```

1. On the server computer, open `http://127.0.0.1:8080/access` and select **Open panel on this computer (Bu bilgisayarda paneli aç)**.
2. Create an overlay or widget and upload your HTML file. A minimal example is included at `example/template.html`.
3. Add/edit variable names and values on the edit page, then select **Apply to broadcast (Yayına uygula)**.
4. Copy the OBS URL from the panel and paste it into an OBS **Browser Source**. Set the source size to match your design.

The management panel listens on `http://127.0.0.1:8080`. Overlay pages use the separate origin `http://127.0.0.1:8081`. For a worked example, see [example/README.md](example/README.md) (Turkish).

## Configuration

The application requires `configuration.yaml` in its working directory. Application settings are **not** read from environment variables.

The included file uses loopback defaults:

```yaml
addr: "127.0.0.1:8080"
public_addr: "127.0.0.1:8081"
public_origin: "http://127.0.0.1:8081"
admin_origins: []
db_path: "overlay.db"
```

- `addr` and `public_addr`: management and overlay listening addresses.
- `public_origin`: the externally reachable overlay origin used to generate OBS URLs; do not include a path.
- `admin_origins`: complete allowed management origins, without paths. An empty list with a loopback bind automatically allows `http://127.0.0.1:8080` and `http://localhost:8080` at the configured management port. A non-empty list replaces those defaults.
- `db_path`: SQLite path, relative to the working directory unless absolute. The database is created on startup if missing.

Management and overlay origins must differ. Missing, invalid, or unknown settings are rejected at startup. Restart after editing the file.

## Trusted LAN use and device pairing

Use your server's actual LAN IP; `192.168.1.20` is only an example:

```yaml
addr: "0.0.0.0:8080"
public_addr: "0.0.0.0:8081"
public_origin: "http://192.168.1.20:8081"
admin_origins:
  - "http://127.0.0.1:8080"
  - "http://192.168.1.20:8080"
db_path: "overlay.db"
```

A non-loopback bind requires an explicit, non-empty `admin_origins` list. `0.0.0.0` is a listening address, not a browser destination. Allow the configured TCP ports in your firewall as needed; the application does not change firewall settings.

1. Restart the application after changing configuration.
2. First authorize local access on the server through `http://127.0.0.1:8080/access`.
3. Open **Devices (/cihazlar)** and generate a QR code and six-digit pairing code for another device.
4. On the same trusted network, scan the QR code or open `http://192.168.1.20:8080/access` and enter the code.
5. OBS uses the overlay URL copied from the panel, on the separate overlay origin.

Pairing codes and QR links are single-use and expire after five minutes. Authorized devices have a persistent cookie lasting 180 days. Revoke a device from the access screen or sign out on that device.

**HTTP LAN traffic is not encrypted.** Codes, sessions, and content may be observed on the network. Use a trusted LAN and do not expose the application to the public internet. HTTPS is recommended, but the application has no built-in HTTPS listener and does not trust forwarded headers. Do not assume TLS-terminating proxy support without separately securing local bootstrap and client-address behavior.

Cookies are not isolated by port. Separate origins work alongside strict Origin/CSRF checks; the overlay origin must not receive management authorization. An OBS URL grants read access to its page; keep it private.

## Persistent data and sharing

SQLite stores overlay HTML, values, and sensitive access information. Keep the database and backups private. Database files, WAL/SHM sidecars, and locally built executables are excluded by `.gitignore`. Do not commit them or include them in source archives.

For a consistent SQLite backup, stop the application cleanly before copying its database, or use SQLite's supported online backup mechanism. Do not delete WAL/SHM files while the application is running.

The repository's loopback `configuration.yaml` is a shareable default. Review personal changes to it before committing.

## Tests

```sh
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
```

JavaScript behavior checks run through the Go suite when Node.js is available on PATH or via `OBS_NODE`. Without Node.js they are skipped; a passing Go run does not then establish JavaScript behavior. Use `go test -v ./...` to see skips.

Optional browser checks require Python Playwright and Chrome:

```sh
python3 tests/browser_refactor.py
python3 tests/browser_access.py
```

Set `CHROME_BIN` if Chrome is not at `/opt/google/chrome/chrome`. These tests use temporary working directories, private YAML configuration/databases, and separate loopback ports. They do not use the project's live database. The access test simulates a second host using `overlay.test`; this is not a physical LAN or real OBS test.

See [tests/README.md](tests/README.md) and [KULLANICI_TESTLERI.md](KULLANICI_TESTLERI.md) (Turkish). Dated results and limitations are recorded under `test_output/`; earlier reports apply to the versions they tested.

## License and theme

Project license: [MIT](LICENSE). The management panel uses [Bootstrap Kehribar Gece](https://github.com/MuhammedOzby/bootstrap-kehribar-gece); its bundled license is preserved at [template/assets/bootstrap-kehribar-gece/LICENSE](template/assets/bootstrap-kehribar-gece/LICENSE). The theme is not injected into uploaded overlay HTML.
