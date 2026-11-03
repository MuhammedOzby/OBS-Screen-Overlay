# Template Yayın Refaktörü — Test ve Doğrulama Raporu

- Tarih: 2026-11-01 (UTC 2026-11-01T07:55:15Z)
- Kapsam: `internal/store/template.go` okunabilirlik refaktörü ve `internal/store/template_test.go` yeni testleri. (İlgili testler: store publish/snapshot/rollback/concurrency/negative-rev; `internal/live` hub testleri).
- Not: Bu görev, önceki bütünsel denetimde üretilen `required return signature / as per instruction / moved` tarzı yorumları ve `CompileHTML` üç dönüşlü kullanımını temizlemeyi, `upload.ValidatePlaceholders` kullanımını, validation'ın `liveMu` öncesine alınmasını, rows kapatma güvenliğini ve gereksiz tekrarı azaltmayı içeriyordu.

## Çalıştırılan komutlar ve sonuçlar

1. `go build ./...`
   - Sonuç: BAŞARILI (BUILD_OK).
2. `go test -count=1 ./...`
   - Sonuç: TÜM PAKETLER GEÇTİ (`main`, `internal/live`, `internal/store`, `internal/upload`, `routes`). `lib [no test files]`.
   - Not: Bu çalıştırdığım ilk sırasında `routes` paketi `routes/template_test.go:153` derleme hatasıyla FAIL ediyordu; sonraki çalıştırmada temiz geçti (eş zamanlı/son duruma göre değişmiş olabilir; dosya bu görevde değiştirilmedi).
3. `go vet ./...`
   - Sonuç: TEMİZ (VET_CLEAN).
4. `go test -race -count=1 ./...`
   - Sonuç: TÜM PAKETLER GEÇTİ (`-count=1`, race ile).

## Yeni testler (internal/store/template_test.go)

- `TestPublishTemplate_SuccessPreservesFields` — PASS: source/revision/template_version ilerleme + alan listesi/değer/SSE/sıra korunması.
- `TestPublishTemplate_PublishesOnlyOpenFieldsAfterCommit` — PASS: commit sonrası yayın snapshot’i yalnızca SSE açık alanı taşıyor.
- `TestPublishTemplate_FailuresDoNotMutateOrPublish` — PASS: stale/invalid/missing sonrası mutation yok ve event yok.
- `TestPublishTemplate_RollbackOnUpdateAbort` — PASS: trigger ile abort → rollback, publish yok.
- `TestPublishTemplate_ConcurrentSameRevisionExactlyOneSucceeds` — PASS: aynı revision ile iki concurrent publish → tam olarak biri success, biri conflict.
- `TestPublishTemplate_NegativeRevision` — PASS: negatif revision → ErrValidation.

Not: İlk çalıştığımda `TestPublishTemplate_PublishesOnlyOpenFieldsAfterCommit` ve `TestPublishTemplate_FailuresDoNotMutateOrPublish` `initial snapshot: snapshot gelmedi` ile FAIL etmişti; `SubscribeLive` ilk snapshot’ı kanala push etmediği (doğrudan dönüp kanala yazmadığı) nedeniyle testlerde kanal beklemesi yapılan iki test düzeltildi ve tekrar geçti.

## Karar

- Refaktör ve yeni store testleri tamam; `go build`, `go vet`, `go test -count=1 ./...` ve `go test -race -count=1 ./...` hepsi geçti.
- `routes/` tarafına dokunulmadı; `routes/template_test.go` bu görevde değiştirilmedi.

## İlgili bulgular

- Bu uygulama görevi özel bulguları giderdi: template.go okunabilirlik/ValidatePlaceholders/liveMu sırası/rows kapatma; store seviyesi publish test koruması eklendi.
- Önceki bütünsel denetimde `internal/store/template.go:114` ve `internal/store/delete.go:28` hub nil-guard eksikliği raporlandı; kullanıcı kararıyla "gerçek bug değil" sayıldı ve bu görevi kapsam dışı bırakıldı; refaktörde hub/guard değişikliği yapılmadı.
