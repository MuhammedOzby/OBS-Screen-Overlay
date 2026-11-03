package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
)

// TestDeleteHTTP, DELETE /api/pages/:id'in kaydı 204 ile sildiğini, silinen
// kaydın GetPage'de sql.ErrNoRows döndürdüğünü, tekrar silinmesinin 404 JSON
// hatası verdiğini ve hiç var olmayan id'nin de 404 olduğunu doğrular.
func TestDeleteHTTP(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.CreatePage(t.Context(), store.Page{
		ID:          "p",
		Name:        "Sayfa",
		Type:        "widget",
		PublicToken: "t",
		Fields:      []store.Field{{Name: "x", Value: "1"}},
	}); err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/api/pages/:id", deletePage(s))

	del := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/pages/"+id, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1) Başarılı silme: 204 ve gövdesiz yanıt.
	rec := del("p")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d (istenen 204), body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("204 gövdesi boş olmalı, body = %s", rec.Body.String())
	}

	// 2) Silinen kayıt GetPage'de sql.ErrNoRows olarak raporlanır.
	if _, err := s.GetPage(t.Context(), "p"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetPage hatası = %v, istenen sql.ErrNoRows", err)
	}

	// 3) Aynı id tekrar silinir: 404 JSON hata.
	rec = del("p")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d (istenen 404), body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("yanıt JSON okunamadı: %v", err)
	}
	if out.Error != "Kayıt bulunamadı" {
		t.Errorf("hata mesajı = %q, istenen Kayıt bulunamadı", out.Error)
	}

	// 4) Hiç var olmayan id: 404.
	if rec := del("missing"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d (istenen 404), body = %s", rec.Code, rec.Body.String())
	}
}

// TestDeleteStoreClosed, store kapatıldıktan sonra DELETE isteğinin 500 ve
// genel (bilgi sızdırmayan) hata JSON'i döndürdüğünü doğrular.
func TestDeleteStoreClosed(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/api/pages/:id", deletePage(s))

	req := httptest.NewRequest(http.MethodDelete, "/api/pages/p", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d (istenen 500), body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("yanıt JSON okunamadı: %v", err)
	}
	if out.Error != "Bir hata oluştu, lütfen daha sonra tekrar deneyin" {
		t.Errorf("hata mesajı = %q, istenen genel hata mesajı", out.Error)
	}
}
