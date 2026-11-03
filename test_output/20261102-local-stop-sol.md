# Yerel sunucuyu durdurma — 2 Kasım 2026

- Kapsam: Kullanıcı isteğiyle bu oturumda başlatılan yerel sunucuyu kapatma.
- `ss -ltnp '( sport = :8080 or sport = :8081 )'` iki listener'ın PID 2062523'e ait olduğunu gösterdi.
- `kill -TERM 2062523` exit 0 döndü.
- Arka plan süreci kapanırken `context deadline exceeded` ve exit status 1 bildirdi; zarif kapanış zaman aşımına uğradı.
- Son `ss -ltnp '( sport = :8080 or sport = :8081 )'` çıktısında dinleyici yok. Sunucu kapalı.
- Tam otomatik testler veya veri bütünlüğü kontrolü çalıştırılmadı. Kullanıcı kabul listesi `KULLANICI_TESTLERI.md` dosyasına eklendi; testler yapılmış sayılmadı.
