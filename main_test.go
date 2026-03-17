package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuymn/exportsurf/pkg/report"
)

type candidateReport struct {
	Symbol           string   `json:"symbol"`
	Kind             string   `json:"kind"`
	DefinedIn        string   `json:"src"`
	InternalRefCount int      `json:"internal_ref_count"`
	Confidence       string   `json:"confidence"`
	Reasons          []string `json:"reasons"`
}

func TestScanJSONContract(t *testing.T) {
	t.Run("basic fixture", func(t *testing.T) {
		got := runCandidateCLI(t, "./testdata/fixtures/basic/...", "--json")

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.Candidate",
				Kind:             "type",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:3",
				InternalRefCount: 4,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedConst",
				Kind:             "const",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:7",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
				Kind:             "var",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:9",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected scan output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("external tests are opt-in", func(t *testing.T) {
		withoutTests := runCandidateCLI(t, "./testdata/fixtures/withtests/...", "--json")
		wantWithoutTests := []candidateReport{}
		if !reflect.DeepEqual(withoutTests, wantWithoutTests) {
			t.Fatalf("unexpected output without external tests\nwant: %#v\ngot: %#v", wantWithoutTests, withoutTests)
		}

		withTests := runCandidateCLI(
			t,
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
		got := runCandidateCLI(t, "./testdata/fixtures/testrunner/...", "--json")

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output for test entrypoint fixture\nwant: %#v\ngot: %#v", want, got)
		}
	})
}

func TestScanJSONContractCurrentModuleHasNoCandidates(t *testing.T) {
	got := runCandidateCLI(t, "--json")

	want := []candidateReport{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected scan output for current module\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestRunTextOutput(t *testing.T) {
	t.Run("default output is text", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"./testdata/fixtures/basic/..."}, &stdout)
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
		err := run([]string{"./testdata/fixtures/withtests/..."}, &stdout)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		if stdout.String() != "" {
			t.Fatalf("expected empty output, got:\n%s", stdout.String())
		}
	})

	t.Run("control characters are escaped", func(t *testing.T) {
		var stdout bytes.Buffer
		err := writeOutput(&stdout, []report.Candidate{
			report.NewCandidate(
				"example.com/adversarial/lib.Type\nName",
				"type",
				"lib/\x1b[31mowned.go:3",
				1,
				report.ConfidenceLow,
				[]string{"bad\tinput", "line\nbreak"},
			),
		}, false, false)
		if err != nil {
			t.Fatalf("writeOutput failed: %v", err)
		}

		want := "lib/\\x1B[31mowned.go:3: Type\\x0AName (type) [low: bad\\x09input, line\\x0Abreak]\n"
		if stdout.String() != want {
			t.Fatalf("unexpected escaped text output\nwant:\n%s\ngot:\n%s", want, stdout.String())
		}
	})
}

func TestRunRejectsConflictingFormatFlagsWithoutStdout(t *testing.T) {
	var stdout bytes.Buffer

	err := run([]string{"./testdata/fixtures/basic/...", "--json", "--sarif"}, &stdout)
	if err == nil {
		t.Fatal("expected error for conflicting format flags")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for conflicting format flags, got %q", stdout.String())
	}
}

