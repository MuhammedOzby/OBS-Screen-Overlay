# Kullanıcı kabul testleri

Bu liste gerçek tarayıcı, telefon ve OBS üzerinde kullanıcı tarafından uygulanacak kontrollerdir. Kutular henüz kabul edilmemiştir.

## Hazırlık

- Uygulamayı proje klasöründe `go run .` ile başlat.
- Bilgisayarda `http://127.0.0.1:8080/access` adresinden yerel erişimi başlat.
- Test için ayrı bir kayıt oluştur; önemli tasarımlarını değiştirme.
- Örnek olarak `example/template.html` kullanılabilir. Ayrıntılar: `example/README.md`.
- OBS Browser Source'a panelin verdiği **OBS bağlantısını** ekle; yönetim panelinin adresini ekleme.

## 1. Temel kullanım ve kalıcılık

- [ ] İsim, tür ve tek HTML dosyasıyla kayıt oluşturabiliyorum.
- [ ] Listeleme, tür filtresi ve isim/tür düzenleme çalışıyor.
- [ ] Değişken ekleyebiliyor, adını/değerini değiştirebiliyor ve kaldırabiliyorum.
- [ ] `{{deneme}}` içeren HTML için panelde `deneme` değişkenini tanımlayınca değer yayında görünüyor.
- [ ] Uygulamayı kapatıp aynı veritabanıyla açınca kayıtlar, HTML, değerler ve SSE tercihleri korunuyor.
- [ ] Yalnız test kaydını silince listeden kalkıyor; diğer kayıtlar etkilenmiyor.

## 2. Gerçek OBS Browser Source — canlı değerler

- [ ] SSE açık `deneme` değişkeninin değerini değiştirip **Yayına uygula** dediğimde OBS'de metin değişiyor; sayfa yenilenmiyor.
- [ ] Aynı `{{deneme}}` etiketi iki yerdeyse ikisi de güncelleniyor.
- [ ] Birden fazla değişkeni düzenleyince düğmeye basmadan yayına gitmiyor; uygulayınca birlikte değişiyor.
- [ ] Değere `<b>deneme</b>` yazınca düz metin olarak görünüyor, kalın HTML olarak işlenmiyor.
- [ ] HTML içindeki kendi JavaScript'im ve animasyonlarım metin güncellemesinde kesilmiyor.
- [ ] Başka bir kaydı güncellemek bu OBS kaynağını etkilemiyor.

## 3. SSE kapalı sayaç ve yeniden bağlanma

Bu bölüm için kendi JavaScript'iyle çalışan, başlangıcını bir değişkenden alan sayaç kullan. Sayaç başlangıç alanının SSE seçimini kapat; başka bir metin alanının SSE'si açık kalsın.

- [ ] İlk açılışta sayaç kayıtlı başlangıç değerini alıyor ve çalışıyor.
- [ ] Canlı metni güncellemek çalışan sayacı sıfırlamıyor.
- [ ] Sayaç başlangıç değerini panelden değiştirmek çalışan sayacı etkilemiyor.
- [ ] OBS kaynağını tamamen yenileyince sayaç yeni kayıtlı başlangıç değerini alıyor.
- [ ] Bir alanın SSE'sini kapatınca mevcut görünümü korunuyor; sonraki değer değişiklikleri açık sayfaya uygulanmıyor.
- [ ] SSE'yi yeniden açınca güncel kayıtlı değer uygulanıyor.
- [ ] Sunucuyu kapatıp yeniden başlatınca OBS bağlantısı kendiliğinden toparlanıyor; SSE açık metin son değere geliyor, SSE kapalı çalışan sayaç sıfırlanmıyor.

Not: Aynı bilgisayardaki `127.0.0.1` bağlantısını denemek için Wi-Fi'yi kapatmak yeterli değildir. Sunucuyu durdurup başlatmak bağlantıyı gerçekten keser. Bu sırada OBS kaynağını elle yenileme veya görünürlüğünü değiştirme.

## 4. Tasarım değiştirme

- [ ] Ayrı **Tasarımı yayınla** işlemiyle yeni HTML yükleyince OBS yeni tasarımı kontrollü olarak yüklüyor.
- [ ] Aynı isimli değişkenlerin kayıtlı değerleri ve SSE tercihleri korunuyor.
- [ ] Geçersiz HTML yükleme denemesi hata gösteriyor; önceki çalışan tasarım bozulmuyor.
- [ ] Bağlantısı kesik bir yayın sayfası, bu sırada değişen tasarımı yeniden bağlanınca yüklüyor. Bunu fiziksel LAN cihazında ağ bağlantısını keserek deneyebilirsin; sunucudaki panel açık kalmalı.

## 5. Telefon / ikinci cihaz — fiziksel LAN ve QR

Varsayılan başlatma yalnız bilgisayardan erişilebilir. Bu bölüm için `README.md` içindeki **Güvenilir yerel ağ örneği** ile, gerçek LAN IP adresini kullanarak yeniden başlat. Her iki cihaz aynı güvenilir ağda olsun. HTTP trafiği şifreli değildir; internete açma.

- [ ] Bilgisayarda yerel erişimi başlatıp `/cihazlar` ekranından kod/QR oluşturabiliyorum.
- [ ] Telefon LAN IP:8080 adresini açabiliyor; altı haneli kodla eşleşip paneli yönetebiliyor.
- [ ] Yeni bir eşleştirme oluşturarak telefon kamerasıyla QR tarayınca eşleşebiliyorum.
- [ ] Kullanılmış kod/QR ikinci kez erişim vermiyor.
- [ ] Beş dakikası dolan kod/QR erişim vermiyor; yenisiyle eşleşebiliyorum.
- [ ] Telefon tarayıcısını kapatıp yeniden açınca oturum korunuyor (gizli sekme kullanma, cookie'leri silme).
- [ ] Telefonda çıkış yapınca tekrar yönetim erişimi için eşleştirme gerekiyor.
- [ ] Bilgisayardan telefonun cihaz erişimini iptal edince telefondaki sonraki yönetim isteği reddediliyor.
- [ ] Eşleştirilmemiş başka bir tarayıcı yönetim paneline doğrudan giremiyor.

## Sonuç kaydı

Test tarihi:

İşletim sistemi / tarayıcı / telefon modeli:

OBS sürümü:

Başarısız madde ve tekrar üretme adımları:

Beklenen / gerçekleşen:

Ekran görüntüsü veya hata mesajı:

Paylaşılan görüntülerde QR, eşleştirme kodu, cookie ve OBS bağlantısındaki gizli token'ı gizle. Tüm maddeler doğrulanmadan gerçek cihaz/OBS kabulü tamamlandı sayılmaz.
