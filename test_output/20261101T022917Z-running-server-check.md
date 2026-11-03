# Çalışan sunucu kontrolü

- Tarih: 1 Kasım 2026, 02:29 UTC.
- Kapsam: Kullanıcının 8080 portundaki yükleme hatasının teşhisi.
- `curl -sS -i http://127.0.0.1:8080/yeni`: 200, eski “Demo / Değişiklikler kaydedilmez” şablonu döndü.
- `curl -sS -i -X POST http://127.0.0.1:8080/api/pages`: 404.
- Sonuç: 8080 üzerinde yeni backend değil, eski demo süreci çalışıyor. Kullanıcının çalışan süreci durdurulmadı; kaynak dizininden yeniden başlatması gerekiyor.
- Bu kontrolde kod değişmedi; yeniden başlatma sonrası yükleme henüz doğrulanmadı.
