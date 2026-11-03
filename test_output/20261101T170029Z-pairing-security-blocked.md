# QR / altı haneli kod eşleştirme — engellenen çalışma

- Tarih: 1 Kasım 2026, 17:00 UTC.
- Durum: tamamlanmadı; güvenlik koruması etkin DEĞİL.

## Kullanıcı gereksinimi

Hesap/parola yok. Ana bilgisayarın yerel ekranında QR ve altı haneli eşleştirme kodu gösterilecek. İkincil cihaz LAN IP:PORT adresinden kod girerek veya QR okutarak eşleşecek; kalıcı cihaz oturumu olacak. QR eşleştirme bilgisini içerecek. Kodun LAN'daki eşleşmemiş istemcilere gösterilmemesi gerekiyor.

## Gerçek değişiklikler

- `internal/store/sqlite.go`: mevcut sayfa/alan şemasını koruyan v2 migration ve `access_sessions` tablosu eklendi. Alanlar: token_hash, csrf_token, device_name, created_at, expires_at. v2 yeniden açılış no-op; >2 reddediliyor.
- `internal/store/sqlite_test.go`: mevcut sürüm beklentisi 2; desteklenmeyen sürüm senaryosu 3 oldu.
- Oturum repository metotları, HTTP koruması, QR/kod ekranı ve istemci entegrasyonu yazılamadı. Sadece tablo bulunması oturum güvenliği sağlamaz.
- Mevcut çalışan süreç yeniden başlatılmadı; uygulama veritabanı üzerinde migration komutu çalıştırılmadı; LAN erişimi açılmadı.

## Engeller ve başarısız ara kontroller

- İlk Nemotron çıktısı talimat dışı tablo ve bozuk migration üretti; kabul edilmedi.
- Sonraki Qwen değişikliği migration'ı v2'ye taşıdı ancak çağrı çıktı sınırıyla kesildi. İlk bağımsız `go test -count=1 ./internal/store` eski testlerin v1 ve unsupported=2 beklentileri nedeniyle başarısız oldu.
- Gemma ile bu iki test beklentisi güncellendi. Nemotron/Gemma ile oturum repository görevleri tekrar tekrar `Timed out waiting for response headers` hatasıyla sonuçlandı. Repo dosyası oluşmadı.
- Yerel model zorunluluğu ve ana ajanın uygulama kodu yazmama talimatı gereği bulut alt ajana geçilmedi; uygulama işi durduruldu.

## Son kaynak üzerinde gerçek kontroller

| Komut | Sonuç |
| --- | --- |
| `go test -count=1 ./...` | exit 0, altı paket başarılı |
| `go vet ./...` | exit 0, çıktı yok |
| `go test -race -count=1 ./...` | exit 0, altı paket başarılı, race bildirimi yok |

Bu çalışmada bağımsız Node testi, gerçek tarayıcı, QR tarama, LAN cihazı veya OBS kabul testi çalıştırılmadı. Eşleştirme güvenliği test edildi/geçti iddiası yok. Migration v1 veri koruması için yeni özel test de henüz eklenmedi.

## Devam işi

Önce migration v1→v2 veri koruma testi ve kalıcı oturum repository testleri. Sonra güvenli yerel kod üretimi/gösterimi, kısa süreli tek kullanımlık QR/kod, kaba kuvvet sınırlaması, hash ile saklanan kalıcı oturum, iptal/çıkış, CSRF/Origin/Host doğrulaması, açık LAN yapılandırması ve gerçek tarayıcı/LAN kabulü. HTTP LAN trafiği şifreli değildir; kalıcı cookie ve kodların ağdan dinlenmesi riskine karşı HTTPS gereksinimi açıkça belgelenmeli.
