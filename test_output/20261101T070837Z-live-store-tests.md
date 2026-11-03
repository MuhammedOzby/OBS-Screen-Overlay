# Canlı yayın store testleri ve taşınabilir panel JS kontrolü

- **Tarih:** 2026-11-01 07:08 UTC (`20261101T070837Z`)
- **Kapsam:** Yalnız testler. `panel_script_test.go` taşınabilir hale getirildi; `SubscribeLive` ve `UpdatePage` için eksik store seviyesi canlı yayın testleri eklendi.
- **Kapsam dışı:** Üretim kodu, yeni `page_deleted` olayı, `PLAN.md`, commit/push.

## Değişen dosyalar

1. `panel_script_test.go`
   - Node binary için `/tmp/opencode/...` hardcoded yolu kaldırıldı.
   - Sıra: `OBS_NODE` ortam değişkeni → `PATH` üzerindeki `node`.
   - Node yoksa `node --check` atlanıyor, yalnız yapısal kontroller çalışıyor.
   - Tüm JS dosyasında literal `\n` yasağı kaldırıldı; bu kontrol geçerli JavaScript string kaçışlarını da reddediyordu.
   - Yerine yalnız eski bozulma örüntüsü kontrol ediliyor:
     - `\n\n// START TEMPLATE FORM PUBLISHING LOGIC` literal dizisi yasak.
     - Aynı fiziksel satırda hem literal `\n` hem `// START TEMPLATE FORM PUBLISHING LOGIC` varsa hata.
   - Mevcut yapısal kontroller korundu: tek template publishing bloğu, tek `template-form` kaydı, form kapsamlı `#html-file`, doğru SSE mesajı, regex kaçışları.
   - Sözdizimi kontrolü `node --check` ile yapılmaya devam ediyor.

2. `internal/store/live_test.go` (yeni)
   - `SubscribeLive` ve `UpdatePage` için store seviyesinde canlı yayın testleri.
   - Mevcut `internal/live/hub_test.go` hub davranışını, `internal/store/update_test.go` DB/rollback davranışını kapsıyordu; ancak store üzerinden snapshot + publish entegrasyonu ve abonelik temizliği eksikti.

3. `test_output/20261101T070837Z-live-store-tests.md`
   - Bu rapor.

## Eklenen store testleri

### `TestSubscribeLiveSnapshotOnlyOpenFields`

- Sayfada SSE açık (`open`) ve kapalı (`closed`) alan var.
- İlk `SubscribeLive` snapshot'ı revision/template_version ve yalnız `open` alanını getiriyor.
- `UpdatePage` sonrası stream'de gelen snapshot yalnız `open` alanını taşıyor.
- Yeniden bağlanan (reconnect) yeni abonelik en güncel revision'ı ve yine yalnız `open` alanını getiriyor.
- SSE kapalı alanın payload'da olmadığı her aşamada doğrulanıyor.

### `TestUpdatePageFailuresDoNotPublish`

Abonelik açıkken şu hatalarda hiçbir snapshot yayınlanmadığı doğrulanıyor:

- Boş isim → `ErrValidation`
- Geçersiz alan adı (`bad-name`) → `ErrValidation`
- Eski revision → `ErrConflict`
- DB trigger ile insert abort → transaction rollback
  - Mevcut `update_test.go`/`delete_test.go` testlerindeki trigger deseni kullanıldı.
  - Trigger, transaction commit'e ulaşmadan insert'i durduruyor; rollback sonrası name, revision ve alanların korunduğu ayrıca kontrol ediliyor.
- Trigger kaldırıldıktan sonra başarılı `UpdatePage` yayın yapıyor; bu, aboneliğin hata testleri sırasında bozulmadığını gösteriyor.

Not: Commit çağrısının kendisini üretim kodu değiştirmeden güvenilir biçimde düşürmek mümkün olmadığı için commit hatası ayrıca simüle edilmedi; yayın zaten yalnız `Commit` başarılı olduktan sonra yapılıyor ve trigger testi commit öncesi hata yolunu kapsıyor.

### `TestSubscribeLiveConcurrentUpdate`

- `SubscribeLive` ile `UpdatePage` aynı anda başlatılıyor; 12 küçük iterasyon çalıştırılıyor.
- Her iterasyonda yeni revision'ın **ya dönen snapshot'ta ya da stream'de** olması bekleniyor.
- `liveMu` sayesinde okuma+abone olma penceresi ile commit+yayın penceresi çakışmıyor.
- Bounded hub kuyruğu taşarsa kanal kapanabilir; bu durum başarısızlık sayılmıyor. Bunun yerine yeniden abone olunup taze snapshot'tan son revision doğrulanıyor.
- Snapshot yeni revision'ı taşıyorsa yalnız SSE açık alanı içerdiği de kontrol ediliyor.
- İterasyon sonunda son snapshot'ın en güncel revision ve değeri taşıdığı doğrulanıyor.

