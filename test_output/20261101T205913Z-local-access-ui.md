# Yerel erişim düğmesi ve hız sınırı mesajı

- Tarih: 1 Kasım 2026 (UTC).
- Kapsam: erişim sayfasında gerçek bağlantı ve Host üzerinden yerel düğme görünürlüğü; eşleştirmede 429 mesajı.
- `go test ./routes`: geçti; IPv4, IPv6, localhost, uzak Host/istemci ve forwarded başlıkları render testleri dahil.
- `go test ./...`: geçti.
- `go vet ./...`: geçti.
- `go test -race ./...`: geçti.
- `/home/ozbay/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node --test tests/access-ui.test.cjs tests/panel.test.cjs`: 15 test geçti; 429 ve 403 mesajları dahil.
- Bu değişiklik için gerçek tarayıcı testi çalıştırılmadı. Yerellik kontrolü yalnız görünürlüktür; backend yetkilendirmesi değiştirilmedi.
