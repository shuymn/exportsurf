package baseline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsEmptySymbol(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(path, []byte(`[{"symbol":""}]`), 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty baseline symbol")
	}
	if !strings.Contains(err.Error(), "symbol is required") {
		t.Fatalf("expected symbol validation error, got %v", err)
	}
}
