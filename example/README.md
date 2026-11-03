# Örnek akış: template.html ile kayıt oluşturma

Bu klasör yalnızca örnek içeriktir: `template.html` (boş, bağımlılıksız şablon), `variables.md` (örnek değişken listesi) ve bu dosya.

## Çalıştır

```bash
go run .
```

Yönetim paneli `http://127.0.0.1:8080`, OBS sayfaları ayrı listener üzerinde `http://127.0.0.1:8081` adresindedir. Bunlar sırasıyla `ADDR` ve `PUBLIC_ADDR` ile değiştirilebilir. Panelde gösterilen adres `PUBLIC_ORIGIN` ile açıkça ayarlanır; varsayılanı `http://127.0.0.1:8081`'dir ve istek Host başlığından alınmaz. Örneğin:

```bash
ADDR=127.0.0.1:8080 PUBLIC_ADDR=127.0.0.1:8082 PUBLIC_ORIGIN=http://127.0.0.1:8082 DB_PATH=/tmp/overlay-example.db go run .
```

`PUBLIC_ORIGIN`, OBS'nin ulaşacağı yayın origin'i olmalı ve yönetim origin'inden ayrı olmalıdır. Veritabanı yolu `DB_PATH` ile ayarlanır; varsayılan `overlay.db`'dir (çalışma dizinine göre).

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
