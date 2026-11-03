# OBS Screen Overlay — Uygulama planı

Durum: Ana işlevler ve QR/kod ile kalıcı cihaz erişimi uygulandı; son Go/Node/Chromium kontrolleri geçti. Fiziksel LAN, telefonla QR tarama ve kullanıcıyla gerçek OBS Browser Source kabulü bekliyor. Aşağıdaki eski aşama kutuları tarihsel kayıtlarla birlikte değerlendirilmelidir.
Tarih: 2 Kasım 2026.
Çalışma talimatı: `AGENT.md`.

## 1. Hedef

Kullanıcının tek dosyalık HTML tasarımını OBS Browser Source'a sunmak; HTML içindeki `{{alan_adi}}` metinlerini panelden, sayfayı yenilemeden güncellemek. Her alan için SSE güncellemesi kapatılabilmelidir. Kullanıcının kendi JavaScript'iyle yönettiği sayaç gibi öğeler korunmalıdır.

## 2. Kesin kapsam

| Konu | Karar |
| --- | --- |
| Kayıt türleri | Screen Overlay ve Widget; aynı altyapı, bağımsız yayın URL'leri. |
| Girdi | Tek tam HTML; CSS, JS, SVG ve gerekirse data URI içeriğe gömülü. |
| Panel | Go html/template, `.tmpl`, vanilla JS; tek yönetici. |
| Panel teması | Bootstrap Kehribar Gece: https://github.com/MuhammedOzby/bootstrap-kehribar-gece ; yalnızca yönetim panelinde. |
| Sunucu | Go + Gin; tek uygulama örneği. |
| Kalıcılık | SQLite; kaynak HTML de veritabanında. |
| Canlı taşıma | SSE; panelden yazma normal HTTP isteği. |
| Yer tutucu | Normal HTML metnindeki `{{alan_adi}}`. |
| Çıktı | `<span data-overlay-field="alan_adi">değer</span>`. |
| Alan ayarı | Düz metin değer ve `sse_enabled`; varsayılan true. |
| Uygulama | Birden çok alanı tek işlemde `Yayına uygula`. |
| Tasarım güncelleme | Ayrı yayın işlemi; kontrollü sayfa reload'u. |
| Dağıtım | Bu planın geliştirme aşamalarında dış ortama otomatik dağıtım yok. |

Kapsam dışında: yerleşik sayaç/animasyon motoru, widget pazarı, sahne oluşturucu, drag-and-drop editör, dosya/asset yönetimi, ZIP yükleme, SPA, WebSocket, çok kullanıcılı rol sistemi, yatay ölçekleme.

## 3. Temel veri modeli

### pages

`id`, `name`, `type`, `source_html`, `public_token`, `revision`, `template_version`, `created_at`, `updated_at`.

- `type`: `screen_overlay` veya `widget`.
- Kaynak HTML değişmeden saklanır; sunum için derlenmiş belge üretilir.
- Token rastgele ve yüksek entropili oluşturulur. Panelde bağlantının tekrar kopyalanabilmesi için MVP'de token veritabanında saklanır; veritabanı ve yedekleri sır olarak korunur. Token loglara yazılmaz ve yalnızca korumalı panel oturumuna gösterilir.
- `revision` değer/politika ve tasarım değişikliklerinde artar; `template_version` yalnızca tasarım değişince artar.

### fields

`page_id`, `name`, `value`, `sse_enabled`, `display_order`.

- Birincil anahtar: `(page_id, name)`.
- İsim ilk görünme sırasına göre panelde listelenir.
- Aynı isimli bütün span'ler bu tek kaydı kullanır.
- HTML değiştirilince ortak isimlerin değeri ve ayarı korunur.

OBS bağlantısı paylaşılabilir salt okunur bir sırdır ve yönetim yetkisi sağlamaz.

## 4. HTTP sözleşmesi taslağı