### `TestSubscribeLivePageIsolation`

- A ve B sayfalarına ayrı abonelikler açılıyor.
- A güncellenince yalnız A abonesi snapshot alıyor; B abonesi boş kalıyor.
- B güncellenince yalnız B abonesi snapshot alıyor; A abonesi boş kalıyor.

### `TestDeletePageClosesSubscriptions`

- A ve B sayfalarına abonelik açılıyor.
- A silinince A abonesinin kanalı kapanıyor; sentetik bir olay/snapshot gelmiyor.
- Yeni `page_deleted` olayı eklenmedi.
- A aboneliğinde `cancel` iki kez çağrılarak idempotentlik doğrulanıyor.
- B abonesi etkilenmiyor ve kendi güncellemesini almaya devam ediyor.
- B silinince onun kanalı da kapanıyor.

## Çalıştırılan komutlar ve gerçek sonuçlar

```text
OBS_NODE=/tmp/opencode/node-v22.14.0-linux-x64/bin/node go test -count=1 -v \
  -run 'TestSubscribeLive|TestUpdatePageFailuresDoNotPublish|TestDeletePageClosesSubscriptions' \
  ./internal/store
=== RUN   TestSubscribeLiveSnapshotOnlyOpenFields
--- PASS: TestSubscribeLiveSnapshotOnlyOpenFields (0.00s)
=== RUN   TestUpdatePageFailuresDoNotPublish
--- PASS: TestUpdatePageFailuresDoNotPublish (0.00s)
=== RUN   TestSubscribeLiveConcurrentUpdate
--- PASS: TestSubscribeLiveConcurrentUpdate (0.00s)
=== RUN   TestSubscribeLivePageIsolation
--- PASS: TestSubscribeLivePageIsolation (0.00s)
=== RUN   TestDeletePageClosesSubscriptions
--- PASS: TestDeletePageClosesSubscriptions (0.00s)
PASS
ok  	github.com/ozbay/obs-screen-overlay/internal/store	0.005s

go test -race -count=10 \
  -run 'TestSubscribeLive|TestUpdatePageFailuresDoNotPublish|TestDeletePageClosesSubscriptions' \
  ./internal/store
ok  	github.com/ozbay/obs-screen-overlay/internal/store	1.114s

OBS_NODE=/tmp/opencode/node-v22.14.0-linux-x64/bin/node go test -count=1 ./...
ok  	github.com/ozbay/obs-screen-overlay	0.021s
ok  	github.com/ozbay/obs-screen-overlay/internal/live	0.002s
ok  	github.com/ozbay/obs-screen-overlay/internal/store	0.065s
ok  	github.com/ozbay/obs-screen-overlay/internal/upload	0.005s
?   	github.com/ozbay/obs-screen-overlay/lib	[no test files]
ok  	github.com/ozbay/obs-screen-overlay/routes	1.072s

OBS_NODE=/tmp/opencode/node-v22.14.0-linux-x64/bin/node go test -race -count=1 ./...
ok  	github.com/ozbay/obs-screen-overlay	1.031s
ok  	github.com/ozbay/obs-screen-overlay/internal/live	1.010s
ok  	github.com/ozbay/obs-screen-overlay/internal/store	1.250s
ok  	github.com/ozbay/obs-screen-overlay/internal/upload	1.023s
?   	github.com/ozbay/obs-screen-overlay/lib	[no test files]
ok  	github.com/ozbay/obs-screen-overlay/routes	2.156s

go vet ./...
(çıktı yok; başarılı)
```

Ek olarak `panel_script_test.go`, `OBS_NODE` verilmeden ve `PATH` üzerinde node yokken de çalıştırıldı; yapısal kontroller geçti ve test yalnız `Node bulunamadı; yalnız yapısal kontroller çalıştı` logunu bıraktı.

## Notlar

- Üretim kodu değiştirilmedi.
- Yeni `page_deleted` olayı eklenmedi.
- `PLAN.md` değiştirilmedi.
- Commit/push/deployment yapılmadı.
- Testlerde sabit `sleep` yok; yalnız kanal bekleme sınırları için 2 saniyelik timeout kullanılıyor.
