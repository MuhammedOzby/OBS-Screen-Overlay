# OBS Screen Overlay — Hermes çalışma talimatı

Bu dosya Hermes için proje talimatıdır. Uygulamaya başlamadan önce bu dosyayı ve `PLAN.md` dosyasını oku. Yalnızca kullanıcının başlatmanı istediği aşamayı uygula; dosyaların verilmesini tek başına tüm projeyi geliştirme veya dağıtma izni sayma.

## 1. Amaç ve kapsam

Go sunucusu üzerinde çalışan, OBS Browser Source için tek dosyalık HTML sayfalarını yöneten basit bir uygulama geliştir.

İki kayıt türü vardır: `screen_overlay` ve `widget`. İkisi de bağımsız, tam HTML belgesidir; aynı veri ve yayın altyapısını kullanır. Tür ayrımı panelde filtreleme ve sınıflandırma içindir. Widget birleştirme, sahne editörü veya sürükle-bırak tasarım sistemi yapma.

Kullanıcı her kayıt için tek `.html` dosyası yükler. CSS, JavaScript ve SVG dosyanın içine gömülebilir. Uygulama ZIP, asset klasörü veya ayrı resim/font yükleme sistemi sunmaz. Harici servisleri otomatik indirme veya HTML'i paketleme. Kullanıcı kendi sayacını, animasyonunu ve diğer davranışlarını HTML içindeki JavaScript ile yazar.

## 2. Onaylanmış teknoloji

- Go ve Gin: HTTP sunucusu, panel ve API.
- Go `html/template` ve `.tmpl`: yalnızca uygulamanın yönetim paneli.
- Yönetim panelinde kullanıcının Bootstrap Kehribar Gece temasını kullan: https://github.com/MuhammedOzby/bootstrap-kehribar-gece . Entegrasyon sırasında tema deposunun belgelerini ve bağımlılıklarını incele; Go template ve vanilla JavaScript mimarisini koru. Temayı kullanıcı tarafından yüklenen OBS HTML belgelerine enjekte etme.
- SQLite: sayfalar, kaynak HTML, alan ayarları ve güncel değerler.
- Vanilla JavaScript: basit panel ve yayın runtime'ı.
- SSE / `EventSource`: sunucudan OBS sayfasına canlı veri.
- Tek uygulama örneği; bellek içi sayfa bazlı abonelik merkezi.

SPA framework, WebSocket, Redis, mikroservis veya kuyruk sistemi ekleme. Bağımlılıkları az tut. Uygulama sırasında kullanacağın sürümlerin resmi belgelerini kontrol et; doğrulanmamış API veya sürüm bilgisi uydurma.

## 3. Alan sözleşmesi

Kaynak HTML örneği:

```html
<div>{{konuk_adi}}</div>
<div>{{sayac_baslangic}}</div>
```

OBS'ye sunulan HTML örneği:

```html
<div><span data-overlay-field="konuk_adi">ÖZBAY</span></div>
<div><span data-overlay-field="sayac_baslangic">300</span></div>
```

- Alan adları ilk sürümde `[a-zA-Z_][a-zA-Z0-9_]*` biçimindedir; büyük/küçük harf duyarlıdır.
- Aynı alanın bütün tekrarları aynı başlangıç değerini ve aynı SSE ayarını kullanır.
- Her alanın `value` ve `sse_enabled` özellikleri vardır. Yeni alanlar için değer boş, SSE açık başlar.
- Kullanıcı değerleri düz metindir. Başlangıç çıktısında escape et; canlı güncellemede `textContent` kullan. `innerHTML` kullanma.
- Kaynak HTML'i Go template olarak parse veya execute etme. `template.HTML` ile panel içine doğrudan yerleştirme.
- HTML ayrıştırıcısı kullan; tüm belge üzerinde regex ile değiştirme yapma. Alan adını doğrulamak için regex kullanılabilir.
- Yalnızca `span` eklenebilen normal HTML metin düğümlerini dönüştür. `script`, `style`, `textarea`, `title`, yorumlar ve attribute değerlerini alan bağlamı olarak işleme. Desteklenmeyen konumlarda alan bekleniyorsa anlaşılır doğrulama hatası ver.
- SVG belgesini koru. SVG `<text>` içine HTML `span` ekleme; bu bağlamda yer tutucu desteği ilk sürümde yoktur. Kullanıcı normal HTML içindeki alanı JavaScript ile okuyarak SVG öğelerini kendisi yönetebilir. HTML namespace içindeki uygun `foreignObject` içeriği ayrıca doğrulanmalıdır.
- Alan dışındaki CSS, JavaScript ve gömülü SVG içeriğini bozma. Derleyici yalnızca alan bağlamalarını ve sistem runtime'ını eklesin.

## 4. SSE kapalı alanların kesin davranışı

Panelde her alan için `SSE ile canlı güncelle` seçimi bulunur. Seçimin kapsamı sayfa değil, alandır.

