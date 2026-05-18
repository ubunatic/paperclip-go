# Paperclip-Go — Feature Parity Plan

> Previous MVP plan archived in `PLAN.archive.md`.  
> All 9 MVP phases + A1–E2 are ✅ DONE. This plan covers what remains.

---

## Scope & Audience

**Target:** A single developer running Paperclip locally or in a VM.  
**Assumption:** Trusted single-user environment — no authentication required, no multi-tenancy enforcement.  
**Auth & multi-user:** Explicitly out of scope unless community interest grows beyond solo use.

This means:
- No auth middleware or RBAC in the near-term phases
- Cross-tenant isolation is defensive/informational, not a hard security boundary
- Secrets can be stored with lightweight protection (env-var reference pattern preferred over mandatory encryption)
- WebSocket, workspaces, and approvals are useful but not blockers to a working system

---

## Status (2026-05-16, PR #79 open — agentic invariants + security layer + test hardening)

**Completed:** A1–A4, B1–B2, C1–C3, D1, E1–E5, F1–F4, G1–G2, H1–H2, I1, J1, K, L, M0, M1, N5–N11  
**PR:** [#79 — Review Go port tests](https://github.com/ubunatic/paperclip-go/pull/79) (branch `claude/review-go-port-tests-TjhEm`)  
**Next:** Phase P proposals (Opus review findings, see below) or further hardening  
**Build:** ✅ green (all test packages, 400+ total tests passing, race detector enabled)  
**Latest migration:** `0018_api_keys.sql`  
**Code quality:** ✅ Approval gate; budget hard-stop; concurrency guard; API key auth; routine run history; race detector in CI

**N5–N11 Session Findings (2026-05-16, Opus deep review):**
- ✅ **Implemented in this session:**
  - Schema fields: `issues.priority` + `issues.estimate`, `agents.budget_limit/budget_used`, `heartbeat_runs.prompt_tokens/completion_tokens/cost` (migration 0016)
  - Routine run history: `routine_runs` table + `RunService` (Record, ListByRoutine) (migration 0017)
  - API key auth: `api_keys` table, SHA-256 hashing service, `APIKeyAuth` middleware, `/api/apikeys` handler (migration 0018)
  - Approval gate in heartbeat runner: skips issues with pending approvals, returns `ErrApprovalPending`
  - Budget hard-stop: checks `budget_limit`/`budget_used` before run, returns `ErrBudgetExceeded`
  - Concurrency guard: `sync.Map` in-process guard + DB COUNT cross-process check, `ErrAlreadyRunning`
  - Context timeout: `context.WithTimeout` wrapping adapter calls (default 5 min)
  - Token/cost recording: extracts from Anthropic API response, stored after run, increments `budget_used`
  - Race detector: `go test -race ./...` in Makefile
  - `ListPendingByIssue` method on approvals service
  - Comprehensive tests for all new features (budget, context, concurrency, approval gate, API keys, routine runs, priority/estimate, auth middleware)
- ⚠️ **Open critical items from Opus review** (see Phase P proposals below)

**M1 Code Review Findings (2026-05-09):**
- ✅ **Fixed issues:**
  - WebSocket handler: Replaced lone `http.Error()` with `respond.Error()` for consistency (`internal/api/ws/handler.go:17`)
  - Scanner interface: Modernized `...interface{}` → `...any` in approvals and routines service (`internal/{approvals,routines}/service.go`)
  - WebSocket SetWriteDeadline: Moved outside select loop to eliminate per-event syscall overhead (`internal/api/ws/handler.go`)
  - Interactions pagination: Added `limit` query parameter to `ListByIssue` with default 100 / max 500 clamping (`internal/interactions/service.go`)
- 💡 **Status corrections from post-M0 review:**
  - Slice init consistency (approvals/routines) — already fixed in commit 024047b; was incorrectly listed as pending
  - ListByEntity limit clamping — already fixed in M0 (uses `DefaultEntityLimit=50`); was incorrectly listed as pending
  - Respond.go logging documentation — already present in respond.go:22-24; was incorrectly listed as pending
  - WS upgrade.go error handling — already used `respond.Error()` throughout; single remaining `http.Error()` was in handler.go (now fixed)

**H1 Code Review Findings (2026-05-06):**
- ✅ **Fixed issues:**
  - Missing `CHECK (status IN (...))` constraint in migration: Added to ensure invalid statuses cannot persist
  - Service instantiation pattern: Moved `workspaceSvc` and `routineSvc` from route closure to services block for consistency with all other services
  - Status validation gap: Added `IsValidWorkspaceStatus()` checks in both handler and service (defense-in-depth)
  - Misleading test name: Renamed `TestFKConstraints` to `TestCreateWithIssueID` to reflect actual test behavior
  - Variable shadowing: Fixed `ids` variable shadowing package import in `TestListByCompany`
- 💡 **Minor notes:**
  - Added explanatory comment to `Run()` method explaining why `workspace_id` is not set at creation time
  - Implemented `ListByCompany` returning empty slice instead of nil for consistency
  - All E2E tests cover happy path (create, retrieve, list, delete with proper status assertions)

**H2 Code Review Findings (2026-05-06):**
- ✅ **Approved for production** — RFC 6455 compliant, goroutine-safe, all tests passing under race detector
- 💡 **Minor notes (LOW severity, no action needed):**
  - E2E test frame parser assumes 2-byte header (works correctly via strings.Contains on full frame)
  - Event.Topic field never populated by publish calls (redundant, but not a bug since routing uses topic parameter)
  - No read-side disconnect detection; relies on write deadline (acceptable for server-push-only design)
  - Scheduler in serve.go uses separate service instances without bus (maintenance hazard only, not a correctness bug)

**M0+ Code Review Findings (2026-05-09):**
- ✅ **All tests passing** — 346 tests, no regressions, no critical bugs found
- 🐛 **Small fixes identified (apply next):**
  - WebSocket error handling: Replace 5 `http.Error()` calls with `respond.Error()` for consistency (`internal/api/ws/upgrade.go:28-46`)
  - Respond.go logging: Add documentation comment explaining silent encode-error logging policy (`internal/respond/respond.go:26`)
  - Slice initialization: Standardize `approvals`/`routines` services to use `make([]*Type, 0)` pattern like secrets/workspaces (consistency, not a bug since handlers have nil guards)
- 🏗️ **Design debt items:**
  - ListByEntity limit clamping: `limit=0` clamps to 500 instead of using sensible default; add separate defaultLimit constant
  - Interactions ListByIssue: Missing pagination limit (unbounded on high-activity issues); add optional limit parameter like ListByEntity
  - WebSocket SetWriteDeadline: Set on every loop iteration (inefficiency); move outside select loop
  - Secrets validation: Multiple `strings.TrimSpace()` checks could be consolidated (acceptable per existing notes, skip for now)

**G1 & G2 Code Review Findings (2026-05-05):**
- ✅ **Fixed issues:**
  - `RowsAffected()` error handling: Now explicitly checks for DB driver errors in `Trigger()` and `ClearDispatched()` (was silently ignoring errors).
  - HTTP status codes: Standardized on `StatusUnprocessableEntity` (422) for all validation errors (list/create required-field validation). Was inconsistent (400 vs 422).
- 🏗️ **Design debt (defer to post-MVP):**
  - `DispatchFingerprint` exposed in JSON responses: Verify with TS API if this is client-facing or should be omitted (likely internal-only implementation detail).
  - API path parity deviation: Go uses query params (`?companyId=X`), TS uses path params (`/companies/{id}/routines`). This is acceptable for Go simplification but should be documented.
- 🚀 **Test coverage gap:**
  - Handler unit tests missing for approvals & routines packages (only E2E coverage exists; handler edge cases like concurrent requests untested).

**G2 implementation (2026-05-04):**
- **Migration:** `routines` table with `dispatch_fingerprint` for dedup, `last_run_at` tracking, `enabled` flag, unique constraint on (company_id, name)
- **Service:** Full CRUD + `DueRoutines()` (cron matching), `MarkDispatched()` (atomic dedup), `ClearDispatched()` (reset for recurring cycles)
- **Cron parser:** 5-field stdlib-only parser with `IsDue()` and `NextAfter()`, handles `*`, `n`, `n-m`, `*/n`, `n,m,...`, fixed `*/n` logic for min > 0 fields (months, days)
- **Scheduler:** Background goroutine (60s tick), fires heartbeat.Run() for due routines, uses fingerprints for dedup, proper context cancellation
- **API handlers:** GET/POST/PATCH/DELETE/trigger endpoints, standardized error codes, E2E test coverage
- **CLI:** `routine create` and `routine list` commands with flag validation
- **Tests:** 30+ unit tests (service, cron edge cases, scheduler mocks), 10-step E2E test, all passing

#### I1 — `issue_thread_interactions` table + API ✅

**Files:** `internal/store/migrations/0014_issue_thread_interactions.sql`, `internal/domain/interaction.go`, `internal/interactions/service.go`, `internal/api/issues/handler.go`, `internal/api/router.go`, `internal/api/api_e2e_test.go`

**Completed (2026-05-05):**
- Migration: `issue_thread_interactions(id, company_id, issue_id, agent_id, comment_id, run_id, kind, status, idempotency_key, result, resolved_at, resolved_by_agent_id, created_at, updated_at)` with UNIQUE(issue_id, idempotency_key) for dedup
- Domain: `InteractionStatus` enum (pending, resolved) and `Interaction` struct with all 12 fields
- Service: Create (with idempotency dedup), GetByID, GetByIdempotencyKey, ListByIssue, Resolve (atomic UPDATE with conflict detection)
- HTTP handlers: 3 routes integrated into issues handler — GET/POST /api/issues/{id}/interactions, POST /api/issues/{id}/interactions/{iid}/resolve
- Router: `interactionSvc` instantiated and wired to issues.Handler()
- Tests: 13 unit tests (service) + comprehensive E2E test (7 cases); all passing
- Acceptance: ✅ Agent can post an interaction on an issue and resolve it atomically; idempotency keys prevent duplicate requests.

---

## Priority Tiers (road to a running version)

Phases grouped by what actually matters for a single-developer working system.

### Tier 1 — Minimum Running Version

| Phase | What | Why |
|---|---|---|
| E3 | `claude_local` heartbeat adapter | Heartbeat calls Claude; the system actually does something |
| F1 | Secrets (lightweight) | Store `ANTHROPIC_API_KEY` and other agent keys; plaintext+env-ref is fine for single dev |
| F2 | Instance settings | Configure server behaviour (deployment mode, origins) |

### Tier 2 — Useful for Daily Operation

| Phase | What | Why |
|---|---|---|
| E4 | `heartbeat_runs` extended fields | Upstream schema sync; liveness + retry state |
| E5 | `issues.origin_fingerprint` | Unlocks routine dedup (needed before G2) |
| G2 | Routines + cron scheduler | Schedule regular heartbeats without manual triggering |
| F4 | `db:backup` CLI | Data safety on VM |

### Tier 3 — Useful but Deferrable

| Phase | What | Why |
|---|---|---|
| G1 | Approvals | Human-in-loop gates; not critical solo |
| F3 | `env` CLI | Convenience wrapper over F1 API |
| I1 | Issue thread interactions | Agent continuation loop; complex |

### Tier 4 — Deferred (community interest)

| Phase | What | When |
|---|---|---|
| H1 | Execution workspaces | If workspace isolation becomes needed |
| H2 | WebSocket live events | If a UI consumer exists |
| Auth / RBAC | Multi-user access control | If others join |

---

## Ground rules

- **Do not modify** `server/`, `ui/`, `packages/`, `cli/`, `tests/`, `scripts/`,
  `docs/`, `evals/`, `skills/`, `package.json`, `pnpm-*.yaml`, `tsconfig*.json`,
  `vitest.config.ts`, `Dockerfile`.
- All Go code lives under `cmd/` and `internal/`.
- Run `make build` and `make test` before committing.
- **Mock LLM calls in tests** — any code that calls an LLM must accept an `Adapter`
  interface so tests inject a `MockAdapter` (deterministic, no network).
- Each phase is sized for a single Haiku agent session (~30–90 min):
  one package or endpoint group, clear acceptance criteria, tests required.

---

## Feature Parity Tracker

Legend: ✅ Done | ⚠️ Partial | 🟡 Stub | 🔲 Planned | ❌ Not started

### API Endpoints

| Area | TS endpoints | Go | Phase |
|---|---|---|---|
| `/api/health` | 1 | ✅ | — |
| `/api/companies` CRUD | 4 | ✅ | — |
| `PATCH /api/companies/{id}` | 1 | ✅ | A1 |
| `/api/agents` CRUD + me + patch | 6 | ✅ | — |
| Issue status enum validation | 1 | ✅ | A2 |
| Agent lifecycle (pause/resume/terminate) | 3 | ✅ | B1 |
| Agent configuration field | 1 | ✅ | B2 |
| `/api/issues` CRUD + checkout/release | 9 | ✅ | — |
| Issue labels | 5+ | ✅ | C1 |
| Issue documents / work-products | 5+ | ✅ | C2 |
| Issue read / archive state | 2 | ✅ | C3 |
| `/api/issues/{id}/comments` | 2 | ✅ | — |
| `/api/activity` GET | 1 | ✅ | — |
| `/api/activity` POST + issue-scoped | 3 | ✅ | D1 |
| `/api/heartbeat/runs` POST + GET list | 2 | ✅ | — |
| Heartbeat run detail GET + cancel | 2 | ✅ | E1 |
| `/api/skills` GET | 1 | ✅ | — |
| Dashboard / sidebar stubs | 4 | ✅ | — |
| `/api/secrets` CRUD | 8+ | ✅ | F1 |
| `/api/instance-settings` CRUD | 5+ | ✅ | F2 |
| `/api/approvals` | 10+ | ✅ | G1 |
| `/api/routines` CRUD + trigger | 15+ | ✅ | G2 |
| `/api/issues/{id}/interactions` | 5+ | ✅ | I1 |
| `/api/execution-workspaces` | 20+ | ✅ | H1 |
| `/api/apikeys` CRUD | 3 | ✅ | N9 |
| `APIKeyAuth` middleware | — | ✅ | N9 |
| `/api/costs` | 20+ | 🟡 | — (deferred) |
| `/api/goals` | 6 | 🟡 | — (deferred) |
| `/api/projects` | 25+ | 🟡 | — (deferred) |
| `/api/plugins` | 30+ | 🟡 | — (deferred) |

### CLI Commands

| Command | TS | Go | Phase |
|---|---|---|---|
| serve / init / doctor | ✅ | ✅ | — |
| company create/list | ✅ | ✅ | — |
| agent create/list | ✅ | ✅ | — |
| issue create/list/get | ✅ | ✅ | — |
| heartbeat run | ✅ | ✅ | — |
| `configure` | ✅ | ✅ | A3 |
| `onboard` (interactive setup) | ✅ | ✅ | A3 |
| `env list/set/get` | ✅ | ✅ | F3 |
| `db:backup` | ✅ | ✅ | F4 |
| `approval list/get` | ✅ | ✅ | G1 |
| `routine create/list` | ✅ | ✅ | G2 |
| `plugin install/list/remove` | ✅ | 🟡 | — (deferred) |

### Schema / Data Model

| Feature | TS | Go | Phase |
|---|---|---|---|
| `issues.labels` (junction table) | ✅ | ✅ | C1 |
| `issues.documents` / `work_products` | ✅ | ✅ | C2 |
| `issues.archived_at` | ✅ | ✅ | C3 |
| `agents.configuration` (YAML/JSON) | ✅ | ✅ | B2 |
| `agents.runtime_state` | ✅ | ✅ | B1 |
| `issues.execution_policy` | ✅ | 🔲 | C2+ |
| `heartbeat_runs` extended fields | ✅ | ✅ | E4 |
| `issues.origin_fingerprint` | ✅ | ✅ | E5 |
| `secrets` table | ✅ | ✅ | F1 |
| `instance_settings` table | ✅ | ✅ | F2 |
| `approvals` table | ✅ | ✅ | G1 |
| `routines` table | ✅ | ✅ | G2 |
| `issue_thread_interactions` table | ✅ | ✅ | I1 |
| `heartbeat_runs.workspace_id` | ✅ | ✅ | H1 |
| `execution_workspaces` table | ✅ | ✅ | H1 |
| WebSocket live events | ✅ | ✅ | H2 |
| `issues.priority` + `issues.estimate` | ✅ | ✅ | N5 |
| `agents.budget_limit/budget_used` | ✅ | ✅ | N6 |
| `heartbeat_runs.prompt_tokens/completion_tokens/cost` | ✅ | ✅ | N7 |
| `routine_runs` table | ✅ | ✅ | N8 |
| `api_keys` table (SHA-256 hash) | — | ✅ | N9 |
| `goals` / `projects` tables | ✅ | 🟡 | — (deferred) |
| Authentication (BetterAuth / RBAC) | ✅ | ❌ | — (deferred) |

### Heartbeat Adapters

| Adapter | TS | Go | Phase |
|---|---|---|---|
| Stub adapter | ✅ | ✅ | — |
| Mock adapter (test-only) | — | ✅ | E2 |
| `claude_local` adapter | ✅ | ✅ | E3 |
| Build version via ldflags | ✅ | ✅ | A4 |

---

## Phases

Each phase has: one agent, one package (or small group), tests required, `make test` green before commit.

---

### Phase A — Quick Wins ✅

| Phase | Description |
|---|---|
| A1 | `PATCH /api/companies/{id}` |
| A2 | Issue status enum validation |
| A3 | `configure` + `onboard` CLI commands |
| A4 | Build version via ldflags |

---

### Phase B — Agent Runtime State ✅

| Phase | Description |
|---|---|
| B1 | Agent `runtime_state` + pause/resume/terminate |
| B2 | Agent `configuration` field (JSON merge-patch) |

---

### Phase C — Issue Enhancements ✅

| Phase | Description |
|---|---|
| C1 | Issue labels (junction table, CRUD) |
| C2 | Issue documents / work-products (JSON columns) |
| C3 | Issue read / archive state (`archived_at`, filters) |

---

### Phase D — Activity Enhancements ✅

| Phase | Description |
|---|---|
| D1 | `POST /api/activity` + `GET /api/issues/{id}/activity` |

---

### Phase E — Heartbeat

#### E1 — Heartbeat run detail + cancel ✅

`GET /api/heartbeat/runs/{id}` and `POST /api/heartbeat/runs/{id}/cancel`.  
Cancel uses atomic conditional UPDATE; 409 if already terminal.

#### E2 — Mock adapter ✅

`MockAdapter` with callback injection lives in `internal/heartbeat/mock_adapter.go`.  
All 17 heartbeat tests pass.

#### E3 — `claude_local` heartbeat adapter ✅

Implemented: LLMClient interface for testable HTTP transport, ClaudeAdapter calling Anthropic Messages API, adapter registration in NewDefaultRegistry() when ANTHROPIC_API_KEY env var is set. Unit tests cover success, API errors, empty responses, and transport failures. All tests pass without a real API key.

#### E4 — `heartbeat_runs` extended fields (upstream sync HI-1) ✅

**Files:** `internal/store/migrations/0008_heartbeat_runs_ext.sql`, `internal/domain/heartbeat.go`, `internal/heartbeat/runner.go`

Completed:
- Migration: 8 new nullable/defaulted columns added
- Domain: 8 new fields added to `HeartbeatRun` struct
- Runner: `scanHeartbeatRun()` and SELECT queries updated
- Tests: All 23 heartbeat tests pass; `make test` green

Result: GET run response includes new fields (null/0 by default).

#### E5 — `issues.origin_fingerprint` (upstream sync HI-2)

**Files:** `internal/store/migrations/0009_issue_origin_fingerprint.sql`, `internal/domain/issue.go`

Tasks:
- Migration: `ALTER TABLE issues ADD COLUMN origin_fingerprint TEXT NOT NULL DEFAULT 'default'`.
- Add `OriginFingerprint string` to `domain.Issue`; include in scan/insert.
- Expose in API response (camelCase: `originFingerprint`).
- Unit test: create issue → field present; PATCH does not overwrite unless explicitly set.

Acceptance: `GET /api/issues/{id}` → `originFingerprint` field present; existing tests green.

---

### Phase F — Secrets & Settings

#### F1 — Secrets table + CRUD

**Files:** `internal/store/migrations/0010_secrets.sql`, `internal/domain/secret.go`, `internal/secrets/service.go`, `internal/api/secrets/handler.go`

Tasks:
- Migration: `secrets(id, company_id, name, value TEXT, created_at, updated_at)`.  
  Single-dev / trusted-VM scope: store values as plaintext. Encryption can be added if multi-user support is needed later.
- CRUD: `GET /api/secrets?companyId=`, `POST /api/secrets`, `GET /api/secrets/{id}`, `PATCH /api/secrets/{id}`, `DELETE /api/secrets/{id}`.
- `GET` list responses omit the value field (`{"id","name","createdAt"}`); `POST` and `GET /{id}` return the value.
- Unit tests: create, list (no values in list), get (value present), update, delete, 404.

Acceptance: `POST /api/secrets` → 201 with value; `GET /api/secrets` → list without values.

**Status: ✅ DONE (2026-05-01)**

Implemented: Migration 0010, domain types (Secret, SecretSummary), service CRUD with error handling, HTTP handlers for all endpoints, 13 unit tests + E2E test, router integration. All tests pass; code review passed cleanly.

#### F2 — Instance settings table + API ✅

**Files:** `internal/store/migrations/0011_instance_settings.sql`, `internal/settings/service.go`, `internal/api/settings/handler.go`

**Completed (2026-05-02):**
- Migration: `instance_settings(key TEXT PRIMARY KEY, value TEXT, updated_at TEXT)` — singleton KV store.
- Service: `GetAll()`, `Patch()`, `SeedDefaults()` — transactional UPSERT, empty-map return on empty table.
- HTTP handlers: `GET /api/instance-settings` and `PATCH /api/instance-settings` — flat JSON map response (no wrapper).
- Startup seeding: `deployment_mode=local_trusted`, `allowed_origins=localhost`.
- Tests: 6 service tests + 4 handler tests + 1 E2E test; all passing.
- Code review: Clean, idiomatic Go, no critical issues. (Removed dead domain type post-review.)

Acceptance: `GET /api/instance-settings` returns `{"deployment_mode":"local_trusted","allowed_origins":"localhost"}`. ✅

#### F3 — `env` CLI subcommand ✅

**Files:** `internal/cli/client.go`, `internal/cli/env.go`, `internal/cli/env_test.go`

**Completed (2026-05-02, post-review):**
- Migration: None (uses F1 secrets table)
- HTTP client wrapper: `HTTPClient` with base URL from config, PAPERCLIP_API_URL env override
- CLI commands: `env list|set|get` with three subcommands
  - `list --company <id>`: Lists secrets via `GET /api/secrets?companyId=X`, tabwriter output with name and creation date
  - `set KEY VALUE --company <id>`: Creates secret via `POST /api/secrets`, prints ID and name
  - `get KEY --company <id>`: Lists all secrets by company, finds by name, fetches full secret via `GET /api/secrets/{id}`, prints value to stdout
- Fallback behavior: Default HTTP client, auto-fallback to DB on `NewHTTPClient()` failure; optional `--db` flag for explicit DB use
- Context handling: Checks context cancellation before falling back to DB (respects user Ctrl+C)
- Tests: 10 unit tests covering HTTP and DB paths, mock HTTP servers, error cases (duplicates, not found)

Acceptance: ✅ `paperclip-go env set FOO bar --company acme` creates secret; `paperclip-go env list --company acme` shows FOO. All tests passing. Context cancellation respected.

#### F4 — `db:backup` CLI command ✅

**Files:** `internal/cli/dbbackup.go`, `internal/cli/dbbackup_test.go`, `internal/config/config.go` (BackupsDir() method)

**Completed (2026-05-02, post-review):**
- Migration: None (uses existing store)
- Command: `db:backup [--out path]` with optional destination flag
- Default behavior: Creates timestamped backup in `<data_dir>/backups/YYYY-MM-DD_HH-MM-SS.db`
- Implementation: Uses `VACUUM INTO` for safe online copy with zero reader/writer blocking
- Security: Path validation with `filepath.Abs()` + `Clean()` to prevent directory traversal; custom paths restricted to data dir; file permissions 0o600 (owner-only access)
- Context handling: Proper `cmd.Context()` and `ExecContext` usage
- Tests: 5 comprehensive unit tests covering default path, custom path, permissions, data integrity, integration (updated for allowedParent validation)
- Code review: Initial review + security fixes applied (SQL injection mitigation via strict path validation)

Acceptance: ✅ `paperclip-go db:backup` creates a secure, timestamped `.db` file in backups dir; `paperclip-go db:backup --out /custom/path.db` validates path within data directory; directory traversal attempts are rejected.

---

### Phase G — Approvals & Routines

> **Design note (G1):** Upstream TS uses `issue_thread_interactions` as the common substrate for approvals and agent continuation (see I1). Decide before starting G1 whether approvals should be a separate table or a thin layer over `issue_thread_interactions`. The simpler path for MVP is a standalone `approvals` table; refactor to interactions-backed if needed post-I1.

#### G1 — Approvals table + API + CLI ✅

**Files:** `internal/store/migrations/0012_approvals.sql`, `internal/domain/approval.go`, `internal/approvals/service.go`, `internal/api/approvals/handler.go`, `internal/cli/approval.go`

**Completed (2026-05-02):**
- Migration: `approvals(id, company_id, agent_id, issue_id, kind, status [pending|approved|rejected], request_body TEXT, response_body TEXT, created_at, resolved_at)` with efficient indexes
- Domain: `ApprovalStatus` enum (pending, approved, rejected) and `Approval` struct with all fields
- Service: CRUD operations (Create, GetByID, ListByCompany), atomic state transitions (Approve, Reject) with 409 conflict on double-resolve
- HTTP handlers: All 5 endpoints implemented — GET list, POST create, GET detail, POST approve, POST reject
- CLI: `approval list --company <id>` and `approval get <id>` commands with table/JSON output
- Router: Replaced stub endpoint with Mount to real handler
- Tests: 13 unit tests (service) + 10 E2E test cases; all passing; code review findings fixed (error handling, validation consolidation)
- Code quality: `errors.Is()` for error comparisons, removed redundant constraints, added symmetric test coverage

Acceptance: ✅ `POST /api/approvals` → 201; `POST /api/approvals/$ID/approve` → `status: "approved"`; `make test` green.

#### G2 — Routines table + API + CLI

**Files:** `internal/store/migrations/0013_routines.sql`, `internal/domain/routine.go`, `internal/routines/service.go`, `internal/api/routines/handler.go`, `internal/cli/routine.go`

Tasks:
- Migration: `routines(id, company_id, agent_id, name, cron_expr TEXT, enabled BOOLEAN DEFAULT 1, last_run_at TEXT, created_at, updated_at)`.  
  Include `dispatch_fingerprint` column for dedup (inline with this migration).
- `GET/POST /api/routines`, `GET/PATCH/DELETE /api/routines/{id}`, `POST /api/routines/{id}/trigger`.
- Cron scheduler: at `serve` startup, goroutine checks due routines every 60 s and fires a heartbeat run. Uses `issues.origin_fingerprint` (E5) for dedup.
- CLI: `paperclip-go routine create --name "daily" --cron "0 9 * * *" --agent $AID`, `paperclip-go routine list`.
- Replace stub with real handler.
- Unit tests: create, list, trigger, disable. Cron check uses a mock clock.

Acceptance: `POST /api/routines` → 201; `POST /api/routines/$ID/trigger` fires a heartbeat run row.

---

### Phase H — Execution Workspaces & Realtime

> These are the most complex phases. Each may need to be split into sub-agents.

#### H1 — Execution workspaces

**Files:** `internal/store/migrations/0014_workspaces.sql`, `internal/domain/workspace.go`, `internal/workspaces/service.go`, `internal/api/workspaces/handler.go`

Tasks:
- Migration: `execution_workspaces(id, agent_id, issue_id, heartbeat_run_id, status, path TEXT, created_at, updated_at)`.
- CRUD under `/api/execution-workspaces`.
- Link `heartbeat_runs.workspace_id` to workspaces (ALTER TABLE on `heartbeat_runs`).
- Unit tests: create, get, list, delete.

Acceptance: `POST /api/execution-workspaces` → 201; heartbeat run can reference a workspace.

#### H2 — WebSocket live events ✅

**Files:** `internal/events/bus.go`, `internal/api/ws/handler.go`, `internal/api/ws/upgrade.go`

**Completed (2026-05-06):**
- In-process event bus: MemBus with `Publish(topic, event)` and `Subscribe(topic) (<-chan Event, func())`
- Concurrent-safe snapshot pattern for publishers; buffered channels (32 events) with non-blocking sends
- RFC 6455-compliant WebSocket handshake (SHA-1 accept key, proper header validation, version check)
- Proper 8-byte payload length encoding for frames up to 64-bit sizes
- Event publication hooks in companies, agents, issues services on Create/Update/Pause/Resume/Terminate
- HTTP handler with graceful disconnection (context cancellation + write deadline)
- 5 unit tests (missing companyId, header validation, frame encoding, version check)
- Full E2E test: company → WS connect → issue.created event → client receives
- All 24 tests passing; race detector clean

Acceptance: ✅ Connect to `/api/ws?companyId=$CID`; POST /api/issues triggers `issue.created` event via WebSocket.

---

### Phase I — Agent Interaction Loop

#### I1 — `issue_thread_interactions` (upstream sync MED-1)

**Files:** `internal/store/migrations/0015_issue_thread_interactions.sql`, `internal/domain/interaction.go`, `internal/interactions/service.go`, `internal/api/issues/handler.go`

Tasks:
- Migration:
  ```
  issue_thread_interactions(
    id, company_id, issue_id, kind, status,
    continuation_policy, idempotency_key,
    source_comment_id, source_run_id,
    title, summary,
    created_by_agent_id, resolved_by_agent_id,
    payload TEXT, result TEXT,
    resolved_at, created_at, updated_at
  )
  ```
- Routes: `POST/GET /api/issues/{id}/interactions`, `POST /api/issues/{id}/interactions/{iid}/resolve`.
- Unit tests: create, list, resolve, idempotency key dedup.

Acceptance: agent can post an interaction on an issue and resolve it.

---

### Phase J — Post-MVP Polish & Quality Debt

#### J1 — Handler Unit Tests (Approvals & Routines) ✅

**Files:** `internal/api/approvals/handler_test.go`, `internal/api/routines/handler_test.go`, `internal/domain/routine.go`

**Completed (2026-05-07):**
- **Approvals handler tests** (11 tests): list validation (missing companyId → 422), create validation (invalid JSON, missing fields), create success (201), get not found (404)/success (200), approve/reject success and already-resolved conflicts (409).
- **Routines handler tests** (12 tests): list validation, create validation (invalid JSON, missing fields, invalid cron, name conflict), create success (201, no `dispatchFingerprint` in response), get not found/success, update invalid cron (422), delete/trigger not found.
- **DispatchFingerprint fix**: Changed `json:"dispatchFingerprint,omitempty"` to `json:"-"` on `Routine.DispatchFingerprint` to hide the internal dedup sentinel from API responses.
- **Test pattern**: Follows `internal/api/labels/handler_test.go` — uses `testutil.NewStore(t)` for hermetic SQLite, direct DB inserts for test data, chi router ServeHTTP calls for HTTP testing.
- **Code review**: Two defects identified and fixed: (1) missing error body assertion in reject test, (2) unnecessary DB setup in invalid-cron test. Test naming aligned with codebase convention (`TestHandlerVerb_Condition`).
- **Tests**: 23 new handler unit tests, all passing; 25 test packages total, all green; no race detector warnings.

Acceptance: ✅ `make test` green; handler error branches (validation, conflicts, not found) exercised independently from E2E; `DispatchFingerprint` field no longer exposed in JSON responses.

### Phase K — Handler Unit Tests (Companies, Agents, Issues) ✅

**Files:** `internal/api/companies/handler_test.go`, `internal/api/agents/handler_test.go`, `internal/api/issues/handler_test.go`

**Completed (2026-05-07, post-J1):**
- **Companies handler tests** (8 tests): create validation (invalid JSON, missing name/shortname), create success (201), update validation (empty name), update success (200), get not found (404), delete conflict (409 if agents exist), list success.
- **Agents handler tests** (10 tests): create validation (invalid JSON, missing companyId/shortname/displayName), create success (201, adapter defaults to "stub"), get not found/success, update validation (no patch fields), update null configuration, pause not found (404), pause invalid transition (409), delete conflict (409), list success (filters by companyId).
- **Issues handler tests** (10 tests): list validation (missing companyId → 422, invalid status → 422), create validation (invalid JSON, missing fields, invalid status), create success (201), get not found (404), update validation, checkout validation (missing agentId), checkout conflict (409), list success.
- **Boilerplate optimization**: Extracted `newTestHandler(t, s)` helper in issues to eliminate repeated 5-line service instantiation across 9 tests.
- **Test data uniqueness**: Randomized company/agent shortnames using UUID suffixes to avoid UNIQUE constraint violations in cross-test scenarios.
- **Pattern consistency**: All three files follow the established pattern from J1 (approvals/routines) — `testutil.NewStore(t)` per test, raw SQL data setup, chi router HTTP testing, error body assertions.
- **Tests**: 28 new handler unit tests (8+10+10), all passing; 25 test packages total remain green; no regressions.

Acceptance: ✅ `make test` green; handler validation branches (missing fields, invalid input, state conflicts) exercised independently from E2E for the three foundational API packages (companies, agents, issues); boilerplate reduced via helper extraction; test data isolation improved.

### Phase L — Handler Unit Tests (Remaining API Packages) ✅

**Files:** `internal/api/activity/handler_test.go`, `internal/api/secrets/handler_test.go`, `internal/api/heartbeat/handler_test.go`, `internal/api/workspaces/handler_test.go`, `internal/api/health/handler_test.go`

**Completed (2026-05-07, post-K):**
- **Activity handler tests** (7 tests): create validation (invalid JSON, missing fields), create success (201), list validation (missing companyId → 400), list success (scoped to company).
- **Secrets handler tests** (17 tests): CRUD endpoints with validation (missing name, whitespace value), duplicates (409 on create/update), not-found (404), list without values, update/delete success.
- **Heartbeat handler tests** (11 tests): create (missing agentId → 422, agent not found → 404, success → 201), list (missing agentId → 400, success), get (not found/success), cancel (not found/terminal status conflict → 409, success).
- **Workspaces handler tests** (14 tests): CRUD with validation (missing path, blank agentId, invalid status), default status (active), duplicate agent+path (409), not-found, delete success.
- **Health handler tests** (4 tests): status code 200, correct status field, all required fields present, version passthrough, features structure validation.
- **Code review refinements**: Extracted `newTestRunner()` helper in heartbeat to eliminate 55 lines of boilerplate; precise list assertions (exact count, not ≥); setup response guards to prevent panics on failure; consistent with K test patterns.
- **Test coverage**: 53 new handler unit tests across 5 packages, all passing; 15/15 API packages now have unit test coverage (100%).

Acceptance: ✅ `make test` green (all 25 packages + 53 handler tests); all API handler packages have comprehensive unit test coverage including validation, success paths, conflicts, and not-found scenarios; boilerplate extracted and optimized per code review feedback.

### Phase M0 — Decode Boilerplate Consolidation + Activity Pagination ✅

**Files:** `internal/respond/respond.go`, `internal/respond/respond_test.go`, `internal/api/{issues,agents,companies,secrets,routines,approvals,activity,workspaces,heartbeat,settings,labels}/handler.go`, `internal/activity/log.go`, `internal/activity/log_test.go`, `internal/api/secrets/handler.go`

**Completed (2026-05-08):**
- **Decode helper**: Added `respond.DecodeJSON(w, r, dst)` in respond package to centralize JSON body decoding (1 MiB limit, proper error responses for 400 bad JSON vs. 413 oversized bodies). Created `respond_test.go` with 4 comprehensive tests (valid, invalid JSON, oversized, empty body).
- **Boilerplate elimination**: Replaced 24 identical `http.MaxBytesReader` + `json.NewDecoder` blocks across 11 handler files with single `respond.DecodeJSON` call. Removed now-unused `encoding/json` imports from 9 files. Net result: 66 lines of boilerplate eliminated.
- **Activity pagination**: Added `LIMIT` clause to `ListByEntity()` with `const MaxEntityLimit = 500` and clamping logic. Updated single call site in issues handler to use exported constant. Added `TestListByEntity_LimitClamped` test covering limit enforcement and clamping edge cases.
- **Secrets handler cleanup**: Consolidated redundant `strings.TrimSpace()` validation checks in create/update handlers and `list` handler for clearer, single-check logic per field.
- **Code review fixes**: Fixed test logic error in `TestDecodeJSON_Valid` (guard condition). Exported `MaxEntityLimit` constant for cross-package reuse in activity handler. All 346 tests passing; no regressions.

Acceptance: ✅ `make test` green; zero `MaxBytesReader`/`json.NewDecoder` boilerplate remaining in handlers; `ListByEntity` has `LIMIT` with clamping; constant properly exported and reused; test coverage comprehensive.

### Phase M1 — WebSocket + Interactions Quality Debt ✅

**Files:** `internal/api/ws/handler.go`, `internal/approvals/service.go`, `internal/routines/service.go`, `internal/interactions/service.go`

**Completed (2026-05-09):**
- **WS handler consistency**: Replaced `http.Error()` with `respond.Error()` in `ws/handler.go:17` — the one remaining inconsistency after M0 was applied to upgrade.go.
- **Scanner interface modernization**: Changed `Scan(dest ...interface{})` → `Scan(dest ...any)` in approvals and routines scanner interfaces (Go 1.18+ convention).
- **WS SetWriteDeadline optimization**: Moved `conn.SetWriteDeadline()` call outside the select loop to eliminate per-event syscall overhead.
- **Interactions pagination**: Added optional `limit` query parameter to `GET /api/issues/{id}/interactions` (default 100, max 500) matching the activity log pattern.

Acceptance: ✅ `make test` green; all WS error paths use `respond.Error()`; interactions listing is bounded; scanner interfaces use modern `any` alias.

---

### Phase N — Completed in Session (2026-05-16) ✅

#### N1–N4 — Proposed (remain open, see below)

#### N5 — Issue priority + estimate fields ✅

**Migration:** `0016_fields.sql` — `issues.priority TEXT NOT NULL DEFAULT 'medium'`, `issues.estimate INTEGER`  
**Domain:** `Issue.Priority`, `Issue.Estimate *int`, `IsValidIssuePriority()`, `validPriorities` map, `ErrInvalidPriority` sentinel  
**Service:** Priority validated in Create and Update; defaults to "medium" on empty; `ErrInvalidPriority` on invalid values  
**Tests:** `internal/issues/priority_test.go` — 6 tests (default, explicit, estimate set/update, invalid create/update)

#### N6 — Agent budget fields ✅

**Migration:** `0016_fields.sql` — `agents.budget_limit INTEGER`, `agents.budget_used INTEGER NOT NULL DEFAULT 0`  
**Domain:** `Agent.BudgetLimit *int`, `Agent.BudgetUsed int`  
**Heartbeat:** Budget hard-stop in `runner.Run()` — checks `budget_used >= budget_limit` before proceeding, returns `ErrBudgetExceeded`  
**Tests:** `internal/heartbeat/budget_test.go` — 3 tests (blocked, allowed, increment after run)

#### N7 — Heartbeat token/cost tracking ✅

**Migration:** `0016_fields.sql` — `heartbeat_runs.prompt_tokens/completion_tokens/cost INTEGER NOT NULL DEFAULT 0`  
**Domain:** `HeartbeatRun.PromptTokens/CompletionTokens/Cost int`, `RunResult.PromptTokens/CompletionTokens/Cost int`  
**Adapter:** Claude adapter extracts `usage.input_tokens` / `usage.output_tokens` from Anthropic API response  
**Runner:** Stores tokens + cost after run; increments `agents.budget_used` by run cost

#### N8 — Routine run history ✅

**Migration:** `0017_routine_runs.sql` — `routine_runs(id, routine_id, agent_id, status, started_at, finished_at, error)`  
**Domain:** `RoutineRun` struct in `domain/routine.go`  
**Service:** `routines.NewRunService(s)` with `Record(ctx, routineID, agentID)` (status="dispatched") and `ListByRoutine(ctx, routineID)`  
**Tests:** `internal/routines/run_service_test.go` — 3 tests (record, list by routine, ordering)

#### N9 — API key auth layer ✅

**Migration:** `0018_api_keys.sql` — `api_keys(id, company_id, name, key_hash TEXT UNIQUE, created_at, revoked_at)`  
**Domain:** `domain.APIKey` (no key_hash exposed)  
**Service:** `apikeys.New(s)` — SHA-256 hashing, raw key returned once at creation; `ErrNotFound`, `ErrRevoked` sentinels  
**Middleware:** `api/middleware.APIKeyAuth(svc, skipPaths...)` — checks `X-Api-Key` header; 401 on missing/invalid/revoked  
**Router:** Conditionally applied when `deployment_mode != "local_trusted"`; skip paths: `/api/health`, `/api/apikeys`  
**Tests:** `internal/apikeys/service_test.go` (6 tests) + `internal/api/middleware/apikey_test.go` (6 tests)

#### N10 — Approval gate in heartbeat ✅

**Runner:** Before dispatching an issue, checks `approvalSvc.ListPendingByIssue(ctx, issue.ID)`; skips issues with pending approvals  
**Approvals service:** Added `ListPendingByIssue(ctx, issueID string) ([]*domain.Approval, error)`  
**Tests:** `internal/heartbeat/approval_test.go` — 2 tests (gate skips issue, gate allows issue without pending approvals)

#### N11 — Context timeout + concurrency guard ✅

**Context timeout:** `context.WithTimeout` wrapping adapter calls in runner (default 5 min via `runner.Timeout`)  
**Concurrency guard:** `sync.Map` in-process `LoadOrStore` guard (deterministic for goroutines) + DB COUNT check (cross-process stale rows)  
**Errors:** `ErrAlreadyRunning` when another run is in-flight for the same agent  
**Race detector:** `go test -race ./...` in Makefile  
**Tests:** `internal/heartbeat/concurrency_test.go` (3 tests) + `internal/heartbeat/context_test.go` (2 tests)

---

### Phase O — Proposed Next Steps (from N1–N4) 🔲

#### O1 — Handler tests for interactions routes

The three interaction routes (`POST/GET /api/issues/{id}/interactions`, `POST .../resolve`) are tested only via E2E. Add unit handler tests following the J1/K/L pattern using `testutil.NewStore(t)`.

**Files:** `internal/api/issues/handler_test.go` (extend) or new `internal/api/interactions/handler_test.go`

#### O2 — `routine create/list` CLI integration tests

The CLI commands exist but have no unit tests. Add tests following the `env_test.go` pattern with a mock HTTP server.

**Files:** `internal/cli/routine_test.go`

#### O3 — Structured logging

Replace scattered `log.Printf` calls with a minimal structured logger (stdlib `slog`, Go 1.21+) across handlers. Adds request-scoped context (method, path, duration) without external deps.

**Files:** `internal/api/router.go`, handler files

#### O4 — `approval create/get` CLI integration tests

Mirrors O2 for the approvals CLI commands.

**Files:** `internal/cli/approval_test.go`

---

### Phase P — Opus Review Proposals 🔲

Deep review by Opus on 2026-05-16 identified the following improvements. Critical security items (P3, P4) should be addressed before exposing the server to non-local traffic.

#### P1 — Atomic heartbeat claim via partial unique index (S)

**Problem:** `sync.Map` guard is in-process only; DB COUNT check + INSERT are not atomic. Two processes can both see 0 in-flight and both insert.  
**Fix:** Add a partial unique index: `CREATE UNIQUE INDEX IF NOT EXISTS heartbeat_runs_agent_inflight ON heartbeat_runs(agent_id) WHERE status = 'running'`. The INSERT then fails with a UNIQUE constraint error if another process already holds the lock; map the error to `ErrAlreadyRunning`.  
**Files:** new migration, `internal/heartbeat/runner.go`

#### P2 — Transactional cost + budget finalization (S)

**Problem:** Token/cost update and `budget_used` increment are two separate `ExecContext` calls; an error on the second is silently swallowed (`_, _ =`). A crash between the two leaves cost recorded but budget not incremented.  
**Fix:** Wrap both writes in a single DB transaction; surface errors instead of swallowing them.  
**Files:** `internal/heartbeat/runner.go`

#### P3 — Inject principal into request context — closes multi-tenant hole (M) ⚠️ CRITICAL

**Problem:** `APIKeyAuth` middleware validates the key but discards `APIKey.CompanyID`. Downstream handlers trust `?companyId=` query params from the caller. Any valid key for Company A can query Company B's data.  
**Fix:** Store the validated `*domain.APIKey` in request context (`context.WithValue`); handlers read company ID from context instead of from query param.  
**Files:** `internal/api/middleware/apikey.go`, all handlers that accept `?companyId=`

#### P4 — Protect `/api/apikeys` — separate bootstrap auth (M) ⚠️ CRITICAL

**Problem:** `/api/apikeys` is on the skip list so API key creation is unauthenticated in non-local_trusted mode. Anyone who can reach the server can create keys.  
**Fix:** Require a separate bootstrap secret (env var `PAPERCLIP_ADMIN_SECRET`) or restrict key creation to `local_trusted` mode only. Remove `/api/apikeys` from the skip list.  
**Files:** `internal/api/middleware/apikey.go`, `internal/api/router.go`

#### P5 — Routine run lifecycle: MarkSucceeded/Failed/Skipped (S)

**Problem:** Routine runs are created with `status="dispatched"` and never updated. There is no way to know if a scheduled run succeeded, failed, or was skipped (approval gate).  
**Fix:** Add `MarkSucceeded(ctx, id string)`, `MarkFailed(ctx, id, errMsg string)`, `MarkSkipped(ctx, id string)` to `RunService`; call from the scheduler after heartbeat.Run() returns.  
**Files:** `internal/routines/run_service.go`, `internal/api/serve.go` (scheduler)

#### P6 — Priority-aware issue selection (S)

**Problem:** Heartbeat runner selects an issue with no ordering — priority field exists but is ignored.  
**Fix:** Add `ORDER BY CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END, created_at` to the issue selection query.  
**Files:** `internal/heartbeat/runner.go`

#### P7 — Budget policy primitives (M/L)

**Problem:** Budget is a per-agent integer; there is no history of what consumed the budget, no reset mechanism, and no way to set per-issue budgets.  
**Fix:** Add `cost_events` table (`agent_id, run_id, amount, recorded_at`); accumulate via trigger or service call; policy engine for soft/hard limits.  
**Files:** new migration, `internal/heartbeat/runner.go`, `internal/agents/service.go`

#### P8 — Heartbeat watchdog (M)

**Problem:** If the server crashes while a run is `status='running'`, that run blocks the agent forever (ErrAlreadyRunning on next tick).  
**Fix:** On startup, find all runs where `status='running'` and `started_at < now - timeout`; mark them `failed` and clear the in-flight state.  
**Files:** `internal/heartbeat/runner.go` (startup hook or separate watchdog goroutine)

#### P9 — Claude adapter productionization (M)

**Problem:** Adapter uses a hardcoded 500-token limit, no system prompt, no retry on transient errors, no timeout per-request (only the runner-level timeout).  
**Fix:** Make `max_tokens` configurable via agent `configuration` JSON; add a system prompt derived from agent `configuration`; add retry with exponential backoff (2x, max 3 attempts) for 429/5xx; propagate per-call deadline from context.  
**Files:** `internal/heartbeat/claude_adapter.go`

#### P10 — `/api/auth` + agent JWT (L)

**Problem:** No authentication for human users. API key auth covers service-to-service but not browser sessions.  
**Fix:** Add `/api/auth/login` endpoint issuing short-lived JWTs; add JWT middleware alongside API key middleware.  
**Files:** new `internal/api/auth/` package, `internal/api/middleware/jwt.go`

#### P11 — Refactor NewRouter into Server struct (S)

**Problem:** `NewRouter()` takes 14+ parameters and is hard to extend without breaking callers.  
**Fix:** Introduce a `Server` struct holding all dependencies; `NewServer(store, settings, ...)` constructor; `Server.Routes() http.Handler` method.  
**Files:** `internal/api/router.go`

#### P12 — Settings hot-reload for deployment_mode (S)

**Problem:** `deployment_mode` is read once at startup (middleware decision). Changing it via PATCH requires a server restart.  
**Fix:** Read `deployment_mode` from the settings service on every request (cached with 1s TTL) instead of capturing at router construction time.  
**Files:** `internal/api/router.go`, `internal/api/middleware/apikey.go`

#### P13 — Wire workspace_id into heartbeat (M)

**Problem:** `heartbeat_runs.workspace_id` column exists but is never populated by the runner.  
**Fix:** Before running an adapter, create (or look up) an `execution_workspace` record for the agent+issue; store its ID in `heartbeat_runs.workspace_id` on INSERT.  
**Files:** `internal/heartbeat/runner.go`, `internal/workspaces/service.go`

#### P14 — API key prefix + last-4 display (S)

**Problem:** Once created, there is no way to identify which key is which from the list endpoint (all fields are opaque UUIDs).  
**Fix:** Store a `prefix` column (first 8 chars of the raw key, safe to expose) and `last4` (last 4 chars) alongside `key_hash`; return these in list/detail responses.  
**Files:** `internal/store/migrations/` (new), `internal/apikeys/service.go`, `internal/domain/apikey.go`

---

## LLM Mocking Convention

All adapters that call external LLMs **must** accept an interface for the HTTP transport:

```go
// internal/heartbeat/llm_client.go
type LLMClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

Tests inject a `mockLLMClient` (defined in `_test.go`) that returns a pre-built `*http.Response` from a string fixture:

```go
func newMockLLMClient(body string, status int) LLMClient {
    return &mockLLMClient{body: body, status: status}
}
```

This keeps every LLM-touching test hermetic and fast — no network, no API key.

---

## Testing conventions

- Every service package has a `_test.go` using `testutil.NewStore(t)` (temp-file SQLite, auto-migrated).
- New migrations must be idempotent and backwards-compatible (ADD COLUMN with DEFAULT).
- E2E tests live in `internal/api/api_e2e_test.go`; add a function per phase (e.g. `TestSecretsE2E`).
- `make test` must stay green after every phase.

---

## Commit discipline

Each phase = one or more commits, one commit per logical unit:
1. Migration SQL
2. Domain type + service (with tests)
3. HTTP handler
4. CLI command (if any)

Commit message format: `feat(<area>): <what> — <why>`  
Example: `feat(secrets): add secrets table + CRUD — needed for agent API key storage`

---

## Quality Debt (post-MVP)

| Item | Severity | Location | Status | Effort |
|------|----------|----------|--------|--------|
| ✅ SQL injection in db:backup VACUUM INTO | CRITICAL | `internal/cli/dbbackup.go:37` | FIXED | — |
| ✅ Context cancellation in env CLI | MEDIUM | `internal/cli/env.go:65-233` | FIXED | — |
| ✅ RowsAffected() error handling in routines | MEDIUM | `internal/routines/service.go:231,310` | FIXED (2026-05-05) | — |
| ✅ HTTP status code consistency (G1/G2) | LOW | `internal/api/routines,approvals/handler.go` | FIXED (2026-05-05) | — |
| ✅ DispatchFingerprint exposure in API | LOW | `internal/domain/routine.go:15` | FIXED (2026-05-07, J1) | — |
| ✅ Handler unit tests missing (G1/G2) | MEDIUM | `internal/api/approvals,routines/` | FIXED (2026-05-07, J1) — 23 tests | — |
| ✅ `MaxBytesReader` boilerplate (24 sites) | LOW | `internal/api/*/handler.go` | FIXED (2026-05-08, M0) — DecodeJSON | — |
| ✅ Unbounded `ListByEntity()` pagination | MEDIUM | `internal/activity/log.go` | FIXED (2026-05-08, M0) — added LIMIT + clamping | — |
| ✅ Slice init consistency (approvals/routines) | MEDIUM | `internal/{approvals,routines}/service.go` | FIXED (2026-05-09, 024047b) | — |
| ✅ ListByEntity limit clamping logic | LOW | `internal/activity/log.go:95-98` | FIXED (2026-05-08, M0) — DefaultEntityLimit=50 | — |
| ✅ Respond.go logging documentation | LOW | `internal/respond/respond.go:22-24` | FIXED (2026-05-08, M0) — doc comment present | — |
| ✅ WebSocket error handling inconsistency | LOW | `internal/api/ws/handler.go:17` | FIXED (2026-05-09, M1) — respond.Error() | — |
| ✅ Scanner interface `...interface{}` → `...any` | LOW | `internal/{approvals,routines}/service.go` | FIXED (2026-05-09, M1) — modernized | — |
| ✅ Interactions ListByIssue pagination | MEDIUM | `internal/interactions/service.go:99` | FIXED (2026-05-09, M1) — limit param, default 100/max 500 | — |
| ✅ WebSocket SetWriteDeadline inefficiency | LOW | `internal/api/ws/handler.go:51` | FIXED (2026-05-09, M1) — moved outside select | — |
| 🔴 Budget check races in-flight lock | CRITICAL | `internal/heartbeat/runner.go` | OPEN — budget checked before `sync.Map` store; a second goroutine can pass budget check then be blocked, wasting one slot | P2 |
| 🔴 Cost/budget update errors silently swallowed | CRITICAL | `internal/heartbeat/runner.go` | OPEN — `_, _ = s.DB.ExecContext(...)` on finalization writes; errors are lost | P2 |
| 🔴 APIKey.CompanyID discarded in middleware | CRITICAL | `internal/api/middleware/apikey.go` | OPEN — multi-tenant data leak: any valid key can query any company's data via query param | P3 |
| 🔴 `/api/apikeys` reachable unauthenticated | CRITICAL | `internal/api/router.go` skip list | OPEN — key creation requires no auth in non-local_trusted mode | P4 |
| 🔴 Skip list exact-match doesn't cover sub-paths | CRITICAL | `internal/api/middleware/apikey.go` | OPEN — `r.URL.Path == skipPath` only; `/api/health/extra` would require auth unexpectedly; `/api/apikeys/123` is still protected | P4 |
| 🟠 COUNT check + INSERT not atomic cross-process | MAJOR | `internal/heartbeat/runner.go` | OPEN — two separate processes can both pass the COUNT=0 check and both insert | P1 |
| 🟠 Issue selection has no ordering (ignores priority) | MAJOR | `internal/heartbeat/runner.go` | OPEN — priority field exists but SELECT has no ORDER BY | P6 |
| 🟠 Approval-pending idles agent forever | MAJOR | `internal/heartbeat/runner.go` | OPEN — same pending-approval issue will be selected every tick, blocking all other issues | P6 |
| 🟠 Routine runs stay "dispatched" on error | MAJOR | `internal/routines/run_service.go` | OPEN — no MarkSucceeded/Failed/Skipped; scheduler doesn't update status | P5 |
| 🟠 Finalization writes use cancelled context | MAJOR | `internal/heartbeat/runner.go` | OPEN — if adapter call times out, the deferred finalization ExecContext uses the already-cancelled ctx | P2 |
| 🟡 `issues` variable shadows package in runner.go | MINOR | `internal/heartbeat/runner.go` | OPEN — local `issues` var shadows `issues` package import | P6 |
| 🏗️ Secrets TrimSpace validation consolidation | LOW | `internal/api/secrets/handler.go:60` | Acceptable | 5 min |
| Cross-tenant isolation at route level | MEDIUM | DELETE/PATCH/state endpoints | Phase P3 | — |
| State machine RBAC | MEDIUM | pause/resume/terminate handlers | Phase P10+ | — |
| Structured logging | LOW-MED | `internal/api/{activity,issues,agents}/handler.go` | Deferred (O3) | 20 min |
| Response wrapping inconsistency | LOW | GET returns `{items}`, POST returns raw object | Deferred | 30 min |

---

## Deferred

These are out of scope for a single-developer deployment. Revisit if community interest grows.

- **Auth / RBAC / multi-user** — BetterAuth, board-claim flow, permission checks (Phase P10)
- **Embedded Postgres** — SQLite is fine for a single-dev VM
- **Plugin host / external adapter processes** — useful at scale, not needed solo
- **Full schema parity** — `goals`, `projects`, `costs`, `budgets` (deferred until needed)
- **Data sharing with the TS instance** — migration path TBD if ever needed
- **Budget policy engine** — cost_events table, per-issue budgets (Phase P7)
- **Agent JWT / browser session auth** — Phase P10

> **Security note:** For production or shared deployments, P3 and P4 are blockers. The current multi-tenant isolation is defense-by-convention only — all companyId filtering is caller-supplied.
