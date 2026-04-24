# Paperclip-Go — Feature Parity Plan

> Previous MVP plan archived in `PLAN.archive.md`.  
> This plan tracks Go→TS parity starting from the completed 9-phase MVP.  
> All 9 MVP phases are ✅ DONE. This plan covers what remains.

---

## Status & Recent Review (2026-04-24)

**Phases Completed:** A1-A4, B1-B2, C1-C2 ✅  
**Build Status:** ✅ `make build && make test` green

**Code Quality Review Summary (2026-04-24):**

| Item | Status | Details |
|------|--------|---------|
| C2 Implementation | ✅ COMPLETE | Documents/work_products JSON arrays added to issues table; PATCH support with replacement semantics; comprehensive E2E + unit tests |
| A-C Security Audit | ✅ FIXED | Cross-tenant isolation assumptions documented; Delete() docstring corrected to reflect actual dependencies (assigned/checked-out/comments/runs, not just in_progress) |
| Code Quality | ✅ FIXED | All code review findings addressed: gofmt compliance, JSON unmarshal error logging, complete test coverage with edge cases |
| Service Tests | ✅ ADDED | Unit tests for documents/workProducts set/clear at service layer; E2E tests cover persistence, clearing, and 404 errors |
| Parity | ✅ Verified | Response schemas match TS (camelCase JSON keys, Agent runtimeState/configuration, Issue documents/workProducts); all HTTP status codes consistent (204, 200, 404, 409, 422) |
| Testing | ✅ IMPROVED | Added service-level unit tests + comprehensive E2E coverage (set, retrieve, clear, cross-field persistence, 404 error case) |
| Design Debt | 📝 Noted | Handler unit tests (A-C packages); route-level cross-tenant isolation; state machine RBAC guards (deferred to Phase D+) |
| Next Phase | → C3 | Archive/read state: `archived_at` column, archive/unarchive endpoints, filter exclusion by default |

---

## Ground rules (unchanged from MVP)

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
| Issue read / archive state | 2 | 🔲 | C3 |
| `/api/issues/{id}/comments` | 2 | ✅ | — |
| `/api/activity` GET | 1 | ✅ | — |
| `/api/activity` POST + issue-scoped | 3 | 🔲 | D1 |
| `/api/heartbeat/runs` POST + GET | 2 | ✅ | — |
| Heartbeat run detail GET | 1 | 🔲 | E1 |
| Heartbeat run cancel | 1 | 🔲 | E1 |
| `/api/skills` GET | 1 | ✅ | — |
| `/api/secrets` CRUD | 8+ | 🔲 | F1 |
| `/api/instance-settings` CRUD | 5+ | 🔲 | F2 |
| `/api/approvals` | 10+ | 🟡 | G1 |
| `/api/costs` | 20+ | 🟡 | — (deferred) |
| `/api/goals` | 6 | 🟡 | — (deferred) |
| `/api/projects` | 25+ | 🟡 | — (deferred) |
| `/api/routines` CRUD | 15+ | 🔲 | G2 |
| `/api/plugins` | 30+ | 🟡 | — (deferred) |
| `/api/execution-workspaces` | 20+ | 🔲 | H1 |
| Dashboard / sidebar stubs | 4 | ✅ | — |

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
| `env list/set/get` | ✅ | 🔲 | F3 |
| `db:backup` | ✅ | 🔲 | F4 |
| `approval list/get` | ✅ | 🔲 | G1 |
| `routine create/list` | ✅ | 🔲 | G2 |
| `plugin install/list/remove` | ✅ | 🟡 | — (deferred) |

### Schema / Data Model