| Metot | Yol | İşlev |
| --- | --- | --- |
| GET/POST | `/login` | Yönetici oturumu. |
| POST | `/logout` | Oturumu kapatma. |
| GET | `/admin/pages` | Tür filtresiyle liste. |
| GET | `/admin/pages/new` | Oluşturma formu. |
| GET | `/admin/pages/:id` | Alan ve HTML yönetimi. |
| POST | `/api/pages` | İsim, tür ve HTML ile oluşturma. |
| PATCH | `/api/pages/:id` | İsim/tür güncelleme. |
| PATCH | `/api/pages/:id/state` | Değerler ve SSE ayarlarını atomik uygulama. |
| POST | `/api/pages/:id/template` | Doğrulanmış yeni HTML'i yayınlama. |
| GET | `/view/:token` | OBS'ye sunulan tam HTML. |
| GET | `/view/:token/events` | Sayfanın salt okunur SSE akışı. |
| GET | `/system/overlay-runtime.js` | Uygulamanın kendi minimal runtime dosyası. |

Admin yolları admin origin'inde; view/events/runtime yayın origin'inde sunulur. Aynı Go sürecinde ayrı listener/router kullanılabilir. Her mutation'da oturum, CSRF ve alan sahipliği doğrulanır.

Örnek durum isteği:

```json
{
  "expected_revision": 7,
  "fields": {
    "konuk_adi": {"value": "ÖZBAY", "sse_enabled": true},
    "sayac_baslangic": {"value": "300", "sse_enabled": false}
  }
}
```

Bilinmeyen alanları ve hatalı tipleri reddet. Beklenen revision uyuşmazsa `409 Conflict` dön; panel kullanıcının taslağını sessizce ezmeden güncel veriyi yeniden yükleme imkânı versin. Uygulama başarısını yalnızca transaction commit edildikten sonra bildir.

## 5. Yayın ve runtime sözleşmesi

HTTP render bütün alanların başlangıç değerlerini güvenli biçimde yazar. Runtime başlangıç `revision` ve `template_version` bilgisini güvenli JSON veri bloğundan veya sistem metadata'sından alır. JSON yerleştirmede `</script>` gibi değerlerle bloktan çıkışa izin verme.

SSE örneği:

```text
event: state
id: 8
data: {"revision":8,"template_version":1,"fields":{"konuk_adi":"Yeni konuk"}}

```

`fields` her olayda yalnızca SSE açık alanların tam güncel haritasıdır. SSE kapalı `sayac_baslangic` burada bulunmaz. Boş harita bile geçerlidir ve DOM temizleme anlamına gelmez.

Runtime:

1. Yeni tasarım sürümü yoksa DOM'u yeniden oluşturmaz.
2. Eski/tekrarlanan revision'ı yok sayar.
3. Payload'da bulunan her alanın bütün `span[data-overlay-field]` eşleşmelerini `textContent` ile günceller.
4. Payload'da olmayan alanı değiştirmez; kaldırmaz veya ilk değerine döndürmez.
5. SSE hatasında `EventSource` bağlantısının toparlanmasına izin verir, sayfayı reload etmez.
6. Açık `template_changed` olayı veya snapshot'ta farklı `template_version` olduğunda bir kez kontrollü reload yapar.

Alanları bir kez indeksleyip düğüm referansları üzerinden güncellemek uygundur. Aynı metin tekrar geldiyse gereksiz DOM yazımından kaçın. Kullanıcının bağlanan span'i kaldırması veya bütünüyle değiştirmesi ilk sürümün desteklediği bir davranış değildir.

Abone olma ve snapshot üretimi sayfa bazında sıralanır. Commit, revision ve yayın sırası tutarlı tutulur. Yeniden bağlanmada event geçmişini yeniden oynatmak zorunlu değildir; güncel canlı snapshot yeterlidir. Kapalı alanlar hiçbir reconnect snapshot'ına eklenmez.

## 6. Aşamalı geliştirme

Her aşama ayrı kullanıcı görevi olarak başlatılabilir. Tamamlanan maddeyi kanıtıyla işaretle. Kapsam/teknoloji değişikliklerini uygulamadan önce kullanıcıyla kararlaştır.

### Aşama 1 — İskelet ve kalıcılık

- [ ] Go modülü, Gin başlangıcı ve sade paket yapısını oluştur.
- [ ] Admin/yayın adreslerini, veri yolunu ve yönetici sırrını yapılandır.
- [ ] SQLite migration ve repository katmanını yaz.
- [ ] Sayfa ve alan kayıtlarının transaction içinde saklanmasını kur.
- [ ] Yönetici oturumu, CSRF ve admin/yayın router ayrımını kur.

Kabul: Uygulama yerelde çalışır; veri yeniden başlatma sonrası korunur; oturumsuz mutation reddedilir. Yayın origin'inden admin mutation yetkisi elde edilemez.

