package routes

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsHTMLFile, dosya adı doğrulamasının .html kabul davranışını
// (büyük/küçük harf duyarsız) doğrular.
func TestIsHTMLFile(t *testing.T) {
	cases := []struct {
		filename string
		want     bool
	}{
		{"test.html", true},
		{"Test.HTML", true},
		{"widget.HTML", true},
		{"test.htm", false},
		{"test.html.bak", false},
		{"test.zip", false},
		{"test", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isHTMLFile(tc.filename); got != tc.want {
			t.Errorf("isHTMLFile(%q) = %v, istenen %v", tc.filename, got, tc.want)
		}
	}
}

// TestCleanupMultipartFormSafe, multipart formı henüz içermeyen request'lerde
// temizlik fonksiyonunun güvenli no-op olduğunu doğrular; parse başarısızlığında
// dekontaminasyonun patlamasını engeller.
func TestCleanupMultipartFormSafe(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/pages", nil)
	cleanupMultipartForm(req)
}

func diskUpload(t *testing.T) (*http.Request, *multipart.FileHeader, string) {
	t.Helper()
	t.Setenv("TMPDIR", t.TempDir())
	body, contentType := multipartBody("Test", "widget", "large.html", strings.Repeat("x", maxMemory+1))
	req := httptest.NewRequest(http.MethodPost, "/api/pages", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	if err := req.ParseMultipartForm(maxMemory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupMultipartForm(req) })
	header := req.MultipartForm.File["html"][0]
	file, err := header.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	diskFile, ok := file.(*os.File)
	if !ok {
		t.Fatalf("expected upload on disk, got %T", file)
	}
	return req, header, diskFile.Name()
}

func TestCleanupMultipartRemovesDiskFile(t *testing.T) {
	req, _, path := diskUpload(t)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temporary upload missing before cleanup: %v", err)
	}
	cleanupMultipartForm(req)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary upload remains after cleanup: %v", err)
	}
}

func TestReadUploadFileOpenFailureReturnsBadRequest(t *testing.T) {
	req, header, _ := diskUpload(t)
	cleanupMultipartForm(req) // Force a real FileHeader.Open error.
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = req
	data, err := readUploadFile(c, header)
	if err == nil || data != nil {
		t.Fatalf("read result = %q, %v; expected failure", data, err)
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %q; expected 400", response.Code, response.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error == "" {
		t.Fatalf("expected JSON error response: %q (%v)", response.Body.String(), err)
	}
}
