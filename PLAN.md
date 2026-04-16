# Paperclip-Go MVP v1 — Implementation Plan

## Context

`paperclip-go` is a fork of [paperclipai/paperclip](https://github.com/paperclipai/paperclip), a TypeScript/Node.js control plane for autonomous AI agent companies. This plan adds a **Go reimplementation alongside the existing TS code** so the user can:

1. Drop Node.js / pnpm / TS toolchain as their primary dev loop.
2. Sync freely from upstream TS without merge conflicts (no existing TS/node files touched).
3. Let cheap agents (Sonnet/Haiku) iterate on a small, clear Go codebase to reach feature parity incrementally.

The MVP is deliberately scoped to what is needed to **run locally and iterate**: a single Go binary that exposes the core Paperclip control-plane API (companies / agents / issues / comments / heartbeat) against SQLite. Auth, plugins, WebSockets, approvals, budgets, routines, and embedded Postgres are explicitly deferred.

Since the Go port starts from scratch, it uses its own data dir (`~/.paperclip-go/`), port (`3200`), and SQLite DB — so it can coexist with an upstream TS instance on the same machine.

---

## Ground rules

- **Do not modify** `server/`, `ui/`, `packages/`, `cli/`, `tests/`, `scripts/`, `docs/`, `evals/`, `skills/`, `package.json`, `pnpm-*.yaml`, `tsconfig*.json`, `vitest.config.ts`, `Dockerfile`. These sync from upstream.
- **Reuse** `/skills/` (read-only at runtime) and — if present — `/ui/dist/` (optional, pre-built SPA).
- **All new Go code** lives under `/cmd` (binaries) and `/internal` (packages). Single `go.mod` at repo root. No Go workspaces.
- **Stdlib-first**, with a small, justified dep list.
- **Agent-friendly layout**: one responsibility per package, small surfaces, godoc on every `package` line, SQL handwritten and visible.

---

## Tech choices

| Concern       | Choice                                   | Why                                                              | Status   |
|---------------|------------------------------------------|------------------------------------------------------------------|----------|
| CLI           | `github.com/spf13/cobra`                 | Mirrors TS commander; standard in Go tooling.                    | ✅ added  |
| HTTP router   | `github.com/go-chi/chi/v5`               | Clean per-resource subrouters; easier middleware than `ServeMux`.| ✅ added  |
| SQLite driver | `modernc.org/sqlite`                     | Pure Go, no CGO — fast to build in sandboxes/CI.                 | pending  |
| UUIDs         | `github.com/google/uuid`                 | Standard.                                                        | pending  |
| Config        | `gopkg.in/yaml.v3`                       | One tiny dep; human-editable config.                             | pending  |
| Logging       | stdlib `log/slog`                        | Structured, zero extra dep.                                      | pending  |
| Migrations    | stdlib `embed` + custom runner           | Avoids golang-migrate dep for MVP; SQL stays visible.            | pending  |
| Testing       | stdlib `testing` + `net/http/httptest`   | Simple Go-style tests; no vitest/playwright port.                | pending  |

Explicitly **not** pulling in: viper, zap/logrus, gorm/ent/sqlc, testify.

---

## Directory layout

Files marked ✅ already exist; the rest are planned.

```
/ (repo root)
  go.mod                               # module: github.com/ubunatic/paperclip-go  ✅
  go.sum                                                                             ✅
  Makefile                                                                           ✅
  AGENTS.md                            # Go-side agent playbook                     ✅
  .gitignore                                                                         ✅

  cmd/
    paperclip-go/
      main.go                          # func main() { cli.Execute() }              ✅

  internal/
    README.md                          # one-line package responsibilities (for Haiku skims)

    app/
      app.go                           # App struct wires Config, Store, Logger, Clock, Router
      clock.go                         # Clock interface, stubable in tests
    config/
      config.go                        # Load(path) (*Config, error); defaults; env/flag overrides
      config_test.go
    logging/
      logging.go                       # slog setup (text|json), request-id helpers
    ids/
      ids.go                           # NewUUID() string

    store/
      store.go                         # *Store wraps *sql.DB; Open(dsn)
      tx.go                            # WithTx(ctx, fn)
      migrations.go                    # embed.FS loader + runner + schema_migrations table
      migrations/
        0001_init.sql                  # all MVP tables in one file

    domain/                            # pure data types, no deps on db/http
      company.go
      agent.go
      issue.go
      comment.go
      heartbeat.go
      activity.go

    companies/   {service.go, service_test.go}
    agents/      {service.go, service_test.go}
    issues/      {service.go, service_test.go}      # checkout/release atomic transitions live here
    comments/    {service.go}
    heartbeat/
      runner.go                        # orchestrates a run
      adapter.go                       # Adapter interface + StubAdapter + registry
      runner_test.go
    activity/
      log.go                           # Record(ctx, actor, action, entity, meta)
    skills/
      loader.go                        # scans /skills/*/SKILL.md + YAML front matter

    api/
      router.go                        # chi router, mounts /api                    ✅
      middleware.go                    # request id, slog access log, recoverer, json content-type
      errors.go                        # typed errors -> HTTP status
      render.go                        # writeJSON, readJSON, decodeAndValidate
      health/handler.go                                                              ✅
      companies/handler.go
      agents/handler.go
      issues/handler.go
      comments/handler.go
      heartbeat/handler.go
      skills/handler.go
      activity/handler.go
      stubs/handler.go                 # approvals/costs/goals/projects/routines/plugins → {"items":[]}

    ui/
      assets.go                        # serve /ui/dist if present; else embed.FS landing page + SPA fallback

    cli/
      root.go                          # cobra root + Execute()                     ✅
      serve.go                         # paperclip-go serve                         ✅
      initcmd.go                       # paperclip-go init
      doctor.go                        # paperclip-go doctor
      company.go                       # create/list
      agent.go                         # create/list
      issue.go                         # create/list/get
      heartbeat.go                     # run
      client.go                        # thin HTTP client (for --remote mode)

    testutil/
      server.go                        # SpawnTestServer(t) — full router on temp sqlite
      factories.go                     # MakeCompany, MakeAgent, MakeIssue
```

---

## Config

Path: `~/.paperclip-go/config.yaml` (override with `--config` or `PAPERCLIP_GO_CONFIG`).

```yaml
data_dir: ~/.paperclip-go              # sqlite file at <data_dir>/paperclip.db
listen_addr: 127.0.0.1:3200            # TS uses 3100; we pick 3200 to coexist
log_level: info                        # debug|info|warn|error
log_format: text                       # text|json
skills_dir: ./skills                   # relative to cwd or absolute
ui_dir: ./ui/dist                      # optional; missing → API-only landing page
deployment_mode: local_trusted         # fixed for MVP; field reserved for later
```

`paperclip-go init` writes this file if absent and creates the data dir.

---

## Database schema (MVP)

Single file `internal/store/migrations/0001_init.sql`, applied in a transaction at startup by a small runner that tracks applied names in `schema_migrations`. Pragmas: `journal_mode=WAL`, `foreign_keys=ON`. UUIDs are stored as `TEXT`; timestamps as ISO-8601 `TEXT` (simple + inspectable).

Tables (MVP only):

- `companies(id, name, shortname UNIQUE, description, created_at, updated_at)`
- `agents(id, company_id, shortname, display_name, role, reports_to, adapter DEFAULT 'stub', created_at, updated_at; UNIQUE(company_id, shortname))`
- `issues(id, company_id, title, body, status [open|in_progress|blocked|done|cancelled], assignee_id, checked_out_by, checked_out_at, parent_issue_id, created_at, updated_at)`
- `comments(id, issue_id, author_agent_id, author_kind [agent|system|operator], body, created_at)`
- `heartbeat_runs(id, agent_id, issue_id, status [running|success|error], started_at, finished_at, summary, error)`
- `activity_log(id, company_id, actor_kind, actor_id, action, entity_kind, entity_id, meta_json, created_at)`
- `schema_migrations(id, name, applied_at)`

**Atomic issue checkout** (core invariant from `doc/SPEC-implementation.md` §5):
`UPDATE issues SET checked_out_by=?, checked_out_at=?, status='in_progress' WHERE id=? AND checked_out_by IS NULL` — then verify `RowsAffected == 1`, else return 409.

Schema drift from the TS Postgres side is accepted. We only need the minimal set to drive the control-plane loop.

---

## HTTP API (MVP)

All under `/api`, JSON, chi subrouters per resource. Middleware chain: `RequestID → AccessLog(slog) → Recoverer → ContentTypeJSON`. Errors as `{"error": {"code","message"}}` with 400/404/409/422/500.

| Method & path                                 | Notes                                            |
|-----------------------------------------------|--------------------------------------------------|
| `GET /api/health`                             | `{"status":"ok","version":"..."}`                |
| `GET/POST /api/companies`, `GET /…/{id}`      |                                                  |
| `GET/POST /api/agents` (`?companyId=`)        |                                                  |
| `GET /api/agents/{id}`, `GET /api/agents/me`  | `me` reads `X-Agent-Id` header for MVP           |
| `GET/POST /api/issues` (`?companyId=`, filters: `status`, `assigneeId`) |                           |
| `GET /api/issues/{id}`, `PATCH /api/issues/{id}` |                                               |
| `POST /api/issues/{id}/checkout`              | body: `{agentId}`; 409 if already held           |
| `POST /api/issues/{id}/release`               | body: `{agentId}`                                |
| `GET/POST /api/issues/{id}/comments`          |                                                  |
| `POST /api/heartbeat/runs`                    | body: `{agentId}`; kicks a stub run              |
| `GET /api/heartbeat/runs` (`?agentId=`)       |                                                  |
| `GET /api/skills`                             | in-memory list from `/skills/*/SKILL.md`         |
| `GET /api/activity` (`?companyId=`)           |                                                  |
| `GET /api/{approvals,costs,goals,projects,routines,plugins}` | stubs returning `{"items":[]}`    |

Non-`/api/*` paths go to `internal/ui/assets.go` which serves `ui/dist` if present with SPA fallback to `index.html`, otherwise an embedded "API-only" landing page.

---

## Heartbeat (stub adapter)

Core interface in `internal/heartbeat/adapter.go`:

```go
type RunContext struct {
    RunID, AgentID, CompanyID string
    Issue *domain.Issue
}
type RunResult struct {
    Summary string
    Comment string // optional; posted on the issue as a system comment
    Err     error
}
type Adapter interface {
    Name() string
    Execute(ctx context.Context, rc RunContext) RunResult
}
```

`StubAdapter` writes a canned summary + comment ("Stub adapter: acknowledged issue"). `Runner.Run(agentID)`:

1. Pick highest-priority open issue assigned to agent (may be nil).
2. Insert `heartbeat_runs` row as `running`.
3. Invoke adapter.
4. On success: post the canned comment, write an `activity_log` entry, mark run `success`. On error: mark run `error` + record error.

Registry is a simple `map[string]Adapter`; MVP registers only `"stub"`. This keeps the interface stable when real adapters (Claude, Cursor) arrive post-MVP.

---

## CLI surface

Cobra commands. Everything except `serve` runs **in-process** against the DB by default (`local_trusted`); pass `--remote` to hit a running server via `internal/cli/client.go`.

```
paperclip-go serve
paperclip-go init
paperclip-go doctor
paperclip-go company create --name "Acme" --shortname acme
paperclip-go company list
paperclip-go agent create --company acme --shortname ceo --role CEO --display-name "CEO"
paperclip-go agent list --company acme
paperclip-go issue create --company acme --title "..." [--assignee ceo]
paperclip-go issue list --company acme [--status open]
paperclip-go issue get <id>
paperclip-go heartbeat run --agent <id>
```

---

## Skills loader

`internal/skills/loader.go`:

1. At startup walk `cfg.SkillsDir` (default `./skills`).
2. For each `*/SKILL.md`, parse the `---` YAML front matter (`name`, `description`) plus the body.
3. Keep in memory as `[]Skill{Name, Description, Path, Body}`.
4. `GET /api/skills` returns the slice.

No DB table for MVP. Re-scan on SIGHUP is a nice-to-have; deferred.

---

## Testing

Goal: **Go-idiomatic, not a vitest/playwright port.**

- **Unit tests** per service package: `companies/service_test.go` opens a temp-file SQLite via `testutil.NewStore(t)` (auto-migrates), exercises each public method.
- **One end-to-end test** in `internal/api/api_e2e_test.go` using `testutil.SpawnTestServer(t)` (full router on temp SQLite) covering the golden path:
  1. `POST /api/companies` → 201 + id.
  2. `POST /api/agents` → ok.
  3. `POST /api/issues` → ok.
  4. `POST /api/issues/{id}/checkout` → 200; second call → 409.
  5. `POST /api/heartbeat/runs` → run row + canned comment visible via `GET /api/issues/{id}/comments`.
  6. `GET /api/skills` → returns ≥1 skill loaded from the repo's real `/skills/` dir.
- **One CLI smoke test** shelling `go run ./cmd/paperclip-go company list` against a fixture data dir.

`make test` runs `go test ./internal/...`. No build tag separation needed at MVP scale.

---

## Build & run

`Makefile`:

```
.PHONY: build run init doctor test tidy
build:   go build -o bin/paperclip-go ./cmd/paperclip-go
run:     go run ./cmd/paperclip-go serve
init:    go run ./cmd/paperclip-go init
doctor:  go run ./cmd/paperclip-go doctor
test:    go test ./internal/...
tidy:    go mod tidy
```

Binary: `./bin/paperclip-go`. No release pipeline in MVP.

`.gitignore` append (clearly delimited block so merges from upstream are clean):

```
# --- paperclip-go (Go port) ---
/bin/
*.test
*.out
coverage.out
.paperclip-go-data/
```

---

## Agent iteration playbook

Two new docs help Sonnet/Haiku stay cheap:

- **`/AGENTS.md`** (top-level, ✅ exists): declares the TS files are read-only; lists the Go layout; maps TS → Go ("`server/src/routes/issues.ts` → `internal/api/issues/handler.go` + `internal/issues/service.go`"); describes the porting workflow (read TS route → read TS service → add failing Go test → implement → wire handler); lists MVP non-goals; has a "When stuck" section (`make doctor`, `go test ./internal/<pkg>`).
- **`/internal/README.md`**: one-liners per package (mirrors the tree above). Paired with godoc on every `package x` declaration so `go doc ./internal/...` is the ground truth.

Together this means porting a new upstream TS feature is a scripted loop: find TS file → look up Go target in the map → add test → implement → run `make test`.

---

## Phased implementation (validatable checkpoints)

Each phase ends with `make test` green and a documented `curl` recipe in `AGENTS.md`.

1. ✅ **DONE: Skeleton + serve + health** — `go.mod`, cobra root, `serve` on `:3200`, `GET /api/health` → `{"status":"ok"}`. `make run` works.
2. ✅ **DONE: Config + `init` + `doctor`** — YAML loader, writes default config + data dir, `doctor` reports status.
   - Created `internal/config/config.go` with Config struct, Load/Write/DBPath methods
   - Implemented `paperclip-go init` command that writes `~/.paperclip-go/config.yaml` and creates data directory
   - Implemented `paperclip-go doctor` command that validates installation and reports configuration
   - Updated `serve` command to load config and use ListenAddr from configuration
   - Added comprehensive unit tests in `internal/config/config_test.go`
   - All tests passing: `go test ./...` ✓
3. ✅ **DONE: Store + migrations + companies** — SQLite, embedded `0001_init.sql`, companies CRUD (service + handler + CLI). First unit test + first e2e test.
   - Created SQLite store with migrations
   - Implemented companies service with full CRUD operations
   - Added HTTP handlers for company endpoints
   - Created CLI commands for company management
   - All tests passing ✓
4. ✅ **DONE: Agents + activity log** — agents CRUD, `/api/agents/me`, activity table + `GET /api/activity`.
   - Created domain models for Agent and Activity
   - Implemented agents service with CRUD + GetByShortname + ListByCompany
   - Implemented activity log service with Record + List operations
   - Added HTTP handlers for agents (GET/POST, /me endpoint) and activity
   - Created CLI commands: agent create/list with company filtering
   - Added companies.GetByShortname() helper
   - Full unit + E2E test coverage
   - Code review cycle completed with all fixes applied
   - PR #13 merged ✓
5. **Issues + comments + checkout** — full issue lifecycle with atomic checkout/release; nested comments.
6. **Skills loader** — walk `/skills/`, expose `/api/skills`.
7. **Heartbeat stub** — Adapter interface, `StubAdapter`, `POST /api/heartbeat/runs`, CLI `heartbeat run`; e2e covers the full loop.
8. **UI serving + stub endpoints** — serve `/ui/dist` if present, SPA fallback; stub endpoints for approvals/costs/goals/projects/routines/plugins so the UI (if used) does not 404.

---

## Risks & explicit non-goals

Non-goals for MVP (documented in `AGENTS.md`):

- Auth / BetterAuth / board claim flow (always `local_trusted`).
- WebSocket live events (poll for now).
- Plugin host, external adapter processes.
- Approvals, budgets, costs, routines/cron, company portability.
- Embedded Postgres (SQLite only).
- Data sharing with the TS instance (separate data dir, separate DB).
- Full Drizzle-schema parity.

Risks & mitigations:

- **UI expects endpoints Go lacks** → stub routes returning `{"items":[]}`.
- **Agents accidentally editing TS** → enforced by layout (`/cmd`, `/internal` only) + prominent `AGENTS.md`.
- **Schema drift from TS** → accepted; MVP is isolated, not a port of Drizzle.
- **modernc.org/sqlite perf** → fine for single-user MVP; swap to PG later if needed behind the `store` interface.
- **Heartbeat stub hardens incorrectly** → keep `Adapter` interface small; first real adapter (`claude_local`) slots in as a second implementation.

---

## Critical files

Already created (Phase 1 ✅):

- `go.mod` / `go.sum`
- `Makefile`
- `AGENTS.md`
- `cmd/paperclip-go/main.go`
- `internal/api/router.go`
- `internal/api/health/handler.go`
- `internal/cli/root.go`
- `internal/cli/serve.go`

Still to create (Phases 2–8):

- `internal/store/migrations/0001_init.sql`
- `internal/heartbeat/runner.go`
- `internal/testutil/server.go`
- (all other packages listed in the layout above)

---

## Verification

After the user approves and implementation runs through Phase 8:

```sh
# 1. Build
make build
ls bin/paperclip-go

# 2. Init config + data dir
./bin/paperclip-go init
cat ~/.paperclip-go/config.yaml

# 3. Doctor
./bin/paperclip-go doctor

# 4. Run tests
make test

# 5. Start server in one terminal
./bin/paperclip-go serve
# expect: slog line "server listening on 127.0.0.1:3200"

# 6. In another terminal, drive the API end-to-end
curl -s localhost:3200/api/health | jq
CID=$(curl -s -XPOST localhost:3200/api/companies -d '{"name":"Acme","shortname":"acme"}' -H 'content-type: application/json' | jq -r .id)
AID=$(curl -s -XPOST localhost:3200/api/agents -d "{\"companyId\":\"$CID\",\"shortname\":\"ceo\",\"displayName\":\"CEO\",\"role\":\"CEO\"}" -H 'content-type: application/json' | jq -r .id)
IID=$(curl -s -XPOST localhost:3200/api/issues -d "{\"companyId\":\"$CID\",\"title\":\"first task\",\"assigneeId\":\"$AID\"}" -H 'content-type: application/json' | jq -r .id)
curl -s -XPOST localhost:3200/api/issues/$IID/checkout -d "{\"agentId\":\"$AID\"}" -H 'content-type: application/json'
curl -s -XPOST localhost:3200/api/heartbeat/runs -d "{\"agentId\":\"$AID\"}" -H 'content-type: application/json'
curl -s "localhost:3200/api/issues/$IID/comments" | jq   # should show the stub comment
curl -s localhost:3200/api/skills | jq '.items | length'  # >= 1

# 7. Exercise the CLI (in-process, no running server needed)
./bin/paperclip-go company list
./bin/paperclip-go issue list --company acme
./bin/paperclip-go heartbeat run --agent "$AID"
```

The MVP is done when every step above succeeds and `make test` is green. From there, Sonnet/Haiku agents port upstream TS features one at a time using the TS→Go map in `AGENTS.md`.
