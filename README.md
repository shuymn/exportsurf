# exportsurf

`exportsurf` scans a Go module and reports exported symbols that have no external references — candidates that may be safe to unexport.

It is a report tool for public API review, not a linter. The output provides reference counts, confidence levels, and reasons to help you make informed decisions.

## Install

```bash
go install github.com/shuymn/exportsurf@latest
```

## Usage

```bash
exportsurf scan ./...                # text output (default)
exportsurf scan ./... --json         # JSON output
exportsurf scan ./... --sarif        # SARIF v2.1.0 output
exportsurf scan ./... --baseline ./baseline.json  # filter accepted symbols
exportsurf scan ./... --fail-on-findings          # exit non-zero on candidates (CI)
```

Flags can be combined. `--sarif` and `--json` are mutually exclusive.

## Config

Config is auto-discovered from the working directory in this order: `.exportsurf.yaml`, `.exportsurf.yml`, `exportsurf.yaml`, `exportsurf.yml`. Use `--config <path>` to specify an explicit path (overrides auto-discovery).

```yaml
exclude:
  packages:
    - github.com/your/module/cmd/tool
  symbols:
    - github.com/your/module/pkg/api.LegacyExport

rules:
  include_funcs: true
  include_types: true
  include_vars: true
  include_consts: true
  include_methods: true
  include_fields: true
  treat_tests_as_external: false
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

- `exclude` — exact-match filters for packages and symbols.
- `rules.include_*` — which symbol kinds to scan. All default to `true`.
- `rules.treat_tests_as_external` — count external `_test.go` references as external uses. The CLI flag `--treat-tests-as-external` is an additive override.
- `rules.mark_low_confidence.*` — which patterns trigger low confidence. All default to `true`.

## Output

Default output is go vet-style text:

```
lib/lib.go:3: Candidate (type)
lib/lib.go:7: ExportedConst (const)
```

`--json` emits an array of candidate objects:

```json
[
  {
    "symbol": "github.com/your/module/lib.Candidate",
    "kind": "type",
    "defined_in": "lib/lib.go:3",
    "internal_ref_count": 4,
    "confidence": "high",
    "reasons": []
  }
]
```

| Field | Description |
|-------|-------------|
| `symbol` | Fully qualified symbol name |
| `kind` | `func`, `type`, `var`, `const`, `method`, `field` |
| `defined_in` | Source file and line |
| `internal_ref_count` | References within the defining package |
| `confidence` | `high` or `low` |
| `reasons` | Why confidence was downgraded (e.g. `package_main`, `reflect_usage`) |

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
