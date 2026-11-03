package routes

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
	"obs-screen-overlay/internal/upload"
)

// createPage, POST /api/pages için multipart/form-data yüklemeyi işler.
// Formda tam olarak tek name, tek type ve tek html dosya alanı beklenir;
// başka değer alanları ya da dosya alanları reddedilir. Başarıda 201 ve
// {id, redirect} JSON'u döner; yanıt ve loglarda token ya da kaynak
// HTML yer almaz.
func createPage(s *store.Store) gin.HandlerFunc {
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

		// Tam 1 name, tam 1 type ve tam 1 html dosyası; başka alan yok.
		if len(form.Value) != 2 ||
			len(form.Value["name"]) != 1 ||
			len(form.Value["type"]) != 1 {
			writeError(c, http.StatusBadRequest, "Formda tam olarak 1 'name' ve 1 'type' alanı olmalı")
			return
		}
		if len(form.File) != 1 || len(form.File["html"]) != 1 {
			writeError(c, http.StatusBadRequest, "Formda tam olarak 1 'html' dosyası olmalı")
			return
		}

		if !isHTMLFile(form.File["html"][0].Filename) {
			writeError(c, http.StatusBadRequest, "Yalnızca .html uzantılı dosya yüklenebilir")
			return
		}

		name := strings.TrimSpace(form.Value["name"][0])
		if name == "" {
			writeError(c, http.StatusBadRequest, "İsim boş olamaz")
			return
		}

		typ := strings.TrimSpace(form.Value["type"][0])
		if typ != "screen_overlay" && typ != "widget" {
			writeError(c, http.StatusBadRequest, "Tür 'screen_overlay' veya 'widget' olmak zorunda")
			return
		}

		data, err := readUploadFile(c, form.File["html"][0])
		if err != nil {
			return
		}

		if err := upload.ValidatePlaceholders(data); err != nil {
			c.Error(err)
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		// ID ve PublicToken, sunucu tarafında, crypto/rand ile üretilir;
		// kullanıcıdan bu bilgiler alınmaz.
		id, err := randomHex(16)
		if err != nil {
			c.Error(err)
			writeError(c, http.StatusInternalServerError, "Benzersiz kimlik oluşturulamadı")
			return
		}
		token, err := randomHex(32)
		if err != nil {
			c.Error(err)
			writeError(c, http.StatusInternalServerError, "Benzersiz kimlik oluşturulamadı")
			return
		}

		if err := s.CreatePage(c.Request.Context(), store.Page{
			ID:          id,
			Name:        name,
			Type:        typ,
			SourceHTML:  string(data),
			PublicToken: token,
		}); err != nil {
			// Detay yalnızca sunucu tarafında loglanır; istemciye
			// dökülmez ve yanıt genel bir mesaj içerir.
			c.Error(err)
			writeError(c, http.StatusInternalServerError, "Bir hata oluştu, lütfen daha sonra tekrar deneyin")
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":       id,
			"redirect": "/duzelt/" + id,
		})
	}
}

// randomHex, crypto/rand ile n bayt rastgele veriyi hex metni olarak döndürür.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// writeError, {error: mesaj} biçiminde JSON hata yanıtı döndürür.
func writeError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}
