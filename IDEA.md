# OBS Screen Overlay — Proje fikri

## Amaç

OBS Browser Source üzerinde kullanılan ekran tasarımlarını ve widget'ları basit bir yönetim panelinden yönetmek; tasarım içindeki belirli metinleri canlı yayında, sayfayı yenilemeden değiştirebilmek.

Uygulama tasarımı oluşturmaz. Kullanıcı kendi HTML dosyasını hazırlar; sistem bu dosyayı sunar, tanımlanan alanlara veri sağlar ve istenen alanları canlı günceller.

## İki içerik türü

- **Screen Overlay:** Yayın ekranında kullanılan çerçeve ve ekran tasarımları.
- **Widget:** Sayaç, bilgi kutusu veya benzeri bağımsız bileşenler.

İki tür de tam HTML belgesi kabul eder ve aynı altyapıyı kullanır. Her kaydın OBS'ye eklenebilen bağımsız bir bağlantısı bulunur.

## Tek HTML dosyası

Her tasarım daima tek HTML dosyasından oluşur.

Kullanıcı CSS, JavaScript ve SVG içeriğini dosyanın içine gömebilir. Sistem ayrı dosya yükleme, asset yönetimi veya ZIP paketi gerektirmez.

Sayaçlar, animasyonlar ve diğer dinamik davranışlar kullanıcı tarafından HTML içindeki JavaScript ile hazırlanır.

## Düzenlenebilir alanlar

Kullanıcı, panelden veri vermek istediği metinleri yer tutucularla belirtir:

```html
<div>{{konuk_adi}}</div>
<div>{{konuk_unvani}}</div>
```

Yeni sayfa oluşturulurken yalnızca isim, tür ve HTML belgesi alınır; değişken değerleri bu adımda istenmez. Kullanıcı değişken/değer çiftlerini düzenleme ekranında ekler, düzenler ve kaldırır. Arayüzde sabit satır sınırı bulunmaz. HTML'deki yer tutucular düzenleme ekranındaki değişken adlarıyla eşleşir; otomatik keşif zorunlu alan listesi değildir.

OBS'ye sunulan belgede yer tutucular şu yapıya dönüştürülür:

```html
<div><span data-overlay-field="konuk_adi">ÖZBAY</span></div>
<div><span data-overlay-field="konuk_unvani">Konuk</span></div>
```

Aynı alan birden fazla yerde kullanıldığında bütün tekrarlar aynı değeri alır.

Alan değerleri düz metindir; HTML veya JavaScript olarak çalıştırılmaz. İlk sürümde yer tutucular, span eklenebilen normal HTML metin alanlarında desteklenir. JavaScript, CSS, attribute veya SVG text içeriğine doğrudan yer tutucu yerleştirme desteği yoktur.

## Canlı güncelleme

Kullanıcı panelde bir veya birden fazla alanın değerini düzenler ve “Yayına uygula” düğmesine basar.

Sunucu değişiklikleri kaydeder ve SSE üzerinden ilgili OBS sayfalarına iletir. Sayfa yalnızca ilgili span elemanlarının metnini günceller.

Metin değişikliği sırasında sayfa yenilenmez. Sayfanın kendi JavaScript'i ve animasyonları çalışmaya devam eder.

## Alan bazında SSE seçimi

Her alanda “SSE ile canlı güncelle” seçeneği bulunur. Yeni alanlarda bu seçenek varsayılan olarak açıktır.

- Sayfa ilk açıldığında bütün alanlara kayıtlı değerleri yazılır.
- SSE açık alanlar panelden yapılan değişiklikleri canlı alır.
- SSE kapalı alanların içeriğine sistem açılıştan sonra dokunmaz.
- SSE yeniden bağlandığında kapalı alanlar sıfırlanmaz.
- Tam sayfa yenilendiğinde bütün alanlar güncel kayıtlı değerleri alır.
- SSE yeniden açılırsa alanın güncel kayıtlı değeri uygulanır.

Örneğin kullanıcı, sayacın başlangıç değerini panelden sağlayabilir. Bu alan için SSE'yi kapatarak sayacı kendi JavaScript'iyle çalıştırır. Sistem çalışan sayacın üzerine tekrar değer yazmaz.

SSE seçeneği bir erişim veya gizlilik ayarı değildir; otomatik güncelleme davranışını belirler.

## Basit yönetim paneli

Panelde şu işlemler bulunur:

- Screen Overlay ve Widget kayıtlarını listeleme.
- İsim ve tür belirleyerek tek HTML dosyası yükleme.
- Düzenleme ekranında değişken/değer çiftlerini ekleme, düzenleme ve kaldırma.
- Her alan için SSE güncellemesini açma veya kapatma.
- Değerleri tek işlemde yayına uygulama.
- OBS bağlantısını kopyalama ve sayfayı görüntüleme.
- Mevcut tasarımın yerine yeni HTML dosyası yayınlama.

HTML tasarımını değiştirmek, metin güncellemesinden ayrı bir işlemdir. Yeni tasarım yayınlandığında açık sayfalar kontrollü olarak yenilenebilir. Aynı isimli alanların değerleri ve SSE ayarları korunur.

## Teknik yaklaşım

- Go ve Gin ile tek sunucu uygulaması.
- Yönetim panelinde Go html/template ve .tmpl dosyaları.
- Kalıcı veriler için SQLite.
- Canlı veri aktarımı için SSE.
- Panel ve yayın sayfasında sade JavaScript.

Kullanıcının yüklediği HTML, Go template olarak çalıştırılmaz. Yer tutucular uygulamanın kendi alan bağlama sistemiyle işlenir.

Yönetim paneli oturumla korunur. OBS bağlantısı yalnızca ilgili sayfayı görüntüleme yetkisi verir. Kullanıcı HTML'i JavaScript çalıştırabildiği için yayın belgeleri yönetim panelinden farklı bir origin üzerinde sunulur.

## Kapsam dışında

İlk sürümde görsel tasarım editörü, sahne oluşturucu, yerleşik sayaç, animasyon motoru, dosya yöneticisi, widget pazarı veya çok kullanıcılı rol sistemi bulunmaz.

## Başarı ölçütü

Kullanıcı tek HTML dosyasını yükleyip OBS bağlantısını alabilmeli; panelden metinleri yayını kesmeden değiştirebilmeli ve SSE'yi kapattığı alanlarda kendi JavaScript davranışlarını kesintisiz sürdürebilmelidir.

## İlgili dosyalar

- AGENT.md: Hermes'in çalışma kuralları ve uygulama sözleşmeleri.
- PLAN.md: Geliştirme aşamaları ve kabul senaryoları.

Bu dosya projenin amacını ve kapsamını açıklar. Uygulama ayrıntıları AGENT.md ve PLAN.md üzerinden takip edilir.
