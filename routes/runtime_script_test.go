package routes

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// runtimeBehaviorTestPath is the Node built-in test suite that executes the
// real overlay runtime script inside a VM context and asserts its behavior.
const runtimeBehaviorTestPath = "tests/runtime.test.cjs"

// nodeForRuntimeBehaviorTest resolves the Node binary used by the runtime
// behavior tests. OBS_NODE takes precedence; otherwise a node binary found on
// PATH is used. An empty result means the behavior tests are skipped.
func nodeForRuntimeBehaviorTest(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("OBS_NODE"); p != "" {
		return p
	}
	if p, err := exec.LookPath("node"); err == nil {
		return p
	}
	return ""
}

// TestOverlayRuntimeScript runs tests/runtime.test.cjs, which executes the real
// overlay-runtime.js source through Node's built-in test runner and VM context,
// and fails when the runtime behavior does not match the SSE contract.
func TestOverlayRuntimeScript(t *testing.T) {
	node := nodeForRuntimeBehaviorTest(t)
	if node == "" {
		t.Skip("Node bulunamadı; overlay runtime davranış testleri çalıştırılamıyor")
	}
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("depo kökü çözülemedi: %v", err)
	}
	testPath := filepath.Join(repoRoot, runtimeBehaviorTestPath)
	for _, file := range []string{testPath, filepath.Join(repoRoot, "routes", "overlay-runtime.js")} {
		if _, err := os.ReadFile(file); err != nil {
			t.Fatalf("runtime test girdisi okunamadı (%s): %v", file, err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--test", testPath)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("overlay runtime davranış testleri başarısız: %v\n%s", err, out)
	}
}
