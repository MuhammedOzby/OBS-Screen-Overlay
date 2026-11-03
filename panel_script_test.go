package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// OBS_NODE takes precedence over a Node binary on PATH.
func nodeForPanelCheck() string {
	if p := os.Getenv("OBS_NODE"); p != "" {
		return p
	}
	if p, err := exec.LookPath("node"); err == nil {
		return p
	}
	return ""
}

func TestPanelScriptBehavior(t *testing.T) {
	node := nodeForPanelCheck()
	if node == "" {
		t.Skip("Node bulunamadı; panel davranış testleri çalıştırılamıyor")
	}
	// Files opened only by the Node subprocess are invisible to Go's test cache.
	for _, file := range []string{"template/assets/panel.js", "tests/panel.test.cjs"} {
		if _, err := os.ReadFile(file); err != nil {
			t.Fatalf("panel test girdisi okunamadı (%s): %v", file, err)
		}
	}
	abs, err := filepath.Abs("tests/panel.test.cjs")
	if err != nil {
		t.Fatalf("panel.js yolu çözülemedi: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, node, "--test", abs).CombinedOutput()
	if err != nil {
		t.Fatalf("panel davranış testleri başarısız: %v\n%s", err, out)
	}
}
