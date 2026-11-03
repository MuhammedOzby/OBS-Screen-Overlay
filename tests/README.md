# Kontroller

```sh
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
```

JavaScript davranış testleri Go testlerine dahildir. Node PATH üzerinde olmalı
veya `OBS_NODE` ile çalıştırılabilir dosyanın yolu verilmelidir. Node yoksa ilgili
testler açıkça atlanır; JavaScript doğrulandı sayılmaz. Atlanan testleri görmek
için `go test -v ./...` kullanın.

## Gerçek tarayıcı

Playwright Python paketi ve Chrome bulunan ortamda:

```sh
python tests/browser_refactor.py
python tests/browser_access.py
```

Chrome yolu `CHROME_BIN` ile değiştirilebilir (varsayılan:
`/opt/google/chrome/chrome`). Bu kontrol gerçek sunucuyu ayrı loopback portları
ve geçici SQLite dosyasıyla başlatır; mevcut `overlay.db` dosyasını kullanmaz.
Oluşturma, canlı güncelleme, tasarım yayınlama, hata/çatışmada taslak koruma,
JavaScript kapalı form davranışı ve silme akışını sınar. PATCH ve tasarım
yayınlama istekleri sırayla bekletilerek diğer formun ve çift gönderimin
engellendiği de doğrulanır; ardından gerçek sunucu yanıtıyla devam edilir.
Ağ hatası senaryosunda yayınlama isteği bilinçli olarak engellenir. Bu test
gerçek OBS kabulünün yerine geçmez.

`browser_access.py` yerel oturumu arayüzden açar; cihaz eşleştirme, kodun
tek kullanımlılığı, kalıcı cookie ve sunucu yeniden başlatma, QR bağlantısı,
CSRF/origin yalıtımı, cihaz iptali ve çıkışı gerçek sunucuyla sınar. Chrome
`overlay.test` adını loopback'e eşler: iki farklı host/origin simülasyonudur,
fiziksel LAN testi değildir. PNG imzası ve fragment bağlantısı kontrol edilir;
QR görüntüsü taranıp içeriği çözülmez. Gerçek OBS kabulü de yapılmaz. Erişim
API ve arayüz entegrasyonu henüz tamamlanmadıysa test başarısız/engellidir;
başarılı sayılmamalıdır. Mevcut ortam örneği:

```sh
/tmp/opencode/obs-panel-test-venv/bin/python tests/browser_access.py
/tmp/opencode/obs-panel-test-venv/bin/python tests/browser_refactor.py
```

Çalıştırılan kontrollerin gerçek sonuçlarını benzersiz bir Markdown raporuyla
`test_output/` altında saklayın.
