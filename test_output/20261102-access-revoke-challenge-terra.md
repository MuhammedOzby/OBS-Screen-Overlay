# Toplu iptal ve erişim sınırları — 2 Kasım 2026

Kapsam: `internal/access`. Toplu cihaz iptali aynı mutex altında önce veritabanındaki oturumları siler, yalnız başarıda aktif challenge'ı temizler. Pair oturum eklemesi ve challenge tüketimi aynı mutex kullanır. Tek cihaz iptali challenge'ı değiştirmez. İptalden önce kimliği doğrulanmış fakat henüz challenge oluşturmamış istek, mutex altında oturumun tekrar kontrolü sayesinde yeni challenge oluşturamaz. Kilit gerektiren DB işlemleri mutex → store sırasındadır; ilk middleware oturum okuması mutex alınmadan tamamlanır.

Kontroller: toplu iptalde eşleştirilmiş oturum ve bekleyen QR iptali; tek cihaz iptalinde challenge korunması; kapalı store ile silme hatasında challenge korunması ve cookie verilmemesi; eşzamanlı pair/toplu iptal; iptal edilmiş oturumla challenge oluşturulamaması; 50 farklı IP ile global limit; 1024 girişte harita sınırı; eksik `X-Overlay-Request` ve cross-site Fetch Metadata reddi. Global limit normal akışta 1024 girişe ulaşmayı engellediğinden harita sınırı testi haritayı doğrudan doldurur. Store hata testi gerçek SQLite COMMIT hatası enjekte etmez.

- `go test -count=1 ./internal/access` — geçti.
- `go vet ./internal/access` — geçti.
- `go test -race -count=1 ./internal/access` — geçti.

Tam proje ve tarayıcı testleri bu takip çalışmasında tekrar çalıştırılmadı.