| Feature | TS | Go | Phase |
|---|---|---|---|
| `issues.labels` (junction table) | ✅ | ✅ | C1 |
| `issues.documents` / `work_products` | ✅ | ✅ | C2 |
| `issues.execution_policy` | ✅ | 🔲 | C2+ |
| `agents.configuration` (YAML/JSON) | ✅ | ✅ | B2 |
| `agents.runtime_state` | ✅ | ✅ | B1 |
| `secrets` table | ✅ | 🔲 | F1 |
| `routines` table | ✅ | 🔲 | G2 |
| `goals` / `projects` tables | ✅ | 🟡 | — (deferred) |
| `approvals` table | ✅ | 🔲 | G1 |
| `instance_settings` table | ✅ | 🔲 | F2 |
| `heartbeat_runs.workspace_id` | ✅ | 🔲 | H1 |
| WebSocket live events | ✅ | 🔲 | H2 |
| Authentication (BetterAuth / RBAC) | ✅ | ❌ | — (deferred) |

### Heartbeat Adapters

| Adapter | TS | Go | Phase |
|---|---|---|---|
| Stub adapter | ✅ | ✅ | — |
| Mock adapter (test-only) | — | 🔲 | E2 |
| `claude_local` adapter | ✅ | 🔲 | E3 |
| Build version via ldflags | ✅ | ✅ | A4 |

---

## Phases

Each phase has: one agent, one package (or small group), tests required, `make test` green before commit.

---

### Phase A — Quick Wins (no new tables)

> Fixes and small additions that require no schema changes. Each sub-task can be done independently.

#### A1 — `PATCH /api/companies/{id}` ✅

**Files:** `internal/companies/service.go`, `internal/api/companies/handler.go`, `internal/companies/service_test.go`

Tasks: ✅ COMPLETE
- Add `Update(ctx, id, fields)` method to companies service using an explicit patch/fields type (for example, pointer fields such as `*string` for `name` and `description`) so the service can distinguish "not provided" from "provided as empty".
- Add `PATCH /{id}` route in companies handler: decode into that patch type, call service, and apply only fields that are present; this must allow setting values to zero values such as clearing `description` to `""`; return 200 + updated company.
- Unit test: update name, update description, update both, clear description to empty string, 404 on missing id.

Acceptance: `curl -XPATCH localhost:3200/api/companies/$CID -d '{"name":"New"}' -H 'content-type:application/json'` → 200 with updated name.

#### A2 — Issue status enum validation ✅

**Files:** `internal/issues/service.go`, `internal/domain/issue.go`

Tasks: ✅ COMPLETE
- Define `ValidStatuses` set in `domain/issue.go` — ✅ Already existed
- In `issues.Service.Create` and `issues.Service.Update`, validate `status` field against the set; return `ErrInvalidStatus` (→ 422) for unknown values — ✅ Update already validated, Create now validates
- Unit test: valid status accepted, invalid status rejected with correct error — ✅ Added TestCreateValidStatus, TestCreateInvalidStatus

Acceptance: ✅ `POST /api/issues` with `"status":"bogus"` → 422.

#### A3 — `configure` + `onboard` CLI commands ✅

**Files:** `internal/cli/configure.go`, `internal/cli/onboard.go`

Tasks: ✅ COMPLETE
- `configure`: prints the active config path and YAML content (read-only view for MVP).
- `onboard`: interactive prompts for `name`, `shortname`, calls `POST /api/companies`, prints the created company ID. If `--remote` not given, opens the DB directly.
- Add both commands to `internal/cli/root.go`.

Acceptance: `paperclip-go configure` prints config; `paperclip-go onboard` creates a company via prompts.

#### A4 — Build version via ldflags ✅

**Files:** `cmd/paperclip-go/main.go`, `internal/api/health/handler.go`, `Makefile`

Tasks: ✅ COMPLETE
- Declare `var Version = "dev"` in `main.go`; pass to `cli.Execute(version)`.
- Thread version string into health handler response.
- In `Makefile`, add `-ldflags "-X main.Version=$(git describe --tags --always --dirty)"` to the `build` target.
- Update `TestHealthE2E` to accept any non-empty string.

Acceptance: `make build && ./bin/paperclip-go serve` → `GET /api/health` returns non-`"dev"` version when git tag is present.

---

### Phase B — Agent Runtime State