| Durum | Davranış |
| --- | --- |
| İlk HTTP sayfa açılışı | SSE ayarından bağımsız olarak bütün alanlar kayıtlı değerleriyle doldurulur. |
| SSE açık alan | Panelde uygulanan değer bütün eşleşen span'lere yazılır. |
| SSE kapalı alan | Runtime o alanın DOM içeriğine dokunmaz. |
| SSE kopması ve yeniden bağlanma | Yalnızca halen SSE açık alanlar eşitlenir. Kapalı alanlar sıfırlanmaz. |
| Kapalı alanın panelde değerinin değişmesi | Yeni değer kaydedilir; açık OBS sayfasındaki içerik değişmez. |
| Tam sayfa yenilemesi | Bütün alanlar güncel kayıtlı başlangıç değerleriyle yeniden oluşur. |
| SSE'nin kapatılması | Açık istemcilere yeni politika iletilir; alanın mevcut DOM içeriği korunur. |
| SSE'nin açılması | Alanın güncel kayıtlı değeri açık istemcilere uygulanır. |

SSE kapalı olması gizlilik sınırı değildir; başlangıç HTML'i zaten bu değeri içerir. Bu seçenek otomatik DOM güncelleme davranışını belirler.

Kullanıcı sayaç kodu gibi davranışları kendisi yazar. Yerleşik sayaç, sayaç başlat/durdur/sıfırla komutları veya özel widget SDK'sı ekleme. Alan bağlama için kararlı arayüz `data-overlay-field` attribute'udur.

## 5. Canlı veri ve tutarlılık

- Panelde birden çok alanı düzenle, tek `Yayına uygula` işlemiyle gönder. Her tuş vuruşunu otomatik yayımlama.
- Değerler ve SSE ayarları tek veritabanı işlemiyle kaydedilir. Başarılı commit sonrasında yayın yapılır.
- Her sayfa için artan `revision` kullan. Değişiklikler ve abonelik snapshot'ı aynı sayfa için sıralı olsun; eski bir commit daha yeni durumdan sonra yayınlanmasın.
- İlk sürümde her SSE durum olayında yalnızca SSE açık alanların tam güncel değerlerini gönder. Kapalı alanları `null`, boş metin veya eski değerle gönderme.
- Runtime payload'daki alanları günceller; payload'da bulunmayan DOM düğümlerine dokunmaz. Önceki alan listesini temizleyerek kapalı alanları sıfırlama.
- İlk abonelikte ve yeniden bağlanmada güncel canlı snapshot gönder. HTTP render ile SSE aboneliği arasındaki değişiklik kaybolmasın.
- Snapshot elde etme ve abone olma işlemini tutarlı yap; arada değişiklik kaçırma. Eski/tekrarlanan revision'ları istemcide yok say.
- Runtime ve bağlantı başına kaynakları sayfa kapanışında temizle. Yavaş istemci yayıncıyı bloklamasın; kuyruk taşarsa bağlantıyı kapatıp güncel snapshot ile yeniden bağlanmaya izin ver.
- SSE heartbeat, iptal edilen request context'i, flush ve reverse proxy buffering ayarlarını dikkate al. Bağlantı kopunca sayfayı reload etme.
- Runtime tek bağlantı açsın. Kullanıcı yüklemesi tek dosyadır; sunucunun kendi runtime JS dosyasını HTTP ile sunması bu kuralı ihlal etmez.

## 6. Basit panel

- Tek yönetici oturumu; kayıt olma, ekipler ve rol sistemi yok.
- Screen Overlay / Widget listesi, kayıt oluşturma ve düzenleme.
- İsim, tür, HTML dosyası yükleme/değiştirme.
- Yeni sayfa oluşturma yalnızca isim, tür ve HTML belgesini alır; oluşturma akışında değişken/değer istenmez.
- Değişken/değer çiftleri düzenleme sayfasında kullanıcı tarafından eklenir, düzenlenir ve kaldırılır; arayüzde sabit satır sınırı yoktur. Alan başına SSE seçimi korunur. HTML'deki yer tutucular bu adlarla eşleşir; HTML'den keşif, düzenleme için zorunlu alan listesi değildir.
- `Yayına uygula`, `OBS bağlantısını kopyala` ve sayfayı yeni sekmede görüntüleme.
- Türkçe ve sade arayüz. İlk sürümde gömülü canlı önizleme, kod editörü, tasarım oluşturucu ve asset yöneticisi yok.

## 7. HTML değiştirme

Yeni HTML önce tamamen doğrulanır; hata halinde mevcut çalışan şablon ve değerler korunur. Aynı isimli alanların değerleri ve SSE ayarları korunur; yeni alanlar varsayılanlarla eklenir. Kaldırılan alanlar aktif alan listesinden çıkarılır.

Metin güncellemesi sayfa yenilemez. HTML değişikliği için panelde `Tasarımı yayınla` gibi ayrı ve açık bir işlem kullan. Başarılı tasarım değişikliği açık OBS istemcilerine ayrı `template_changed` olayı gönderip kontrollü reload yaptırabilir. İlk bağlantı/yeniden bağlantı snapshot'ında `template_version` taşı; kopukluk sırasında değişen tasarım da algılansın. Tasarım değişmedikçe SSE reconnect reload sebebi değildir.

