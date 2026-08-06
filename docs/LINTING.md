# Go Linting & Quality Checklist

Use this checklist to keep Go codebases healthy. Content is drawn from Effective Go, Go Code Review Comments, and common Go tooling docs.

## Baseline Tools

- `gofmt`: Mandatory formatting; run before committing.
- `goimports`: Formats and prunes/organizes imports.
- `go vet`: Static analysis for suspicious constructs (format string mistakes, unreachable code, copylocks, etc.).
- `golangci-lint`: Aggregates formatters and linters configured in `.golangci.yml`. Run via `make lint`.
- `go test -race`: Detects data races; run on packages with concurrency.
- Comment policy test: Parses repository Go files and rejects prose comments;
  only Go directives, line directives, and the standard generated-code marker
  are accepted.

### Formatters (golangci-lint)

Configured in `.golangci.yml` under `formatters.enable`:

- `gofmt`
- `goimports`

These run as part of `make lint` / `golangci-lint run`. `make fmt` and `make fmt-check` also apply or verify them directly.

### Linters (golangci-lint)

Configured in `.golangci.yml` under `linters.enable`:

- `depguard` — Clean Architecture import boundaries for core packages
- `errcheck` — unchecked returned errors
- `errorlint` — error wrapping and comparison idioms
- `govet` — suspicious constructs
- `ineffassign` — ineffectual assignments
- `revive` — style and convention checks
- `staticcheck` — advanced static analysis
- `unused` — unused code
- `wsl_v5` — blank lines between logical statement blocks (see [STYLE_GUIDE.md](STYLE_GUIDE.md#vertical-spacing--logical-blocks))

## Common Lint Rules

- Unused: remove unused variables/imports; do not keep dead code.
- Errors: check returned errors; wrap with `%w`; avoid swallowing errors. Do not use panics for normal errors.
- Naming: avoid stutter; keep exported names meaningful; follow standard initialisms (`ID`, `HTTP`).
- Imports: avoid blank imports unless for side effects (document why); avoid dot-imports in production code.
- Vertical spacing: separate logical steps with blank lines; keep short error-check chains tight. Enforced by `wsl_v5`.
- Concurrency: guard shared maps; close channels only from senders; avoid goroutine leaks (tie to context).
- Defer: close resources immediately after open (`defer f.Close()`); beware of defers in loops for large iterations.
- Range pitfalls: avoid taking the address of loop variables; use index-based capture.
- Struct tags: keep JSON/DB tags consistent; avoid unused tags; ensure backticks are balanced.
- Composite literals: use field names for structs when many fields or exported fields are involved.
- Context: place `context.Context` as the first parameter when meaningful; do not store contexts inside structs.
- Comments: keep Go source and tests comment-free except for policy-approved
  compiler, line, and generated-code markers.

## Suggested CI Gates

- `make fmt-check` (or `make fmt` locally)
- `go vet ./...`
- `golangci-lint run ./...` (formatters + linters from `.golangci.yml`, including `wsl_v5`)
- `go test ./...` (add `-race` for critical paths)

This repository provides these gates through `make check`, `.golangci.yml`, and
`.github/workflows/quality.yml`. Run `make vuln` separately for dependency
vulnerability scanning.

## References

- Paraphrased from: Effective Go (go.dev/doc/effective_go), Go Code Review Comments (go.dev/wiki/CodeReviewComments), and common Go tooling practices.