> Adds `runtime_state` and `configuration` fields to agents without breaking existing tests.

#### B1 — Agent `runtime_state` field ✅

**Files:** `internal/store/migrations/0002_agent_runtime.sql`, `internal/domain/agent.go`, `internal/agents/service.go`, `internal/api/agents/handler.go`

Tasks: ✅ COMPLETE
- Migration: `ALTER TABLE agents ADD COLUMN runtime_state TEXT DEFAULT 'idle'` (values: `idle|running|paused|terminated`).
- Add `RuntimeState` to `domain.Agent`.
- `PATCH /api/agents/{id}` already exists; extend to accept `runtimeState` field.
- Add `POST /api/agents/{id}/pause`, `POST /api/agents/{id}/resume`, `POST /api/agents/{id}/terminate` handlers — each updates `runtime_state` and writes an activity log entry.
- Unit tests: each lifecycle transition, invalid transition returns 422.

Acceptance: `POST /api/agents/$AID/pause` → 200 with `runtimeState: "paused"`.

#### B2 — Agent `configuration` field ✅

**Files:** `internal/store/migrations/0003_agent_config.sql`, `internal/domain/agent.go`, `internal/agents/service.go`

Tasks: ✅ COMPLETE
- Migration: `ALTER TABLE agents ADD COLUMN configuration TEXT DEFAULT '{}'` (stored as JSON string).
- Add `Configuration map[string]any` (serialized to/from JSON) to `domain.Agent`.
- `PATCH /api/agents/{id}` accepts `configuration` key; merge-patches existing config.
- Unit tests: set config, retrieve config, partial update preserves existing keys.
- E2E test added for configuration PATCH endpoint.

Acceptance: ✅ PATCH /api/agents/$AID -d '{"configuration":{"model":"claude-opus-4"}}' → 200; GET /api/agents/$AID → config persisted.

---

### Phase C — Issue Enhancements

#### C1 — Issue labels ✅

**Files:** `internal/store/migrations/0004_labels.sql`, `internal/domain/label.go`, `internal/labels/service.go`, `internal/api/labels/handler.go`, `internal/api/issues/handler.go`

Tasks: ✅ COMPLETE
- Migration: `labels(id, company_id, name, color)` and `issue_labels(issue_id, label_id)` junction.
- `GET /api/issues/{id}` returns `labels []Label` in response.
- `POST /api/issues/{id}/labels` adds a label by id.
- `DELETE /api/issues/{id}/labels/{labelId}` removes.
- `GET/POST /api/labels` (scoped to `companyId`) for label management.
- Unit tests: add label, list labels on issue, remove label, duplicate add is idempotent.

Acceptance: create label, attach to issue, list issue → `labels` array populated.

#### C2 — Issue documents / work-products

**Files:** `internal/store/migrations/0005_issue_docs.sql`, `internal/domain/issue.go`, `internal/issues/service.go`, `internal/api/issues/handler.go`

Tasks:
- Migration: `ALTER TABLE issues ADD COLUMN documents TEXT DEFAULT '[]'` and `work_products TEXT DEFAULT '[]'` (stored as JSON arrays).
- Add `Documents []any` and `WorkProducts []any` to `domain.Issue`.
- `PATCH /api/issues/{id}` accepts these fields; replace (not merge) on update.
- Unit tests: set documents, retrieve, clear.

Acceptance: `PATCH /api/issues/$IID -d '{"documents":[{"title":"spec","url":"..."}]}'` → 200; GET returns documents.

#### C3 — Issue read/archive state

**Files:** `internal/store/migrations/0006_issue_state.sql`, `internal/domain/issue.go`, `internal/issues/service.go`

Tasks:
- Migration: `ALTER TABLE issues ADD COLUMN archived_at TEXT DEFAULT NULL`.
- `POST /api/issues/{id}/archive` sets `archived_at`; `POST /api/issues/{id}/unarchive` clears it.
- `GET /api/issues` default filter excludes archived; `?includeArchived=true` includes them.
- Unit tests: archive, list (excluded), list with flag (included), unarchive.

