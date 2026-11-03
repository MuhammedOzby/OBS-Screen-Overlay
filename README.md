# OBS Screen Overlay

Go ve SQLite ile tek HTML dosyasından OBS yayını ve ayrı bir yönetim paneli. Hesap veya parola yoktur; cihazlar eşleştirilir.

## Yerel kullanım

```sh
go run .
```

Önce sunucu bilgisayarında `http://127.0.0.1:8080/access` adresini açıp yerel erişimi başlatın. Yönetim paneli bu adrestedir; OBS yayını ayrı origin olan `http://127.0.0.1:8081` üzerindedir. Sonra `/cihazlar` ekranından başka bir cihaz için QR ve altı haneli kod oluşturun. Kod/QR tek kullanımlıktır ve 5 dakika geçerlidir. Uzak cihaz aynı ağdan yönetim adresini açıp kodla eşleşebilir. Cihaz oturumu 180 günlük cookie ile kalıcıdır; cihazı erişim ekranından iptal edebilir veya cihazda çıkış yapabilirsiniz.

Uygulamayı proje kökünden çalıştırın; çalışma dizinindeki `configuration.yaml` zorunludur. Ayarlar ortam değişkenlerinden okunmaz. Yerel varsayılan dosya:

```yaml
addr: "127.0.0.1:8080"
public_addr: "127.0.0.1:8081"
public_origin: "http://127.0.0.1:8081"
admin_origins: []
db_path: "overlay.db"
```

`addr` ve `public_addr` dinleme adresleri, `public_origin` OBS bağlantısının dışarıdan görünen origin'i, `db_path` SQLite dosyasıdır (göreli yol çalışma dizinine göredir). `admin_origins` tam origin izin listesidir; yol eklemeyin. Boş liste ve loopback bind için yönetim portuna göre `http://127.0.0.1:8080` ve `http://localhost:8080` otomatik kullanılır. Dolu liste bunların yerine geçer. Yayın origin'i yönetim origin'lerinden farklı olmalıdır. Eksik, hatalı veya bilinmeyen ayarlar başlangıçta reddedilir.

## Güvenilir yerel ağ örneği

Sunucunun gerçek LAN IP adresini kullanın:

```yaml
addr: "0.0.0.0:8080"
public_addr: "0.0.0.0:8081"
public_origin: "http://192.168.1.20:8081"
admin_origins:
  - "http://127.0.0.1:8080"
  - "http://192.168.1.20:8080"
db_path: "overlay.db"
```

Loopback dışına bağlanmak için `admin_origins` açıkça ve boş olmayan bir liste olarak verilmelidir. Dosyayı düzenledikten sonra `go run .` ile başlatın. Önce sunucu bilgisayarında yerel `/access` ile erişimi başlatın; `/cihazlar` üzerinden üretilen kodla LAN cihazından `http://192.168.1.20:8080` adresinde eşleşin. OBS, ayrı yayın origin'indeki panel bağlantısını kullanır.

**HTTP LAN trafiği şifreli değildir:** kodlar, oturumlar ve içerik ağda görülebilir. Yalnız güvenilir LAN kullanın; açık internete çıkarmayın. HTTPS önerilir; uygulamanın yerleşik HTTPS listener'ı yoktur ve erişim kontrolü forwarded başlıklarına güvenmez. Bu nedenle TLS sonlandıran proxy arkasında özellikle yerel bootstrap ve istemci adresi davranışını güvenli biçimde ayrıca çözmeden proxy desteği varsaymayın.

Cookie'ler portlara göre izole edilmez. Ayrı origin, katı Origin/CSRF kontrolleriyle birlikte kullanılır; yayın origin'ine yönetim yetkisi verilmemelidir. OBS bağlantısı okuma yetkisi veren bir sırdır. SQLite dosyasını ve yedeklerini gizli tutun; oturum ve yayın erişimine ilişkin hassas bilgiler içerir. Örnek HTML ve panel akışı için `example/README.md` dosyasına bakın.
