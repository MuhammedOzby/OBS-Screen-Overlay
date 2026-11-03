package upload

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestCompileHTMLFieldsAndPreservation(t *testing.T) {
	source := []byte(document(`<style> .a { color: red; } </style><script>const x = "<tag>&";</script>`, `<svg viewBox='0 0 1 1'><text>unchanged &amp; exact</text><foreignObject><div>{{Name}}</div></foreignObject></svg><p>{{name}}|{{Name}}|{{name}}|{{missing}}|{{empty}}</p>`))
	copySource := bytes.Clone(source)
	value := `< & " ' </script>`
	out, fields, err := CompileHTML(source, map[string]string{"name": value, "Name": "üst", "empty": "", "unused": "ignored"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(source, copySource) {
		t.Fatal("source changed")
	}
	if !reflect.DeepEqual(fields, []string{"Name", "name", "missing", "empty"}) {
		t.Fatalf("fields = %#v", fields)
	}
	expected := strings.ReplaceAll(string(source), "{{Name}}", `<span data-overlay-field="Name">üst</span>`)
	expected = strings.ReplaceAll(expected, "{{name}}", `<span data-overlay-field="name">&lt; &amp; &#34; &#39; &lt;/script&gt;</span>`)
	expected = strings.ReplaceAll(expected, "{{missing}}", `<span data-overlay-field="missing"></span>`)
	expected = strings.ReplaceAll(expected, "{{empty}}", `<span data-overlay-field="empty"></span>`)
	if string(out) != expected {
		t.Fatalf("unexpected output:\n%s\nexpected:\n%s", out, expected)
	}
	doc, err := html.Parse(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" && len(n.Attr) > 0 && n.Attr[0].Val == "name" {
			count++
			if n.FirstChild == nil || n.FirstChild != n.LastChild || n.FirstChild.Type != html.TextNode || n.FirstChild.Data != value {
				t.Errorf("value is not a single text node: %#v", n.FirstChild)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if count != 2 {
		t.Fatalf("found %d repeated fields", count)
	}
}

func TestCompileHTMLRejectsUnsupportedContexts(t *testing.T) {
	cases := map[string]string{
		"attribute":          `<p title="{{x}}">a</p>`,
		"comment":            `<!-- {{x}} -->`,
		"script":             `<script>{{x}}</script>`,
		"style":              `<style>{{x}}</style>`,
		"textarea":           `<textarea>{{x}}</textarea>`,
		"title":              `<title>{{x}}</title>`,
		"table foster":       `<table>{{x}}<tr><td>ok</td></tr></table>`,
		"table cell":         `<table><tr><td>{{x}}</td></tr></table>`,
		"select":             `<select><option>{{x}}</option></select>`,
		"xmp":                `<xmp>{{x}}</xmp>`,
		"iframe":             `<iframe>{{x}}</iframe>`,
		"noembed":            `<noembed>{{x}}</noembed>`,
		"noframes":           `<noframes>{{x}}</noframes>`,
		"noscript":           `<noscript>{{x}}</noscript>`,
		"template":           `<template><div>{{x}}</div></template>`,
		"svg text":           `<svg><text>{{x}}</text></svg>`,
		"svg direct":         `<svg><foreignObject>{{x}}</foreignObject></svg>`,
		"math":               `<math><mtext>{{x}}</mtext></math>`,
		"entity placeholder": `<p>&#123;{x}}</p>`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			source := []byte(wrapper(body))
			if err := ValidatePlaceholders(source); err == nil {
				t.Error("validator accepted")
			}
			if _, _, err := CompileHTML(source, nil); err == nil {
				t.Error("compiler accepted")
			}
		})
	}
	// plaintext consumes the closing document tags, so basic validation fails too.
	if _, _, err := CompileHTML([]byte(wrapper(`<plaintext>{{x}}`)), nil); err == nil {
		t.Error("plaintext accepted")
	}
}

func TestCompileHTMLInvalidDocuments(t *testing.T) {
	for _, source := range [][]byte{nil, []byte("<html></html>"), []byte(wrapper("\xff")), []byte(wrapper("\x00")), []byte(strings.Repeat("x", MaxHTMLSize+1))} {
		if _, _, err := CompileHTML(source, nil); err == nil {
			t.Errorf("accepted invalid input of size %d", len(source))
		}
	}
	output, fields, err := CompileHTML([]byte(wrapper("no fields")), nil)
	if err != nil || string(output) != wrapper("no fields") || len(fields) != 0 {
		t.Fatalf("empty fields: %q %v %v", output, fields, err)
	}
}
