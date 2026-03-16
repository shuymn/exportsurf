package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type candidateReport struct {
	Symbol              string   `json:"symbol"`
	Kind                string   `json:"kind"`
	DefinedIn           string   `json:"defined_in"`
	InternalRefCount    int      `json:"internal_ref_count"`
	ExternalRefPkgCount int      `json:"external_ref_pkg_count"`
	ExternalRefExamples []string `json:"external_ref_examples"`
	Confidence          string   `json:"confidence"`
	Reasons             []string `json:"reasons"`
}

func TestScanJSONContract(t *testing.T) {
	t.Run("basic fixture", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/basic/...", "--json")

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/cmd/tool.CommandCandidate",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/basic/cmd/tool/main.go:3",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"package main", "package under cmd"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.Candidate",
				Kind:                "type",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:3",
				InternalRefCount:    4,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedConst",
				Kind:                "const",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:7",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
				Kind:                "var",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:9",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.GeneratedCandidate",
				Kind:                "const",
				DefinedIn:           "testdata/fixtures/basic/lib/generated.go:5",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"generated file"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.UsedExternally",
				Kind:                "type",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:5",
				InternalRefCount:    0,
				ExternalRefPkgCount: 1,
				ExternalRefExamples: []string{"testdata/fixtures/basic/app/app.go:5"},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("external tests are opt-in", func(t *testing.T) {
		withoutTests := runCandidateCLI(t, "scan", "./testdata/fixtures/withtests/...", "--json")
		wantWithoutTests := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/withtests/lib.TestOnly",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/withtests/lib/lib.go:3",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}
		if !reflect.DeepEqual(withoutTests, wantWithoutTests) {
			t.Fatalf("unexpected output without external tests\nwant: %#v\ngot: %#v", wantWithoutTests, withoutTests)
		}

		withTests := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/withtests/...",
			"--json",
			"--treat-tests-as-external",
		)
		wantWithTests := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/withtests/lib.TestOnly",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/withtests/lib/lib.go:3",
				InternalRefCount:    0,
				ExternalRefPkgCount: 1,
				ExternalRefExamples: []string{"testdata/fixtures/withtests/lib/external_test.go:10"},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}
		if !reflect.DeepEqual(withTests, wantWithTests) {
			t.Fatalf(
				"unexpected output when external tests are treated as external\nwant: %#v\ngot: %#v",
				wantWithTests,
				withTests,
			)
		}
	})

	t.Run("go test entrypoints are downgraded but helpers remain", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/testrunner/...", "--json")

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/testrunner/lib.BenchmarkEntrypoint",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/testrunner/lib/lib_test.go:7",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"go test entrypoint"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/testrunner/lib.ExampleEntrypoint",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/testrunner/lib/lib_test.go:11",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"go test entrypoint"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/testrunner/lib.FuzzEntrypoint",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/testrunner/lib/lib_test.go:9",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"go test entrypoint"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/testrunner/lib.HelperAPI",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/testrunner/lib/lib_test.go:13",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/testrunner/lib.Placeholder",
				Kind:                "const",
				DefinedIn:           "testdata/fixtures/testrunner/lib/lib.go:3",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/testrunner/lib.TestEntrypoint",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/testrunner/lib/lib_test.go:5",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"go test entrypoint"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output for test entrypoint fixture\nwant: %#v\ngot: %#v", want, got)
		}
	})
}

