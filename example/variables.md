# Örnek değişkenler

Düzenleme ekranı için örnek ad / değer / SSE listesi:

| Ad | Değer | SSE |
| --- | --- | --- |
| `konuk_adi` | `Özbay` | `true` |
| `konuk_unvani` | `Konuk` | `true` |
| `sayac_baslangic` | `300` | `false` |

## Yer tutucu kullanımı

`{{konuk_adi}}` yalnızca normal HTML metni olarak kullanılır:

```html
<div>{{konuk_adi}}</div>
<div>{{konuk_unvani}}</div>
<div>{{sayac_baslangic}}</div>
```

Desteklenmeyen yerler (yükleme sırasında reddedilir):

- attribute değerleri: `<img alt="{{konuk_adi}}">`
- `script` / `style` içeriği
- SVG `<text>` içeriği ve diğer SVG/foreign namespace bağlamları
- yorumlar, `textarea`, `title`, `select`/`option`, tablo hücreleri

Alan adları `[a-zA-Z_][a-zA-Z0-9_]*` biçiminde ve büyük/küçük harf duyarlıdır.

## Durum

Değişken/değer ve alan başına SSE ayarları düzenleme ekranında kaydedilir. `Yayına uygula`, SSE açık alanları açık OBS sayfalarında yenilemeden günceller. SSE kapalı alanın kayıtlı değeri tam sayfa yenilemesinde yüklenir; açık sayfadaki JavaScript ile yönetilen içerik korunur.
