# Runtime refactor — overlay runtime davranış test raporu

- Tarih: 1 Kasım 2026.
- Kapsam: `routes/overlay-runtime.js` runtime refactor; yeni `tests/runtime.test.cjs`; yeni `routes/runtime_script_test.go`. Panel/yükleme/store incelemeleri yapan diğer ajan dosyalarına ve `tests/panel.test.cjs`, `tests/browser_refactor.py` dosyalarına dokunulmadı.

## Refactor değişiklikleri

- Artık kullanılmayan "Fix 1/Fix 4/Fix 5/Fix 6" ve "Original ... removed per instructions" yorumları kaldırıldı; davranış belgelemesi fonksiyon adlarıyla yapıldı.
- `initializeEventSource` gövdesindeki bozuk indent düzeltildi.
- Tekrarlanan version validation ve reload prosedürü sadeleştirildi: `isSafePositiveInt`, `closeEventSource`, `reloadForTemplateChange` yardımcıları; state ve `template_changed` handler'ları tek kullanım mantığı paylaşıyor.
- No-op error callback (`eventSource.onerror = function(error) {}`) kaldırıldı; reconnect hataları EventSource'ın kendi doğal (otomatik yeniden denemeli) davranışına bırakıldı.
- Gereksiz state mutasyonu (`Object.assign(currentState, payload)`) kaldırıldı; `events_url` yalnızca ilk metadata nesnelerinden okunuyor.

## Korunan davranışlar

- İlk metadata doğrulama: `revision` ve `template_version` için safe positive integer; uyuşmazsa runtime idles.
- Tek EventSource; `hasReloaded` reload guard'i reload öncesi kontrol ediliyor.
- State payload doğrulama: alanlar non-null object ve tüm değerler string; aksi halde ignore.
- Stale revision ignore (`revision <= currentRevision`).
- Tekrarlı alanlar için tüm span'ların `textContent` üzerinden yazılması; aynı değerde gereksiz yazım yok.
- Payload'ta olmayan alanlara DOM'a dokunulmuyor.
- `template_version` değişimi veya `template_changed` event'i → tek reload.

## Testler

- `tests/runtime.test.cjs`: Node built-in test runner + `node:vm`; gerçek `routes/overlay-runtime.js` kaynağı context içinde çalıştırılıyor, code string assertion içermiyor.
- 13 davranış testi: tek EventSource açılışı, metadata malformed/unsafe idles, malformed JSON, unsafe version, alan map doğrulaması, tekrarlı alan textContent, payload dışı alan koruma, stale ignore, reload once (state + template_changed), reconnect error handler yok, pagehide/pageshow lifecycle.

## Kontroller ve gerçek sonuçlar

- Komut: `/home/ozbay/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node --check routes/overlay-runtime.js` → exit 0.
- Komut: `/home/ozbay/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node --test tests/runtime.test.cjs` → 13/13 PASS, 0 fail.
- Komut: `go test -count=1 -v ./routes/ -run TestOverlayRuntimeScript` → PASS (`nodeForRuntimeBehaviorTest` fallback ile local Node kullanarak `tests/runtime.test.cjs` çalıştırdı).
- Komut: `go vet ./...` → OK.
- Komut: `go test -count=1 ./...` → tüm paketler OK (main, internal/live, internal/store, internal/upload, routes).
- Komut: `go test -race -count=1 ./...` → `internal/store` paketinde `TestPublishTemplate_PublishesOnlyOpenFieldsAfterCommit` ve `TestPublishTemplate_FailuresDoNotMutateOrPublish` FAIL (timing/snapshot). Bu görev kapsamında değiştirilen dosya değil; internal/store'a dokunulmadı. Ayrı görevde ele alınmalı.
- Node bulunamadı senaryosu: `env PATH=/usr/bin:/bin` ile de test fallback local Node ile geçti; `OBS_NODE` env'i boş iken PATH/Stat fallback zinciri çalıştı.

## Sonuç

- Runtime refactor tamamlandı; davranış koruması ve yeni davranış testleri geçti.
- Sonraki adım: ayrı bir görevde `internal/store` publish timing testlerinin stabilitesi.
