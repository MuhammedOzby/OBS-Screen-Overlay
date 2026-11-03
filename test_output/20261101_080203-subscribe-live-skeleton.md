# SubscribeLive iskeleti — store (liveMu + hub + SubscribeLive)

- Tarih: 2026-11-01 08:02 (UTC)
- Kapsam: Store struct'una `liveMu sync.Mutex` ve `hub *live.Hub` alanları; `Open`'da `live.NewHub(16)` başlatma; yeni `internal/store/live.go` içinde `SubscribeLive` yöntemi. Update/delete entegrasyonu ve yeni testler bu adımda YAPILMADI (sonraki adım).

## Değişiklikler

1. `internal/store/sqlite.go`
   - `Store` struct'a `liveMu sync.Mutex`, `hub *live.Hub` ekleni.
   - `Open`'da `store := &Store{db: db, hub: live.NewHub(16)}`.
   - Import'lara `sync` ve `github.com/ozbay/obs-screen-overlay/internal/live` eklendi.
2. `internal/store/live.go` (yeni dosya)
   - `func (s *Store) SubscribeLive(ctx context.Context, token string) (live.Snapshot, <-chan live.Snapshot, func(), error)`
   - `liveMu` lock + defer unlock.
   - `GetPageByPublicToken(ctx, token)` ile sayfa okunur; hata dönüldür.
   - `live.Snapshot{Revision, TemplateVersion, Fields}` doldurulur; `Fields` map yalnız `SSEEnabled` olan alanları içerir (AGENT.md §5).
   - `s.hub.Subscribe(page.ID)` ile kanal + cancel alınır.
   - Döndürülen `cancel` idempottur (hub tarafında unsubscribe idempotent).

## Çalıştırılan komutlar ve sonuçlar

| Komut | Sonuç |
| --- | --- |
| `go build ./...` | Başarılı (hata değil) |
| `go vet ./...` | Başarılı (hata değil) |
| `go test ./internal/store/ ./internal/live/` | `ok` (store 0.061s), `ok` (live, cached) |

## Çalıştırılmayan kontroller

- `go test -race ./...` (bu adımda çalıştırılmadı; hub/SubscribeLive birlikte çalıştığında çalıştırılmalı)
- `SubscribeLive` için yeni birim testleri (sonraki adım)
- Update/delete işlemlerinde `hub.Publish` entegrasyonu (sonraki adım)
- SSE uç noktası ve runtime (sonraki adımlar)

## Sonraki adım

Update/delete commit'inden sonra `liveMu` içinde snapshot okuyup `hub.Publish` yapan tek bir commit-sonrası yayın noktası eklenmeli; ardından `go test -race ./...` ve `SubscribeLive` birim testleri.
