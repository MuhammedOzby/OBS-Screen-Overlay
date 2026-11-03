# Erişim API sözleşmesi düzeltmesi — 1 Kasım 2026

`GET /access/devices` yanıtı `devices` dizisi içeren nesneye dönüştürüldü. Eşleşmede boş/eksik cihaz adı `Eşleşmiş cihaz` olur; 128 baytı aşan ad reddedilir. Yerel olmayan pairing oluşturma reddi ve store kapalıyken oturum oluşturmanın başarısız olması test edildi: cookie verilmez, kayıt oluşmaz, challenge korunur ve store yeniden açıldığında aynı challenge ile eşleşme başarılıdır. Bu test kapalı store ile insert hatasını doğrular; ayrıca gerçek SQLite COMMIT hatası enjekte edilmedi.

- `go test -count=1 ./internal/access` — geçti.
- `go vet ./internal/access` — geçti.
- `go test -race -count=1 ./internal/access` — geçti.

Tam proje veya tarayıcı testleri çalıştırılmadı.
