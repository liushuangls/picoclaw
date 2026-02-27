# Repository Guidelines

## Language Policy

- User-facing communication: Chinese (questions, explanations, plans, summaries, reviews, etc.).
- Source code: follow project conventions (this repo is primarily Go); in code files, everything except comments must be English; all code comments must be Chinese.
- Otherwise: use the most appropriate language for correctness/clarity (commands, identifiers, file names, protocol fields, error messages, upstream docs, etc.).
- Avoid mixing languages in the same paragraph unless it improves precision (e.g., quoting identifiers).

## Project Structure & Module Organization
- `cmd/picoclaw/` contains the CLI entrypoint and command handlers under `internal/`.
- `pkg/` holds core application packages (`agent`, `tools`, `providers`, `channels`, `config`, etc.).
- `config/` contains example runtime config (`config.example.json`).
- `docs/` contains channel, migration, and design documentation.
- `workspace/` stores built-in agent identity/memory files and bundled skills.
- `assets/` contains images and demo media used in docs.
- Tests live next to code as `*_test.go` files.

## Build, Test, and Development Commands
- `make deps` — download and verify Go modules.
- `make build` — run `go generate ./...` and build the platform binary in `build/`.
- `make build-all` — cross-compile release binaries for major targets.
- `make test` — run all unit tests (`go test ./...`).
- `make check` — full local gate: deps, format, vet, and tests.
- `make lint` / `make fix` — run lint checks or auto-fix lint issues.
- `make run ARGS='status'` — build and run locally with CLI args.

## Coding Style & Naming Conventions
- Language/runtime baseline: Go `1.25.x` (`go.mod`).
- Format via `make fmt` (golangci formatters: `gofmt`, `gofumpt`, `goimports`, `gci`, `golines`).
- Keep lines within 120 chars where practical.
- Follow Go naming: exported `PascalCase`, unexported `camelCase`, lowercase package names.
- Use descriptive file names with standard suffixes like `*_test.go`, `*_linux.go`.

## Testing Guidelines
- Use Go’s `testing` package; `testify` is available for assertions/mocks.
- Prefer table-driven tests for parser/routing/provider logic.
- Add or update tests in the same package as the change.
- Run focused checks during development (example: `go test -run TestSessionKey -v ./pkg/routing`).
- Before opening a PR, run `make check` locally.

## Commit & Pull Request Guidelines
- Follow Conventional Commits style seen in history (e.g., `fix(providers): ...`, `docs: ...`).
- Keep commits scoped to one logical change and reference issues when relevant (`(#123)`).
- Open PRs against `main`; do not push directly to protected branches.
- Complete `.github/pull_request_template.md`, including AI disclosure, related issue, and test environment.
- Include logs/screenshots when behavior changes are user-visible.
