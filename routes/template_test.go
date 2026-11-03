package routes

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
)

// publishBodyFields, verilen alan ad=değer çiftlerine ek olarak tek html
// dosyasını içeren multipart/form-data gövdesi üretir; duplicate ve bilinmeyen
// form alanı senaryoları için kullanılır.
func publishBodyFields(fields [][2]string, fileName, html string) ([]byte, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for _, f := range fields {
		if err := w.WriteField(f[0], f[1]); err != nil {
			panic(err)
		}
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

// publishBody, publishTemplate isteği için tek expected_revision alanı ve
// tek html dosyasından oluşan multipart gövdesini hazırlar.
func publishBody(expectedRevision string, fileName, html string) ([]byte, string) {
	return publishBodyFields([][2]string{{"expected_revision", expectedRevision}}, fileName, html)
}

// publishDoc, test HTML belgeleri için gereken tam belge iskeletini kurar.
func publishDoc(content string) string {
	return "<!doctype html><html><head><meta charset=\"utf-8\"></head><body>" + content + "</body></html>"
}

// publishRequest, publishTemplate rotasına gerçek HTTP isteği atar ve yanıtı
// recorder döner.
func publishRequest(t *testing.T, r *gin.Engine, id, expectedRevision, fileName, html string) *httptest.ResponseRecorder {
	t.Helper()
	body, ct := publishBody(expectedRevision, fileName, html)
	req := httptest.NewRequest(http.MethodPost, "/api/pages/"+id+"/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// TestPublishTemplateHTTP, POST /api/pages/:id/template handler'ının gerçek
// HTTP davranışını doğrular: success, stale conflict, invalid HTML, missing
// kayıt, duplicate/bilinmeyen alan, uzantı, boyut ve malformed multipart
// senaryoları.
func TestPublishTemplateHTTP(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	if err := s.CreatePage(t.Context(), store.Page{
		ID:          "p",
		Name:        "Sayfa",
		Type:        "widget",
		PublicToken: "t",
	}); err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages/:id/template", publishTemplate(s))

	const validHTML = "<!doctype html><html><head><meta charset=\"utf-8\"></head><body><div>{{konuk_adi}}</div></body></html>"
	const invalidHTML = "<div>fragment</div>"

	// Sayfa, CreatePage sonrası revision=1 ve template_version=1 ile başlar.
	pageState := func() store.Page {
		t.Helper()
		p, err := s.GetPage(t.Context(), "p")
		if err != nil {
			t.Fatalf("GetPage: %v", err)
		}
		return p
	}

	// 1) Başarılı yayın: 200 {revision:2}; source, revision ve template_version kalıcı.
	rec := publishRequest(t, r, "p", "1", "test.html", validHTML)
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
		t.Fatalf("yanıt revision = %d, istenen 2", out.Revision)
	}
	p := pageState()
	if p.SourceHTML != validHTML {
		t.Errorf("source = %q, istenen %q", p.SourceHTML, validHTML)
	}
	if p.Revision != 2 {
		t.Errorf("revision = %d, istenen 2", p.Revision)
	}
	if p.TemplateVersion != 2 {
		t.Errorf("template_version = %d, istenen 2", p.TemplateVersion)
	}

	// 2) Stale revision: beklenen 1, mevcut 2 -> 409 ve mutation yok.
	if rec := publishRequest(t, r, "p", "1", "test.html", validHTML); rec.Code != http.StatusConflict {
		t.Fatalf("stale status = %d (istenen 409), body = %s", rec.Code, rec.Body.String())
	}
	p = pageState()
	if p.SourceHTML != validHTML || p.Revision != 2 || p.TemplateVersion != 2 {
		t.Errorf("stale sonrası mutation var: source=%q revision=%d template_version=%d", p.SourceHTML, p.Revision, p.TemplateVersion)
	}

	// 3) Geçersiz HTML: 400 ve mevcut kayıt bozunmadan kalır.
	if rec := publishRequest(t, r, "p", "2", "test.html", invalidHTML); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d (istenen 400), body = %s", rec.Code, rec.Body.String())
	}
	p = pageState()
	if p.SourceHTML != validHTML || p.Revision != 2 || p.TemplateVersion != 2 {
		t.Errorf("invalid sonrası mutation var: source=%q revision=%d template_version=%d", p.SourceHTML, p.Revision, p.TemplateVersion)
	}

	// 4) Var olmayan kayıt: 404.
	if rec := publishRequest(t, r, "missing", "1", "test.html", validHTML); rec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d (istenen 404), body = %s", rec.Code, rec.Body.String())
	}

	// 5) Duplicate expected_revision alanı: 400 ve mutation yok.
	body, ct := publishBodyFields([][2]string{
		{"expected_revision", "2"},
		{"expected_revision", "2"},
	}, "test.html", validHTML)
	req := httptest.NewRequest(http.MethodPost, "/api/pages/p/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("duplicate-field status = %d (istenen 400), body = %s", rec.Code, rec.Body.String())
	}
	p = pageState()
	if p.SourceHTML != validHTML || p.Revision != 2 || p.TemplateVersion != 2 {
		t.Errorf("duplicate alan sonrası mutation var: source=%q revision=%d template_version=%d", p.SourceHTML, p.Revision, p.TemplateVersion)
	}

	// 6) Bilinmeyen form alanı: 400 ve mutation yok.
	body, ct = publishBodyFields([][2]string{
		{"expected_revision", "2"},
		{"extra", "değer"},
	}, "test.html", validHTML)
	req = httptest.NewRequest(http.MethodPost, "/api/pages/p/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown-field status = %d (istenen 400), body = %s", rec.Code, rec.Body.String())
	}
	p = pageState()
	if p.SourceHTML != validHTML || p.Revision != 2 || p.TemplateVersion != 2 {
		t.Errorf("bilinmeyen alan sonrası mutation var: source=%q revision=%d template_version=%d", p.SourceHTML, p.Revision, p.TemplateVersion)
	}

	// 7) Uzantı kontrolü: .html dışı dosya adı -> 400 ve mutation yok.
	if rec := publishRequest(t, r, "p", "2", "test.zip", validHTML); rec.Code != http.StatusBadRequest {
		t.Fatalf("extension status = %d (istenen 400), body = %s", rec.Code, rec.Body.String())
	}
	p = pageState()
	if p.SourceHTML != validHTML || p.Revision != 2 || p.TemplateVersion != 2 {
		t.Errorf("uzantı sonrası mutation var: source=%q revision=%d template_version=%d", p.SourceHTML, p.Revision, p.TemplateVersion)
	}

	// 8) Boyut kontrolü: 2 MiB + 1 -> 413.
	if rec := publishRequest(t, r, "p", "2", "test.html", strings.Repeat("a", maxHTMLBytes+1)); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status = %d (istenen 413), body = %s", rec.Code, rec.Body.String())
	}
	p = pageState()
	if p.SourceHTML != validHTML || p.Revision != 2 || p.TemplateVersion != 2 {
		t.Errorf("boyut sonrası mutation var: source=%q revision=%d template_version=%d", p.SourceHTML, p.Revision, p.TemplateVersion)
	}

	// 9) Malformed multipart: 400.
	malformedReq := httptest.NewRequest(http.MethodPost, "/api/pages/p/template", bytes.NewReader([]byte("this is not a multipart body")))
	malformedReq.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	malformedRec := httptest.NewRecorder()
	r.ServeHTTP(malformedRec, malformedReq)
	if malformedRec.Code != http.StatusBadRequest {
		t.Fatalf("malformed status = %d (istenen 400), body = %s", malformedRec.Code, malformedRec.Body.String())
	}
	p = pageState()
	if p.SourceHTML != validHTML || p.Revision != 2 || p.TemplateVersion != 2 {
		t.Errorf("malformed sonrası mutation var: source=%q revision=%d template_version=%d", p.SourceHTML, p.Revision, p.TemplateVersion)
	}
}
