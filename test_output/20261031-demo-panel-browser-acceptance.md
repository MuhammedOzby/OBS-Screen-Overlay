# Demo yönetim paneli — son denetim

- Tarih: 31 Ekim 2026
- Kapsam: Liste, oluşturma ve düzenleme demo ekranları; tek dosyalık seçim; vanilla JS form davranışı ve responsive görünüm.
- Uygulama kodunu `ollama/qwen3.8:27b` yazdı. Ana ajan inceleme ve test yaptı; bulunan hataların düzeltmeleri de Qwen'e verildi. Çıktı sınırına takılan görevler küçültüldü.

## Komutlar ve sonuçlar

| Komut | Sonuç |
| --- | --- |
| `go test ./...` | Başarılı; test dosyası yok. |
| `go vet ./...` | Başarılı. |
| `go build -o /tmp/opencode/obs-panel-demo .` | Başarılı. |
| `go test -race ./...` | Başarılı; test dosyası olmadığından hub/eşzamanlılık kanıtı değildir. |
| `gofmt -l main.go lib routes` | Temiz. |
| `/tmp/opencode/obs-panel-demo` | Yerel test sunucusu `127.0.0.1:8080` üzerinde çalıştı. |
| `/tmp/opencode/obs-panel-test-venv/bin/python - <<'PY' … PY` | Playwright ile gerçek `/opt/google/chrome/chrome` üzerinde aşağıdaki assertion'lar geçti. İki ayrı tarayıcı kontrolü çalıştırıldı. |

## Tarayıcı sonuçları

- Liste: `/` iki kayıt; `screen_overlay` ve `widget` birer kayıt; geçersiz filtre iki kayıt. Geçerli sayfalar HTTP 200.
- Oluşturma: yalnız isim, tür ve bir dosya seçimi; değişken alanı yok, `multiple` yok.
- `.zip`, boş `.html` ve 2 MiB üzeri dosya reddedildi; büyük harfli `.HTML` kabul edildi. Bu yalnız istemci metadata doğrulamasıdır; içerik doğrulaması/backend upload değildir.
- Düzenleme: üç örnek alan, kapalı sayaç SSE seçimi, yeni alanın varsayılan açık SSE seçimi.
- Değişken ekleme/kaldırma, 18 satıra çıkma, yeni eklenen satırın yinelenen adını reddetme ve geçerli formu doğrulama geçti.
- HTML benzeri değer textarea'da düz metin kaldı; `img` oluşturulmadı.
- Widget türü doğru seçildi; ayrı tasarım formu demo bildirimi verdi. OBS düğmeleri devre dışı; sahte yayın URL'si yok.
- Bilinmeyen düzenleme ID'si listeye yönlendi.
- Liste/yeni/düzenleme ekranları 360, 768 ve 1440 px genişlikte kontrol edildi; yatay belge taşması veya ana form kontrollerinin ekran dışına çıkması yok.
- Masaüstü ve dar ekran görüntüleri incelendi. 360 px'te OBS düğme metinlerinin sıkışması bulundu, Qwen CSS düzeltmesi yaptı ve yeniden test geçti.
- JavaScript hatası veya GET dışı ağ isteği oluşmadı.
- JavaScript kapalıyken oluşturma düğmesi devre dışı kaldı.
- Panel CSS, JS ve yerel tema CSS HTTP 200 döndü.
- Ekran görüntüleri: `/tmp/opencode/obs-panel-{360,768,1440}.png`, `/tmp/opencode/obs-panel-360-final.png`.

## Sınırlar

Gerçek kayıt, SQLite, oturum/CSRF, sunucu taraflı HTML yükleme/doğrulama, SSE ve OBS yayını uygulanmadı; bu çalışma yalnız demo paneldir. OBS kabulü yapılmadı. Demo doğrulama mesajı başarılı gerçek kayıt/yayın anlamına gelmez. Test sunucusu çalışma sonunda kapatılacak; çalıştırma: proje dizininde `go run .`.
