package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/store"
)

// publishTemplate, POST /api/pages/:id/template ile şablon yayınlamayı işler.
// Formda tam olarak tek expected_revision ve tek .html dosyası beklenir.
// Başarıda 200 ve {revision:newRev} JSON'u döner.
// Hata eşlemesi: ErrValidation=400, ErrConflict=409, sql.ErrNoRows=404, diğerleri=500 veya MaxBytesError=413.
func publishTemplate(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer cleanupMultipartForm(c.Request)

		form, err := parseUploadForm(c)
		if err != nil {
			return
		}
		if form == nil {
			writeError(c, http.StatusBadRequest, "multipart/form-data bir form gönderin")
			return
		}

		// Tam 1 expected_revision alanı ve tam 1 html dosyası; başka alan yok.
		if len(form.Value) != 1 || len(form.Value["expected_revision"]) != 1 ||
			len(form.File) != 1 || len(form.File["html"]) != 1 {
			writeError(c, http.StatusBadRequest, "Formda tam olarak 1 'expected_revision' ve 1 'html' dosyası olmalı")
			return
		}

		// 1. Expected Revision Kontrolü
		revisionStr := form.Value["expected_revision"][0]
		expectedRevision, err := strconv.ParseInt(revisionStr, 10, 64)
		if err != nil || expectedRevision < 0 {
			writeError(c, http.StatusBadRequest, "expected_revision geçerli, nonnegative bir base10 int64 olmalı")
			return
		}

		// 2. HTML Dosya Kontrolü
		htmlFile := form.File["html"][0]
		if !isHTMLFile(htmlFile.Filename) {
			writeError(c, http.StatusBadRequest, "Yalnızca .html uzantılı dosya yüklenebilir")
			return
		}

		// 3. Dosya Okuma ve Boyut Kontrolü
		data, err := readUploadFile(c, htmlFile)
		if err != nil {
			return
		}

		// 4. Yayınlama ve Hata Yönetimi
		id := c.Param("id")
		ctx := c.Request.Context()

		// store.PublishTemplate'den dönen revizyonu yakalar.
		newRev, publishErr := s.PublishTemplate(ctx, id, expectedRevision, string(data))

		if publishErr != nil {
			c.Error(publishErr)
			switch {
			case errors.Is(publishErr, store.ErrValidation):
				writeError(c, http.StatusBadRequest, "Geçersiz şablon içeriği veya revizyon bilgisi")
			case errors.Is(publishErr, store.ErrConflict):
				writeError(c, http.StatusConflict, "Revizyon çatışması: Kayıt bu sırada değişti")
			case errors.Is(publishErr, sql.ErrNoRows):
				writeError(c, http.StatusNotFound, "Kayıt bulunamadı")
			default:
				writeError(c, http.StatusInternalServerError, "Şablon yayınlanırken bir sunucu hatası oluştu")
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"revision": newRev})
	}
}
