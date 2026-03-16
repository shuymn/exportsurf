## Context

`exportsurf` は exported symbol を対象にした公開 API 面の棚卸しツールであり、lint や vet のように fail させる診断が主目的ではない。false positive を避けるには、`unused` の断定よりも「unexport してよさそうな候補」を証拠付きで出す方が用途に合う。初期段階では package-level の exported symbol を対象にした最小の CLI contract を先に閉じる必要がある。

## Decision

- `exportsurf` は `go/packages` + `go/types` ベースの standalone CLI として実装する
- `go vet -vettool` や `unitchecker` は正本にしない
- 出力は diagnostic ではなく candidate report とし、JSON contract は最終的に `symbol`, `kind`, `defined_in`, `internal_ref_count`, `external_ref_pkg_count`, `external_ref_examples`, `confidence`, `reasons` を中心に拡張する
- MVP では exported package-level の `func`, `type`, `var`, `const` のみを対象にする
- methods, fields, explain, Markdown/SARIF, `multichecker` adapter は後続テーマに分離する
- suppress は inline annotation ではなく baseline と config 方式で扱う前提にする
- `package main`, `cmd/**`, generated, test runner entrypoint のような low-confidence surface は diagnostic error にせず、除外または reason 付き候補として扱う

## Rejected Alternatives

- `unitchecker` / `go vet -vettool` を使う: compilation unit 単位の診断ドライバに寄るため、候補抽出ツールの UX とずれる
- `multichecker` を正本にする: standalone 実行は可能だが、diagnostic object 中心の設計に引っ張られ、候補抽出・スコアリング・diff を中心にした contract を置きにくい

## Consequence

- scan engine と CLI を薄く分離しやすくなる
- candidate report を正本にできるため、confidence / reasons / example evidence / baseline diff を自然に拡張できる
- linter 互換の fail semantics は最初から持たない

## Revisit trigger

- `go vet` / `golangci-lint` 連携が必須要件になったとき
- methods や fields を MVP に含める必要が出たとき
- JSON 以外の report contract を先に固定する必要が出たとき
