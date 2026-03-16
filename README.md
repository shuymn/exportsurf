# exportsurf

`exportsurf` is a Go CLI that scans a repository and reports exported package-level identifiers with internal and external reference evidence for API review.

It is designed as a report tool for public API review, not as a linter or `go vet`-style diagnostic. The output is intended to help you review exported symbols with the evidence needed to decide whether they may be safe to unexport.

## Current Scope

The current implementation scans exported package-level:

- `func`
- `type`
- `var`
- `const`

The scanner currently:

- reports candidates as JSON
- counts internal references within the defining package
- counts external package references
- includes `external_ref_examples` and `reasons` for evidence-rich candidate review
- downgrades confidence for `package main`
- downgrades confidence for packages under `cmd/**`
- downgrades confidence for generated files
- downgrades confidence for `go test` entrypoints such as `TestXxx`, `BenchmarkXxx`, `FuzzXxx`, and `ExampleXxx`

## Usage

Build the binary:

```bash
task build
```

Run the scanner against the current module:

```bash
./bin/exportsurf scan ./... --json
```

Treat external `_test.go` references as external uses:

```bash
./bin/exportsurf scan ./... --json --treat-tests-as-external
```

You can also run it without building first:

```bash
go run . scan ./... --json
```

## Output

`scan --json` emits an array of candidate-report objects.

Example:

```json
[
  {
    "symbol": "github.com/shuymn/exportsurf/testdata/fixtures/basic/lib.UsedExternally",
    "kind": "type",
    "defined_in": "testdata/fixtures/basic/lib/lib.go:5",
    "internal_ref_count": 0,
    "external_ref_pkg_count": 1,
    "external_ref_examples": [
      "testdata/fixtures/basic/app/app.go:5"
    ],
    "confidence": "high",
    "reasons": []
  }
]
```

Field meanings:

- `symbol`: fully qualified symbol name
- `kind`: symbol kind
- `defined_in`: source file and line of the definition
- `internal_ref_count`: references found inside the defining package
- `external_ref_pkg_count`: number of external packages that reference the symbol
- `external_ref_examples`: example source locations for detected external references
- `confidence`: current confidence label for the candidate
- `reasons`: annotations that explain why confidence was downgraded or why the candidate needs extra review

## Development

Use `task` as the primary entrypoint for local development.

Common commands:

```bash
task build
task test
task lint
task fmt
task check
```

`task check` runs lint, compile checks, tests, and module verification.

If you use Git hooks locally:

```bash
lefthook install
```

## Repository Layout

- `main.go`: CLI entrypoint
- `internal/scan`: package loading and reference aggregation
- `pkg/report`: report serialization
- `testdata/fixtures`: contract fixtures for scanner behavior
- `docs/adr`: design decisions
