package store

import (
	"database/sql"
	"errors"
	"testing"
)

func TestGetPageByPublicToken(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := Page{ID: "id", Name: "page", Type: "widget", SourceHTML: "html", PublicToken: "secret", Fields: []Field{{Name: "off", Value: "saved", SSEEnabled: false}}}
	if err := s.CreatePage(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetPageByPublicToken(t.Context(), "secret")
	if err != nil || got.ID != p.ID || len(got.Fields) != 1 || got.Fields[0].Value != "saved" || got.Fields[0].SSEEnabled {
		t.Fatalf("lookup = %+v, %v", got, err)
	}
	for _, token := range []string{"", "id", "wrong", "SECRET"} {
		if _, err := s.GetPageByPublicToken(t.Context(), token); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("invalid token: %v", err)
		}
	}
}
