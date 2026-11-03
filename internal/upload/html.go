// Package upload, OBS kayıtlarına yüklenen tek dosyalık HTML
// belgelerinin doğrulanması için yardımcıları içerir.
package upload

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// MaxHTMLSize, tek HTML belgesi için izin verilen azami boyut.
const MaxHTMLSize = 2 * 1024 * 1024

var (
	errEmptyHTML   = errors.New("HTML içeriği boş")
	errHTMLTooBig  = errors.New("HTML boyutu 2 MiB sınırını aşıyor")
	errInvalidUTF8 = errors.New("HTML içeriği geçerli UTF-8 değil")
	errNULByte     = errors.New("HTML içeriği NUL bayt içeriyor")
	errNoDoctype   = errors.New("<!doctype html> bulunamadı")
	errNoHTMLOpen  = errors.New("<html> etiketi bulunamadı")
	errNoHTMLEnd   = errors.New("</html> etiketi bulunamadı")
	errNoBodyOpen  = errors.New("<body> etiketi bulunamadı")
	errNoBodyEnd   = errors.New("</body> etiketi bulunamadı")

	errPlaceholderAttr    = errors.New("placeholder, attribute değerinde kullanılamaz")
	errPlaceholderText    = errors.New("placeholder, normal HTML metni dışındaki bağlamda kullanılamaz")
	errPlaceholderComment = errors.New("placeholder, yorum içinde kullanılamaz")
)

// placeholderRE, geçerli template placeholder {{ad}} biçimlerini tanımlar.
var placeholderRE = regexp.MustCompile(`\{\{[a-zA-Z_][a-zA-Z0-9_]*\}\}`)

// placeholderTextBlocked, placeholder metnin kabul edilmediği HTML metin
// bağlamlarıdır: raw-text/RCDATA, template, select/option ve tablo.
var placeholderTextBlocked = map[atom.Atom]struct{}{
	atom.Script:    {},
	atom.Style:     {},
	atom.Textarea:  {},
	atom.Title:     {},
	atom.Select:    {},
	atom.Option:    {},
	atom.Table:     {},
	atom.Xmp:       {},
	atom.Iframe:    {},
	atom.Noembed:   {},
	atom.Noframes:  {},
	atom.Plaintext: {},
	atom.Noscript:  {},
	atom.Template:  {},
}

// placeholderTextAllowed, bir metin düğümünün normal HTML metni
// bağlamında olduğunu bildirir: en yakın kapsayan element HTML
// namespace'inde olmalı ve element zincirinde yasaklı bağlam bulunmamalıdır. HTML namespace
// foreignObject elementindeki normal HTML element metni bu kurala göre
// izinli sayılır.
func placeholderTextAllowed(n *html.Node) bool {
	p := n.Parent
	for p != nil && p.Type != html.ElementNode {
		p = p.Parent
	}
	if p == nil || p.Namespace != "" {
		return false
	}
	for e := p; e != nil; e = e.Parent {
		if e.Type == html.ElementNode {
			if _, blocked := placeholderTextBlocked[e.DataAtom]; blocked {
				return false
			}
		}
	}
	return true
}

// ValidateHTML yüklenen tam HTML belgesinin temel bütünlüğünü doğrular.
// Boş içerik, boyut sınırı aşımı, geçersiz UTF-8 ve NUL bayt reddedilir.
// golang.org/x/net/html tokenizer ile belgede gerçek <!doctype html>,
// <html>, </html>, <body> ve </body> varlığı denetlenir; script ve yorum
// içindeki sahte etiketler sayılmaz. HTML değiştirilmez ve template
// olarak parse edilmez; placeholder doğrulaması burada yapılmaz.
func ValidateHTML(data []byte) error {
	if len(data) == 0 {
		return errEmptyHTML
	}
	if len(data) > MaxHTMLSize {
		return errHTMLTooBig
	}
	if !utf8.Valid(data) {
		return errInvalidUTF8
	}
	if bytes.IndexByte(data, 0) != -1 {
		return errNULByte
	}

	var (
		doctypeHTML bool
		htmlStart   bool
		htmlEnd     bool
		bodyStart   bool
		bodyEnd     bool
	)

	zt := html.NewTokenizer(bytes.NewReader(data))
	for zt.Next() != html.ErrorToken {
		tok := zt.Token()
		switch tok.Type {
		case html.DoctypeToken:
			if strings.EqualFold(tok.Data, "html") {
				doctypeHTML = true
			}
		case html.StartTagToken:
			switch tok.Data {
			case "html":
				htmlStart = true
			case "body":
				bodyStart = true
			}
		case html.EndTagToken:
			switch tok.Data {
			case "html":
				htmlEnd = true
			case "body":
				bodyEnd = true
			}
		}
	}
	if err := zt.Err(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("HTML ayrıştırılamadı: %w", err)
	}

	switch {
	case !doctypeHTML:
		return errNoDoctype
	case !htmlStart:
		return errNoHTMLOpen
	case !htmlEnd:
		return errNoHTMLEnd
	case !bodyStart:
		return errNoBodyOpen
	case !bodyEnd:
		return errNoBodyEnd
	}
	return nil
}

// ValidatePlaceholders önce ValidateHTML temel bütünlük denetimini
// çalıştırır; ardından derleyiciyle aynı tokenizer ve ağaç kontrollerini
// kullanarak normal HTML metni dışındaki {{ad}} placeholderlarını reddeder. Attribute
// değerleri, yorumlar, script/style/textarea/title içerikleri, SVG/MathML
// gibi foreign namespace ve select/option/tablo metin bağlamlarındaki
// placeholderlar ve entity ile kodlanmış placeholderlar reddedilir. HTML namespace foreignObject altındaki
// normal HTML metni ile JS/CSS içindeki normal parantezler etkilenmez.
// Belge dönüştürülmez ya da alan kaydedilmez; kaynak aynen saklanır.
func ValidatePlaceholders(data []byte) error {
	_, _, err := compileHTML(data, nil)
	return err
}

func validatePlaceholderTree(data []byte) (*html.Node, error) {
	if err := ValidateHTML(data); err != nil {
		return nil, err
	}

	// Yapısal denetim ValidateHTML tarafında tamamlandı; Parse'ın
	// en iyi çaba ağacını kullanıp placeholder bağlamlarına bakarız.
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("HTML ayrıştırılamadı: %w", err)
	}

	seen := make(map[*html.Node]bool)
	var walk func(n *html.Node) error
	walk = func(n *html.Node) error {
		switch n.Type {
		case html.ElementNode:
			for _, a := range n.Attr {
				if placeholderRE.MatchString(a.Val) {
					return errPlaceholderAttr
				}
			}
		case html.CommentNode:
			if placeholderRE.MatchString(n.Data) {
				return errPlaceholderComment
			}
		case html.TextNode:
			if placeholderRE.MatchString(n.Data) && !placeholderTextAllowed(n) {
				return errPlaceholderText
			}
		}
		if seen[n] {
			return nil
		}
		seen[n] = true
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	return doc, walk(doc)
}
