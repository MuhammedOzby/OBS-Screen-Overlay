# Yerel başlatma kontrolü — 2 Kasım 2026

- Kapsam: Kullanıcının yerel denemesi için mevcut kaynakla sunucuyu başlatma. Kod değişikliği yapılmadı.
- Başlangıç: `ss -ltnp '( sport = :8080 or sport = :8081 )'` çıktısında dinleyici yoktu; restart gerekmedi.
- Başlatma (arka planda): `env ADDR=127.0.0.1:8080 PUBLIC_ADDR=127.0.0.1:8081 PUBLIC_ORIGIN=http://127.0.0.1:8081 ADMIN_ORIGINS=http://127.0.0.1:8080,http://localhost:8080 DB_PATH=overlay.db GIN_MODE=release go run .`
- Mevcut `overlay.db` kullanıldı; LAN erişimi açılmadı.
- `curl --silent --show-error --output /dev/null --write-out 'Yerel erişim HTTP %{http_code}\n' http://127.0.0.1:8080/access` → HTTP 200, exit 0.
- `curl --silent --show-error --output /dev/null --write-out 'Yayın runtime HTTP %{http_code}\n' http://127.0.0.1:8081/system/overlay-runtime.js` → HTTP 200, exit 0.
- Bu işlemde tam test/vet/race, tarayıcı, fiziksel LAN veya OBS kabul testi çalıştırılmadı; yalnız HTTP erişilebilirliği kontrol edildi.
