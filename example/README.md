# Örnek akış: template.html ile kayıt oluşturma

Bu klasör yalnızca örnek içeriktir: `template.html` (boş, bağımlılıksız şablon), `variables.md` (örnek değişken listesi) ve bu dosya.

## Çalıştır

```bash
go run .
```

Yönetim paneli `http://127.0.0.1:8080`, OBS sayfaları ayrı origin olan `http://127.0.0.1:8081` üzerindedir. Uygulama **proje kökünden** çalıştırılır; `example/` klasöründen değil.

Ayarlar çalışma dizinindeki `configuration.yaml` dosyasından okunur. `ADDR`, `PUBLIC_ADDR`, `PUBLIC_ORIGIN` ve `DB_PATH` ortam değişkenleri uygulama ayarlarını değiştirmez.

Örneğin farklı yayın portu ve veritabanı için, YAML içeriğini şöyle düzenleyin ve uygulamayı yeniden başlatın:

```yaml
addr: "127.0.0.1:8080"
public_addr: "127.0.0.1:8082"
public_origin: "http://127.0.0.1:8082"
admin_origins: []
db_path: "overlay-example.db"
```

`public_origin`, OBS'nin ulaşacağı yayın origin'idir ve yönetim origin'inden ayrı olmalıdır. Göreli `db_path` çalışma dizinine göredir. Kullanım sonrası kişisel ayarları paylaşmadan önce gözden geçirin.

## Kayıt oluşturma

1. Önce `http://127.0.0.1:8080/access` ile yerel erişimi başlatın; ardından `http://127.0.0.1:8080/yeni` adresini açın.
2. İsim girin; tür seçin (`screen_overlay` veya `widget`).
3. HTML dosyası olarak bu klasördeki `example/template.html` seçin.
4. `Kaydı oluştur` düğmesine basın.

Oluşturma sonrası:

- Kayıt liste sayfasında (`/`) seçili tür filtresinde görünür.
- Düzenleme sayfasına (`/duzelt/<id>`) yönlenirsiniz.
- Kayıt SQLite'ta kalıcıdır; sunucu yeniden başlatıldığında korunur.

## İsteğe bağlı adım

İsterseniz `template.html`'in `body` içine `<div>{{konuk_adi}}</div>` satırını ekleyip yeni kayıt oluşturabilirsiniz; `{{konuk_adi}}` yalnızca normal HTML metni olarak geçerlidir (bkz. `variables.md`). Düzenleme ekranında değişkeni ve değerini kaydedin; OBS bağlantısını kopyalayın veya yeni sekmede açın. Kayıtlı değerler sayfa açıldığında veya yenilendiğinde görünür; SSE açık alanlar `Yayına uygula` ile açık OBS sayfalarında canlı güncellenir. `Tasarımı yayınla`, yeni HTML'i ayrı olarak yayınlar ve açık yayın sayfalarını yeniler; kaydedilmemiş değişken taslağını yayınlamaz. Bağlantıyı gizli tutun: bağlantıyı bilen herkes sayfayı okuyabilir.

## Güvenlik

Cihaz eşleştirme, LAN yapılandırması ve HTTP güvenlik sınırları için kök dizindeki `README.md` dosyasına bakın. OBS bağlantısını ve veritabanı yedeklerini gizli tutun; uygulamayı açık internete çıkarmayın.
