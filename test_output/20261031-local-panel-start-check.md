# Yerel panel başlatma kontrolü

- Tarih: 31 Ekim 2026.
- Kapsam: Mevcut demo paneli tasarımı değiştirmeden loopback üzerinde başlatma.
- Başlatma: `go run .` (arka planda çalışır).
- Kontrol: `curl --silent --show-error --output /dev/null --write-out 'HTTP %{http_code}\n' http://127.0.0.1:8080/`
- Gerçek sonuç: `HTTP 200`.
- Adres: http://127.0.0.1:8080/
- Bu çalışmada tarayıcı, OBS veya tam test paketi çalıştırılmadı. Demo kayıt ve gerçek yayın sağlamaz.
