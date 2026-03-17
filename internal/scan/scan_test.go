package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCurrentModuleDoesNotPanic(t *testing.T) {
	repoRoot := repoRoot(t)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Run panicked for current module: %v", recovered)
		}
	}()

	if _, err := Run(Options{
		Patterns:   []string{"./..."},
		WorkingDir: repoRoot,
		Rules: RulesFlags{
			Funcs: true, Types: true, Vars: true, Consts: true,
			Methods: true, Fields: true,
		},
	}); err != nil {
		t.Fatalf("Run returned error for current module: %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