### Aşama 2 — Tek HTML ve alan derleyicisi

- [ ] HTML boyut/UTF-8/belge doğrulamasını ekle.
- [ ] Metin düğümlerinden alanları keşfet ve tekrarsız listele.
- [ ] Güvenli span dönüşümü, başlangıç değerleri ve runtime eklemeyi yaz.
- [ ] Desteklenmeyen bağlamları anlaşılır hata ile bildir; gömülü JS/CSS/SVG'yi koru.
- [ ] HTML değiştirmede ortak alan ayarlarını koru; hata halinde eski kaydı koru.

Kabul: Birden çok ve tekrarlanan alan doğru oluşur. `<`, `&`, tırnak ve HTML benzeri değerler yalnızca metindir. SVG kendi halinde çalışır; SVG `<text>` içine span yerleştirilmez.

### Aşama 3 — Atomik durum ve SSE

- [ ] Revision kontrollü durum güncelleme API'sini yaz.
- [ ] Sayfa bazlı hub, tutarlı ilk snapshot ve SSE akışını kur.
- [ ] Runtime'da yalnızca SSE açık alanları güncelle.
- [ ] SSE aç/kapat politikasını bütün bağlı istemcilere uygula.
- [ ] Heartbeat, reconnect, yavaş istemci ve disconnect temizliğini tamamla.
- [ ] Tasarım sürümü ve kontrollü reload davranışını ekle.

Kabul: İki açık sekmede aynı alan anlık değişir. Kapalı alanın JS ile değiştirilmiş içeriği; başka alan güncellenince, SSE kopunca veya yeniden bağlanınca bozulmaz. Bağımsız sayfalara veri sızmaz.

### Aşama 4 — Basit yönetim paneli

- [ ] Türkçe `.tmpl` liste, oluşturma ve düzenleme ekranlarını yap.
- [ ] Bootstrap Kehribar Gece temasını yönetim paneline entegre et; kaynak: https://github.com/MuhammedOzby/bootstrap-kehribar-gece .
- [ ] Screen Overlay / Widget filtresini ekle.
- [ ] Oluşturmada yalnızca isim/tür/HTML al; değişken formunu bu akışa ekleme.
- [ ] Düzenleme ekranında kullanıcı tanımlı değişken/değer çiftlerini ekleme, düzenleme ve kaldırmayı bağla; sabit arayüz satır sınırı koyma. HTML yer tutucuları bu adlarla eşleşir; keşif zorunlu alan listesi değildir.
- [ ] Alan başına SSE seçimi ve tek `Yayına uygula` düğmesini ekle.
- [ ] OBS URL kopyalama ve yeni sekmede görüntülemeyi ekle.
- [ ] Yeni HTML için ayrı `Tasarımı yayınla` işlemini ekle.
- [ ] Başarı, doğrulama hatası ve revision çatışmasını sade göster.

Kabul: Kullanıcı tek HTML ile kayıt oluşturur, URL'yi OBS'ye ekler ve panelden canlı metin değiştirir. Yönetim paneli Bootstrap Kehribar Gece temasını kullanır; tema yüklenen OBS HTML belgelerine eklenmez. Sayaç yönetimi gibi ek ürün özellikleri yoktur.

### Aşama 5 — Kabul, hata senaryoları ve kullanım notları

- [ ] Go test/vet ve hub race kontrollerini çalıştır.
- [ ] Gerçek tarayıcıda DOM davranışını ve reconnect'i doğrula.
- [ ] Tek HTML örneğiyle konuk adı + SSE kapalı sayaç senaryosunu göster.
- [ ] Kullanıcıyla OBS Browser Source kabulünü yap.
- [ ] Yerel çalıştırma, yapılandırma, kalıcı veri, token gizliliği ve proxy SSE ayarlarını kısa README'ye yaz.

Kabul: Aşağıdaki senaryoların sonucu kayıtlıdır. OBS testi yapılmadıysa açıkça bekliyor yazılır; proje tamamlandı denmez.

## 7. Kabul senaryoları

