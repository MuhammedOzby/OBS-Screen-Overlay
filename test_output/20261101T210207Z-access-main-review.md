# QR/kod ve kalıcı cihaz oturumu — ana denetçi kabulü

- Tarih: 2 Kasım 2026 yerel tarih; son kontrol başlangıcı UTC `20261101T210207Z`.
- Durum: uygulama ve otomatik/tarayıcı kontrolleri geçti. Fiziksel LAN, telefonla QR tarama ve gerçek OBS Browser Source kabulü yapılmadı.
- Kullanıcı onayı: yerel model kısıtlaması kaldırıldı; uygulama işleri `openai/gpt-6.1-terra` alt ajanlarına verildi. Ana ajan kaynak incelemesi ve bağımsız test yaptı.

## Kabul edilen değişiklikler

- SQLite v2 kalıcı cihaz oturumları: ham oturum token'ı değil SHA-256 özeti saklanır. v1 sayfa/alan verilerini koruyan migration, expiry/list/delete ve yeniden açılış testleri eklendi.
- Yönetim listener'ı bütün yolları kapsayan erişim korumasına bağlandı. Giriş HTML'i ve statik dosyalar dışındaki yönetim ekranları oturum ister; mutation istekleri tam Origin ve oturum CSRF belirteci ister. Hesap/parola eklenmedi.
- Yerel erişim başlatma yalnız gerçek loopback istemci ve loopback Host üzerinden; forwarded başlıklarına güvenilmez. Uzak giriş sayfasında yerel panel açma düğmesi gizlidir.
- Yerel `/cihazlar` ekranında izinli LAN origin seçimi, QR ve altındaki altı haneli kod. Kod ve QR beş dakika/tek kullanım; QR URL'si sırları query yerine fragment'ta taşır, giriş JS'i fragment'ı temizler.
- Kod denemeleri IP başına 5 ve global 50 başarısız deneme/5 dakika ile sınırlıdır; IP haritası sınırlıdır. Süre dolması, challenge yenileme, eşzamanlı tek kazanan ve başarısız store yazımında challenge korunması test edildi.
- Eşleşmiş cihaz 180 günlük host-only/HttpOnly/Strict cookie ile kalır. Çıkış, tek cihaz ve tüm cihaz erişim iptali var. Toplu iptal bekleyen challenge'ı da eşleştirmeyle aynı kilit altında iptal eder.
- Panel oluşturma/düzenleme/tasarım/silme istekleri oturumdan CSRF alır. Belirteç alınamazsa mutation gönderilmez. OBS yayın HTML'ine yönetim script'i eklenmedi.
- Varsayılan bind hâlâ loopback. Loopback dışı bind için açık `ADMIN_ORIGINS`; yönetim/yayın origin çakışması reddi; cookie/CSRF/query/panic sırlarını dökmeyen recovery. Yeni QR bağımlılığı resmi README ile kontrol edilen `github.com/skip2/go-qrcode` sabit pseudo-version'ıdır.
- Kullanım ve LAN örneği `README.md` içinde. HTTP LAN şifreli değildir; açık internete açma ve TLS proxy desteği varsayma uyarıları var.

## İncelemede düzeltmeye verilen noktalar

- Cihaz JSON yanıtı UI sözleşmesiyle aynı `{devices:[...]}` biçimine getirildi.
- Gin'in standart recovery'si debug modda Cookie/CSRF gibi başlıkları dökebildiği için sırsız recovery kullanıldı; gerçek panic testi eklendi.
- LAN giriş ekranında yanıltıcı yerel erişim düğmesi gizlendi; 429 için Türkçe bekleme mesajı eklendi.
- Toplu iptalin bekleyen QR/kodu da geçersiz kılması ve iptalle yarışan challenge oluşturma isteğinin oturumunu yeniden doğrulaması sağlandı.

## Ana ajanın son kaynakta bağımsız çalıştırdığı kontroller

| Komut | Gerçek sonuç |
| --- | --- |
| `TMPDIR=/tmp/opencode OBS_NODE=/home/ozbay/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node go test -count=1 ./...` | exit 0; yedi paket başarılı, Node wrapper'ları etkin |
| `go vet ./...` | exit 0; çıktı yok |
| `TMPDIR=/tmp/opencode OBS_NODE=/home/ozbay/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node go test -race -count=1 ./...` | exit 0; yedi paket başarılı, race bildirimi yok |
| `/home/ozbay/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node --test tests/access-ui.test.cjs tests/panel.test.cjs tests/runtime.test.cjs` | 29/29 geçti; 0 fail, 0 skip |
| `TMPDIR=/tmp/opencode /tmp/opencode/obs-panel-test-venv/bin/python tests/browser_access.py` | exit 0; gerçek Chrome kod ve QR URL eşleştirme, tek kullanım, cookie kalıcılığı, süreç yeniden başlatma, public-origin saldırısının reddi, eksik/yanlış CSRF reddi, panel mutation, yayın sayfası, cihaz iptali ve çıkış geçti |
| `TMPDIR=/tmp/opencode /tmp/opencode/obs-panel-test-venv/bin/python tests/browser_refactor.py` | exit 0; gerçek Chrome oluşturma, canlı alanlar/tekrarlar, kapalı alan DOM korunması, tasarım reload, taslak koruma, ortak kilit, 400/409/ağ hatası, silme ve JS kapalı kontroller geçti |
| `go build -o /tmp/opencode/obs-access-reviewed .` | exit 0; proje binary'si yerine ayrı çıktı |

Tam Go/Node/Chrome kontrolleri, son yerel görünürlük ve toplu iptal değişikliklerinden sonra yeniden çalıştırıldı. Daha önce bu görevde 27 testlik panel/runtime turu da geçti; son tur erişim UI testleri dahil 29'dur.

## Sınırlar ve önceki hatalar

- Önceki yerel model hataları/zaman aşımları `20261101T170029Z-pairing-security-blocked.md` içinde korunuyor. Bu yeni çalışma o engelin ardından Terra ile tamamlandı.
- Terra'nın ilk browser fixture'ında Playwright'ın Cookie başlığını göstermemesi nedeniyle assertion hatası oldu; Chrome DevTools ağ olayıyla gerçek aynı isteğin cookie taşıması doğrulandı. Ayrıntı `20261101T205657Z-access-browser-terra.md`. Ana denetçinin iki son tarayıcı çalıştırması da başarılıdır.
- Browser `overlay.test` hostunu loopback'e eşler; bu fiziksel LAN değil, ayrı host/origin simülasyonudur. PNG imzası ve gerçek QR URL navigasyonu doğrulandı; QR görüntüsü kamerayla taranmadı veya decoder ile çözülmedi.
- Başarısız session yazımı testi kapalı store/insert hata yoludur; gerçek SQLite COMMIT arızası bu yeni oturum tablosunda ayrıca enjekte edilmedi. Mevcut sayfa/yayın rollback testleri ayrı olarak sürer.
- Testler geçici veritabanı ve loopback portları kullanır. Mevcut `overlay.db`, çalışan uygulama süreci ve üretim binary'si değiştirilmedi; LAN dinleyicisi açılmadı. Commit, push veya deployment yapılmadı.
