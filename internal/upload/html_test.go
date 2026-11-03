package upload

import (
	"errors"
	"strings"
	"testing"
)

// document, head ve body içeriğiyle tam bir HTML test belgesi üretir.
func document(head, body string) string {
	return "<!DOCTYPE html><html><head><meta charset=\"utf-8\">" + head + "</head><body>" + body + "</body></html>"
}

// wrapper, boş head'lı standart test belgesi üretir.
func wrapper(body string) string {
	return document("", body)
}

func TestValidateHTML(t *testing.T) {
	valid := []struct {
		name  string
		input string
	}{
		{"minimal", "<!doctype html><html><body></body></html>"},
		{"doctype uppercase", "<!DOCTYPE HTML><html><body></body></html>"},
		{"full document", wrapper(`<div>{{name}}</div>`)},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateHTML([]byte(tc.input)); err != nil {
				t.Fatalf("ValidateHTML(%q) = %v; hedef nil", tc.input, err)
			}
		})
	}

	invalid := []struct {
		name  string
		input string
		err   error
	}{
		{"empty", "", errEmptyHTML},
		{"too big", strings.Repeat("a", MaxHTMLSize+1), errHTMLTooBig},
		{"invalid utf8", "<!doctype html><html><body>\xff</body></html>", errInvalidUTF8},
		{"nul byte", "<!doctype html><html><body>\x00</body></html>", errNULByte},
		{"no doctype", "<html><body></body></html>", errNoDoctype},
		{"no html open", "<!doctype html><head></head><body></body></html>", errNoHTMLOpen},
		{"no html close", "<!doctype html><html><body></body>", errNoHTMLEnd},
		{"no body open", "<!doctype html><html><head></head></html>", errNoBodyOpen},
		{"no body close", "<!doctype html><html><body></html>", errNoBodyEnd},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateHTML([]byte(tc.input)); !errors.Is(err, tc.err) {
				t.Fatalf("ValidateHTML: hedef %v, alınan %v", tc.err, err)
			}
		})
	}
}

func TestValidatePlaceholders(t *testing.T) {
	valid := []struct {
		name string
		doc  string
	}{
		{"div text", wrapper(`<div>{{name}}</div>`)},
		{"p multiple", wrapper(`<p>selam {{ad}}, {{soyad_1}}</p>`)},
		{"svg foreignObject div", wrapper(`<svg width="20" height="20"><foreignObject width="20" height="20"><div>{{name}}</div></foreignObject></svg>`)},
		{"js normal braces", wrapper(`<script>function f() { return { a: 1, b: { c: 2 } }; }</script>`)},
		{"css normal braces", wrapper(`<style>.a::before { content: "" } .b { margin: 0 }</style>`)},
		{"textarea single braces", wrapper(`<textarea>{ deger }</textarea>`)},
		{"no placeholder", wrapper(`<section>Hello, world!</section>`)},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePlaceholders([]byte(tc.doc)); err != nil {
				t.Fatalf("ValidatePlaceholders = %v; hedef nil", err)
			}
		})
	}

	invalid := []struct {
		name string
		doc  string
		err  error
	}{
		{"base check first", "<html><body><div>{{name}}</div></body></html>", errNoDoctype},
		{"attribute", wrapper(`<div data-user="{{name}}">selam</div>`), errPlaceholderAttr},
		{"comment", wrapper(`<!-- {{name}} -->`), errPlaceholderComment},
		{"script", wrapper(`<script>var x = "{{name}}";</script>`), errPlaceholderText},
		{"style", wrapper(`<style>.a { content: "{{name}}" }</style>`), errPlaceholderText},
		{"textarea", wrapper(`<textarea>{{name}}</textarea>`), errPlaceholderText},
		{"title", document(`<title>{{name}}</title>`, ""), errPlaceholderText},
		{"svg text", wrapper(`<svg><text>{{name}}</text></svg>`), errPlaceholderText},
		{"mathml text", wrapper(`<math><mtext>{{name}}</mtext></math>`), errPlaceholderText},
		{"svg foreignObject direct text", wrapper(`<svg><foreignObject width="10" height="10">{{name}}</foreignObject></svg>`), errPlaceholderText},
		{"select", wrapper(`<select><option>{{name}}</option></select>`), errPlaceholderText},
		{"table", wrapper(`<table><tr><td>{{name}}</td></tr></table>`), errPlaceholderText},
		{"empty", "", errEmptyHTML},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePlaceholders([]byte(tc.doc)); !errors.Is(err, tc.err) {
				t.Fatalf("ValidatePlaceholders: hedef %v, alınan %v", tc.err, err)
			}
		})
	}
}
