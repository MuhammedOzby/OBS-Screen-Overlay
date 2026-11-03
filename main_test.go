package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		admin, public, origin string
		valid                 bool
	}{
		{"127.0.0.1:8080", "127.0.0.1:8081", "http://127.0.0.1:8081", true},
		{"127.0.0.1:8080", "127.0.0.1:8080", "http://127.0.0.1:8081", false},
		{"127.0.0.1:8080", "127.0.0.1:8081", "http://localhost:8080", false},
		{"127.0.0.1:8080", "127.0.0.1:8081", "http://localhost:8081/path", false},
		{"bad", "127.0.0.1:8081", "http://localhost:8081", false},
	}
	for _, tt := range tests {
		if err := validateConfig(tt.admin, tt.public, tt.origin); (err == nil) != tt.valid {
			t.Errorf("%+v: %v", tt, err)
		}
	}
}

func TestAdminOrigins(t *testing.T) {
	tests := []struct {
		name, addr, public, configured string
		want                           []string
		valid                          bool
	}{
		{"default", "127.0.0.1:9000", "http://127.0.0.1:8081", "", []string{"http://127.0.0.1:9000", "http://localhost:9000"}, true},
		{"ipv6 loopback", "[::1]:9000", "http://127.0.0.1:8081", "", []string{"http://127.0.0.1:9000", "http://localhost:9000", "http://[::1]:9000"}, true},
		{"lan requires explicit", "0.0.0.0:8080", "http://192.168.1.20:8081", "", nil, false},
		{"wildcard requires explicit", ":8080", "http://localhost:8081", "", nil, false},
		{"lan explicit", "0.0.0.0:8080", "http://192.168.1.20:8081", "http://127.0.0.1:8080, http://192.168.1.20:8080", []string{"http://127.0.0.1:8080", "http://192.168.1.20:8080"}, true},
		{"same canonical origin", "127.0.0.1:8080", "http://EXAMPLE.com", "http://example.COM:80", nil, false},
		{"same https default port", "127.0.0.1:8080", "https://example.com:443", "https://EXAMPLE.com", nil, false},
		{"path invalid", "127.0.0.1:8080", "http://localhost:8081", "http://localhost:8080/path", nil, false},
		{"empty entry invalid", "127.0.0.1:8080", "http://localhost:8081", "http://localhost:8080,", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adminOrigins(tt.addr, tt.public, tt.configured)
			if (err == nil) != tt.valid {
				t.Fatalf("got %v, error %v", got, err)
			}
			if tt.valid && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdminRecoveryDoesNotLogSecrets(t *testing.T) {
	var output bytes.Buffer
	previous := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &output
	t.Cleanup(func() { gin.DefaultErrorWriter = previous })
	r := gin.New()
	r.Use(adminRecovery())
	r.GET("/panic", func(c *gin.Context) { panic("panic-secret") })
	req := httptest.NewRequest(http.MethodGet, "/panic?token=query-secret", nil)
	req.Header.Set("Cookie", "overlay_session=cookie-secret")
	req.Header.Set("X-CSRF-Token", "csrf-secret")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	for _, secret := range []string{"panic-secret", "query-secret", "cookie-secret", "csrf-secret"} {
		if strings.Contains(output.String(), secret) {
			t.Errorf("error log contains %q", secret)
		}
	}
	if output.Len() != 0 {
		t.Errorf("unexpected recovery log: %s", output.String())
	}
}