| No | İşlem | Beklenen |
| --- | --- | --- |
| 1 | Sayfayı ilk aç | Açık ve kapalı bütün alanlar kayıtlı değerlerini alır. |
| 2 | Konuk adı ve unvanını birlikte uygula | Tek revizyonda kaydolur ve açık sayfalarda yenilemesiz görünür. |
| 3 | Aynı alan HTML'de iki kez bulunsun | Her iki span güncellenir. |
| 4 | SSE kapalı sayacı JS ile çalıştır | Runtime sayacın üzerine yazmaz. |
| 5 | Sayaç çalışırken başlangıç değerini panelde değiştir | Kayıt değişir; çalışan sayaç etkilenmez. Tam reload yeni başlangıcı alır. |
| 6 | Bağlantıyı kes/aç; bu sırada değerleri değiştir | Canlı alanlar son duruma gelir, kapalı sayaç sıfırlanmaz. |
| 7 | Bir alanın SSE'sini kapat | Mevcut içerik korunur; sonraki durum olayları alanı değiştirmez. |
| 8 | SSE'yi yeniden aç | Güncel kayıtlı değer uygulanır. |
| 9 | Commit başarısız olsun | Başarılı yanıt/yayın üretilmez; eski kayıt korunur. |
| 10 | Eşzamanlı/eskimiş revision gönder | Sessiz veri ezilmez; çatışma bildirilir. |
| 11 | Başka sayfanın token'ıyla bağlan | Yalnızca o token'ın sayfası okunur; yönetim yetkisi yoktur. |
| 12 | Alan değerine HTML/JS benzeri metin yaz | Metin görünür, kod olarak çalışmaz. Kullanıcının asıl HTML JS'i çalışmaya devam eder. |
| 13 | HTML'i değiştir | Ortak alan ayarları korunur; tasarım sürümü değişir ve açık sayfa kontrollü yenilenir. |
| 14 | Tasarım değişirken OBS bağlantısı kopuk olsun | Reconnect snapshot'ı sürüm farkını gösterir; yeni tasarım yüklenir. |
| 15 | Yavaş istemci ve sayfa kapanışı | Diğer istemciler bloklanmaz; goroutine/abonelik sızıntısı oluşmaz. |

## 8. Hermes başlangıç görevi

```text
Önce AGENT.md ve PLAN.md dosyalarını oku. Henüz kod yazmadan mevcut dizini incele ve Aşama 1 için uygulayacağın küçük görevleri kısaca bildir. Kullanıcının Aşama 1'i başlatma talimatı geldiyse yalnızca bu aşamayı uygula, ilgili kontrolleri çalıştır ve tamamlanan maddeleri kanıtıyla işaretle. Kapsamı değiştirme ve sonraki aşamalara kendiliğinden geçme. Commit, push veya deployment yapma.
```

## 9. İlerleme kaydı

### 31 Ekim 2026 — İlk çalışan sürüm

- Aşama 1–4 uygulandı: ayrı admin/yayın listener'ları, SQLite kalıcılığı, yönetici oturumu/CSRF, tek HTML doğrulama ve derleme, revision kontrollü atomik durum, sayfa bazlı SSE ve mevcut temaya bağlı gerçek yönetim işlemleri.
- Aşama 5'in otomatik kontrolleri ve README/örnek HTML tamamlandı. Gerçek OBS Browser Source kabulü henüz yapılmadı; bu madde bekliyor.
- Kanıt: `test_output/20261031T043033Z-v1-integration.md`. Cache'siz Go testleri, vet, race, build, altı frontend testi ve gerçek Chromium kabul akışı geçti. Aynı deneme veritabanıyla süreç yeniden başlatılarak kalıcılık doğrulandı.
- Bağlantı gerçekten kesilmişken değer/tasarım yayınlama ve reconnect davranışı test HTTP proxy'siyle doğrulandı. Masaüstü, kısa yatay ve portre ekran düzenleme geometrisi kontrol edildi.
- Commit, push veya deployment yapılmadı. İlk sürüm etiketi kullanıcı kabulünden sonra kararlaştırılacak.

### 31 Ekim 2026 — Mevcut dizinde demo panel çalışması

