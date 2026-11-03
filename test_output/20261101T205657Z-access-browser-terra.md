# Erişim tarayıcı testleri — 1 Kasım 2026 UTC

Kapsam: gerçek headless Chrome, ayrı loopback yönetim/yayın portları, geçici SQLite; mevcut üretim veritabanı ve bind kullanılmadı. `overlay.test` Chrome host resolver ile loopback'e eşlendi. Uzak origin API kontrolleri Chrome sayfasından `fetch` ile yapıldı; APIRequestContext DNS çözümlemesine dayanılmadı.

## Komutlar ve gerçek sonuçlar

```sh
TMPDIR=/tmp/opencode /tmp/opencode/obs-panel-test-venv/bin/python tests/browser_access.py
TMPDIR=/tmp/opencode /tmp/opencode/obs-panel-test-venv/bin/python tests/browser_refactor.py
```

- `browser_refactor.py`: exit 0. Oluşturma, PATCH, SSE kapalı alan koruma, tasarım yayını ve karşılıklı kilit, hata/çatışma, silme ve JS kapalı kontrolleri geçti.
- `browser_access.py`: son çalıştırma exit 0:

```text
PASS code and QR URL pairing, persistence and one-use challenge; PNG signature checked (not QR decoded)
PASS public-origin rejection, CSRF, panel mutation, OBS view, revoke and logout
Host simulation only; no physical LAN, QR scanning or OBS Browser Source acceptance performed.
```

Yerel arayüz bootstrap, anonim ret/yönlendirme, uzak hostta yerel bootstrap reddi, kod eşleştirme, kalıcı HttpOnly cookie, context yeniden açılması ve aynı DB ile sunucu yeniden başlatılması, kodun tekrar reddi, QR fragment ile eşleştirme, Chrome ağ URL'lerinde fragment/sır olmaması, yayın HTML'inden gerçek cookie taşıyan admin isteğinin 403 alması, yanlış/eksik CSRF reddi, panelden geçerli mutation ve canlı OBS view değeri, cihaz iptali ve çıkış kontrol edildi.

## Ara başarısızlık ve test düzeltmesi

İlk erişim çalıştırması kod/QR eşleştirme ve kalıcılık sonrasında, `request.all_headers()` sonucunda cookie görünmediğinden test assertion'ında exit 1 ile durdu. İkinci tanılama çalıştırmasında Chrome DevTools cookie ilişkilendirmesinde engelleme nedeni bulunmadığı görüldü. Fixture, saldırı isteğini request ID ile Chrome DevTools `Network.requestWillBeSentExtraInfo` olayına bağlayıp gerçek ağ cookie başlığını kontrol edecek şekilde düzeltildi; 403 korunarak sonraki ve son çalıştırma geçti. Cookie değerleri raporlanmadı. Uygulama kaynağı değiştirilmedi.

## Sınırlar

İki farklı host/origin simülasyonu fiziksel LAN testi değildir. PNG imzası kontrol edildi, QR görüntüsünün içeriği çözülmedi; QR URL fragment akışı ayrı doğrulandı. Gerçek OBS Browser Source kabulü yapılmadı.
