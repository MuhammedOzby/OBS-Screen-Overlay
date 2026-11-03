package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
)

// TestUpdateHTTP, PATCH /api/pages/:id'ın başarılı güncellemeyi 200 ile
// kaydettiğini, aynı expected_revision'ın çatışma (409) ve yok id'nin 404
// döndürdüğünü doğrular.
func TestUpdateHTTP(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Şema revision varsayılanı 1 olduğundan expected_revision=1 eşleşir.
	if err := s.CreatePage(t.Context(), store.Page{
		ID:          "p",
		Name:        "old",
		Type:        "widget",
		PublicToken: "t",
	}); err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/api/pages/:id", updatePage(s))

	body := `{"expected_revision":1,"name":"new","type":"widget","fields":[{"name":"konuk","value":"Ali","sse_enabled":false}]}`

	patch := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/pages/"+id, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1) Başarılı güncelleme: 200 ve {revision:2}; name/field/revision kalıcı.
	rec := patch("p")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (istenen 200), body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Revision int64 `json:"revision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("yanıt JSON okunamadı: %v", err)
	}
	if out.Revision != 2 {
		t.Errorf("yanıt revision = %d, istenen 2", out.Revision)
	}
	p, err := s.GetPage(t.Context(), "p")
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if p.Name != "new" {
		t.Errorf("name = %q, istenen new", p.Name)
	}
	if p.Revision != 2 {
		t.Errorf("revision = %d, istenen 2", p.Revision)
	}
	if len(p.Fields) != 1 || p.Fields[0].Name != "konuk" || p.Fields[0].Value != "Ali" || p.Fields[0].SSEEnabled {
		t.Errorf("field beklenenden farklı: %+v", p.Fields)
	}

	// 2) Aynı gövde tekrar: revision 2 olduğundan 409.
	if rec := patch("p"); rec.Code != http.StatusConflict {
		t.Fatalf("status = %d (istenen 409), body = %s", rec.Code, rec.Body.String())
	}

	// 3) Var olmayan id: 404.
	if rec := patch("missing"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d (istenen 404), body = %s", rec.Code, rec.Body.String())
	}
}
