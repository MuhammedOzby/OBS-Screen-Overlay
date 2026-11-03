# HTML alan derleyicisi doğrulaması

- Tarih: 1 Kasım 2026
- Kapsam: bağımsız HTML derleyicisi, mevcut placeholder doğrulaması ve mevcut Go paketleri.
- `gofmt -w internal/upload/html.go internal/upload/compiler.go internal/upload/compiler_test.go`: başarılı.
- `go test -count=1 -v ./internal/upload`: **PASS**, `ok github.com/ozbay/obs-screen-overlay/internal/upload 0.005s`. Alan sırası ve tekrarlar, büyük/küçük harf, düz metin escape, kaynak ve gömülü içeriğin korunması, desteklenmeyen bağlamlar, namespace ve geçersiz belge senaryoları geçti.
- `go test -count=1 ./...`: **PASS**. `internal/store` 0.061s, `internal/upload` 0.005s, `routes` 0.012s; kök ve `lib` paketlerinde test dosyası yok.
- `golang.org/x/net/html.Tokenizer` ve `html.Parse` için kurulu modül belgeleri `go doc` ile incelendi. Parser'ın hatalı yapıları düzeltebildiği ve 512 seviyeden derin belgeyi reddettiği dikkate alındı.
- Tarayıcı/OBS, SSE, runtime ve yayın entegrasyonu bu bağımsız derleyici çalışmasının kapsamında test edilmedi.
