# Erişim yöneticisi — 1 Kasım 2026

Kapsam: bağımsız `internal/access` middleware, pairing ve cihaz oturumu; router/UI entegrasyonu yapılmadı.

## API ve kontroller

`access.New(store, access.Config{AdminOrigins: origins})` ve `manager.Wrap(next)` yönetim listener'ına uygulanır. İzin listesi tam şema/host/port eşleşmesiyle doğrulanır; TLS doğrudan sunucuda sonlanmalıdır, forwarded başlıkları kullanılmaz. `GET /access` ve `GET /assets/...` next'e geçer. Diğer yollar oturum ister; API 401, HTML 303 döner. Mutasyonlar tam Origin ve oturum CSRF değeri gerektirir; local bootstrap ve pairing için Origin ve `X-Overlay-Request: 1` gerekir. Cross-site Fetch Metadata reddedilir. Güvenlik başlıkları ve no-store uygulanır.

- `POST /access/local`: gerçek loopback uzak adres ve loopback Host ile yerel oturum.
- `GET /access/session`: CSRF, local ve son kullanma; yalnız local ise `admin_origins`.
- `POST /access/pairing`: yerel oturum ve CSRF; isteğe bağlı `{ "origin": "..." }` izin listesinden seçilir. `code`, `qr_token`, `expires_at`, `qr_url` (fragment) ve `qr_image` (PNG data URL) döner; varsayılan mümkünse loopback olmayan origin.
- `POST /access/pair`: `code` veya `qr_token`, isteğe bağlı `device_name` (boşsa `Eşleşmiş cihaz`; en fazla 128 bayt); tek kullanımlık, beş dakikalık bellek içi challenge ve sınırlı başarısız deneme sayacı.
- `POST /access/logout`; yerel `GET /access/devices` (`{"devices":[{"id":"...","device_name":"...","created_at":"...","expires_at":"..."}]}`) ve `POST /access/devices/revoke` (`id` veya `all`). Veritabanında yalnız SHA-256 oturum token özeti saklanır; cookie host-only, HttpOnly, Strict ve 180 gündür.

`github.com/skip2/go-qrcode` `v0.0.0-20200617195104-da1b6568686e` olarak çözümlendi. QR fragment içeriği HTTP query'ye yerleştirilmez. `/cihazlar` UI için ayrıdır ve bu paket tarafından sunulmaz.

## Gerçek sonuçlar

- `go test -count=1 ./internal/access` — geçti.
- `go vet ./internal/access` — geçti.
- `go test -race -count=1 ./internal/access` — geçti.

Testler bootstrap, remote/forwarded reddi, CRUD yetkisi, Origin/CSRF, aynı host farklı port, cookie, TLS Secure, logout/revoke, kalıcılık, güvenlik başlıkları, challenge yenileme/süre/tek kullanım/eşzamanlılık, başarısız deneme limiti, QR hedef seçimi ve reddini kapsar. Tam proje testleri ve gerçek tarayıcı/LAN/OBS entegrasyonu bu bağımsız paket çalışmasında yapılmadı.

Takip düzeltmesi ve store hatası testi: `test_output/20261101-access-contract-followup-terra.md`.