Acceptance: archive issue → not in default list; `?includeArchived=true` → visible.

---

### Phase D — Activity Enhancements

#### D1 — POST activity + issue-scoped activity

**Files:** `internal/activity/log.go`, `internal/api/activity/handler.go`

Tasks:
- Add `POST /api/activity` endpoint: accepts `{companyId, actorKind, actorId, action, entityKind, entityId, metaJson?}` and inserts a row.
- Add `GET /api/issues/{id}/activity` route in the issues handler: queries `activity_log WHERE entity_kind='issue' AND entity_id=?` ordered by `created_at`.
- Unit tests: post entry, list by company, list by issue.

Acceptance: `POST /api/activity` creates a row; `GET /api/issues/$IID/activity` returns it.

---

### Phase E — Heartbeat Improvements

#### E1 — Heartbeat run detail + cancel

**Files:** `internal/api/heartbeat/handler.go`, `internal/heartbeat/runner.go`

Tasks:
- Add `GET /api/heartbeat/runs/{id}` returning full run record.
- Add `POST /api/heartbeat/runs/{id}/cancel`: sets `status='cancelled'` if run is `running`; 409 if already terminal.
- Unit tests: get existing run, get missing run (404), cancel running, cancel already finished (409).

Acceptance: start run → GET returns it; POST cancel → status `cancelled`.

#### E2 — Mock adapter for tests

**Files:** `internal/heartbeat/mock_adapter.go`, update existing tests

Tasks:
- Add `MockAdapter` struct in `internal/heartbeat/` implementing `Adapter` interface.
- Constructor: `NewMockAdapter(summaryFn func(RunContext) RunResult)` — lets tests inject deterministic responses.
- Replace ad-hoc test stubs in `runner_test.go` with `MockAdapter`.
- Export `MockAdapter` for use in integration tests.

Acceptance: `runner_test.go` uses `MockAdapter`; `go test ./internal/heartbeat/...` ✅.

#### E3 — `claude_local` heartbeat adapter

**Files:** `internal/heartbeat/claude_adapter.go`, `internal/heartbeat/claude_adapter_test.go`

Tasks:
- Add `ClaudeAdapter` implementing `Adapter`; constructor: `NewClaudeAdapter(apiKey, model string)`.
- `Execute`: calls Anthropic Messages API with the issue title/body as user prompt; returns the response text as `Summary` and `Comment`.
- HTTP client is an interface (`LLMClient`) injected via constructor so tests use `MockLLMClient` (returns canned JSON).
- `MockLLMClient` lives in `claude_adapter_test.go` or `internal/testutil/`.
- Register `"claude_local"` in the adapter registry in `app.go` when `ANTHROPIC_API_KEY` env var is set.
- Unit tests using `MockLLMClient`: success, API error (→ `RunResult.Err`), empty response.

Acceptance: with `ANTHROPIC_API_KEY` set, `paperclip-go heartbeat run --agent $AID` calls Claude; tests pass without a real key (mock).

---

### Phase F — Secrets & Settings

#### F1 — Secrets table + CRUD

**Files:** `internal/store/migrations/0007_secrets.sql`, `internal/domain/secret.go`, `internal/secrets/service.go`, `internal/api/secrets/handler.go`

Tasks:
- Migration: `secrets(id, company_id, name, value_encrypted TEXT, created_at, updated_at)`.  
  `value_encrypted` stores an authenticated-encryption payload (AES-GCM) using a key derived from `config.SecretKey` and a fresh random nonce per secret; store nonce+ciphertext+tag together (for example, base64-encoded). **Do not use XOR or plaintext fallback.** If `config.SecretKey` is not set or invalid, secrets write/update endpoints must fail closed and startup must emit a clear warning that secrets APIs are disabled until a key is configured.
