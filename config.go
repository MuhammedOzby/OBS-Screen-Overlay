package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

type appConfig struct {
	Addr         string   `yaml:"addr"`
	PublicAddr   string   `yaml:"public_addr"`
	PublicOrigin string   `yaml:"public_origin"`
	AdminOrigins []string `yaml:"admin_origins"`
	DBPath       string   `yaml:"db_path"`
}

func loadConfig(path string) (appConfig, []string, error) {
	var config appConfig
	f, err := os.Open(path)
	if err != nil {
		return config, nil, fmt.Errorf("yapılandırma %s açılamadı: %w", path, err)
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f, yaml.Strict())
	if err := decoder.Decode(&config); err != nil {
		return config, nil, fmt.Errorf("yapılandırma %s geçersiz: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("yalnızca bir YAML belgesi desteklenir")
		}
		return config, nil, fmt.Errorf("yapılandırma %s geçersiz: %w", path, err)
	}
	for name, value := range map[string]string{"addr": config.Addr, "public_addr": config.PublicAddr, "public_origin": config.PublicOrigin, "db_path": config.DBPath} {
		if strings.TrimSpace(value) == "" {
			return config, nil, fmt.Errorf("yapılandırma %s: %s zorunludur", path, name)
		}
	}
	if config.AdminOrigins == nil {
		return config, nil, fmt.Errorf("yapılandırma %s: admin_origins listesi zorunludur", path)
	}
	for _, origin := range config.AdminOrigins {
		if strings.Contains(origin, ",") || strings.TrimSpace(origin) != origin || origin == "" {
			return config, nil, fmt.Errorf("yapılandırma %s: geçersiz admin_origins girdisi", path)
		}
	}
	if err := validateConfig(config.Addr, config.PublicAddr, config.PublicOrigin); err != nil {
		return config, nil, fmt.Errorf("yapılandırma %s: %w", path, err)
	}
	origins, err := adminOrigins(config.Addr, config.PublicOrigin, strings.Join(config.AdminOrigins, ","))
	if err != nil {
		return config, nil, fmt.Errorf("yapılandırma %s: %w", path, err)
	}
	return config, origins, nil
}
