package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/access"
	"obs-screen-overlay/internal/store"
	"obs-screen-overlay/routes"
)

// PUBLIC_ORIGIN is the externally visible origin used for panel URLs.
// Reverse proxies may map it to PUBLIC_ADDR; never infer it from a request Host.
func validateConfig(adminAddr, publicAddr, publicOrigin string) error {
	if _, _, err := net.SplitHostPort(adminAddr); err != nil {
		return fmt.Errorf("ADDR: %w", err)
	}
	if _, _, err := net.SplitHostPort(publicAddr); err != nil {
		return fmt.Errorf("PUBLIC_ADDR: %w", err)
	}
	if strings.EqualFold(adminAddr, publicAddr) {
		return errors.New("admin ve yayın listener adresleri aynı olamaz")
	}
	u, err := url.Parse(publicOrigin)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return errors.New("PUBLIC_ORIGIN yalnızca http(s) origin olmalıdır")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "http" {
			port = "80"
		} else {
			port = "443"
		}
	}
	host, adminPort, _ := net.SplitHostPort(adminAddr)
	if port == adminPort && (strings.EqualFold(host, u.Hostname()) || (isLoopback(host) && isLoopback(u.Hostname())) || host == "" || host == "0.0.0.0" || host == "::") {
		return errors.New("PUBLIC_ORIGIN admin origin ile çakışıyor")
	}
	return nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func adminRecovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// canonicalOrigin compares browser origins without conflating distinct hosts.
func canonicalOrigin(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", errors.New("geçersiz origin")
	}
	port := u.Port()
	if port == "" {
		if strings.HasSuffix(u.Host, ":") {
			return "", errors.New("geçersiz origin portu")
		}
		if u.Scheme == "http" {
			port = "80"
		} else {
			port = "443"
		}
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("geçersiz origin portu")
	}
	host := strings.ToLower(u.Hostname())
	if ip := net.ParseIP(host); ip != nil {
		host = ip.String()
	}
	return u.Scheme + "://" + net.JoinHostPort(host, strconv.Itoa(n)), nil
}

func adminOrigins(adminAddr, publicOrigin, configured string) ([]string, error) {
	host, port, err := net.SplitHostPort(adminAddr)
	if err != nil {
		return nil, fmt.Errorf("ADDR: %w", err)
	}
	var origins []string
	if configured == "" {
		if !isLoopback(host) {
			return nil, errors.New("loopback dışı ADDR için ADMIN_ORIGINS zorunludur")
		}
		origins = []string{"http://" + net.JoinHostPort("127.0.0.1", port), "http://" + net.JoinHostPort("localhost", port)}
		if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
			origins = append(origins, "http://"+net.JoinHostPort(ip.String(), port))
		}
	} else {
		for _, origin := range strings.Split(configured, ",") {
			origins = append(origins, strings.TrimSpace(origin))
		}
	}
	public, err := canonicalOrigin(publicOrigin)
	if err != nil {
		return nil, fmt.Errorf("PUBLIC_ORIGIN: %w", err)
	}
	for _, origin := range origins {
		canonical, err := canonicalOrigin(origin)
		if err != nil {
			return nil, fmt.Errorf("ADMIN_ORIGINS: %w", err)
		}
		if canonical == public {
			return nil, errors.New("ADMIN_ORIGINS yayın origin ile çakışıyor")
		}
	}
	return origins, nil
}

func run() error {
	config, origins, err := loadConfig("configuration.yaml")
	if err != nil {
		return err
	}
	s, err := store.Open(config.DBPath)
	if err != nil {
		return err
	}
	defer s.Close()
	manager, err := access.New(s, access.Config{AdminOrigins: origins})
	if err != nil {
		return err
	}
	admin := gin.New()
	admin.Use(adminRecovery())
	if err := admin.SetTrustedProxies(nil); err != nil {
		return err
	}
	if err := routes.RegisterRoutes(admin, s, config.PublicOrigin); err != nil {
		return err
	}
	public := routes.NewPublicRouter(s)
	adminListener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}
	defer adminListener.Close()
	publicListener, err := net.Listen("tcp", config.PublicAddr)
	if err != nil {
		return err
	}
	defer publicListener.Close()
	servers := []*http.Server{{Handler: manager.Wrap(admin), ReadHeaderTimeout: 10 * time.Second}, {Handler: public, ReadHeaderTimeout: 10 * time.Second}}
	errs := make(chan error, 2)
	go func() { errs <- servers[0].Serve(adminListener) }()
	go func() { errs <- servers[1].Serve(publicListener) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err = <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, server := range servers {
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			server.Close()
			if err == nil {
				err = shutdownErr
			}
		}
	}
	return err
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
