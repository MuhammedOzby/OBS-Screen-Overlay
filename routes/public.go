package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"net/url"

	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/store"
	"obs-screen-overlay/internal/upload"
)

// NewPublicRouter exposes only the read-only broadcast endpoint. Do not add
// request logging or Gin recovery middleware: both can log secret URLs.
func NewPublicRouter(s *store.Store) *gin.Engine {
	r := gin.New()
	RegisterPublicRoutes(r, s)
	return r
}

// RegisterPublicRoutes registers the read-only public endpoints on r. It is
// separate from NewPublicRouter so callers and tests can use the real route
// set without duplicating handler wiring.
func RegisterPublicRoutes(r *gin.Engine, s *store.Store) {
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	})

	// New endpoint for runtime JS injection
	r.GET("/system/overlay-runtime.js", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/javascript", OverlayRuntimeJS)
	})

	r.GET("/view/:token", func(c *gin.Context) {
		page, err := s.GetPageByPublicToken(c.Request.Context(), c.Param("token"))
		if errors.Is(err, sql.ErrNoRows) {
			c.Status(http.StatusNotFound)
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		values := make(map[string]string, len(page.Fields))
		for _, field := range page.Fields {
			values[field.Name] = field.Value
		}
		body, _, err := upload.CompileHTML([]byte(page.SourceHTML), values)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		revision := page.Revision
		templateVersion := page.TemplateVersion
		eventsURL := "/view/" + url.PathEscape(c.Param("token")) + "/events"

		finalBody, err := runtimeHTML(body, revision, templateVersion, eventsURL)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", finalBody)
	})
	r.GET("/view/:token/events", func(c *gin.Context) {
		serveEvents(c, s)
	})
}

// runtimeHTML processes the HTML body to inject system metadata and runtime script.
// It uses html.Tokenizer to find the closing </body> tag raw byte offset, satisfying the constraint of not using regex/strings replacement
// for structural detection, but rather byte search for position relative to the end tag.
func runtimeHTML(body []byte, revision int64, version int64, eventsURL string) ([]byte, error) {
	type systemState struct {
		Revision        int64  `json:"revision"`
		TemplateVersion int64  `json:"template_version"`
		EventsURL       string `json:"events_url"`
	}

	// 1. Marshal JSON metadata
	stateData := systemState{
		Revision:        revision,
		TemplateVersion: version,
		EventsURL:       eventsURL,
	}

	metaBytes, err := json.Marshal(stateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal system state: %w", err)
	}

	// 2. Define injected scripts
	// Safe escaping of JSON content into a script tag.
	stateTag := []byte(fmt.Sprintf(`<script type="application/json" id="overlay-system-state">%s</script>`, string(metaBytes)))
	scriptTag := []byte(`<script src="/system/overlay-runtime.js" defer></script>`)

	// 3. Locate the closing </body> tag raw byte offset
	z := html.NewTokenizer(bytes.NewReader(body))
	offset := 0
	n := -1
	for {
		typ := z.Next()
		if typ == html.ErrorToken {
			break
		}
		raw := z.Raw()
		if typ == html.EndTagToken && z.Token().Data == "body" {
			n = offset
			break
		}
		offset += len(raw)
	}

	if n < 0 {
		return nil, errors.New("closing </body> tag not found in compiled body")
	}

	// 4. Reconstruct the body: [Part before body end tag] + [Injections] + [Body end tag + Part after body end tag]

	// Calculate lengths
	bodyPrefix := body[:n]
	bodySuffix := body[n:]

	// Total size needed: Prefix + StateTag + ScriptTag + Suffix
	newBodySize := len(bodyPrefix) + len(stateTag) + len(scriptTag) + len(bodySuffix)
	newBody := make([]byte, 0, newBodySize)

	// Append prefix (Original body up to offset)
	newBody = append(newBody, bodyPrefix...)

	// Inject scripts immediately before </body>
	newBody = append(newBody, stateTag...)
	newBody = append(newBody, scriptTag...)

	// Append the original </body> tag and everything after it
	newBody = append(newBody, bodySuffix...)

	return newBody, nil
}