- CRUD: `GET /api/secrets?companyId=`, `POST /api/secrets`, `GET /api/secrets/{id}`, `PATCH /api/secrets/{id}`, `DELETE /api/secrets/{id}`.
- `GET` responses **omit** the value field (return `{"id","name","createdAt"}`); `POST` response returns value once.
- Unit tests: create, list (no values), get (no value), update, delete, 404, encrypt/decrypt round-trip, tampered ciphertext rejection, and missing-key behavior (writes rejected; no plaintext persistence).

Acceptance: `POST /api/secrets -d '{"companyId":"...","name":"OPENAI_KEY","value":"sk-..."}'` → 201; `GET /api/secrets` → list without values; old `/api/secrets` stub replaced.

#### F2 — Instance settings table + API

**Files:** `internal/store/migrations/0008_instance_settings.sql`, `internal/domain/setting.go`, `internal/settings/service.go`, `internal/api/settings/handler.go`

Tasks:
- Migration: `instance_settings(key TEXT PRIMARY KEY, value TEXT, updated_at TEXT)`.
- `GET /api/instance-settings` → map of all settings.
- `PATCH /api/instance-settings` → merge-update settings.
- Seed with defaults at startup: `deployment_mode=local_trusted`, `allowed_origins=localhost`.
- Unit tests: get defaults, patch, get updated.

Acceptance: `GET /api/instance-settings` returns `{"deployment_mode":"local_trusted",...}`.

#### F3 — `env` CLI subcommand

**Files:** `internal/cli/env.go`

Tasks:
- `paperclip-go env list` — calls `GET /api/secrets` and pretty-prints names.
- `paperclip-go env set KEY VALUE --company <id>` — calls `POST /api/secrets`.
- `paperclip-go env get KEY --company <id>` — calls `GET /api/secrets/{id}` (resolve by name first).
- Uses `internal/cli/client.go` (remote HTTP) by default; `--db` flag for direct DB.

Acceptance: `paperclip-go env set FOO bar --company acme` creates secret; `paperclip-go env list --company acme` shows `FOO`.

#### F4 — `db:backup` CLI command

**Files:** `internal/cli/dbbackup.go`

Tasks:
- `paperclip-go db:backup [--out path]` — copies the SQLite file to `<data_dir>/backups/YYYY-MM-DD_HH-MM-SS.db` (or `--out`).
- Uses `VACUUM INTO` SQL for a clean copy while the server may be running.
- Prints the backup path on success.

Acceptance: `paperclip-go db:backup` creates a `.db` file in the backups dir.

---

### Phase G — Approvals & Routines

#### G1 — Approvals table + API + CLI

**Files:** `internal/store/migrations/0009_approvals.sql`, `internal/domain/approval.go`, `internal/approvals/service.go`, `internal/api/approvals/handler.go`, `internal/cli/approval.go`

Tasks:
- Migration: `approvals(id, company_id, agent_id, issue_id, kind, status [pending|approved|rejected], request_body TEXT, response_body TEXT, created_at, resolved_at)`.
- `GET /api/approvals?companyId=`, `POST /api/approvals`, `GET /api/approvals/{id}`, `POST /api/approvals/{id}/approve`, `POST /api/approvals/{id}/reject`.
- CLI: `paperclip-go approval list --company <id>`, `paperclip-go approval get <id>`.
- Replace the existing `/api/approvals` stub with the real handler.
- Unit tests: create approval, list, approve, reject, 409 on double-resolve.

Acceptance: `POST /api/approvals` → 201; `POST /api/approvals/$ID/approve` → `status: "approved"`.

#### G2 — Routines table + API + CLI

**Files:** `internal/store/migrations/0010_routines.sql`, `internal/domain/routine.go`, `internal/routines/service.go`, `internal/api/routines/handler.go`, `internal/cli/routine.go`