func TestRunBaselineContract(t *testing.T) {
	t.Run("scan --baseline --json filters candidates", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"./testdata/fixtures/basic/...",
			"--baseline",
			"./testdata/baseline/basic.json",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedConst",
				Kind:             "const",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:7",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
				Kind:             "var",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:9",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
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
			"./testdata/fixtures/basic/...",
			"--config",
			"./testdata/config/basic.yaml",
			"--json",
		)

		scanWant := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.Candidate",
				Kind:             "type",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:3",
				InternalRefCount: 4,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
				Kind:             "var",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:9",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
		}

		if !reflect.DeepEqual(scanGot, scanWant) {
			t.Fatalf("unexpected scan output with config\nwant: %#v\ngot: %#v", scanWant, scanGot)
		}

		baselineGot := runCandidateCLI(
			t,
			"./testdata/fixtures/basic/...",
			"--config",
			"./testdata/config/basic.yaml",
			"--baseline",
			"./testdata/baseline/basic.json",
			"--json",
		)

		baselineWant := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.ExportedVar",
				Kind:             "var",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:9",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.NewCandidate",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/basic/lib/lib.go:11",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
		}

		if !reflect.DeepEqual(baselineGot, baselineWant) {
			t.Fatalf("unexpected baseline output with config\nwant: %#v\ngot: %#v", baselineWant, baselineGot)
		}
	})

	t.Run("config can enable treat_tests_as_external", func(t *testing.T) {
		got := runCandidateCLI(
			t,
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
		got := runCandidateCLI(t, "./testdata/fixtures/withtests/...", "--json")

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

func TestConfigDiscovery(t *testing.T) {
	t.Run("discovers .exportsurf.yaml", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)
		if err := os.WriteFile(filepath.Join(dir, ".exportsurf.yaml"), nil, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if got := discoverConfig(); got != ".exportsurf.yaml" {
			t.Fatalf("expected .exportsurf.yaml, got %q", got)
		}
	})

	t.Run("discovers exportsurf.yml", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "exportsurf.yml"), nil, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if got := discoverConfig(); got != "exportsurf.yml" {
			t.Fatalf("expected exportsurf.yml, got %q", got)
		}
	})

	t.Run("priority order prefers dotfile yaml", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)
		for _, name := range []string{".exportsurf.yaml", "exportsurf.yaml"} {
			if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
		if got := discoverConfig(); got != ".exportsurf.yaml" {
			t.Fatalf("expected .exportsurf.yaml, got %q", got)
		}
	})

	t.Run("returns empty when no config found", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)
		if got := discoverConfig(); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})
}

func TestRunMethodScanning(t *testing.T) {
	t.Run("methods reported by default", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"./testdata/fixtures/methods/...",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/methods/lib.Container.Get",
				Kind:             "method",
				DefinedIn:        "testdata/fixtures/methods/lib/lib.go:23",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/methods/lib.MyType.InternalOnly",
				Kind:             "method",
				DefinedIn:        "testdata/fixtures/methods/lib/lib.go:7",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/methods/lib.MyType.Write",
				Kind:             "method",
				DefinedIn:        "testdata/fixtures/methods/lib/lib.go:11",
				InternalRefCount: 1,
				Confidence:       "low",
				Reasons:          []string{"satisfies_interface:io.Writer"},
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

	t.Run("methods excluded with rules.methods false", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"./testdata/fixtures/methods/...",
			"--config",
			"./testdata/config/methods.yaml",
			"--json",
		)

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf(
				"expected no method candidates with rules.methods=false\nwant: %#v\ngot: %#v",
				want,
				got,
			)
		}
	})
}

func TestRunFieldScanning(t *testing.T) {
	t.Run("fields reported by default", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"./testdata/fixtures/fields/...",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.GenericStruct.Value",
				Kind:             "field",
				DefinedIn:        "testdata/fixtures/fields/lib/lib.go:13",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.MyStruct.EmbeddedType",
				Kind:             "field",
				DefinedIn:        "testdata/fixtures/fields/lib/lib.go:6",
				InternalRefCount: 1,
				Confidence:       "low",
				Reasons:          []string{"embedded_field"},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.MyStruct.InternalField",
				Kind:             "field",
				DefinedIn:        "testdata/fixtures/fields/lib/lib.go:4",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/fields/lib.MyStruct.TaggedField",
				Kind:             "field",
				DefinedIn:        "testdata/fixtures/fields/lib/lib.go:7",
				InternalRefCount: 1,
				Confidence:       "low",
				Reasons:          []string{"serialization_tag"},
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

	t.Run("fields excluded with rules.fields false", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"./testdata/fixtures/fields/...",
			"--config",
			"./testdata/config/fields.yaml",
			"--json",
		)

		want := []candidateReport{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf(
				"expected no field candidates with rules.fields=false\nwant: %#v\ngot: %#v",
				want,
				got,
			)
		}
	})
}

