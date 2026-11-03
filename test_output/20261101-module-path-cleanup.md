# Go modül adı temizliği

- Tarih: 1 Kasım 2026.
- Kapsam: `go.mod` ve yerel Go import yollarındaki `github.com/ozbay/obs-screen-overlay` adı `obs-screen-overlay` olarak değiştirildi. Geçmiş raporlar korunmuştur.
- `go test ./...`: başarılı (exit 0); ana paket, internal/live, internal/store, internal/upload ve routes geçti; lib test içermiyor.
- `go vet ./...`: başarılı (exit 0), çıktı yok.
- `go test -race ./...`: başarılı (exit 0); bütün testli paketler geçti.
- `*.go` dosyalarında `github.com/ozbay` araması: eşleşme yok.
- Tarayıcı/OBS kontrolü yapılmadı; değişiklik yalnız modül ve import adlarıyla sınırlı.
