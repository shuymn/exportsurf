# exportsurf

`exportsurf` scans a Go module and reports exported symbols with no external references. Each candidate includes reference counts and confidence annotations for API surface review.

## Install

```bash
go install github.com/shuymn/exportsurf@latest
```

## Usage

```bash
exportsurf ./...                # text output (default)
exportsurf ./... --json         # JSON output
exportsurf ./... --sarif        # SARIF v2.1.0 output
exportsurf ./... --baseline baseline.json    # filter accepted symbols
exportsurf ./... --fail-on-findings          # exit non-zero on candidates (CI)
```

`--sarif` and `--json` are mutually exclusive.

## Baseline

`--baseline` filters out known unused exports. The `--json` output can be used directly as a baseline file.

```bash
exportsurf ./... --json > baseline.json
exportsurf ./... --baseline baseline.json   # exclude symbols listed in baseline
```

## Config

Config is auto-discovered from the working directory in this order: `.exportsurf.yaml`, `.exportsurf.yml`, `exportsurf.yaml`, `exportsurf.yml`. Use `--config <path>` to specify an explicit path (overrides auto-discovery).

```yaml
# Exact-match filters for packages and symbols.
exclude:
  packages:
    - github.com/your/module/cmd/tool
  symbols:
    - github.com/your/module/pkg.FuncName
    - github.com/your/module/pkg.Type.Method

rules:
  # Which symbol kinds to scan. All default to true.
  include_funcs: true
  include_types: true
  include_vars: true
  include_consts: true
  include_methods: true
  include_fields: true
  # Count external _test.go references as external uses. CLI flag --treat-tests-as-external is an additive override.
  treat_tests_as_external: false
  # Which patterns trigger low confidence. All default to true.
  # Set to false to keep matching candidates as high confidence.
  mark_low_confidence:
    package_main: true
    package_under_cmd: true
    generated_file: true
    reflect_usage: true
    plugin_usage: true
    cgo_export: true
    linkname: true
    interface_satisfaction: true
    embedded_field: true
    serialization_tag: true
```

## Output

Default output is go vet-style text:

```
lib/lib.go:3: Candidate (type)
lib/lib.go:7: ExportedConst (const)
```

`--json` emits an array of candidate objects:

```json5
[
  {
    // symbol: fully qualified symbol name
    "symbol": "github.com/your/module/lib.Candidate",
    // kind: func, type, var, const, method, field
    "kind": "type",
    // src: source file and line
    "src": "lib/lib.go:3",
    // internal_ref_count: references within the defining package
    "internal_ref_count": 4,
    // confidence: high or low
    "confidence": "high",
    // reasons: why confidence was downgraded (e.g. package_main, reflect_usage)
    "reasons": []
  }
]
```

`--sarif` emits SARIF v2.1.0 JSON. High-confidence candidates map to `level: "warning"`, low-confidence to `level: "note"`.

## Known Limitations

- Build tags, `GOOS`, and `GOARCH`-dependent references may be missed. The scanner loads packages with default build constraints.

## Development

Use [Task](https://taskfile.dev) as the development interface:

```bash
task check   # lint + build + test (primary gate)
task test    # tests with race detection
task lint    # golangci-lint
task fmt     # format
task build   # build binary
```

Git hooks: `lefthook install`
