package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
)

// maxUpdateBytes, PATCH /api/pages/:id JSON gövdesi için tek üst sınır (8 MiB).
// Sınır aşılınca okuma http.MaxBytesError ile kesilir ve 413 döner.
const maxUpdateBytes = 8 << 20

// updateRequest, PATCH /api/pages/:id için beklenen JSON gövdesidir.
// Tüm alanlar zorunludur: eksik ya da null olan her alan reddedilir.
// Pointer alanlar "eksik"i sıfır değerden ayrı tutmamı sağlar.
type updateRequest struct {
	ExpectedRevision *int64         `json:"expected_revision"`
	Name             *string        `json:"name"`
	Type             *string        `json:"type"`
	Fields           *[]updateField `json:"fields"`
}

// updateField, güncelleme isteğindeki tek bir alanı tanımlar; tüm alanları zorunludur.
type updateField struct {
	Name       *string `json:"name"`
	Value      *string `json:"value"`
	SSEEnabled *bool   `json:"sse_enabled"`
}

// updatePage, PATCH /api/pages/:id güncelleme isteğini işler.
// expected_revision, name, type ve fields alanlı bir JSON gövde alır.
// İş kuralları (name, type, alan adları, yinelemeler) UpdatePage tarafından
// uygulanır; handler yalnızca eksik/null değerleri reddeder ve hataları durum
// kodlarına çevirir: ErrValidation=400, ErrConflict=409, sql.ErrNoRows=404,
// MaxBytesError=413, diğerleri=500. Content-Type tam application/json
// (params hariç) ve gövde tam olarak tek JSON değerinden oluşmalı
// (trailing içerik=400). Başarıda 200 ve {revision} JSON'u döner.
func updatePage(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		r := c.Request

		medype, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || medype != "application/json" {
			writeError(c, http.StatusUnsupportedMediaType, "Content-Type application/json olmalı")
			return
		}

		// Toplam istem üst sınırı: 8 MiB. Sınır aşılınca 413 ile kesilir.
		r.Body = http.MaxBytesReader(c.Writer, r.Body, int64(maxUpdateBytes))

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var payload updateRequest
		if err := dec.Decode(&payload); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(c, http.StatusRequestEntityTooLarge, "İstek boyutu sınırı aşıyor (en fazla 8 MiB)")
				return
			}
			if errors.Is(err, io.EOF) {
				writeError(c, http.StatusBadRequest, "JSON body boş olamaz")
				return
			}
			c.Error(err)
			writeError(c, http.StatusBadRequest, "JSON okunamadı")
			return
		}

		// Gövde tam olarak tek bir JSON değerinden oluşmalı: ikinci Decode
		// kesinlikle io.EOF vermeli. Trailing ikinci JSON veya garbage → 400.
		if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(c, http.StatusRequestEntityTooLarge, "İstek boyutu sınırı aşıyor (en fazla 8 MiB)")
				return
			}
			writeError(c, http.StatusBadRequest, "Gövde tam olarak tek bir JSON nesnesi olmalı (trailing içerik yok)")
			return
		}

		// Zorunlu alanlar: eksik veya null olan her biri reddedilir.
		if payload.ExpectedRevision == nil {
			writeError(c, http.StatusBadRequest, "expected_revision eksik veya null")
			return
		}
		if payload.Name == nil {
			writeError(c, http.StatusBadRequest, "name eksik veya null")
			return
		}
		if payload.Type == nil {
			writeError(c, http.StatusBadRequest, "type eksik veya null")
			return
		}
		if payload.Fields == nil {
			writeError(c, http.StatusBadRequest, "fields eksik veya null")
			return
		}

		id := c.Param("id")
		if id == "" {
			writeError(c, http.StatusNotFound, "Kayıt bulunamadı")
			return
		}

		fields := make([]store.Field, 0, len(*payload.Fields))
		for i, f := range *payload.Fields {
			if f.Name == nil || f.Value == nil || f.SSEEnabled == nil {
				writeError(c, http.StatusBadRequest,
					fmt.Sprintf("fields[%d]: name, value ve sse_enabled zorunlu, eksik veya null olamaz", i))
				return
			}
			fields = append(fields, store.Field{
				Name:       *f.Name,
				Value:      *f.Value,
				SSEEnabled: *f.SSEEnabled,
			})
		}

		rev, err := s.UpdatePage(c.Request.Context(), id,
			*payload.ExpectedRevision, *payload.Name, *payload.Type, fields)
		if err != nil {
			c.Error(err)
			switch {
			case errors.Is(err, store.ErrValidation):
				writeError(c, http.StatusBadRequest, "Geçersiz girdi")
			case errors.Is(err, store.ErrConflict):
				writeError(c, http.StatusConflict, "Revizyon çatışması: kayıt bu sırada değişti")
			case errors.Is(err, sql.ErrNoRows):
				writeError(c, http.StatusNotFound, "Kayıt bulunamadı")
			default:
				writeError(c, http.StatusInternalServerError, "Bir hata oluştu, lütfen daha sonra tekrar deneyin")
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"revision": rev})
	}
}
