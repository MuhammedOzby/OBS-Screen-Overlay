package routes

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/store"
)

// deletePage, DELETE /api/pages/:id silme isteğini işler.
// :id parametresini Store.DeletePage'e iletir. Kayıt bulunamazsa 404,
// beklenmeyen veritabanı hatalarında 500 döner; başarıda 204 (gövdesiz) döner.
func deletePage(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		err := s.DeletePage(c.Request.Context(), id)
		if err != nil {
			c.Error(err)
			if errors.Is(err, sql.ErrNoRows) {
				writeError(c, http.StatusNotFound, "Kayıt bulunamadı")
				return
			}
			writeError(c, http.StatusInternalServerError, "Bir hata oluştu, lütfen daha sonra tekrar deneyin")
			return
		}

		c.Status(http.StatusNoContent)
	}
}
