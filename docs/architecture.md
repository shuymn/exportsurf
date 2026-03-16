## Goal
Exported identifier を「unused error」ではなく「unexport 候補レポート」として抽出し、証拠付きで出力する。

## Constraints
- standalone CLI を正本にする
- `go/packages` + `go/types` の公式 API だけで始める
- MVP では exported package-level symbol のみ扱う
- fail-lint semantics を持ち込まない
- suppress は baseline と config で扱い、inline `nolint` に寄せない

## Core Boundaries
- `internal/scan` -- package load, symbol collection, reference aggregation
- `pkg/report` -- candidate report schema と JSON/Markdown/SARIF emitter
- `cmd/exportsurf` -- CLI, config, exit semantics

## Key Tech Decisions
- symbol ごとに internal/external reference を集計する
- `scan` の正本は diagnostic ではなく candidate report とする
- candidate report は `symbol`, `kind`, `defined_in`, `internal_ref_count`, `external_ref_pkg_count`, `external_ref_examples`, `confidence`, `reasons` を中心に持つ
- external test 参照は flag/config で切り替える
- `package main` / `cmd/**` / generated / test runner entrypoint は除外または low confidence に落とし、report に理由を残せる形にする
- baseline は accepted candidates の保存形式として使う
- repo 固有の除外対象と rule は config file で外出しする

## Open Questions
- `treat_tests_as_external` の default を config と CLI flag のどちらが優先するか
- `go/packages` の load semantics が workspace 境界で不安定なら spike に切る
- `package main` や plugin-like surface を exclude と low-confidence のどちらで扱うか
- reflect / plugin / registry heuristics の拡張は non-blocking

## Revisit Trigger
- `go vet` / `golangci-lint` 連携が必須になったとき
- methods / fields を MVP に含める要求が出たとき
- explain command や JSON 以外の public report contract が先に必要になったとき
