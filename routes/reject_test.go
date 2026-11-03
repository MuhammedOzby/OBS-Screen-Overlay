package routes

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
)

// multipartBody, createPage istemi için multipart form verisi hazırlar.
func multipartBody(name, typ, fileName, html string) ([]byte, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if err := w.WriteField("name", name); err != nil {
		panic(err)
	}
	if err := w.WriteField("type", typ); err != nil {
		panic(err)
	}
	fw, err := w.CreateFormFile("html", fileName)
	if err != nil {
		panic(err)
	}
	if _, err := fw.Write([]byte(html)); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return b.Bytes(), w.FormDataContentType()
}

// TestCreateRejects, reddedilen her yüklemede beklenti durumuna göre
// 400/413 döndüğünü ve store'a tek satır bile yazılmadığını doğrular.
func TestCreateRejects(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages", createPage(s))

	doc := func(content string) string {
		return "<!doctype html><html><head><meta charset=\"utf-8\"></head><body>" + content + "</body></html>"
	}
	base := doc("")

	cases := []struct {
		name     string
		inName   string
		inType   string
		fileName string
		html     string
		want     int
	}{
		{"name whitespace", "   ", "widget", "test.html", base, http.StatusBadRequest},
		{"type invalid", "Test", "foo", "test.html", base, http.StatusBadRequest},
		{"filename zip", "Test", "widget", "test.zip", base, http.StatusBadRequest},
		{"fragment", "Test", "widget", "test.html", "<div>{{konuk_adi}}</div>", http.StatusBadRequest},
		{"UTF8 invalid", "Test", "widget", "test.html", doc("\xff"), http.StatusBadRequest},
		{"empty", "Test", "widget", "test.html", "", http.StatusBadRequest},
		{"placeholder attribute", "Test", "widget", "test.html", doc(`<div data-x="{{konuk_adi}}">selam</div>`), http.StatusBadRequest},
		{"SVG text", "Test", "widget", "test.html", doc(`<svg><text>{{konuk_adi}}</text></svg>`), http.StatusBadRequest},
		{"html 2MiB+1", "Test", "widget", "test.html", strings.Repeat("a", maxHTMLBytes+1), http.StatusRequestEntityTooLarge},
		{"body 3MiB", "Test", "widget", "test.html", strings.Repeat("a", 3*1024*1024), http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, ct := multipartBody(tc.inName, tc.inType, tc.fileName, tc.html)
			req := httptest.NewRequest(http.MethodPost, "/api/pages", bytes.NewReader(body))
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status = %d (beklenen %d), body = %s", rec.Code, tc.want, rec.Body.String())
			}
			pages, err := s.ListPages(t.Context(), "")
			if err != nil {
				t.Fatalf("ListPages: %v", err)
			}
			if len(pages) != 0 {
				t.Fatalf("reddedilmesine rağmen %d sayfa kaydedildi", len(pages))
			}
		})
	}
}