func TestRunFailOnFindings(t *testing.T) {
	t.Run("findings with flag causes error", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"./testdata/fixtures/basic/...", "--json", "--fail-on-findings"}, &stdout)
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
		err := run([]string{"./testdata/fixtures/withtests/...", "--json", "--fail-on-findings"}, &stdout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("findings without flag exits normally", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"./testdata/fixtures/basic/...", "--json"}, &stdout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("text output with fail-on-findings", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"./testdata/fixtures/basic/...", "--fail-on-findings"}, &stdout)
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
		got := runCandidateCLI(t, "./testdata/fixtures/confidence_main/...", "--json")

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/confidence_main.ExportedFromMain",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/confidence_main/main.go:3",
				InternalRefCount: 1,
				Confidence:       "low",
				Reasons:          []string{"package_main"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("mark_main_low_confidence false removes package main reason", func(t *testing.T) {
		got := runCandidateCLI(
			t,
			"./testdata/fixtures/confidence_main/...",
			"--config",
			"./testdata/config/mark_main_false.yaml",
			"--json",
		)

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/confidence_main.ExportedFromMain",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/confidence_main/main.go:3",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("reflect cgo linkname patterns detected", func(t *testing.T) {
		got := runCandidateCLI(t, "./testdata/fixtures/confidence/...", "--json")

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.CgoExportedFunc",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/confidence/lib/lib.go:8",
				InternalRefCount: 1,
				Confidence:       "low",
				Reasons:          []string{"cgo_export"},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.LinkedFunc",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/confidence/lib/linkname.go:6",
				InternalRefCount: 1,
				Confidence:       "low",
				Reasons:          []string{"linkname"},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.NormalFunc",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/confidence/lib/lib.go:10",
				InternalRefCount: 1,
				Confidence:       "high",
				Reasons:          []string{},
			},
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/confidence/lib.ReflectedType",
				Kind:             "type",
				DefinedIn:        "testdata/fixtures/confidence/lib/lib.go:5",
				InternalRefCount: 2,
				Confidence:       "low",
				Reasons:          []string{"reflect_usage"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
		}
	})

	t.Run("plugin usage detected", func(t *testing.T) {
		got := runCandidateCLI(t, "./testdata/fixtures/confidence_plugin/...", "--json")

		want := []candidateReport{
			{
				Symbol:           "github.com/shuymn/exportsurf/testdata/fixtures/confidence_plugin/lib.ExportedFunc",
				Kind:             "func",
				DefinedIn:        "testdata/fixtures/confidence_plugin/lib/lib.go:5",
				InternalRefCount: 1,
				Confidence:       "low",
				Reasons:          []string{"plugin_usage"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected output\nwant: %#v\ngot: %#v", want, got)
		}
	})
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func TestRunSARIFOutput(t *testing.T) {
	t.Run("sarif output for basic fixture", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"./testdata/fixtures/basic/...", "--sarif"}, &stdout)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		var log sarifLog
		if err := json.Unmarshal(stdout.Bytes(), &log); err != nil {
			t.Fatalf("failed to parse SARIF: %v\n%s", err, stdout.Bytes())
		}

		if log.Version != "2.1.0" {
			t.Fatalf("unexpected version: %s", log.Version)
		}
		if len(log.Runs) != 1 {
			t.Fatalf("expected 1 run, got %d", len(log.Runs))
		}

		r := log.Runs[0]
		if r.Tool.Driver.Name != "exportsurf" {
			t.Fatalf("unexpected tool name: %s", r.Tool.Driver.Name)
		}
		if len(r.Tool.Driver.Rules) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(r.Tool.Driver.Rules))
		}
		if len(r.Results) != 4 {
			t.Fatalf("expected 4 results, got %d", len(r.Results))
		}

		for _, res := range r.Results {
			if res.RuleID == "" {
				t.Fatal("empty ruleId")
			}
			if res.Level == "" {
				t.Fatal("empty level")
			}
			if res.Message.Text == "" {
				t.Fatal("empty message text")
			}
			if len(res.Locations) == 0 {
				t.Fatal("no locations")
			}
			loc := res.Locations[0]
			if loc.PhysicalLocation.ArtifactLocation.URI == "" {
				t.Fatal("empty artifact URI")
			}
			if loc.PhysicalLocation.Region.StartLine == 0 {
				t.Fatal("zero start line")
			}
		}
	})

	t.Run("sarif with baseline filters results", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{
			"./testdata/fixtures/basic/...",
			"--sarif", "--baseline", "./testdata/baseline/basic.json",
		}, &stdout)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		var log sarifLog
		if err := json.Unmarshal(stdout.Bytes(), &log); err != nil {
			t.Fatalf("failed to parse SARIF: %v", err)
		}

		if len(log.Runs[0].Results) != 3 {
			t.Fatalf("expected 3 results with baseline, got %d", len(log.Runs[0].Results))
		}
	})

	t.Run("sarif and json are mutually exclusive", func(t *testing.T) {
		var stdout bytes.Buffer
		err := run([]string{"./testdata/fixtures/basic/...", "--sarif", "--json"}, &stdout)
		if err == nil {
			t.Fatal("expected error for --sarif with --json")
		}
	})
}

func chdir(t *testing.T, dir string) string {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("chdir cleanup: %v", err)
		}
	})
	return origDir
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
