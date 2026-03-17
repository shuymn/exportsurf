# Open Questions

- Question: scan の non-JSON デフォルト出力形式
  - Class: `blocking`
  - Resolution: `decision`
  - Status: `resolved` — go vet 風 `defined_in: message` 形式
- Question: `scan --baseline` の semantics
  - Class: `blocking`
  - Resolution: `decision`
  - Status: `resolved` — `diff` 削除、`scan` に `--baseline` 統合（指定時は baseline フィルタ）
- Question: `explain` サブコマンドの要否
  - Class: `blocking`
  - Resolution: `decision`
  - Status: `resolved` — 不要、計画から除外
- Question: Config 構造を nested に変更するか
  - Class: `non-blocking`
  - Resolution: `decision`
  - Status: `resolved` — nested 構造に変更（exclude, include, low_confidence）
- Question: Exit code semantics
  - Class: `blocking`
  - Resolution: `decision`
  - Status: `resolved` — `--fail-on-findings` フラグで制御。デフォルト exit 0、フラグ指定時は候補ありで non-zero
- Question: Build tags / GOOS / GOARCH による参照の見落とし
  - Class: `risk-bearing`
  - Resolution: `decision`
  - Status: `resolved` — 完全解決は困難。README に制約として記載

# Theme Backlog

- [x] Theme: Scan CLI unification — diff 削除・--baseline 統合・テキストデフォルト出力

- [x] Theme: Method scanning — exported method を候補対象に追加

- [x] Theme: Struct field scanning — exported struct field を候補対象に追加
  - Outcome: `include_fields: true` 設定時、exported struct field も候補として報告される。embedded field・serialization tag 付き field は low confidence で報告される
  - Goal: scan が exported struct field を候補に含め、embedded field と serialization tag を検出して confidence に反映
  - Must Not Break: 既存スキャン結果、`include_fields` 未設定時のデフォルト動作
  - Non-goals: method scanning, promoted field の参照パス解析（`go/types` の `Uses` 解決に委譲）
  - Acceptance (EARS):
    - When `include_fields: true` is set in config, the tool shall report exported struct fields with internal-only references as candidates with kind "field"
    - When `include_fields` is not set or false, the tool shall not report fields (default)
    - When a field is reported, the symbol field shall include the parent type (e.g., `pkg.Type.Field`)
    - When a field is an embedded field, the candidate shall have low confidence with reason `"embedded field"`
    - When a field has a serialization struct tag (`json:`, `xml:`, `yaml:`), the candidate shall have low confidence with reason `"has serialization tag"`
    - When a field is on a generic type, the symbol shall use the base type name without type parameters (e.g., `pkg.Container.Value`)
  - Evidence: `run=task check; oracle=test assertions; visibility=independent; controls=[agent,context]; missing=[]; companion=none`
  - Gates: `static`, `integration`
  - Executable doc: integration test — fixture with exported struct fields: regular fields, embedded fields, fields with serialization tags, fields on generic types, some externally referenced; verify confidence and reasons
  - Why not split vertically further?: field の定義収集・参照追跡・embedded/tag 検出は同じ型情報に依存し不可分
  - Escalate if: promoted field の参照追跡で `go/types` の `Uses` 解決が不十分なケースが見つかった場合

- [x] Theme: Confidence scoring configuration — mark_main_low_confidence 設定化 + reflect/plugin/cgo/linkname パターン検出
  - Outcome: confidence 判定がユーザー設定可能になり、reflect/plugin/cgo/linkname パターンが low confidence として検出される
  - Goal: `mark_main_low_confidence` を config で切り替え可能にし、各種パターンを検出して low confidence に分類
  - Must Not Break: 既存の confidence 判定（設定未指定時のデフォルトは現状維持: package main → low）
  - Non-goals: confidence レベルの追加（medium など）、カスタム reason 定義、registry パターン検出（spike 結果次第で follow-up）、間接 reflect（encoding/json 等経由。struct tag 検出は Theme 3 で対応）
  - Acceptance (EARS):
    - When `mark_main_low_confidence: false` is set in config, package main symbols shall not receive "package main" reason
    - When not set, the default behavior shall remain unchanged (package main → low confidence)
    - When a candidate's defining package uses `reflect.TypeOf` or `reflect.ValueOf` on the candidate's type, the candidate shall have reason "reflect usage" and low confidence
    - When a candidate is defined in a package that uses `plugin.Open`, the candidate shall have reason "plugin usage" and low confidence
    - When a function has a `//export` cgo directive, the candidate shall have reason "cgo export" and low confidence
    - When a symbol is referenced via `//go:linkname` directive, the candidate shall have reason "go:linkname" and low confidence
  - Evidence: `run=task check; oracle=test assertions; visibility=independent; controls=[agent,context]; missing=[]; companion=none; notes=reflect/plugin heuristics require spike for false positive evaluation`
  - Gates: `static`, `integration`
  - Executable doc: integration test — fixtures with reflect/plugin/cgo/linkname usage patterns; verify confidence and reasons
  - Why not split vertically further?: 全パターンが同じ confidence 判定パイプラインと reason 生成ロジックに影響する
  - Escalate if: reflect/plugin 検出の spike で false positive 率が許容できない場合

- [ ] Theme: SARIF output format
  - Outcome: `--sarif` フラグで SARIF 形式の出力が可能になる
  - Goal: SARIF v2.1.0 形式の出力を追加。Markdown 変換は外部ツール（sarif-to-md-rs, go-sarif-to-markdown-table 等）に委譲
  - Must Not Break: 既存の JSON / テキスト出力
  - Non-goals: Markdown 出力（SARIF 経由で外部変換可能）、SARIF viewer 統合、GitHub Code Scanning upload 自動化
  - Acceptance (EARS):
    - When `scan --sarif` is run, the tool shall output valid SARIF v2.1.0 JSON to stdout
    - When `scan --sarif --baseline <path>` is run, the tool shall output filtered candidates in SARIF format
    - If `--sarif` is combined with `--json`, the tool shall return an error
  - Evidence: `run=task check; oracle=test assertions + SARIF schema validation; visibility=independent; controls=[agent,context]; missing=[]; companion=none`
  - Gates: `static`, `integration`
  - Executable doc: integration test — verify SARIF output validates against SARIF v2.1.0 JSON schema
  - Why not split vertically further?: SARIF 出力は単一の output formatter
  - Escalate if: SARIF spec の特定要素（rule, result, location mapping）の解釈が曖昧な場合
