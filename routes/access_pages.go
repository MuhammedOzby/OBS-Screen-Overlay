package routes

import (
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
)

func AccessPage(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.HTML(http.StatusOK, "access.html.tmpl", gin.H{"Local": localAccessUI(c.Request)})
}

// localAccessUI only controls presentation; the access gate independently
// enforces the authorization. Forwarded headers are deliberately ignored.
func localAccessUI(r *http.Request) bool {
	remote, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip, err := netip.ParseAddr(remote)
	if err != nil || !ip.Unmap().IsLoopback() {
		return false
	}
	host := r.Host
	if strings.HasPrefix(host, "[") || strings.Contains(host, ":") {
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			if strings.HasPrefix(r.Host, "[") && strings.HasSuffix(r.Host, "]") {
				host = strings.Trim(r.Host, "[]")
			} else {
				return false
			}
		}
	}
	if host == "localhost" {
		return true
	}
	ip, err = netip.ParseAddr(host)
	return err == nil && ip.Unmap().IsLoopback()
}

func DevicesPage(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "devices.html.tmpl", nil)
}
