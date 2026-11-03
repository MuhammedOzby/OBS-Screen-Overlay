package routes

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/store"
)

func TestEditPublicURLUsesConfiguredOriginAndStoredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := store.Page{ID: "record", Name: "Test", Type: "widget", PublicToken: "stored-secret", SourceHTML: "<!doctype html><html><body></body></html>"}
	if err := s.CreatePage(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	tpl := template.Must(template.ParseFiles("../template/edit.html.tmpl", "../template/index.html.tmpl", "../template/new.html.tmpl"))
	r := gin.New()
	r.SetHTMLTemplate(tpl)
	r.GET("/duzelt/:id", Edit(s, "https://broadcast.example:9443"))
	r.GET("/", Index(s))
	r.GET("/yeni", New)
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "hostile.example"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	w := get("/duzelt/record")
	body := w.Body.String()
	url := "https://broadcast.example:9443/view/stored-secret"
	if w.Code != http.StatusOK || strings.Count(body, url) != 2 || strings.Contains(body, "hostile.example") {
		t.Fatalf("edit response %d: %s", w.Code, body)
	}
	for _, want := range []string{`id="obs-url"`, `readonly`, `id="copy-obs-url"`, `target="_blank" rel="noopener noreferrer"`, "SSE açık alanlar kaydedildiğinde yayına uygulanır", ">Yayına uygula<"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s", want)
		}
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Error(w.Header())
	}
	for _, path := range []string{"/", "/yeni"} {
		if body := get(path).Body.String(); strings.Contains(body, p.PublicToken) {
			t.Errorf("token exposed in %s", path)
		}
	}
}