Tasks:
- Migration: `routines(id, company_id, agent_id, name, cron_expr TEXT, enabled BOOLEAN DEFAULT 1, last_run_at TEXT, created_at, updated_at)`.
- `GET/POST /api/routines`, `GET/PATCH/DELETE /api/routines/{id}`, `POST /api/routines/{id}/trigger` (immediate run).
- Cron scheduler: at `serve` startup, launch a goroutine that checks due routines every 60 s and fires a heartbeat run for the agent.
- CLI: `paperclip-go routine create --name "daily" --cron "0 9 * * *" --agent $AID`, `paperclip-go routine list --company acme`.
- Replace stub with real handler.
- Unit tests: create, list, trigger, disable. Cron check uses a mock clock.

Acceptance: `POST /api/routines` → 201; `POST /api/routines/$ID/trigger` fires a heartbeat run row.

---

### Phase H — Execution Workspaces & Realtime

> These are the most complex phases. Each may need to be split into sub-agents.

#### H1 — Execution workspaces

**Files:** `internal/store/migrations/0011_workspaces.sql`, `internal/domain/workspace.go`, `internal/workspaces/service.go`, `internal/api/workspaces/handler.go`

Tasks:
- Migration: `execution_workspaces(id, agent_id, issue_id, heartbeat_run_id, status, path TEXT, created_at, updated_at)`.
- CRUD endpoints under `/api/execution-workspaces`.
- Link `heartbeat_runs.workspace_id` to workspaces.
- Unit tests: create, get, list, delete.

Acceptance: `POST /api/execution-workspaces` → 201; heartbeat run can reference a workspace.

#### H2 — WebSocket live events

**Files:** `internal/api/ws/handler.go`, `internal/events/bus.go`

Tasks:
- Add an in-process event bus (`Publish(topic, payload)` / `Subscribe(topic) <-chan Event`).
- Publish events from companies/agents/issues/heartbeat services on create/update/delete.
- `GET /api/ws` upgrades to WebSocket; client subscribes to a `companyId`; server fans out events.
- Use an external WebSocket package (for example, `golang.org/x/net/websocket`) or implement the upgrade manually via plain HTTP hijack.
- Unit tests: publish event → subscriber receives it; disconnect cleans up subscription.

Acceptance: connect to `/api/ws?companyId=$CID`; create an issue via API → WS message arrives.

---

## LLM Mocking Convention

All adapters that call external LLMs **must** accept an interface for the HTTP transport:

