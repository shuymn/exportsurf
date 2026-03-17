package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
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

func TestRunIsDeterministicAcrossConcurrentCalls(t *testing.T) {
	repoRoot := repoRoot(t)
	opts := Options{
		Patterns:   []string{"./testdata/fixtures/basic/..."},
		WorkingDir: repoRoot,
		Rules: RulesFlags{
			Funcs: true, Types: true, Vars: true, Consts: true,
			Methods: true, Fields: true,
		},
	}

	want, err := Run(opts)
	if err != nil {
		t.Fatalf("initial Run failed: %v", err)
	}

	const workers = 8

	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			got, err := Run(opts)
			if err != nil {
				errs <- fmt.Errorf("concurrent Run failed: %w", err)
				return
			}
			if !reflect.DeepEqual(got, want) {
				errs <- fmt.Errorf(
					"concurrent Run produced non-deterministic results: want %#v got %#v",
					want,
					got,
				)
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
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
