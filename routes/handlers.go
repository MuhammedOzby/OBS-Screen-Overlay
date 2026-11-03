package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/store"
	"obs-screen-overlay/lib"
)

// RegisterRoutes, panelin rotalarını ve statik varlıkları r'e kaydedir.
// Şablonlar burada bir kez parse edilir; hata durumunda nil şablonla
// sunucu ayağa kalkmaz, hata ana programa geri döner.
// s nil olamaz: rotalar veritabanına bağlıdır.
func RegisterRoutes(r *gin.Engine, s *store.Store, publicOrigin ...string) error {
	if s == nil {
		return errors.New("store nil olamaz")
	}
	tpl, err := lib.LoadTemplates()
	if err != nil {
		return fmt.Errorf("panel şablonları yüklenemedi: %w", err)
	}
	r.SetHTMLTemplate(tpl)
	r.Static("/assets", "template/assets")

	r.GET("/", Index(s))
	r.GET("/access", AccessPage)
	r.GET("/cihazlar", DevicesPage)
	r.GET("/yeni", New)
	r.GET("/duzelt/:id", Edit(s, publicOrigin...))
	r.POST("/api/pages", createPage(s))
	r.PATCH("/api/pages/:id", updatePage(s))
	r.POST("/api/pages/:id/template", publishTemplate(s))
	r.DELETE("/api/pages/:id", deletePage(s))

	return nil
}

// recordView, index.html.tmpl'in listeleme görünümünde kullandığı veridir.
// (Go template'leri paket dışı fonksiyon çağırabildikleri için TypeLabel
//
//	burada önceden hesaplanır.)
type recordView struct {
	ID    string
	Name  string
	Type  string
	Label string
}

// indexData, index.html.tmpl'in beklediği veri yapısıdır.
type indexData struct {
	Records []recordView
	Filter  string
}

// newData, new.html.tmpl'in beklediği veri yapısıdır.
type newData struct {
	Types []string
}

// editData, edit.html.tmpl'in beklediği veri yapısıdır.
// Revision, PATCH /api/pages/:id istemindeki expected_revision olarak
// şablonda data-revision olarak taşınır (iyimser eşzamanlama / 409).
type editData struct {
	Record    lib.Record
	Revision  int64
	PublicURL string
}

// pageToRecord, store.Page'ı panel görünümü için lib.Record'a çevirir;
// Fields dahil edilir. SourceHTML ve PublicToken liste verisine taşınmaz.
func pageToRecord(p store.Page) lib.Record {
	rec := lib.Record{
		ID:   p.ID,
		Name: p.Name,
		Type: p.Type,
	}
	for _, f := range p.Fields {
		rec.Fields = append(rec.Fields, lib.Field{
			Name:       f.Name,
			Value:      f.Value,
			SSEEnabled: f.SSEEnabled,
		})
	}
	return rec
}

// Index, kayıtları filtreleme parametresiyle döner.
// GET ?filter=hepsi (varsayılan) | screen_overlay | widget
func Index(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter := c.Query("filter")
		switch filter {
		case "hepsi", "screen_overlay", "widget":
		default:
			filter = "hepsi"
		}
		// "hepsi" = filtre yok: ListPages boş türde tüm kayıtları döndürür.
		if filter == "hepsi" {
			filter = ""
		}

		pages, err := s.ListPages(c.Request.Context(), filter)
		if err != nil {
			c.Error(err)
			writeError(c, http.StatusInternalServerError, "Bir hata oluştu, lütfen daha sonra tekrar deneyin")
			return
		}

		views := make([]recordView, 0, len(pages))
		for _, page := range pages {
			views = append(views, recordView{
				ID:    page.ID,
				Name:  page.Name,
				Type:  page.Type,
				Label: lib.TypeLabel(page.Type),
			})
		}

		c.HTML(http.StatusOK, "index.html.tmpl", indexData{Records: views, Filter: filter})
	}
}

// New, yeni kayıt formu sayfasını döner.
// Form gönderimi POST /api/pages (createPage) tarafından işlenir.
func New(c *gin.Context) {
	c.HTML(http.StatusOK, "new.html.tmpl", newData{
		Types: []string{"screen_overlay", "widget"},
	})
}

// defaultPublicOrigin, public origin yapılandırılmadığında kullanılan varsayılan değer.
const defaultPublicOrigin = "http://127.0.0.1:8081"

// Edit, duyarlı olan, sayfa düzenleme sayfasını döner.
func Edit(s *store.Store, publicOrigin ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		origin := defaultPublicOrigin
		if len(publicOrigin) > 0 {
			origin = publicOrigin[0]
		}

		page, err := s.GetPage(c.Request.Context(), id)

		// Gereken başlıkları ayarla
		c.Header("Cache-Control", "no-store")
		c.Header("Referrer-Policy", "no-referrer")

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// sql.ErrNoRows -> 404
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
			// Diğer hatalar -> 500
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Başarılı yanıt
		editData := editData{
			Record:    pageToRecord(page),
			Revision:  page.Revision,
			PublicURL: origin + "/view/" + url.PathEscape(page.PublicToken),
		}

		c.HTML(http.StatusOK, "edit.html.tmpl", editData)
	}
}
