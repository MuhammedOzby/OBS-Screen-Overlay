# Demo panel — başlangıç denetimi

- Tarih: 31 Ekim 2026
- Kapsam: Mevcut dosyaların ve arka plan işlemlerinin denetimi; uygulama öncesi durum.
- Kod değişiklikleri kullanıcı isteğiyle Qwen `ollama/qwen3.8:27b` modeline küçük görevler halinde devredildi.

## Komutlar ve gerçek sonuçlar

- `go test ./...`: **başarısız**. `lib/record.go:46:1: syntax error: unexpected }, expected expression`. Başlangıç kodu derlenmiyor.
- Süreç listesi ve proje çalışma dizinini kullanan süreçler incelendi: Bonsai adlı süreç veya çalışan proje sunucusu bulunmadı. Kullanıcının aktif Qwen/OpenCode oturumu ve ilgisiz uygulamalar kapatılmadı.
- `curl --max-time 10 -sS http://127.0.0.1:11434/api/ps`: yalnızca `qwen3.8:latest` (27.3B) yüklü; Bonsai yok.
- Tema deposundaki `README.md`, `package.json`, `scss/_theme.scss`, `scss/_root.scss` incelendi. Paket sürümü `6.0.0-alpha.1`; mevcut yerel CSS upstream `dist/css/bootstrap.min.css` ile SHA-256 bazında aynı. Bootstrap 5 sınıflarının tümünün mevcut olduğu varsayılmamalı.
- `/opt/google/chrome/chrome --version`: Google Chrome 154.0.8037.97.
- Tarayıcı testi için proje dışında `/tmp/opencode/obs-panel-test-venv` sanal ortamına Playwright kuruldu; proje bağımlılıkları değiştirilmedi.

## Sınırlar

Bu rapor bir başarı/kabul raporu değildir. Şablon dosyaları başlangıçta yok; panel ve OBS davranışı henüz doğrulanmadı. PLAN.md içindeki önceki tamamlanma kaydı mevcut dizinle uyuşmuyor.