```go
// internal/heartbeat/llm_client.go
type LLMClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

Tests inject a `mockLLMClient` that returns a pre-built `*http.Response` from a string fixture:

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

## Upstream TS Sync — Go Integration Plan (2026-04-22)

Upstream sync commit `fc1c27d` brought TS migrations 0057–0064. Analysis of relevant changes:

### HIGH PRIORITY — Affect existing Go tables

#### HI-1: `heartbeat_runs` extended fields (Phase E1-ext)
**Migration:** `internal/store/migrations/0005_heartbeat_runs_ext.sql`
```sql
ALTER TABLE heartbeat_runs ADD COLUMN liveness_state TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN liveness_reason TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN continuation_attempt INTEGER NOT NULL DEFAULT 0;
ALTER TABLE heartbeat_runs ADD COLUMN last_useful_action_at TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN next_action TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN scheduled_retry_at TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN scheduled_retry_attempt INTEGER NOT NULL DEFAULT 0;
ALTER TABLE heartbeat_runs ADD COLUMN scheduled_retry_reason TEXT;
```
**Domain:** Add nullable fields to `domain.HeartbeatRun`. **Complexity: S**

#### HI-2: `issues.origin_fingerprint` (Phase G2-ext)
**Migration:** `internal/store/migrations/0006_issue_origin_fingerprint.sql`
```sql
ALTER TABLE issues ADD COLUMN origin_fingerprint TEXT NOT NULL DEFAULT 'default';
```
Needed for routine-execution dedup index (G2). Expose in domain type; don't add to API response yet. **Complexity: S**

### MEDIUM PRIORITY — New tables with planned Go analogues

#### MED-1: `issue_thread_interactions` (new Phase I1)
New table linking issues ↔ heartbeat_runs ↔ comments for the agent continuation/approval loop.
Columns: `id, company_id, issue_id, kind, status, continuation_policy, idempotency_key, source_comment_id, source_run_id, title, summary, created_by_agent_id, resolved_by_agent_id, payload, result, resolved_at, created_at, updated_at`.
- **New routes:** `POST/GET /api/issues/{id}/interactions`, `POST /api/issues/{id}/interactions/{iid}/resolve`
- **Note:** G1 approvals and I1 interactions overlap conceptually — consider making approvals a thin layer over this table rather than a separate one. Resolve before starting G1.
- **Complexity: M**

### LOW PRIORITY / DEFER

| Item | Upstream migration | Recommendation |
|------|-------------------|----------------|
| `routine_runs.dispatch_fingerprint` | 0062 | Add inline when implementing G2 `routine_runs` |
| `issue_reference_mentions` | 0060 | Defer — no Go handler planned |
| `plugin_database_namespaces` | 0059 | Skip — plugins are an explicit non-goal |
| `join_requests` cleanup | 0057 | Skip — auth/RBAC deferred |

### Recommended Sequencing

| Order | Item | Go migration # | Complexity | Unblocks |
|-------|------|----------------|------------|---------|
| 1 | `heartbeat_runs` ext fields | 0005 | S | E1 run detail/cancel |
| 2 | `issues.origin_fingerprint` | 0006 | S | G2 routines dedup |
| 3 | `issue_thread_interactions` | 0007 | M | agent continuation loop |
| 4 | `routine_runs.dispatch_fingerprint` | inline G2 | — | G2 |

---

## Code Review & Quality Audit (2026-04-24)

**Phases A-C Status:** ✅ Complete. All tests pass (`make test` green).

### Applied Fixes

| Issue | File | Status |
|-------|------|--------|
| Misleading docstring on Delete | `internal/agents/service.go:153` | ✅ Fixed |
| Cross-tenant security assumption | `internal/api/agents/handler.go:124` | 📝 Documented |

### Design Debt (Future Phases)

1. **Cross-tenant isolation at route level** (Medium)
   - Current: Handlers accept agent ID only; tenant validation depends on auth middleware
   - Risk: Low if auth layer enforces `companyId` scoping; medium if auth is not present
   - Recommendation: Add optional `?companyId=` query param to DELETE, PATCH, state-transition endpoints for defensive isolation. Or require company_id in route path (e.g., `/api/companies/{companyId}/agents/{agentId}`)
   - Timeline: Phase D or later (after auth infrastructure is in place)

2. **State machine validation + RBAC** (Medium)
   - Current: `Pause`, `Resume`, `Terminate`, `Update` have no permission guards; any authenticated user can call them
   - Risk: Medium — no permission-based access control; no audit trail for who changed state
   - Recommendation: Add optional role/permission checks in handlers; wrap state transitions with auth context
   - Timeline: Phase F onwards (when auth framework is available)

3. **Activity log reliability** (Low)
   - Current: Activity log errors in `Pause`, `Resume`, `Terminate` are logged but don't fail the operation
   - Risk: Low — state transitions succeed even if audit fails; acceptable trade-off for graceful degradation
   - Recommendation: Add metrics/monitoring for audit log failures; consider circuit-breaker if failures persist
   - Timeline: Phase F (instrumentation & monitoring)

4. **Handler unit test coverage** (Low)
   - Current: Handlers use E2E tests; no isolated handler-level tests for error cases (malformed JSON, oversized bodies, missing params)
   - Recommendation: Add `handler_test.go` per package (agents, companies, issues) covering 400/404/409/422 cases
   - Timeline: Next phase or parallel effort

### TS Parity Verification (Phases A-C)

- ✅ Endpoints: All routes match TS (DELETE, PATCH, POST pause/resume/terminate, configuration merge)
- ✅ Status codes: 204 (delete), 200 (state transitions), 404, 409, 422 per spec
- ✅ Response schemas: Agent includes `runtimeState`, `configuration`; camelCase JSON keys
- ✅ Error handling: Consistent error shapes and HTTP status codes
- ✅ Database: Migrations idempotent, all new columns have defaults, no breaking changes

---

## Deferred (explicit non-goals beyond this plan)

- BetterAuth / RBAC / board-claim flow
- Embedded Postgres
- Plugin host / external adapter processes
- Full Drizzle-schema parity (`goals`, `projects`, `costs`, `budgets`)
- Data sharing with the TS instance

These remain deferred until there is a concrete need.

---

## Review Notes & Quality Debt (2026-04-22)

### Fixed Issues

1. **Security: Cross-Company Label Removal (CVE-like)**
   - **Status**: ✅ FIXED
   - **File**: `internal/labels/service.go`
   - **Issue**: `UnlinkFromIssue()` lacked company validation; attacker with label+issue IDs could unlink labels across companies
   - **Fix**: Added transaction with company match validation mirroring `LinkToIssue()`
   - **Test**: Added `TestUnlinkFromIssueWrongCompany` to prevent regression

2. **Code Quality: Unused Error Handling (3 instances)**
   - **Status**: ✅ FIXED
   - **Files**: `internal/agents/service.go` (Pause/Resume/Terminate methods)
   - **Issue**: `json.Marshal()` errors silently ignored via `_` placeholder
   - **Fix**: Replaced with explicit error returns: `if err != nil { return nil, fmt.Errorf("marshaling: %w", err) }`

3. **Documentation: Implicit FK Cascade**
   - **Status**: ✅ FIXED
   - **File**: `internal/issues/service.go` Delete() method
   - **Issue**: Labels deleted via DB FK cascade but not obvious from code
   - **Fix**: Added explicit comment: `// Labels are cascade-deleted via issue_labels FK constraint`

