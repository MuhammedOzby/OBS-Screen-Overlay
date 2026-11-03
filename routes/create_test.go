package routes

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
)

func TestCreatePageSuccess(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages", createPage(s))

	html := "<!doctype html><html><body>{{konuk_adi}}</body></html>"
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("name", " Test "); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("type", "widget"); err != nil {
		t.Fatal(err)
	}
	fw, err := w.CreateFormFile("html", "test.HTML")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(html)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/pages", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var out struct {
		ID       string `json:"id"`
		Redirect string `json:"redirect"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("JSON okunamadı: %v", err)
	}
	if out.ID == "" || out.Redirect == "" {
		t.Fatalf("id/redirect boş: %+v", out)
	}

	p, err := s.GetPage(t.Context(), out.ID)
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if p.Name != "Test" {
		t.Errorf("name = %q, istenen Test", p.Name)
	}
	if p.SourceHTML != html {
		t.Errorf("source birebir değil: %q", p.SourceHTML)
	}
	if len(p.Fields) != 0 {
		t.Errorf("fields beklenirken %d değer var", len(p.Fields))
	}
	if len(p.PublicToken) != 64 {
		t.Errorf("token uzunluğu = %d, istenen 64", len(p.PublicToken))
	}
	if len(out.ID) != 32 {
		t.Errorf("id uzunluğu = %d, istenen 32", len(out.ID))
	}
}