- Yukarıdaki ilk çalışan sürüm kaydı mevcut dosya ağacının durumunu temsil etmiyor: bu çalışma başlangıcında yalnız iskelet ve eksik demo Go yardımcıları vardı. Önceki kayıt silinmedi; tam backend'in hazır olduğu varsayılmamalı.
- Kullanıcı isteğiyle yalnız yönetim paneli demosu yapıldı: Kehribar Gece temalı liste, oluşturma ve düzenleme ekranları; tek `.html` dosyası seçimi; değişken ekleme/kaldırma ve alan bazlı SSE seçim arayüzü.
- Oluşturma yalnız isim/tür/HTML alır. Demo verileri kaydedilmez, gerçek yayın URL'si veya SSE bağlantısı üretilmez.
- Uygulama Qwen'e tek dosyalık küçük işler halinde yaptırıldı; ana denetçi Go kontrolleri ve gerçek Chrome/Playwright denetimini yaptı. Kanıt: `test_output/20261031-demo-panel-browser-acceptance.md`.
- Bu demo, Aşama 1–3 veya Aşama 4'ün gerçek backend'e bağlanma kabulünü tamamlamaz. OBS kabulü bekliyor.

### 31 Ekim 2026 — Kullanıcı kararı — Güvenlik işleri sona ertelendi

- Kullanıcı güvenlik uygulamasını backend geliştirmesinin son aşamasına erteledi. Yönetici oturumu, parola yönetimi, CSRF ve Origin doğrulaması kalan işler olarak saklanacak; kullanıcı kalan işleri sorduğunda bildirilecek. Bu karar güvenlik gereksinimlerini iptal etmez. Geliştirme loopback üzerinde sürer; güvenlik tamamlanmadan dış erişime açılmaz.

### 1 Kasım 2026 — SQLite temel katmanı

- internal/store içinde SQLite bağlantısı, sürümlü transaction migration, pages/fields şeması
- atomik CreatePage ve tutarlı GetPage eklendi
- Değişkenler ayrı satır; sayfa başına sabit sayı sınırı yok
- 10.000 alanın kapanış/yeniden açılış sonrası tüm değer/SSE/sıra bilgileriyle korunması test edildi
- Ana denetçi: go test -count=1 -v ./..., go vet ./..., go test -race -count=1 ./... hepsi geçti
- Kanıt: test_output/20261101-sqlite-foundation-review.md
- Panel hâlâ demo, main/routes bağlanmadı; listeleme/güncelleme/silme repository ve uygulama entegrasyonu bekliyor

### 1 Kasım 2026 — Yeni kayıt backend

- `POST /api/pages` gerçek backend'e bağlandı: multipart isim/tür/tek HTML; UTF-8, tam belge, 2 MiB ve placeholder konum doğrulaması; rastgele id/token; Fields boş başlar.
- Listeleme/tür filtresi ve düzenleme ekranının okuma tarafı gerçek veriyle çalışıyor; form JS'i fetch hata/başarı durumlarını gösteriyor.
- `example/template.html`, `variables.md` ve README hazır.
- Ana denetçi: Go test/vet/race ve gerçek Chromium kalıcılık/validation denetimi geçti.
- Kanıt: `test_output/20261101-create-page-backend-review.md`.
- Sonraki işler: güncelleme/silme, template publish, SSE, OBS kabulü ve ertelenmiş güvenlik işleri.

### 1 Kasım 2026 — Düzenleme ekranının kayıt bağlantısı

- İsim/tür ve kullanıcı tanımlı değişken/değer/SSE tercihleri `PATCH /api/pages/:id` ile tek transaction'da kaydediliyor. Değişken ekleme, ad değiştirme ve kaldırma destekleniyor; sabit satır sayısı sınırı yok.
- Revision çatışmasında 409 ve taslağı koruyan arayüz; güncel kaydı yüklemek için açık kullanıcı işlemi. HTML kaynağı/token/tasarım sürümü değişmez.
- Tasarım yayını ve OBS bağlantıları henüz hazır olmadığından devre dışı; kayıt başarı mesajı canlı yayın başarısı iddia etmiyor.
- Ana denetçi Go test/vet/race/build, JS sözdizimi ve gerçek Chromium kontrollerini çalıştırdı; hepsi geçti. Atomik rollback, kalıcılık, 10.000 alan, boş alan listesi, iki sekme çatışması ve mobil görünüm doğrulandı.
- Kanıt: `test_output/20261101-edit-page-backend-review.md`.
- Sonraki işler: kayıt silme, HTML değiştirme/tasarım yayınlama, alan derleme ve SSE/OBS yayını ile ertelenen güvenlik işleri. Gerçek OBS kabulü bekliyor.

