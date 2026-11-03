package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const localConfig = `addr: "127.0.0.1:8080"
public_addr: "127.0.0.1:8081"
public_origin: "http://127.0.0.1:8081"
admin_origins: []
db_path: "overlay.db"
`

func writeConfig(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "configuration.yaml")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigDefaultsAndEnvironmentIgnored(t *testing.T) {
	for _, key := range []string{"ADDR", "PUBLIC_ADDR", "PUBLIC_ORIGIN", "ADMIN_ORIGINS", "DB_PATH"} {
		t.Setenv(key, "invalid-env-value")
	}
	for _, path := range []string{"configuration.yaml", writeConfig(t, localConfig)} {
		cfg, origins, err := loadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Addr != "127.0.0.1:8080" || cfg.PublicAddr != "127.0.0.1:8081" || cfg.PublicOrigin != "http://127.0.0.1:8081" || cfg.DBPath != "overlay.db" || len(cfg.AdminOrigins) != 0 {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		if !reflect.DeepEqual(origins, []string{"http://127.0.0.1:8080", "http://localhost:8080"}) {
			t.Fatalf("unexpected origins: %v", origins)
		}
	}
}

func TestLoadConfigLAN(t *testing.T) {
	data := strings.Replace(localConfig, "127.0.0.1:8080\"", "0.0.0.0:8080\"", 1)
	data = strings.Replace(data, "admin_origins: []", "admin_origins: [\"http://127.0.0.1:8080\", \"http://192.168.1.20:8080\"]", 1)
	_, origins, err := loadConfig(writeConfig(t, data))
	if err != nil || !reflect.DeepEqual(origins, []string{"http://127.0.0.1:8080", "http://192.168.1.20:8080"}) {
		t.Fatalf("origins=%v err=%v", origins, err)
	}
}

func TestLoadConfigRejectsInvalid(t *testing.T) {
	cases := map[string]string{
		"empty":               "",
		"malformed":           "addr: [\n",
		"unknown":             localConfig + "unknown: true\n",
		"duplicate":           localConfig + "addr: 'localhost:80'\n",
		"second document":     localConfig + "---\nunknown: true\n",
		"missing addr":        strings.Replace(localConfig, "addr: \"127.0.0.1:8080\"\n", "", 1),
		"missing list":        strings.Replace(localConfig, "admin_origins: []\n", "", 1),
		"null list":           strings.Replace(localConfig, "admin_origins: []", "admin_origins: null", 1),
		"wrong list type":     strings.Replace(localConfig, "admin_origins: []", "admin_origins: nope", 1),
		"empty db":            strings.Replace(localConfig, "db_path: \"overlay.db\"", "db_path: ''", 1),
		"invalid address":     strings.Replace(localConfig, "127.0.0.1:8080", "invalid", 1),
		"public collision":    strings.Replace(localConfig, "http://127.0.0.1:8081", "http://127.0.0.1:8080", 1),
		"lan no allowlist":    strings.Replace(localConfig, "127.0.0.1:8080", "0.0.0.0:8080", 1),
		"allowlist collision": strings.Replace(localConfig, "admin_origins: []", "admin_origins: ['http://127.0.0.1:8081']", 1),
		"comma list entry":    strings.Replace(localConfig, "admin_origins: []", "admin_origins: ['http://localhost:8080,http://evil:80']", 1),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := loadConfig(writeConfig(t, data)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	if _, _, err := loadConfig(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected missing file error")
	}
}
