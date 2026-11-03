# Access session store doğrulaması

- Tarih: 1 Kasım 2026
- Kapsam: SQLite v2 oturum repository, süre dolması, silme, yeniden açma ve v1→v2 sayfa/alan koruma migration testi.
- `gofmt -w internal/store/access_sessions.go internal/store/access_sessions_test.go internal/store/sqlite_test.go`: çıkış 0, çıktı yok.
- `go test -count=1 ./...`: çıkış 0; kök, internal/live, internal/store, internal/upload, lib ve routes paketleri geçti.
- `go vet ./...`: çıkış 0, çıktı yok.
- `go test -race -count=1 ./...`: çıkış 0; kök, internal/live, internal/store, internal/upload, lib ve routes paketleri geçti.
- Tarayıcı, gerçek OBS ve LAN eşleştirmesi bu repository görevi kapsamında denenmedi.