### 1 Kasım 2026 — Bağımsız HTML alan derleyicisi

- Bu konuşmaya özel kullanıcı tercihiyle GPT-6.1 Terra kullanıldı; kalıcı model tercihi değiştirilmedi.
- `internal/upload.CompileHTML` tam belgeyi doğrular, alanları ilk görünüm sırasıyla tekrarsız keşfeder ve bütün tekrarları escape edilmiş düz metin değerleriyle `span[data-overlay-field]` olarak üretir. Eksik değerler boş, kullanılmayan panel değişkenleri serbesttir.
- Alan dışındaki kaynak baytları korunur; desteklenmeyen bağlamlar ve entity ile kodlanmış yer tutucular reddedilir. Yükleme doğrulaması derleyiciyle aynı bağlam kontrollerini kullanır.
- Ana denetçi `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...` çalıştırdı; hepsi geçti. Kanıt: `test_output/20261101T034257910131Z-html-compiler-review.md`; uygulama test raporu: `test_output/20261101-html-compiler-terra.md`.
- Runtime enjeksiyonu, yayın/SSE entegrasyonu ve tasarım değiştirme bu görevde yapılmadı. Tarayıcı ve OBS kabulü yapılmadı; Aşama 2 bütünüyle tamamlanmış değildir.

### 1 Kasım 2026 — OBS bağlantısı panel entegrasyonu

- Düzenleme ekranında kayıtlı public token ve açıkça yapılandırılmış `PUBLIC_ORIGIN` ile OBS bağlantısı gösterilir; kopyalama, elle seçim yedeği ve güvenli yeni sekme bağlantısı eklendi. Kaydedilmiş değerler açılış/yenilemede görünür; SSE henüz bağlı değildir.
- Go testleri, vet, race, JavaScript sözdizimi ve geçici veritabanıyla gerçek headless Chrome kontrolü geçti. Pano başarı ve hata durumları tarayıcıda stub ile denendi; işletim sistemi panosu ve OBS kabulü test edilmedi. Kanıt: `test_output/20261101-panel-public-url-terra.md`.
- Oturum/CSRF/Origin güvenliği, SSE ve gerçek OBS kabulü bekliyor. Bu konuşmaya özel GPT-6.1 Terra tercihi kullanıldı; kalıcı model tercihi değiştirilmedi.

### 1 Kasım 2026 — Kayıt silme ve temel sayfa yönetimi

- Liste Sil düğmesi onaydan sonra `DELETE /api/pages/:id` çağırır; kayıt ve bağlı değişkenleri mevcut FK CASCADE ile atomik kaldırır. Başarı 204, bulunamayan kayıt 404, beklenmeyen DB hatası genel 500 döner.
- İptalde istek yapılmaz; hata halinde satır korunur ve yeniden deneme açılır. Başka sekmede silinen kayıt listeden temizlenir. Filtre ve boş liste görünümü korunur. JavaScript olmadan silme devre dışıdır.
- Temel oluşturma/listeleme/filtreleme/düzenleme/silme akışları tamamlandı. HTML değiştirme ve tasarım yayınlama bu silme görevinin kapsamında değildir; ilgili önceki kalan işler iptal edilmemiştir.
- Ana denetçi tam Go test/vet/race/build, JavaScript sözdizimi ve gerçek Chromium kontrollerini çalıştırdı; hepsi geçti. Cascade atomikliği, hata/iptal/404, kalıcılık ve başka sayfaya dokunulmaması doğrulandı.
- Kanıt: `test_output/20261101-delete-page-backend-review.md`.
- HTML alan derleyicisi veya yayın tarafına bu görevde müdahale edilmedi; diğer konuşmaların değişiklikleri korundu. Güvenlik ve OBS kabulü hâlâ ayrı işlerdir.

### 1 Kasım 2026 — SSE panel bağlantısı ve canlı akış kabulü

