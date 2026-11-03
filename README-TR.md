# OBS Screen Overlay

> **AI ile geliştirme:** Bu proje, yapay zekâ araçlarının desteğiyle geliştirilmiştir.

[English](README.md) | **Türkçe**

OBS Browser Source için tek dosyalık HTML overlay ve widget'larını yöneten Go ve SQLite uygulaması. HTML belgesini yükleyin, Türkçe yönetim panelinden metin alanlarını düzenleyin ve Server-Sent Events (SSE) ile canlı uygulayın.

Her overlay veya widget'ın kendi salt okunur OBS bağlantısı vardır. Hesap/parola akışı yoktur; cihazlara yerel erişim veya QR/kod eşleştirmesiyle yetki verilir. OBS LAN Controller'dan ayrı bir uygulamadır; onun canlı önizleme eklentisini gerektirmez.

## Özellikler ve sınırlar

- En fazla 2 MiB boyutunda tek, tam, UTF-8 `.html` dosyası yüklenir. CSS, JavaScript ve SVG dosyaya gömülebilir.
- Desteklenen normal HTML metinlerinde `{{guest_name}}` gibi yer tutucular kullanılır. Değerler ve alan başına canlı güncelleme ayarı panelden düzenlenir.
- **Yayına uygula**, değerleri ve canlı güncelleme ayarlarını birlikte kaydeder. SSE açık alanlar OBS sayfası yenilenmeden değişir.
- SSE kapalı alanın mevcut DOM içeriği, kendi JavaScript'inizin yaptığı değişiklikler dahil korunur. Kayıtlı değer tam sayfa yenilemesinde yüklenir; SSE açıldığında güncel kayıtlı değer uygulanır.
- **Tasarımı yayınla**, değiştirilen HTML'i ayrı işlemle yayınlar ve açık overlay sayfalarını yeniler.
- HTML, alan ayarları ve cihaz oturumları SQLite'ta saklanır.

Yerleşik sayaç motoru, sürükle-bırak tasarımcı, asset yöneticisi, ZIP yükleme veya ayrı resim/font yükleme sistemi yoktur. Yalnız güvendiğiniz HTML/JavaScript yükleyin. Bu sürüm yerel/güvenilir LAN kullanımını hedefler; fiziksel LAN, telefon kamerasıyla QR tarama ve gerçek OBS Browser Source kabulü kayıtlı test planında hâlâ açıktır.

## Gereksinimler

- `go.mod` dosyasının belirttiği **Go 1.27.3 veya üzeri**.
- `github.com/mattn/go-sqlite3` için **CGO açık olmalı ve GCC bulunmalı**. `go env CGO_ENABLED` çıktısı `1` olmalıdır.
- Proje kökünden çalıştırın. `configuration.yaml`, `template/` ve varlıkları çalışma dizininden okunur. Yalnız derlenmiş uygulama dosyası tam dağıtım değildir.
- JavaScript testleri için Node.js isteğe bağlıdır. Python, Playwright ve Chrome tarayıcı testleri içindir; uygulamanın çalışma bağımlılıkları değildir.

## İndirme ve çalıştırma

```sh
git clone https://github.com/MuhammedOzby/OBS-Screen-Overlay.git
cd OBS-Screen-Overlay
go run .
```

Ya da aynı dizinde derleyip çalıştırın:

```sh
go build -o obs-screen-overlay .
./obs-screen-overlay
```

1. Sunucu bilgisayarında `http://127.0.0.1:8080/access` açın ve **Bu bilgisayarda paneli aç** seçeneğini kullanın.
2. Overlay veya widget oluşturup HTML dosyanızı yükleyin. Basit örnek: `example/template.html`.
3. Düzenleme sayfasında değişken adlarını ve değerlerini ekleyin/düzenleyin; **Yayına uygula** düğmesine basın.
4. Paneldeki OBS bağlantısını kopyalayıp OBS'de **Browser Source (Tarayıcı)** kaynağına yapıştırın. Kaynak boyutunu tasarımınıza göre ayarlayın.

Yönetim paneli `http://127.0.0.1:8080`, overlay sayfaları ayrı origin olan `http://127.0.0.1:8081` üzerinden sunulur. Ayrıntılı örnek: [example/README.md](example/README.md).

## Yapılandırma

Çalışma dizinindeki `configuration.yaml` zorunludur. Uygulama ayarları **ortam değişkenlerinden okunmaz**.

Depodaki dosya yalnız loopback kullanır:

```yaml
addr: "127.0.0.1:8080"
public_addr: "127.0.0.1:8081"
public_origin: "http://127.0.0.1:8081"
admin_origins: []
db_path: "overlay.db"
```

- `addr` ve `public_addr`: yönetim ve overlay dinleme adresleri.
- `public_origin`: OBS bağlantıları için dışarıdan erişilebilen overlay origin'i; yol eklemeyin.
- `admin_origins`: yol içermeyen tam yönetim origin izin listesi. Boş liste ve loopback bind için yönetim portuna göre `http://127.0.0.1:8080` ve `http://localhost:8080` otomatik kullanılır. Dolu liste bu varsayılanların yerine geçer.
- `db_path`: mutlak değilse çalışma dizinine göre SQLite yolu. Veritabanı yoksa başlangıçta oluşturulur.

