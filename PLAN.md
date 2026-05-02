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

## Status (2026-05-02, post-review)

**Completed:** A1–A4, B1–B2, C1–C3, D1, E1–E5, F1–F4 ✅ (security review & fixes applied)  
**Next:** G1 — Approvals table + API + CLI (ready to start)  
**Build:** ✅ green (all 27+ test packages, 10+ CLI tests)  
**Latest migration:** `0011_instance_settings.sql`

**Recent fixes (2026-05-02):**
- **F4 (db:backup):** Fixed SQL injection vulnerability in VACUUM INTO; now validates absolute paths against allowed parent directory, prevents directory traversal
- **F3 (env CLI):** Added context cancellation checks in HTTP fallback paths; respects user Ctrl+C instead of silently falling back to DB
- **Tests:** All updates to account for path validation in db:backup (5 tests updated)

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
| `/api/approvals` | 10+ | 🔲 | G1 |
| `/api/routines` CRUD + trigger | 15+ | 🔲 | G2 |
| `/api/issues/{id}/interactions` | 5+ | 🔲 | I1 |
| `/api/execution-workspaces` | 20+ | 🔲 | H1 |
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
| `approval list/get` | ✅ | 🔲 | G1 |
| `routine create/list` | ✅ | 🔲 | G2 |
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
| `approvals` table | ✅ | 🔲 | G1 |
| `routines` table | ✅ | 🔲 | G2 |
| `issue_thread_interactions` table | ✅ | 🔲 | I1 |
| `heartbeat_runs.workspace_id` | ✅ | 🔲 | H1 |
| `execution_workspaces` table | ✅ | 🔲 | H1 |
| WebSocket live events | ✅ | 🔲 | H2 |
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

#### G1 — Approvals table + API + CLI

**Files:** `internal/store/migrations/0012_approvals.sql`, `internal/domain/approval.go`, `internal/approvals/service.go`, `internal/api/approvals/handler.go`, `internal/cli/approval.go`

Tasks:
- Migration: `approvals(id, company_id, agent_id, issue_id, kind, status [pending|approved|rejected], request_body TEXT, response_body TEXT, created_at, resolved_at)`.
- `GET /api/approvals?companyId=`, `POST /api/approvals`, `GET /api/approvals/{id}`, `POST /api/approvals/{id}/approve`, `POST /api/approvals/{id}/reject`.
- CLI: `paperclip-go approval list --company <id>`, `paperclip-go approval get <id>`.
- Replace the existing `/api/approvals` stub.
- Unit tests: create, list, approve, reject, 409 on double-resolve.

Acceptance: `POST /api/approvals` → 201; `POST /api/approvals/$ID/approve` → `status: "approved"`.

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

#### H2 — WebSocket live events

**Files:** `internal/api/ws/handler.go`, `internal/events/bus.go`

Tasks:
- In-process event bus: `Publish(topic, payload)` / `Subscribe(topic) <-chan Event`.
- Publish events from companies/agents/issues/heartbeat services on create/update/delete.
- `GET /api/ws` upgrades to WebSocket; client subscribes to `companyId`; server fans out events.
- Unit tests: publish → subscriber receives; disconnect cleans up subscription.

Acceptance: connect to `/api/ws?companyId=$CID`; create issue via API → WS message arrives.

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
| Redundant validation in secrets handler | LOW | `internal/api/secrets/handler.go:31` | Acceptable | <1 min |
| Inconsistent error handling in env CLI | LOW | `internal/cli/env.go:200+` | Minor | 2 min |
| HTTP client lifecycle inefficiency | LOW | `internal/cli/env.go` | Minor | 5 min |
| Structured logging | LOW-MED | `internal/api/{activity,issues,agents}/handler.go` | Deferred | 20 min |
| Unbounded `ListByEntity()` pagination | MEDIUM | `internal/activity/log.go` | Deferred | 15 min |
| `MaxBytesReader` boilerplate (8 sites) | LOW | `internal/api/*/handler.go` | Deferred | 20 min |
| Response wrapping inconsistency | LOW | GET returns `{items}`, POST returns raw object | Deferred | 30 min |
| Handler unit tests missing | MEDIUM | agents, issues, companies packages | Deferred | 1–2 h |
| Cross-tenant isolation at route level | MEDIUM | DELETE/PATCH/state endpoints | Phase F+ | — |
| State machine RBAC | MEDIUM | pause/resume/terminate handlers | Phase F+ | — |

---

## Deferred

These are out of scope for a single-developer deployment. Revisit if community interest grows.

- **Auth / RBAC / multi-user** — BetterAuth, board-claim flow, permission checks
- **Embedded Postgres** — SQLite is fine for a single-dev VM
- **Plugin host / external adapter processes** — useful at scale, not needed solo
- **Full schema parity** — `goals`, `projects`, `costs`, `budgets` (deferred until needed)
- **Data sharing with the TS instance** — migration path TBD if ever needed
- **WebSocket live events (H2)** — only matters with a live UI consumer
- **Execution workspaces (H1)** — only needed when sandboxing agent execution