- Kullanıcının bu görev için istediği DeepSeek V4.1 Flash Free ile küçük parçalar halinde çalışıldı. Model bu ortamda OpenRouter yerine `opencode/deepseek-v4.1-flash-free` olarak erişilebilirdi; fark bildirildi. Kalıcı model tercihi değiştirilmedi.
- Dosyalarda mevcut hub/store/runtime altyapısı korundu. Gerçek panel router'ına eksik `PATCH /api/pages/:id` kaydı eklendi; plan taslağındaki `/state` yerine panelin mevcut yolu kullanılıyor. Başarılı transaction sonrası SSE açık alanlar yayınlanıyor.
- Panel JS'ini durduran kaçışlı/tekrarlı template bloğu, bir kopyanın mantığı korunarak onarıldı. SSE açıklamaları ve `Yayına uygula` düğmesi güncellendi. Tasarım yayınlama özelliğinin kabulü bu göreve dahil değildir.
- Gerçek HTTP üzerinden iki abone, SSE aç/kapat, kapalı alanların payload dışında kalması, 409 sonrası yayın olmaması ve sayfa izolasyonu test edildi. Store snapshot/subscribe yarışı, transaction rollback ve silme abonelik temizliği testleri eklendi.
- Ana denetçi test/vet/race/build ve gerçek Chrome kontrollerini çalıştırdı. İki sayfada yenilemesiz metin değişimi, tekrarlanan alanlar, düz metin güvenliği ve JS ile değiştirilmiş kapalı alanın korunması doğrulandı. Proxy ile bağlantı gerçekten kesilip yeniden açıldığında son değer geldi; kapalı alan ve belge korunarak reload olmadığı doğrulandı.
- Kanıt: `test_output/20261101-sse-deepseek-main-review.md`; ilk başarısız test düzeneği denemeleri de raporda açıklanmıştır.
- Gerçek OBS Browser Source kabulü, HTML/tasarım değiştirme akışının tamamlanması ve ertelenen oturum/parola/CSRF/Origin güvenliği ayrı işler olarak bekliyor. Dış erişim, commit, push veya deployment yapılmadı.

### 1 Kasım 2026 — Refaktör ve test güvenilirliği denetimi

- Kullanıcının bu göreve özel serbest model seçimiyle önce Flint 40B Free, kota sonrası Ollama modelleri kullanıldı; kalıcı model tercihi değiştirilmedi. Alt model bildirimleri bağımsız kaynak incelemesi ve gerçek testlerle denetlendi; hatalı ve yarım çıktılar kabul edilmedi.
- Ortak HTML yükleme yardımcıları, store tasarım yayını cursor/transaction temizliği, runtime bağlantı/reload yardımcıları ve kullanılmayan demo kodu temizlendi. Şema, bağımlılıklar ve hub altyapısı korunuyor.
- Panelin mevcut tasarım yayınlama kontrolleri JavaScript ile etkinleşiyor. POST/PATCH ortak kilidi ve hata sonrası taslak/dosya koruma düzeltildi. Yeni kayıt navigasyonu boyunca çift gönderim kilidi korunuyor.
- Yorum/metin sayan panel testi yerine gerçek olay işleyicilerini çalıştıran testler eklendi. Gerçek COMMIT başarısızlığı/rollback/no-event, tam kayıt korunması, eşzamanlı tek kazanan, disk upload temizliği ve hata yanıtları doğrulandı. Önceki raporlar silinmedi.
- Ana denetçi: 26 Node davranış testi, cache'siz tam Go testleri, vet, race ve gerçek Chromium kontrolleri geçti. Tasarım yayını/reload, kaydedilmemiş taslağın yayınlanmaması, karşılıklı kilit, hata/çatışma ve gerçek SSE reconnect doğrulandı. Kanıt: `test_output/20261101-refactor-main-review.md`; başarısız ara denemeler ayrıca raporlandı.
- Gerçek OBS Browser Source kabulü ve ertelenen oturum/parola/CSRF/Origin güvenliği hâlâ bekliyor. Mevcut çalışan süreç yeniden başlatılmadı; dış erişim, commit, push veya deployment yapılmadı.

### 1 Kasım 2026 — Kullanıcı kararı — Kalıcı yerel alt ajan model seçimi

- Alt ajan modelini seçme yetkisi kalıcı olarak ana ajana verildi; önceki sabit Qwen tercihi kaldırıldı. Alt ajanlar yalnız yerel Ollama modelleriyle çalışır; ana ajanın kendi modeli bu kararla değişmez.
- Ana ajan gerektiğinde uygun Ollama modellerini ayrıca onay istemeden indirebilir ve yerel modeller arasında geçiş yapabilir. Bu izin gereksiz indirme veya bulut sağlayıcı kullanımı izni değildir.
- Ana ajanın planlama, kaynak incelemesi ve bağımsız test sorumluluğu korunur. Ayrıntılı kalıcı talimat `AGENT.md` içindeki model tercihi bölümünde güncellendi. Bu karar kaydında uygulama kodu değiştirilmedi ve model indirilmedi.

