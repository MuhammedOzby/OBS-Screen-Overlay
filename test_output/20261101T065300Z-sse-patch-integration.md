# SSE panel bağlantısı ve gerçek streaming regresyon testi

- **Tarih:** 2026-11-01 06:53 UTC (`20261101T065300Z`)
- **Kapsam:** Panelin kullandığı `PATCH /api/pages/:id` yolunu gerçek admin router'a bağlamak ve `httptest.NewServer` üzerinden PATCH → SSE entegrasyon testi eklemek.
- **Kapsam dışı:** Template publish, silme davranışı, güvenlik, runtime refactor, HTML template değişikliği, OBS kabulü.

## Değişen dosyalar

1. `routes/handlers.go`
   - `RegisterRoutes` içine `r.PATCH("/api/pages/:id", updatePage(s))` eklendi.
   - Yeni bir `/state` yolu icat edilmedi; mevcut panelin kullandığı yol kullanıldı.

2. `routes/public.go`
   - Mevcut public route kayıtları `RegisterPublicRoutes(r, s)` içine alındı.
   - `NewPublicRouter(s)` davranışı korunarak bu fonksiyonu çağıracak şekilde küçültüldü.
   - Amaç, testin gerçek public route kayıt setini kullanması; handler kopyalaması yapılmaması.

3. `routes/sse_integration_test.go` (yeni)
   - Gerçek `RegisterRoutes` admin router'ı ve `RegisterPublicRoutes` public router'ı `httptest.NewServer` ile ayağa kaldırılıyor.
   - SSE çerçeveleri arka planda `bufio.Scanner` ile ayrıştırılıyor; sabit `sleep` yerine beklenen olaylarda 2 sn, olay beklenmeyen kontrollerde 250 ms timeout kullanılıyor.
   - Test edilenler:
     - İlk abonelik snapshot'ı ve revision 1.
     - İki istemcinin aynı revision'ı alması.
     - SSE kapalı alanın payload'da bulunmaması.
     - SSE aç/kapat: kapatınca alanın payload'dan çıkması, açınca güncel değerin gelmesi.
     - Stale revision'ın 409 dönmesi ve hiçbir SSE olayı yayınlamaması.
     - Başarısız PATCH sonrası akışın hâlâ kullanılabilir olması.
     - Bir sayfanın güncellemesinin diğer sayfanın abonesine sızmaması.

## Çalıştırılan komutlar ve gerçek sonuçlar

```text
go test -count=1 ./routes
ok  	github.com/ozbay/obs-screen-overlay/routes	1.071s

go test -count=1 -run '^TestSSEPatchIntegration$' -v ./routes
=== RUN   TestSSEPatchIntegration
--- PASS: TestSSEPatchIntegration (1.01s)
PASS
ok  	github.com/ozbay/obs-screen-overlay/routes	1.009s

go test -count=5 -run '^TestSSEPatchIntegration$' ./routes
ok  	github.com/ozbay/obs-screen-overlay/routes	5.029s

go test -race -count=1 ./routes
ok  	github.com/ozbay/obs-screen-overlay/routes	2.148s

go test -race -count=1 ./...
ok  	github.com/ozbay/obs-screen-overlay	1.013s
ok  	github.com/ozbay/obs-screen-overlay/internal/live	1.012s
ok  	github.com/ozbay/obs-screen-overlay/internal/store	1.249s
ok  	github.com/ozbay/obs-screen-overlay/internal/upload	1.028s
?   	github.com/ozbay/obs-screen-overlay/lib	[no test files]
ok  	github.com/ozbay/obs-screen-overlay/routes	2.151s

go vet ./...
(çıktı yok; başarılı)

go test -count=1 ./...
ok  	github.com/ozbay/obs-screen-overlay	0.006s
ok  	github.com/ozbay/obs-screen-overlay/internal/live	0.002s
ok  	github.com/ozbay/obs-screen-overlay/internal/store	0.065s
ok  	github.com/ozbay/obs-screen-overlay/internal/upload	0.006s
?   	github.com/ozbay/obs-screen-overlay/lib	[no test files]
ok  	github.com/ozbay/obs-screen-overlay/routes	1.072s
```

## Panel JS sözdizimi kontrolü

İddia yeniden kontrol edildi; dosya gerçekten bozuk:

- `template/assets/panel.js` içinde 18, 152, 199 ve 254. satırlar aynı 3.960 baytlık bloğu içeriyor.
- Her birinde 117 adet literal `\n` (ters bölü + n) byte dizisi var; bunlar gerçek satır sonu değil.
- İlk satır `})();\n\n// START...` biçiminde; üst düzeyde string/comment dışında kalan ters bölü JavaScript lexer'ı için geçersizdir. Yani bu haliyle dosyanın tamamının sözdizimi hatası vermesi beklenir.
- Bu adımda `panel.js` değiştirilmedi; ayrı görev olarak ele alınmalı.
- Ortamda Node/Deno/Chromium gibi bir JS çalıştırıcısı bulunmadığı için tarayıcı çalıştırması yapılmadı; kanıt byte düzeyindedir.

## Başarısız veya engellenen kontroller

- Başarısız Go testi yok.
- Tarayıcı/OBS testi yapılmadı; bu görevin kapsamı dışında.
- `PLAN.md` değiştirilmedi.
- Commit/push/deployment yapılmadı.

## Notlar

- Test, `RegisterRoutes` şablonları çalışma dizinine göre yüklediği için `t.Chdir("..")` ile repo kökünden çalışıyor; test sonunda çalışma dizini otomatik geri yükleniyor.
- Yeni testte sabit uyku yok; olumlu olaylar 2 sn timeout ile, olay beklenmeyen durumlar 250 ms sınırlı bekleme ile kontrol ediliyor.
- `/api/pages/:id` PATCH yolu artık gerçek router'da kayıtlı olduğundan panelin mevcut `fetch` çağrısı 404 yerine handler'a ulaşır.
