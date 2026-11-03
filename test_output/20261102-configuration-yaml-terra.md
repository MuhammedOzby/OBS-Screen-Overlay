# YAML yapılandırma doğrulaması

- Tarih: 2 Kasım 2026
- Kapsam: zorunlu `configuration.yaml`, yerel ve LAN origin doğrulaması, ortam değişkenlerinin yok sayılması, hatalı ve eksik ayarlar.
- `go test -count=1 ./...`: geçti (tüm Go paketleri).
- `go vet ./...`: geçti.
- `go test -race -count=1 ./...`: geçti (tüm Go paketleri).
- YAML API kaynağı: https://pkg.go.dev/github.com/goccy/go-yaml@v1.19.2 (`NewDecoder`, `Strict`, `Decode`).
- Fiziksel LAN, tarayıcı ve OBS kabulü bu dar kapsamda çalıştırılmadı. Commit ve deployment yapılmadı.
