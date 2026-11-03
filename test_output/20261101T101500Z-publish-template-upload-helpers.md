# Publish template multipart yardımcıları — test raporu

- Tarih: 1 Kasım 2026.
- Uygulama: Alt yapı (subagent) — `routes/create.go`, `routes/template.go`, yeni `routes/upload.go`, yeni `routes/template_test.go`, yeni `routes/upload_test.go`.
- Kapsam: `create` ve `publishTemplate` handler'larındaki kopyalanmış multipart parse / body limit / `.html` filename / bounded read / cleanup mantığının küçük ortak helper'lara ayırılması; publish handler'ı için gerçek HTTP testleri; geçerli creation testlerinin korunması.

## Yapılan değişiklikler

1. `routes/upload.go` (yeni): package seviyesi `maxHTMLBytes` (`upload.MaxHTMLSize`) ve `maxMemory` (64 KiB) sabitleri; `cleanupMultipartForm`, `parseUploadForm` (MaxBytesReader + `ParseMultipartForm`, hata sentinel'ları), `isHTMLFile`, `readUploadFile` (limit+1 okuma, 413 sınırı) ortak helper'ları.
2. `routes/create.go`: yerel kopya sabitler ve parse/read mantığı silinip yukarıdaki helper'lar kullanıldı; handler akışı (alan sayımı, isim/tür doğrulama, `ValidatePlaceholders`) aynen korundu.
3. `routes/template.go`: yerel kopya sabitleri (`MaxHTMLBytes = 2 << 20`, `MaxMemory`) ve simulation/assuming yorumları silinip helper'lar kullanıldı; publish akışı (alan sayımı, `expected_revision` parse, filename, read, store hata eşlemesi) aynen korundu.

## Çalıştırılan komutlar ve gerçek sonuçlar

1. `go vet ./routes/...` → PASS.
2. `gofmt -l routes/` → yalnızca önceden var olan `routes/events.go` çıktı (değişiklik kapsamı dışında, dokunulmadı); yeni/ değiştirilen dosyalar formatlı.
3. `go test -count=1 -run 'TestPublishTemplateHTTP|TestIsHTMLFile|TestCleanupMultipartFormSafe|TestCreatePageSuccess|TestCreateRejects' -v ./routes/` → PASS:
   - `TestCreatePageSuccess` PASS.
   - `TestCreateRejects` tüm alt testler PASS (name whitespace, type invalid, filename zip, fragment, UTF8 invalid, empty, placeholder attribute, SVG text, html 2MiB+1, body 3MiB).
   - `TestPublishTemplateHTTP` PASS.
   - `TestIsHTMLFile` PASS.
   - `TestCleanupMultipartFormSafe` PASS.
4. `go vet ./...` → PASS.
5. `go test -count=1 ./...` → PASS (main, lib, internal/live, internal/store, internal/upload, routes).
6. `go build ./...` → PASS.

## Yeni/ güncellenen testler

- `routes/template_test.go` (yeni): `TestPublishTemplateHTTP` — gerçek handler HTTP testleri:
  - success: 200 `{revision:2}`, persisted source + revision + template_version doğrulandı.
  - stale409: beklenen 1 / mevcut 2 → 409, mutation yok.
  - invalidHTML400: geçersiz HTML → 400, mutation yok.
  - missing404: bilinmeyen id → 404.
  - duplicate/unknown form fields → 400, mutation yok.
  - extension400: `test.zip` → 400, mutation yok.
  - oversized413: 2 MiB + 1 → 413, mutation yok.
  - malformed multipart → 400, mutation yok.
- `routes/upload_test.go` (yeni): `TestIsHTMLFile`, `TestCleanupMultipartFormSafe`.
- `routes/create_test.go`, `routes/reject_test.go`, handler davranışı: dokunulmadı; geçerli ve reddedilen create testleri korunarak geçti.

## Sınırlar ve notlar

- Tam testler (`go test -race ./...`, tam `-v` output) ana ajan tarafından çalıştırılacak; burada yalnızca dar test seti + paket geneli hızlı testler çalıştırıldı.
- Auth, test gevşetme veya scope genişletmesi yapılmadı; istek sözleşmesi (status kodları, limitler, alan kuralları) korundu.
- `routes/events.go` gofmt çıktısı önceden mevcuttu; bu görevde değiştirilmedi.