### 1 Kasım 2026 — QR/kod ile cihaz eşleştirme kararı ve engellenen başlangıç

- Kullanıcı hesap/parola yerine QR içine gömülü eşleştirme bilgisi veya altındaki altı haneli kodla erişim istedi. LAN IP:PORT üzerinden ikincil cihaz da kodla eşleşecek ve kalıcı oturum açacak. Önceki parola yönetimi işi bu kullanıcı kararıyla değişti; CSRF/Origin ve oturum güvenliği gereksinimleri sürüyor.
- Yalnız SQLite v2 migration ve `access_sessions` tablosu eklendi; mevcut migration test beklentileri güncellendi. Oturum repository, QR/kod, kalıcı cookie ve HTTP güvenlik koruması henüz uygulanmadı. **Uygulama hâlâ korumasızdır; LAN'a açılmadı.**
- Yerel alt ajanlarda hatalı ilk çıktılar kabul edilmedi; tekrar eden model yanıt zaman aşımları nedeniyle çalışma durdu. Ana ajan uygulama kodunu kendisi yazmadı ve bulut alt ajana geçmedi.
- Ana denetçi son kaynakta `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...` çalıştırdı; hepsi exit 0. QR/tarayıcı/LAN/OBS kabulü yapılmadı. Kanıt: `test_output/20261101T170029Z-pairing-security-blocked.md`.

### 1 Kasım 2026 — Kullanıcı kararı — GPT-6.1 Terra

- Kullanıcı bundan sonra alt ajanlarda GPT-6.1 Terra kullanılmasını istedi; önceki yalnız yerel Ollama zorunluluğu kaldırıldı. Model `openai/gpt-6.1-terra` olarak doğrulandı.
- Ana ajan planlama, kaynak incelemesi ve bağımsız test görevlerini sürdürür; uygulama kodu Terra alt ajanına verilir. Yarım kalan QR/kod eşleştirme ve kalıcı cihaz oturumu çalışmasına devam ediliyor.

### 2 Kasım 2026 — QR/kod ile kalıcı cihaz erişimi

- Kullanıcının istediği hesap/parolasız akış tamamlandı: ana bilgisayarda `127.0.0.1` üzerinden yerel panel erişimi; `/cihazlar` ekranında QR ve altındaki altı haneli kod; LAN cihazında kodla veya QR URL'siyle eşleştirme; 180 günlük kalıcı cihaz oturumu.
- Kod/QR beş dakika ve tek kullanımlık; başarısız deneme limitleri, hash ile kalıcı oturum kaydı, HttpOnly/Strict cookie, çıkış ve cihaz iptali eklendi. Toplu iptal bekleyen eşleştirmeyi de geçersiz kılar.
- Yönetim uçları oturum, mutation'lar tam Origin/CSRF ile korunur. Yerel yetki gerçek bağlantı/Host üzerinden doğrulanır; forwarded başlıkları kullanılmaz. Loopback dışı bind açık `ADMIN_ORIGINS` gerektirir. Kullanıcı kararı gereği parola yönetimi eklenmedi.
- Uygulama işleri Terra'ya küçük parçalarda verildi; ana denetçi kaynakları inceleyip düzeltmeler istedi ve son kaynakta test/vet/race/build, 29 Node testi ve iki gerçek Chromium akışını bağımsız çalıştırdı; hepsi geçti. Kanıt: `test_output/20261101T210207Z-access-main-review.md` (son tur başlangıcı UTC 1 Kasım).
- `README.md` yerel/LAN yapılandırmasını ve HTTP LAN trafiğinin şifreli olmadığını açıklar. Varsayılan yerel bind korundu; mevcut çalışan süreç yeniden başlatılmadı ve LAN erişimi açılmadı.
- Fiziksel LAN, telefon kamerasıyla QR tarama ve gerçek OBS Browser Source kabulü hâlâ bekliyor. İki host/origin tarayıcı simülasyonu fiziksel LAN kabulü değildir. Commit, push ve deployment yapılmadı.
