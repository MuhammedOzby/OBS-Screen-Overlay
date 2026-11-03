# Yerel sunucu yeniden başlatma

- Tarih: 1 Kasım 2026, 02:32 UTC.
- Kullanıcı eski süreci kapatma ve yenisini başlatma izni verdi.
- `ss -ltnp 'sport = :8080'`: loopback listener PID 1040001.
- `ps -o pid,ppid,args -p 1040001,1039950` ve `readlink /proc/1040001/cwd`: bu projenin eski `go run .` süreci doğrulandı.
- `kill -TERM 1040001 1039950`: eski sunucu ve başlatıcısı kapatıldı.
- Proje dizininde `go run .` arka planda başlatıldı.
- `curl -sS -i -X POST http://127.0.0.1:8080/api/pages`: 400 JSON `{"error":"Form okunamadı"}`. Dosyasız isteğin beklenen reddi; yeni POST rotası artık mevcut, önceki 404 giderildi.
- Kullanıcı verisi değiştirilmedi; bu kontrolde yeni kayıt oluşturulmadı.