func TestBaselineDiffContract(t *testing.T) {
	got := runCandidateCLI(
		t,
		"diff",
		"./testdata/fixtures/basic/...",
		"--baseline",
		"./testdata/baseline/basic.json",
	)

	want := []candidateReport{
		{
			Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/cmd/tool.CommandCandidate",
			Kind:                "func",
			DefinedIn:           "testdata/fixtures/basic/cmd/tool/main.go:3",
			InternalRefCount:    0,
			ExternalRefPkgCount: 0,
			ExternalRefExamples: []string{},
			Confidence:          "low",
			Reasons:             []string{"package main", "package under cmd"},
		},
		{
			Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedConst",
			Kind:                "const",
			DefinedIn:           "testdata/fixtures/basic/lib/lib.go:7",
			InternalRefCount:    1,
			ExternalRefPkgCount: 0,
			ExternalRefExamples: []string{},
			Confidence:          "high",
			Reasons:             []string{},
		},
		{
			Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
			Kind:                "var",
			DefinedIn:           "testdata/fixtures/basic/lib/lib.go:9",
			InternalRefCount:    1,
			ExternalRefPkgCount: 0,
			ExternalRefExamples: []string{},
			Confidence:          "high",
			Reasons:             []string{},
		},
		{
			Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
			Kind:                "func",
			DefinedIn:           "testdata/fixtures/basic/lib/lib.go:11",
			InternalRefCount:    1,
			ExternalRefPkgCount: 0,
			ExternalRefExamples: []string{},
			Confidence:          "high",
			Reasons:             []string{},
		},
		{
			Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.UsedExternally",
			Kind:                "type",
			DefinedIn:           "testdata/fixtures/basic/lib/lib.go:5",
			InternalRefCount:    0,
			ExternalRefPkgCount: 1,
			ExternalRefExamples: []string{"testdata/fixtures/basic/app/app.go:5"},
			Confidence:          "high",
			Reasons:             []string{},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected diff output\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestConfigContract(t *testing.T) {
	t.Run("scan and diff respect config-driven excludes", func(t *testing.T) {
		scanGot := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/basic/...",
			"--config",
			"./testdata/config/basic.yaml",
			"--json",
		)

		scanWant := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.Candidate",
				Kind:                "type",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:3",
				InternalRefCount:    4,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
				Kind:                "var",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:9",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.GeneratedCandidate",
				Kind:                "const",
				DefinedIn:           "testdata/fixtures/basic/lib/generated.go:5",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"generated file"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.UsedExternally",
				Kind:                "type",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:5",
				InternalRefCount:    0,
				ExternalRefPkgCount: 1,
				ExternalRefExamples: []string{"testdata/fixtures/basic/app/app.go:5"},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(scanGot, scanWant) {
			t.Fatalf("unexpected scan output with config\nwant: %#v\ngot: %#v", scanWant, scanGot)
		}

		diffGot := runCandidateCLI(
			t,
			"diff",
			"./testdata/fixtures/basic/...",
			"--config",
			"./testdata/config/basic.yaml",
			"--baseline",
			"./testdata/baseline/basic.json",
		)

		diffWant := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
				Kind:                "var",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:9",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.UsedExternally",
				Kind:                "type",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:5",
				InternalRefCount:    0,
				ExternalRefPkgCount: 1,
				ExternalRefExamples: []string{"testdata/fixtures/basic/app/app.go:5"},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(diffGot, diffWant) {
			t.Fatalf("unexpected diff output with config\nwant: %#v\ngot: %#v", diffWant, diffGot)
		}
	})

	t.Run("config can enable treat_tests_as_external", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/withtests/...",
			"--config",
			"./testdata/config/withtests.yaml",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/withtests/lib.TestOnly",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/withtests/lib/lib.go:3",
				InternalRefCount:    0,
				ExternalRefPkgCount: 1,
				ExternalRefExamples: []string{"testdata/fixtures/withtests/lib/external_test.go:10"},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan output when config enables external tests\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("missing config keeps default behavior", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/withtests/...", "--json")

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/withtests/lib.TestOnly",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/withtests/lib/lib.go:3",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan output without config\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("empty config keeps default behavior", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "empty.yaml")
		if err := os.WriteFile(configPath, nil, 0o600); err != nil {
			t.Fatalf("write empty config: %v", err)
		}

		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/withtests/...",
			"--config",
			configPath,
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/withtests/lib.TestOnly",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/withtests/lib/lib.go:3",
				InternalRefCount:    0,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan output with empty config\nwant: %#v\ngot: %#v", want, got)
		}
	})
}

func runCandidateCLI(t *testing.T, args ...string) []candidateReport {
	t.Helper()

	var stdout bytes.Buffer
	if err := run(args, &stdout); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	var got []candidateReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode JSON output: %v\n%s", err, stdout.Bytes())
	}

	return got
}
