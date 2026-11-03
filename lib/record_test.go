package lib

import "testing"

func TestTypeLabel(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{name: "screen_overlay", typ: "screen_overlay", want: "Screen Overlay"},
		{name: "widget", typ: "widget", want: "Widget"},
		{name: "bilinmeyen tip", typ: "banner", want: "banner"},
		{name: "boş tip", typ: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TypeLabel(tt.typ); got != tt.want {
				t.Errorf("TypeLabel(%q) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}