Yönetim ve overlay origin'leri farklı olmalıdır. Eksik, hatalı veya bilinmeyen ayarlar başlangıçta reddedilir. Dosyayı düzenledikten sonra yeniden başlatın.

## Güvenilir LAN ve cihaz eşleştirme

Sunucunun gerçek LAN IP'sini kullanın; `192.168.1.20` yalnız örnektir:

```yaml
addr: "0.0.0.0:8080"
public_addr: "0.0.0.0:8081"
public_origin: "http://192.168.1.20:8081"
admin_origins:
  - "http://127.0.0.1:8080"
  - "http://192.168.1.20:8080"
db_path: "overlay.db"
```

Loopback dışına bağlanmak için açık ve boş olmayan `admin_origins` listesi gerekir. `0.0.0.0` dinleme adresidir; tarayıcıya yazılmaz. Gerekiyorsa güvenlik duvarında yapılandırılan TCP portlarına izin verin; uygulama güvenlik duvarını değiştirmez.

1. Yapılandırmayı değiştirdikten sonra uygulamayı yeniden başlatın.
2. Önce sunucu bilgisayarında `http://127.0.0.1:8080/access` ile yerel erişimi başlatın.
3. **Cihazlar (/cihazlar)** ekranından diğer cihaz için QR ve altı haneli kod oluşturun.
4. Aynı güvenilir ağdaki cihazda QR okutun veya `http://192.168.1.20:8080/access` açıp kodu girin.
5. OBS'de panelden kopyalanan, ayrı overlay origin'indeki bağlantıyı kullanın.

Kod ve QR bağlantısı tek kullanımlıktır, beş dakika geçerlidir. Yetkilendirilmiş cihazın kalıcı cookie'si 180 gün sürer. Erişim ekranından cihazı iptal edebilir veya cihazdan çıkış yapabilirsiniz.

**HTTP LAN trafiği şifreli değildir.** Kodlar, oturumlar ve içerik ağda görülebilir. Yalnız güvenilir LAN kullanın; açık internete çıkarmayın. HTTPS önerilir ancak yerleşik HTTPS listener'ı yoktur ve uygulama forwarded başlıklarına güvenmez. Yerel bootstrap ve istemci adresi davranışı ayrıca güvenli biçimde çözülmeden TLS sonlandıran proxy desteği varsaymayın.

Cookie'ler portlara göre izole edilmez. Ayrı origin, katı Origin/CSRF kontrolleriyle birlikte kullanılır; overlay origin'ine yönetim yetkisi verilmemelidir. OBS bağlantısı sayfaya okuma yetkisi verir; gizli tutun.

## Kalıcı veri ve paylaşım

SQLite; overlay HTML'ini, değerleri ve hassas erişim bilgilerini saklar. Veritabanını ve yedeklerini gizli tutun. Veritabanı, WAL/SHM dosyaları ve yerelde derlenen uygulamalar `.gitignore` ile dışlanır. Commit'e veya kaynak arşivine eklemeyin.

Tutarlı SQLite yedeği için uygulamayı temizce durdurup veritabanını kopyalayın veya SQLite'ın desteklediği çevrimiçi yedekleme yöntemini kullanın. Uygulama çalışırken WAL/SHM dosyalarını silmeyin.

Depodaki loopback `configuration.yaml` paylaşılabilir varsayılandır. Kişisel düzenlemelerinizi commit öncesinde kontrol edin.

## Testler

```sh
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
```

Node.js PATH üzerinde veya `OBS_NODE` ile sağlanırsa JavaScript davranış kontrolleri Go testlerinde çalışır. Node.js yoksa atlanır; Go testinin geçmesi bu durumda JavaScript davranışını doğrulamaz. Atlamaları görmek için `go test -v ./...` kullanın.

İsteğe bağlı tarayıcı kontrolleri Python Playwright ve Chrome gerektirir:

```sh
python3 tests/browser_refactor.py
python3 tests/browser_access.py
```

Chrome `/opt/google/chrome/chrome` konumunda değilse `CHROME_BIN` verin. Bu testler geçici çalışma dizinleri, özel YAML yapılandırması/veritabanları ve ayrı loopback portları kullanır; projenin aktif veritabanını kullanmaz. Erişim testi `overlay.test` ile ikinci host'u simüle eder; fiziksel LAN veya gerçek OBS testi değildir.

Ayrıntılar: [tests/README.md](tests/README.md), [KULLANICI_TESTLERI.md](KULLANICI_TESTLERI.md). Tarihli sonuçlar ve sınırlar `test_output/` altındadır; eski raporlar test ettikleri sürümler için geçerlidir.

## Lisans ve tema

Proje lisansı: [MIT](LICENSE). Yönetim paneli [Bootstrap Kehribar Gece](https://github.com/MuhammedOzby/bootstrap-kehribar-gece) kullanır; dağıtılan lisansı [template/assets/bootstrap-kehribar-gece/LICENSE](template/assets/bootstrap-kehribar-gece/LICENSE) konumunda korunur. Tema, yüklenen overlay HTML'ine eklenmez.
