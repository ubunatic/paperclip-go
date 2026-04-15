# AGENTS-GO — Go-side agent playbook

## Layout

| Path | Purpose |
|---|---|
| `cmd/paperclip-go/` | Binary entry point — only calls `cli.Execute()` |
| `internal/cli/` | Cobra root command + sub-commands |
| `internal/` | All other packages (one responsibility each) |
| `go.mod` / `go.sum` | Module root — run `go mod tidy` after adding deps |
| `Makefile` | `build`, `test`, `lint`, `clean` |

## Rules

- **Do not touch** `server/`, `ui/`, `packages/`, `cli/`, `tests/`, `scripts/`,
  `docs/`, `evals/`, `skills/`, `package.json`, `pnpm-*.yaml`, `tsconfig*.json`,
  `vitest.config.ts`, or `Dockerfile`. These sync from upstream TS.
- All Go code lives under `cmd/` and `internal/`.
- Stdlib-first. Current allowed external deps: `cobra`, `chi`, `modernc.org/sqlite`,
  `google/uuid`, `gopkg.in/yaml.v3`. Add others only with justification.
- Run `make build` and `make test` before committing.
- Module path: `github.com/ubunatic/paperclip-go`

## Quick start

```sh
make build          # produces bin/paperclip-go
./bin/paperclip-go  # smoke-test
make test           # run all Go tests
make lint           # go vet
```
