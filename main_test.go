package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
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
		wantWithoutTests := []candidateReport{}
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
		wantWithTests := []candidateReport{}
		if !reflect.DeepEqual(withTests, wantWithTests) {
			t.Fatalf(
				"unexpected output when external tests are treated as external\nwant: %#v\ngot: %#v",
				wantWithTests,
				withTests,
			)
		}
	})

	t.Run("go test entrypoints are excluded", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/testrunner/...", "--json")

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output for test entrypoint fixture\nwant: %#v\ngot: %#v", want, got)
		}
	})
}

func TestRunTextOutput(t *testing.T) {
	t.Run("default output is text", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"scan", "./testdata/fixtures/basic/..."}, &stdout)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		want := "testdata/fixtures/basic/lib/lib.go:3: Candidate (type)\n" +
			"testdata/fixtures/basic/lib/lib.go:7: ExportedConst (const)\n" +
			"testdata/fixtures/basic/lib/lib.go:9: ExportedVar (var)\n" +
			"testdata/fixtures/basic/lib/lib.go:11: NewCandidate (func)\n"

		if stdout.String() != want {
			t.Fatalf("unexpected text output\nwant:\n%s\ngot:\n%s", want, stdout.String())
		}
	})

	t.Run("text output with baseline", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{
			"scan",
			"./testdata/fixtures/basic/...",
			"--baseline",
			"./testdata/baseline/basic.json",
		}, &stdout)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		want := "testdata/fixtures/basic/lib/lib.go:7: ExportedConst (const)\n" +
			"testdata/fixtures/basic/lib/lib.go:9: ExportedVar (var)\n" +
			"testdata/fixtures/basic/lib/lib.go:11: NewCandidate (func)\n"

		if stdout.String() != want {
			t.Fatalf("unexpected text output\nwant:\n%s\ngot:\n%s", want, stdout.String())
		}
	})

	t.Run("no candidates produces empty output", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"scan", "./testdata/fixtures/withtests/..."}, &stdout)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		if stdout.String() != "" {
			t.Fatalf("expected empty output, got:\n%s", stdout.String())
		}
	})
}

func TestRunBaselineContract(t *testing.T) {
	t.Run("scan --baseline --json filters candidates", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/basic/...",
			"--baseline",
			"./testdata/baseline/basic.json",
			"--json",
		)

		want := []candidateReport{
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
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan --baseline --json output\nwant: %#v\ngot: %#v", want, got)
		}
	})
}

func TestConfigContract(t *testing.T) {
	t.Run("scan and scan --baseline respect config-driven excludes", func(t *testing.T) {
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
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(scanGot, scanWant) {
			t.Fatalf("unexpected scan output with config\nwant: %#v\ngot: %#v", scanWant, scanGot)
		}

		baselineGot := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/basic/...",
			"--config",
			"./testdata/config/basic.yaml",
			"--baseline",
			"./testdata/baseline/basic.json",
			"--json",
		)

		baselineWant := []candidateReport{
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
		}

		if !reflect.DeepEqual(baselineGot, baselineWant) {
			t.Fatalf("unexpected baseline output with config\nwant: %#v\ngot: %#v", baselineWant, baselineGot)
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

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan output when config enables external tests\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("missing config keeps default behavior", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/withtests/...", "--json")

		want := []candidateReport{}
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

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan output with empty config\nwant: %#v\ngot: %#v", want, got)
		}
	})
}

func TestRunMethodScanning(t *testing.T) {
	t.Run("methods not reported by default", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/methods/...",
			"--json",
		)

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf(
				"expected no candidates without include_methods\nwant: %#v\ngot: %#v",
				want,
				got,
			)
		}
	})

	t.Run("methods reported with include_methods config", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/methods/...",
			"--config",
			"./testdata/config/methods.yaml",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/methods/lib.Container.Get",
				Kind:                "method",
				DefinedIn:           "testdata/fixtures/methods/lib/lib.go:23",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/methods/lib.MyType.InternalOnly",
				Kind:                "method",
				DefinedIn:           "testdata/fixtures/methods/lib/lib.go:7",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/methods/lib.MyType.Write",
				Kind:                "method",
				DefinedIn:           "testdata/fixtures/methods/lib/lib.go:11",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"satisfies interface io.Writer"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf(
				"unexpected method scanning output\nwant: %#v\ngot: %#v",
				want,
				got,
			)
		}
	})
}

