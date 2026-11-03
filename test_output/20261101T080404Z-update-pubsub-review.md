# UpdatePage publish-after-commit — doğrulama raporu

- **Tarih:** 2026-11-01 ~08:04 (UTC, sunucu saati `20261101_080404`)
- **Değişen dosya:** yalnızca `internal/store/update.go`
- **Kapsam:** `UpdatePage` başarıyla commit edildikten sonra SSE açık alanları taşıyan bir `live.Snapshot`'un hub'a yayınlanması; başarısız transaction / validation / conflict durumlarında yayın yapılmaması.

## Yapılan değişiklikler (özet)

1. `UpdatePage` başında `s.liveMu.Lock()` + `defer s.liveMu.Unlock()` eklendi; yayın-yapma işlemi `SubscribeLive`'in oku-abone ol penceresiyle aynı kilit altında tutuluyor (AGENT.md §5: "snapshot elde etme ve abone olma işlemini tutarlı yap").
2. Transaction içinde mevcut sayfanın `template_version` değeri OKANILDI (`SELECT template_version ... WHERE id = ?`). Sayfa yoksa `sql.ErrNoRows` ile erken dönülüyor; bu yol davranış olarak değişmedi (önceki `RowsAffected()==0` kontrolü de aynı `sql.ErrNoRows` / `ErrConflict` semantiğini veriyordu).
3. `tx.Commit()` başarılı olduktan SONRA, verilen `fields` diliminden `SSEEnabled == true` olanların `name -> value` eşleşmeleri bir baştan oluşturulmuş (`make` ile, dolayısıyla `nil` değil) `Fields` map'ine konuyor.
4. `s.hub.Publish(id, live.Snapshot{Revision: expectedRevision+1, TemplateVersion: okunanSürüm, Fields: map})` çağrılıyor.
5. `s.hub` nil ise (ör. `Open` üzerinden değil, sıfır değerli bir Store) panik oluşmaması için publish adım atlanıyor; bu normal çalışma yolunda değil (`Open` her zaman hub kurar).
6. DB yeniden okuma yok; snapshot yalnızca in-memory `fields` diliminden ve okunan `template_version` değerinden oluşturuluyor.

Başarısız branch'lerde (validation, `sql.ErrNoRows`, `ErrConflict`, insert failure, commit failure) `return` erken yaptığı için publish'e asla ulaşılmıyor.

## Çalıştırılan komutlar

```
gofmt -l internal/store/            # çıktı: (boş) — biçim bozukluğu yok
go build ./...                      # BUILD OK
go vet ./internal/store/            # PASS
go test ./internal/store/ -v        # PASS (tüm TestUpdatePage_* dahil)
go test -race ./internal/store/ ./internal/live/
go test ./...
```

## Sonuçlar (gerçek çıktılar)

- `gofmt -l`: hiçbir dosya listelenmedi → biçim bozukluğu yok.
- `go build ./...`: `BUILD_OK` / `BUILD_VET_OK` çıktısı, hata yok.
- `go vet ./internal/store/`: hata yok.
- `go test ./internal/store/ -v` (tam liste sonu):
  ```
  --- PASS: TestUpdatePage_Success (0.00s)
  --- PASS: TestUpdatePage_ConflictAndNotFound (0.00s)
  --- PASS: TestUpdatePage_RollbackOnInsertAbort (0.00s)
  PASS
  ok   github.com/ozbay/obs-screen-overlay/internal/store 0.057s
  ```
  Diğer tüm store testleri (schema, migration, list, delete, pages, token, persistence) da PASS.
- `go test -race ./internal/store/ ./internal/live/`:
  ```
  ok  github.com/ozbay/obs-screen-overlay/internal/store  1.220s
  ok  github.com/ozbay/obs-screen-overlay/internal/live   (cached)
  ```
  Veri ırkı/data race uyarısı yok.
- `go test ./...`:
  ```
  ok  github.com/ozbay/obs-screen-overlay 0.004s
  ok  github.com/ozbay/obs-screen-overlay/internal/live (cached)
  ok  github.com/ozbay/obs-screen-overlay/internal/store 0.061s
  ok  github.com/ozbay/obs-screen-overlay/internal/upload 0.005s
  ?   github.com/ozbay/obs-screen-overlay/lib [no test files]
  ok  github.com/ozbay/obs-screen-overlay/routes 0.014s
  ```

## Başarısız / engellenen kontroller

- YOK. Tüm çalıştırılan kontroller başarılı.
- Not: Görev kapsamı `internal/store` idi; `internal/live` ve kök paket testleri de `-race` ile birlikte çalıştırılıp tamamı PASS verdi.

## API / SQL semantiği

- `UpdatePage` imzası ve geri dönüş değerleri DEĞİŞMEDİ: `(int64, error)`, `expectedRevision+1` / `sql.ErrNoRows` / `ErrConflict` / `ErrValidation` aynı.
- SQL semantiği korundu: tek transaction, optimistic concurrency (`WHERE id = ? AND revision = ?`), `revision = revision + 1`, field sil+re-insert sıralaması aynı.
- Yeni bağımlılık yok; yalnızca mevcut `internal/live` paketi import edildi.
