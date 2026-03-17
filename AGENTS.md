<!-- Do not restructure or delete sections. Update inline when behavior changes. -->
<!-- Maintenance: Update when tasks, hooks, or project scope changes. -->
<!-- Audience: All docs under docs/ and this file are written for coding agents (LLMs), not humans. Use direct instructions, not tutorials or explanations of concepts the agent already knows. Apply this rule when creating or updating any documentation. -->

## Build, Test, and Development Commands

- Use `task` as the default interface for local tooling and verification so project-local caches under `.cache/` are used.
- `task test` — runs with race detection, shuffle, and 10x count
- `task check` — CI-equivalent local verification (`lint` + `build:check` + `test` + `tidy`); run it before push or when final validation is needed
- Never edit `go.mod` or `go.sum` manually; use `go get`, `go mod tidy`, etc.
- Use `go test -run TestName ./path/to/pkg` only for focused runs; if you bypass `task`, preserve equivalent cache settings

## Git Conventions

- When asked to commit without a specific format, follow Conventional Commits: `<type>(<scope>): <imperative summary>`
- Never use `--no-verify` when committing or pushing; fix the underlying hook failure instead

## Documentation Scope

- Keep this file limited to always-on repository rules.
- Treat files under `docs/` as opt-in reference material; do not read them by default.
- Read `docs/coding.md`, `docs/testing.md`, or `docs/tooling.md` only when the task needs repository-specific coding, testing, or tooling rules.
- Read `docs/review.md` only for code review tasks or when broader review conventions are explicitly requested.
- Read `docs/adr/` only when historical rationale matters.