func TestRunFieldScanning(t *testing.T) {
	t.Run("fields not reported by default", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/fields/...",
			"--json",
		)

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf(
				"expected no candidates without include_fields\nwant: %#v\ngot: %#v",
				want,
				got,
			)
		}
	})

	t.Run("fields reported with include_fields config", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/fields/...",
			"--config",
			"./testdata/config/fields.yaml",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.GenericStruct.Value",
				Kind:                "field",
				DefinedIn:           "testdata/fixtures/fields/lib/lib.go:13",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.MyStruct.EmbeddedType",
				Kind:                "field",
				DefinedIn:           "testdata/fixtures/fields/lib/lib.go:6",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"embedded field"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.MyStruct.InternalField",
				Kind:                "field",
				DefinedIn:           "testdata/fixtures/fields/lib/lib.go:4",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.MyStruct.TaggedField",
				Kind:                "field",
				DefinedIn:           "testdata/fixtures/fields/lib/lib.go:7",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"has serialization tag"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf(
				"unexpected field scanning output\nwant: %#v\ngot: %#v",
				want,
				got,
			)
		}
	})
}

func TestRunDiffRemoved(t *testing.T) {
	var stdout bytes.Buffer
	err := run([]string{
		"diff",
		"--baseline", "./testdata/baseline/basic.json",
		"./testdata/fixtures/basic/...",
	}, &stdout)
	if err == nil {
		t.Fatal("expected error for diff subcommand, got nil")
	}
}

func TestRunFailOnFindings(t *testing.T) {
	t.Run("findings with flag causes error", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"scan", "./testdata/fixtures/basic/...", "--json", "--fail-on-findings"}, &stdout)
		if !errors.Is(err, errFindingsFound) {
			t.Fatalf("expected errFindingsFound, got: %v", err)
		}

		// Output should still be written
		var got []candidateReport
		if jsonErr := json.Unmarshal(stdout.Bytes(), &got); jsonErr != nil {
			t.Fatalf("failed to decode JSON output: %v\n%s", jsonErr, stdout.Bytes())
		}
		if len(got) == 0 {
			t.Fatal("expected candidates in output")
		}
	})

	t.Run("no findings with flag exits normally", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"scan", "./testdata/fixtures/withtests/...", "--json", "--fail-on-findings"}, &stdout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("findings without flag exits normally", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"scan", "./testdata/fixtures/basic/...", "--json"}, &stdout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("text output with fail-on-findings", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"scan", "./testdata/fixtures/basic/...", "--fail-on-findings"}, &stdout)
		if !errors.Is(err, errFindingsFound) {
			t.Fatalf("expected errFindingsFound, got: %v", err)
		}

		if stdout.String() == "" {
			t.Fatal("expected text output before error")
		}
	})
}

func TestRunConfidenceScoring(t *testing.T) {
	t.Run("default preserves package main reason", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/confidence_main/...", "--json")

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/confidence_main.ExportedFromMain",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/confidence_main/main.go:3",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"package main"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("mark_main_low_confidence false removes package main reason", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"scan",
			"./testdata/fixtures/confidence_main/...",
			"--config",
			"./testdata/config/mark_main_false.yaml",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/confidence_main.ExportedFromMain",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/confidence_main/main.go:3",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("reflect cgo linkname patterns detected", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/confidence/...", "--json")

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.CgoExportedFunc",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/confidence/lib/lib.go:8",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"cgo export"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.LinkedFunc",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/confidence/lib/linkname.go:6",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"go:linkname"},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.NormalFunc",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/confidence/lib/lib.go:10",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "high",
				Reasons:             []string{},
			},
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.ReflectedType",
				Kind:                "type",
				DefinedIn:           "testdata/fixtures/confidence/lib/lib.go:5",
				InternalRefCount:    2,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"reflect usage"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("plugin usage detected", func(t *testing.T) {
		got := runCandidateCLI(t, "scan", "./testdata/fixtures/confidence_plugin/...", "--json")

		want := []candidateReport{
			{
				Symbol:              "github.com/shuymn/exportsurf/testdata/fixtures/confidence_plugin/lib.ExportedFunc",
				Kind:                "func",
				DefinedIn:           "testdata/fixtures/confidence_plugin/lib/lib.go:5",
				InternalRefCount:    1,
				ExternalRefPkgCount: 0,
				ExternalRefExamples: []string{},
				Confidence:          "low",
				Reasons:             []string{"plugin usage"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
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