## 8. Erişim ve çalışma ortamı

- OBS URL'si tahmin edilemez, sayfaya özel, salt okunur token taşır; token yönetim API'sine yetki vermez.
- Bütün yönetim uçları oturum doğrulaması, değişiklik istekleri ayrıca CSRF koruması gerektirir. Parola ve token'ları loglama veya koda gömme.
- Tam HTML/JS çalıştırıldığı için yönetim paneliyle yayın belgesini farklı origin'lerde sun. Yerel geliştirmede farklı portlar kullanılabilir; cookie'ler portlara göre izole değildir. Admin API'sinde Origin/CSRF doğrulaması ve uygun SameSite cookie politikası uygula, CORS'u gelişigüzel açma.
- Uzak sunucuda admin ve yayın origin'lerini açık yapılandır; admin cookie'sini host-only tut. Yönetim panelinde yüklenen HTML'i aynı origin'de çalıştırma.
- Yerel varsayılan bind loopback olsun. İnternete açma, DNS/TLS değişikliği veya deployment yapma; bunlar ayrı kullanıcı görevidir.
- HTML yükleme boyutunu sınırla, UTF-8/tam belge doğrulaması yap; yüklenen HTML'i kendiliğinden güvenli kabul etme. Kullanıcının JS'ini kaldırarak gereksinimi bozma; izolasyon uygula.

## 9. Çalışma yöntemi

1. Mevcut kaynakları ve `PLAN.md` durumunu incele. Kullanıcının değişikliklerinin üzerine yazma.
2. İstenen aşama için küçük ve çalışır bir değişiklik yap. Görev kapsamını kendi kendine büyütme.
3. İlgili kontrolleri çalıştır; çıktıyı özetle. Çalıştırılmayan testi geçti gibi sunma. Proje boyunca yapılan test ve doğrulama çalıştırmalarını `test_output/` klasöründe Markdown (`.md`) raporları olarak sakla. Her raporda tarih, kapsam, çalıştırılan komutlar, gerçek sonuçlar ve varsa başarısız veya engellenen kontroller yer alsın. Önceki raporların üzerine yazma; her çalışma için ayrı, benzersiz adlı rapor oluştur.
4. Planın kapsamını ve kabul ölçütlerini kullanıcı onayı olmadan değiştirme. Tamamlanan maddeyi işaretlemek ve test kanıtını eklemek kapsam değişikliği değildir.
5. Sonuçta ne değiştiğini, hangi kontrollerin geçtiğini ve sonraki adımı kısa bildir.

### Kalıcı model tercihi — Kullanıcı kararı, 1 Kasım 2026

- Kullanıcının son kararıyla alt ajanlarda **GPT-6.1 Terra** (`openai/gpt-6.1-terra`) kullan. Önceki yalnız yerel Ollama kısıtlaması kaldırılmıştır. Bu karar ana ajanın kendi modelini değiştirmez.
- Bu görevler için yerel model çalıştırma veya model indirme; kullanıcı yerel modellerin zaman aşımı ve hataları nedeniyle Terra'ya geçilmesini istedi.
- Ana ajan uygulama kodu yazmaz; işi planlar, Terra alt ajanının değişikliklerini inceler ve bağımsız test eder. Bulduğu hataların düzeltmesini de alt ajana verir. Alt ajanın tamamlandı bildirimini kaynak ve gerçek testlerle doğrulamadan kabul etme.
- İşleri küçük, sınırları açık parçalara böl; büyük görevi tek seferde gönderme.
- Terra kullanılamıyorsa engeli kullanıcıya bildir; kendiliğinden başka modele geçme veya uygulama kodunu kendin yazma.

Kullanıcı adım adım ilerlemek isterse yalnızca o adımı tamamla. Otomatik commit, push ve dağıtım yapma.

## 10. Zorunlu doğrulama

Alan dönüştürme ve escape davranışı; tekrarlanan alanlar; HTML içi JS/SVG'nin korunması; açık alanın canlı değişmesi; kapalı alanın sayaç tarafından değiştirildikten sonra snapshot/reconnect sırasında korunması; SSE aç/kapat; transaction başarısızlığında yayın yapılmaması; eski revision ve abonelik yarışı; sayfalar arası izolasyon; yetkisiz yönetim erişimi doğrulanmalıdır.

`go test ./...`, `go vet ./...` ve hub eşzamanlılığı için uygun ortamda `go test -race ./...` çalıştır. DOM davranışını gerçek tarayıcı veya uygun browser testiyle doğrula; yalnızca sunucu testini tarayıcı testi olarak sunma. OBS Browser Source üzerinde son kabul kullanıcıyla yapılır.

Dosya adı kullanıcı isteğiyle `AGENT.md` olarak verilmiştir. Hermes'in otomatik okuması garanti edilmez; başlangıç görevinde bu dosyayı ve `PLAN.md` dosyasını açıkça okut.
