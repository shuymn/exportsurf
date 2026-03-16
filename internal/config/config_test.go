package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("empty file keeps defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.yaml")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatalf("write empty config: %v", err)
		}

		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load returned error for empty file: %v", err)
		}

		if len(got.ExcludePackages) != 0 || len(got.ExcludeSymbols) != 0 || got.TreatTestsAsExternal {
			t.Fatalf("Load returned non-zero config for empty file: %#v", got)
		}
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "unknown.yaml")
		if err := os.WriteFile(path, []byte("unknown_key: true\n"), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		if _, err := Load(path); err == nil {
			t.Fatal("Load succeeded for config with unknown field")
		}
	})
}
