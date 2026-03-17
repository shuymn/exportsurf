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

	if _, err := Run(NewOptions(
		[]string{"./..."},
		repoRoot,
		false,
		nil,
		nil,
		NewRulesFlags(true, true, true, true, true, true),
		LowConfidenceFlags{},
	)); err != nil {
		t.Fatalf("Run returned error for current module: %v", err)
	}
}

func TestRunIsDeterministicAcrossConcurrentCalls(t *testing.T) {
	repoRoot := repoRoot(t)
	opts := NewOptions(
		[]string{"./testdata/fixtures/basic/..."},
		repoRoot,
		false,
		nil,
		nil,
		NewRulesFlags(true, true, true, true, true, true),
		LowConfidenceFlags{},
	)

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