4. **Error Handling: FK Violation Context**
   - **Status**: ✅ FIXED
   - **File**: `internal/labels/service.go` LinkToIssue()
   - **Issue**: FK violation handler returned generic error; couldn't distinguish "label gone" vs "issue gone"
   - **Fix**: Enhanced to query both entities in transaction and return specific error type

### Design Debt (Non-Critical)

| Item | Impact | Recommendation |
|------|--------|-----------------|
| Missing handler unit tests | Medium | Add `internal/api/{agents,issues}/handler_test.go` covering error cases (404, 409, 422) |
| No config schema validation | Low | Define allowed agent config keys; consider JSON schema in `Update()` |
| Response shape validation | Low | Verify against TS schema; suggest adding `SchemaTest` in E2E |

### Parity Status

✅ **Verified:**
- All response JSON uses camelCase (companyId, createdAt, etc.)
- HTTP status codes align with TS (409 for conflicts, 422 for validation, 404 for missing)
- No missing endpoints in Phases A-C
- Error response shapes consistent

🔲 **Not Checked (defer to Phase C2+):**
- Pagination, filtering on large lists (Documents, Routines)
- Batch operations
- Soft-delete vs hard-delete semantics

### Next Recommended Phases

1. **C2 — Documents/Work-Products** (high value, low risk)
   - Schema: Add `documents` and `work_products` JSON arrays to `issues` table
   - No cross-tenant concerns; tests validate schema round-trip only
   - ~1–2 hours

2. **C3 — Archive/Read State** (enables soft-delete UX)
   - Schema: Add `archived_at`, optionally `last_read_at` to `issues` table
   - Impacts: `GET /api/issues` default filter, GET with `?includeArchived=true`
   - ~1–2 hours

3. **D1 — Activity POST + Issue-Scoped** (unblocks audit trail)
   - Reuse existing `activity_log` table; add POST handler
   - New route: `GET /api/issues/{id}/activity` scoped to that issue
   - ~1 hour

### Testing Notes

- **Current**: All Go tests pass (26 label tests including new regression test)
- **Build**: `make build && make test` ✅ green
- **Gaps**: Handler packages (agents, issues, companies) lack unit tests; only E2E coverage exists
- **Recommendation**: Consider adding `handler_test.go` per package in next phase for 404/409/422 error cases
