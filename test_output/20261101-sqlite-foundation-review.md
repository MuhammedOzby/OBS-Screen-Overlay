# SQLite temeli — denetim ve test raporu

- Tarih: 1 Kasım 2026.
- Uygulama: `ollama/laguna-xs-2.1:latest`, küçük görevler halinde. Ana ajan kaynak incelemesi ve bağımsız test çalıştırmaları yaptı.
- Kapsam: `internal/store` bağlantı, transaction içinde sürümlü migration, `pages`/`fields` şeması, atomik `CreatePage` ve snapshot tutarlı `GetPage`. Panel/routes/main değiştirilmedi; çalışan demo henüz bu katmanı kullanmıyor.
- Sürücü: `github.com/mattn/go-sqlite3 v1.14.52`; resmi README incelendi. Ortam CGO_ENABLED=1, Go go1.27.3-X:nodwarf5. Derleme için C derleyicisi gerekir.
- Her değişken `fields` tablosunda ayrı satırdır. `(page_id,name)` benzersizdir; farklı sayfalarda aynı değişken adı kullanılabilir. Alan sayısına uygulama tarafından sabit üst sınır konmadı. Gerçek kapasite disk/bellek/SQLite kaynaklarıyla sınırlıdır.
- Alanlar tek prepared statement ile satır satır aynı transaction'a yazılır; dev bir SQL parametre listesi kurulmaz.

## Ana denetçinin çalıştırdığı komutlar

1. `go test -count=1 ./...`: başarılı (ara kontrol).
2. `go test -count=1 -v ./...`: başarılı; store paketinde 26 üst seviye test ve bütün alt testler geçti. Süre 0.059s.
3. `go vet ./...`: başarılı; çıktı yok.
4. `go test -race -count=1 ./...`: başarılı; store 1.223s.

## Doğrulanan davranışlar

- Tek sayfada 10.000 değişken oluşturma; dosyayı kapatıp aynı dosyayı tekrar açma; bütün ad/değer/SSE/sıra değerlerinin korunması. HTML, revision ve template_version da doğrulandı.
- Boş değişken listesi, serbest sayfa başlığı ve alan adı biçimi doğrulaması.
- Yinelenen değişken adına bağlı SQL hatasında transaction rollback ve sayfanın kaydedilmemesi.
- Yabancı anahtar, benzersiz token, tür/SSE/sıra/revision kısıtları; sayfa silindiğinde değişkenlerin cascade silinmesi.
- Migration tekrar çalıştırılabilirliği, ileri şema sürümünü reddetme, özel karakterli dosya yolu, bulunamayan sayfa.

## İncelemede düzelttirilenler ve sınırlar

- İlk model çıktısındaki eksik şema kısıtları ve transaction dışındaki migration sürüm okuması düzeltildi.
- Sayfa başlığına yanlışlıkla uygulanan alan adı regex'i kaldırıldı.
- İlk test verilerinin geçersiz tür/eski index beklentileri düzeltildi. Başlangıçta iddia edilen test başarıları nihai kanıt sayılmadı; yukarıdaki bağımsız son çalıştırmalar esas alındı.
- İlk çok-alan testinde dosya yeniden açılmıyordu; bu eksik düzeltilip test tekrar çalıştırıldı.
- Bir alt ajan çağrısı özetleme hatasıyla bitti; oluşturduğu dosyalar ayrıca okunup denetlendi.
- Testler geçici veritabanları kullanır. Üretim/demo veritabanı oluşturulmadı, panel bağlanmadı.
- Listeleme/güncelleme/silme repository metotları, uygulama başlangıcına bağlama, HTTP API, SSE ve güvenlik bu teslimin parçası değil. Güvenlik kullanıcı kararıyla en sonda.
- UI değişmediği için bu çalışmada tarayıcı/OBS testi yapılmadı. Aşama 1 bütünü tamamlanmış değildir.
