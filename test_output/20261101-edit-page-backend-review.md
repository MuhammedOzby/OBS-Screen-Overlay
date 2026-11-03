# Düzenleme ekranı — bağımsız denetim

- Tarih: 1 Kasım 2026.
- Kapsam: isim/tür/değişken ekleme, düzenleme, kaldırma ve alan bazlı SSE tercihlerini SQLite'a atomik kaydetme. Uygulama kodu Qwen tarafından yazıldı, ana ajan inceledi ve test etti.
- `PATCH /api/pages/:id`: zorunlu revision, isim, tür ve sıralı fields listesi; 8 MiB istek sınırı, sabit alan sayısı sınırı yok. Eksik/null/bilinmeyen/hatalı alanlar reddedilir.
- İsim/tür ve bütün alanlar tek transaction; stale revision 409; kaynak HTML, token ve tasarım sürümü korunur.

## Çalıştırılan kontroller

Ana ajan tarafından ilk uygulama sonrasında ve son metin düzenlemesinden sonra iki kez çalıştırıldı:

| Komut | Gerçek sonuç |
| --- | --- |
| `go test -count=1 ./...` | PASS: store, upload, routes; diğer paketlerde test yok |
| `go vet ./...` | PASS, çıktı yok |
| `go test -race -count=1 ./...` | PASS, race raporu yok |
| `go build -o /tmp/opencode/obs-edit-review .` | PASS |
| `/tmp/opencode/node-v22.14.0-linux-x64/bin/node --check template/assets/panel.js` | PASS |
| `/tmp/opencode/obs-panel-test-venv/bin/python /tmp/opencode/review-edit-page.py` | PASS, sekiz senaryo grubu |

Son durumda `gofmt -l internal/store/update.go internal/store/update_test.go routes/update.go routes/update_test.go routes/handlers.go` çalıştırıldı: çıktı yok, PASS.

Qwen ayrıca değişiklik parçaları sırasında store/routes testleri, Go build/vet, gofmt ve JavaScript sözdizimi kontrolleri yaptığını bildirdi. Yukarıdaki sonuçlar ana ajanın bağımsız çalıştırmalarıdır. Birkaç Qwen görev çağrısı çıktı sınırı veya `no user query found` hatasıyla tamamlanamadı; işler daha küçük parçalara ayrılarak aynı modele yeniden verildi. Başka modele geçilmedi.

## Go test kapsamı

- `internal/store/update_test.go`: değer/SSE/sıra ve metadata güncelleme; revision artışı; HTML/token/tasarım sürümünü koruma; conflict/not-found; alan insert trigger'ı ile transaction ortasında hata üretip metadata, revision ve eski alanların geri alındığını doğrulama.
- `routes/update_test.go`: gerçek handler üzerinde başarılı JSON güncellemesi, aynı revision'la tekrar denemede 409 ve bulunamayan kimlikte 404.

## Gerçek Chromium / Playwright

Geçici SQLite ve `127.0.0.1:18182` test listener'ı kullanıldı, iki çalıştırma da tamamıyla geçti:

1. Yeni kayıt oluşturma akışı bozulmadı. Düzenlemede isim/tür, çok satırlı Türkçe/HTML benzeri metinler ve SSE açık/kapalı tercihleri kaydedilip yeniden açıldığında korundu; değerler panelde kod olarak çalışmadı.
2. 360/768/1440 piksel genişlikte yatay taşma yok.
3. Yinelenen değişken adı istemcide engellendi; ad değiştirme/kaldırma ve tekrar kaydetmede güncel revision kullanımı doğrulandı.
4. İki sekmede stale kayıt 409 verdi; kullanıcı taslağı korundu, açık kullanıcı işlemiyle güncel kayıt yüklendi.
5. Ağ kesintisi ve JSON olmayan 500 yanıtı sonrası taslak korundu ve tekrar deneme düğmesi açıldı.
6. Boş isim, hatalı tür/alan adı, duplicate, negatif revision, eksik/null/bilinmeyen alan, hatalı değer tipi, trailing JSON/garbage, yanlış Content-Type ve 8 MiB üstü payload reddedildi. DB revision ve HTML/token/template_version/created_at değişmedi.
7. 10.000 değişken API ile kaydedildi; ardından boş liste gönderilerek tümü kaldırıldı.
8. Süreç aynı DB ile yeniden başlatıldı: isim ve revision korundu. JavaScript kapalıyken kaydetme devre dışı. JavaScript hatası yok. Tasarım bölümü devre dışı kaldı.

## Sınırlar

- SSE taşıma, OBS URL ve gerçek OBS Browser Source kabulü bu görevin kapsamında uygulanmadı/test edilmedi. Kaydetme başarı mesajı canlı yayın yapıldığını iddia etmez.
- HTML değiştirme/tasarım yayınlama ve kayıt silme sonraki işlerdir.
- Önceden ertelenen oturum/CSRF/Origin güvenliği hâlâ bekliyor; dış erişime açılmadı.
- Mevcut kullanıcı sürecine ve gerçek `overlay.db` içeriğine test amacıyla dokunulmadı; test listener'ı kapatıldı. Yeni backend'in çalışan uygulamada etkin olması için güncel sürümle yeniden başlatma gerekir.
- Commit, push veya deployment yapılmadı.
