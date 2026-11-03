# Erişim arayüzü doğrulaması

- Tarih: 1 Kasım 2026 (UTC)
- Kapsam: erişim ve cihaz sayfaları, panel CSRF istemcisi.
- `go test ./routes ./lib`: geçti.
- `go test ./...`: geçti.
- `go vet ./...`: geçti.
- `go test -race ./...`: geçti.
- `/home/ozbay/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node --test tests/panel.test.cjs`: 13 test geçti; CSRF başlığı ve oturum alınamadığında mutasyon gönderilmemesi doğrulandı.
- Aynı Node çalıştırıcısıyla `--check` kullanılarak `access.js`, `access-client.js`, `panel.js` sözdizimi kontrolü geçti.
- İlk `node` komutları PATH üzerinde Node bulunmadığından çalışmadı (127); ardından yukarıdaki mevcut çalıştırıcı kullanıldı.
- Gerçek tarayıcı, LAN ve OBS kabulü bu çalışmada yapılmadı. Güvenlik çekirdeğiyle uçtan uca entegrasyon ana denetçinin kontrolünü bekliyor.
