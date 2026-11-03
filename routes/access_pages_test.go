package routes

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccessPageLocalButton(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.SetHTMLTemplate(template.Must(template.ParseFiles("../template/access.html.tmpl")))
	r.GET("/access", AccessPage)
	tests := []struct {
		name, remote, host, forwarded string
		local                         bool
	}{
		{"ipv4", "127.0.0.1:1234", "127.0.0.1:8080", "", true},
		{"localhost", "127.0.0.2:1234", "localhost:8080", "", true},
		{"ipv6", "[::1]:1234", "[::1]:8080", "", true},
		{"ipv6 no port", "[::1]:1234", "[::1]", "", true},
		{"mapped ipv4", "[::ffff:127.0.0.1]:1234", "127.0.0.1", "", true},
		{"remote host", "127.0.0.1:1234", "overlay.test:8080", "127.0.0.1", false},
		{"remote peer", "192.168.1.20:1234", "localhost:8080", "127.0.0.1", false},
		{"missing peer port", "127.0.0.1", "localhost", "", false},
		{"invalid host", "127.0.0.1:1234", "localhost:bad:port", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/access", nil)
			req.RemoteAddr, req.Host = tc.remote, tc.host
			req.Header.Set("X-Forwarded-For", tc.forwarded)
			req.Header.Set("X-Forwarded-Host", "localhost:8080")
			req.Header.Set("Forwarded", "for=127.0.0.1;host=localhost:8080")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d", w.Code)
			}
			visible := strings.Contains(w.Body.String(), `<div><hr><button class="btn btn-subtle" id="local-login"`)
			hidden := strings.Contains(w.Body.String(), `<div hidden><hr><button class="btn btn-subtle" id="local-login"`)
			if visible != tc.local || hidden == tc.local {
				t.Errorf("local=%v, visible=%v, hidden=%v", tc.local, visible, hidden)
			}
		})
	}
}
