# Panel SSE kullanılabilirliği ve panel.js onarımı

- **Tarih:** 2026-11-01 07:01 UTC (`20261101T070150Z`)
- **Kapsam:** `template/assets/panel.js` sözdizimi hatasını minimum düzeyde onarmak, SSE başarı metnini ve edit sayfasındaki SSE açıklaması/apply-button etiketini güncellemek, panel.js için küçük bir regresyon testi eklemek.
- **Kapsam dışı:** Template publish özelliğini geliştirmek/etkinleştirmek, silme davranışı, güvenlik, runtime/Go prod refactor, tasarım bölümü, `PLAN.md`.

## Değişen dosyalar

1. `template/assets/panel.js`
   - Dört adet birebir aynı, literal `\n` ile serileştirilmiş bloktan **yalnız birincisi** gerçek satır sonlarına çevrilerek korundu.
   - Diğer üç kopya kaldırıldı; yerlerine yalnızca kendilerinden önceki IIFE'yi kapatan `})();` satırı bırakıldı.
   - SSE başarı metni `Değişiklikler kaydedildi. SSE yayını henüz bağlı değil.` → `Değişiklikler kaydedildi. SSE açık alanlar yayına uygulandı.` olarak değiştirildi.
   - Dosya boyutu: 26.894 bayt → 14.717 bayt.
   - Yeni SHA-256: `04b59a0a317933012cc8be4e8e93f9c95a2f62fe0b875e1a9bf75abedd5845ff`.
   - Özgün dosya yedeği: `/tmp/opencode/panel.js.original-backup`.

2. `template/edit.html.tmpl`
   - Satır 25: SSE açıklaması `SSE ile canlı güncelleme henüz bağlı değil.` → `SSE açık alanlar kaydedildiğinde yayına uygulanır.`
   - Satır 70: apply-button etiketi `Değişiklikleri kaydet` → `Yayına uygula`.
   - Tasarım/template kartı (satır 78–88) ve diğer tüm içerik değiştirilmedi.

3. `panel_script_test.go` (yeni)
   - `panel.js` için yapısal regresyon testi.
   - İlk adı `panel_js_test.go` idi; Go bu adı `js` GOOS eki sanıp dosyayı yok saydı (`IgnoredGoFiles`). Bu nedenle `panel_script_test.go` olarak yeniden adlandırıldı.
   - Node varsa `node --check` çalıştırır; yoksa yalnız yapısal kontrollerle devam eder.
   - Node araması: `OBS_NODE` → `/tmp/opencode/node-v22.14.0-linux-x64/bin/node` → `PATH`.

4. `routes/panel_url_test.go`
   - Edit sayfası çıktısında eski SSE metnini arayan mevcut test, yeni metne ve `Yayına uygula` etiketine göre güncellendi.

## Korunan template bloğu nasıl korundu

Önce dosya yeniden okundu ve dört kaçışlı satırın gerçekten birebir kopya olduğu doğrulandı:

- Kaçışlı satırlar (1 tabanlı): 18, 152, 199, 254
- Her biri: 4.027 bayt
- Ortak SHA-256 öneki: `cf39cf68a4c87a24`
- Her birinde 117 adet literal `\n` (ters bölü + n) dizisi
- Her blok `})();\n\n// START TEMPLATE FORM PUBLISHING LOGIC` ile başlıyor, `// END TEMPLATE FORM PUBLISHING LOGIC\n` ile bitiyor.

Uygulanan dönüşüm kör bir replace değildi; adım adım doğrulandı:

1. Dört satırın SHA-256 özetlerinin aynı olduğu assert edildi.
2. Yalnız **ilk blok** alındı ve sadece literal `\n` dizileri gerçek satır sonuna çevrildi.
3. `/\.html$/i` gibi regex kaçışlarına dokunulmadı; `\\.` dizilerinin korunduğu ayrıca assert edildi.
4. Diğer üç blok, birebir aynı oldukları doğrulandıktan sonra yalnızca baştaki `})();` IIFE kapatıcısıyla değiştirildi.
5. Blokların başındaki `})();` satırlarının kendilerinden önceki IIFE'leri kapattığı doğrulandı; bu yüzden kopyalar kaldırılırken bu kapatıcılar korundu.
6. Dönüşüm sonrası ilk blok yeniden literal `\n` biçimine çevrilerek özgün kaçışlı satırla karşılaştırıldı: **birebir aynı**.
7. Kalan tüm satırlar (diğer IIFE'ler) özgün dosyayla yeniden karşılaştırıldı: **değişmedi**.

Sonuç sayaçları:

```text
literal \n dizisi:                         0
START TEMPLATE FORM PUBLISHING LOGIC:      1
END TEMPLATE FORM PUBLISHING LOGIC:        1
document.getElementById('template-form'):  1
document.getElementById('html-file'):      1
form.querySelector('#html-file'):          1
/template fetch uç noktası:                1
```

`edit-form` doğrulamasındaki `const file = form.querySelector('#html-file');` sorgusu form kapsamlı olarak korundu; template formunun global `document.getElementById('html-file')` sorgusuyla karışmadığı testle sabitlendi.

## JS sözdizimi kontrolü

```text
/tmp/opencode/node-v22.14.0-linux-x64/bin/node --check template/assets/panel.js
NODE_CHECK_OK
```

Onarım öncesi aynı komut `SyntaxError: Invalid or unexpected token` veriyordu (satır 18, literal `\n` dizisi).

## Çalıştırılan testler ve gerçek sonuçlar

```text
go test -count=1 -run '^TestPanelJSStructure$' -v .
=== RUN   TestPanelJSStructure
--- PASS: TestPanelJSStructure (0.01s)
PASS
ok  	github.com/ozbay/obs-screen-overlay	0.014s

go test -count=1 ./...
ok  	github.com/ozbay/obs-screen-overlay	0.019s
ok  	github.com/ozbay/obs-screen-overlay/internal/live	0.002s
ok  	github.com/ozbay/obs-screen-overlay/internal/store	0.065s
ok  	github.com/ozbay/obs-screen-overlay/internal/upload	0.007s
?   	github.com/ozbay/obs-screen-overlay/lib	[no test files]
ok  	github.com/ozbay/obs-screen-overlay/routes	1.071s

go test -race -count=1 ./...
ok  	github.com/ozbay/obs-screen-overlay	1.028s
ok  	github.com/ozbay/obs-screen-overlay/internal/live	1.012s
ok  	github.com/ozbay/obs-screen-overlay/internal/store	1.225s
ok  	github.com/ozbay/obs-screen-overlay/internal/upload	1.026s
?   	github.com/ozbay/obs-screen-overlay/lib	[no test files]
ok  	github.com/ozbay/obs-screen-overlay/routes	2.151s

go vet ./...
VET_OK
```

İlk `go test ./...` çalıştırmasında `routes/panel_url_test.go` eski SSE metnini aradığı için başarısız oldu; test yeni metne göre güncellendikten sonra tüm paketler geçti. Başka başarısız test yok.

## Notlar

- `gofmt -l` yalnızca bu görevde dokunulmayan, önceden biçimsiz olan `internal/store/template.go` ve `routes/events.go` dosyalarını listeliyor. Bu adımda bu dosyalar değiştirilmedi.
- Template publish mantığı silinmedi veya yeniden yazılmadı; yalnızca serileştirme hatası giderildi.
- Yeni template özelliği geliştirilmedi/etkinleştirilmedi.
- `PLAN.md` değiştirilmedi.
- Commit/push/deployment yapılmadı.
- Tarayıcı/OBS kabul testi yapılmadı; bu adımın kapsamı dışında.
