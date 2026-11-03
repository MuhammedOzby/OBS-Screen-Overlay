# YAML yapılandırması — bağımsız doğrulama

- Tarih: 2 Kasım 2026.
- Kapsam: Kök `configuration.yaml`, dosyadan yükleme, ortam değişkenlerinin ayarları etkilememesi, katı YAML doğrulaması ve mevcut origin korumaları kaynak üzerinden incelendi.
- `go test -count=1 ./...`: geçti; 7 paket başarılı.
- `go vet ./...`: geçti; çıktı/hata yok.
- `go test -race -count=1 ./...`: geçti; 7 paket başarılı, yarış bildirilmedi.
- Yeni testler: yerel dosya, ortam değişkenlerinin yok sayılması, LAN izin listesi, eksik dosya/alan, hatalı/çok belgeli YAML, bilinmeyen/tekrarlanan anahtar ve origin çakışmaları.
- Tarayıcı, fiziksel LAN ve gerçek OBS Browser Source kabulü bu yapılandırma değişikliği kapsamında çalıştırılmadı.
- Commit, deployment yapılmadı; mevcut veritabanı ve derlenmiş uygulama değiştirilmedi.
