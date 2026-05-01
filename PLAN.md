# Paperclip-Go — Feature Parity Plan

> Previous MVP plan archived in `PLAN.archive.md`.  
> All 9 MVP phases + A1–E2 are ✅ DONE. This plan covers what remains.

---

## Status (2026-05-01)

**Completed:** A1–A4, B1–B2, C1–C3, D1, E1, E2  
**Next:** E3 — `claude_local` heartbeat adapter  
**Build:** ✅ green (all 26 test packages, 17 heartbeat tests)  
**Latest migration:** `0007_activity_rename_kind_to_type.sql`

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
| `/api/secrets` CRUD | 8+ | 🔲 | F1 |
| `/api/instance-settings` CRUD | 5+ | 🔲 | F2 |
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
| `issues.archived_at` | ✅ | ✅ | C3 |
| `agents.configuration` (YAML/JSON) | ✅ | ✅ | B2 |
| `agents.runtime_state` | ✅ | ✅ | B1 |
| `issues.execution_policy` | ✅ | 🔲 | C2+ |
| `heartbeat_runs` extended fields | ✅ | 🔲 | E4 |
| `issues.origin_fingerprint` | ✅ | 🔲 | E5 |
| `secrets` table | ✅ | 🔲 | F1 |
| `instance_settings` table | ✅ | 🔲 | F2 |
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
| `claude_local` adapter | ✅ | 🔲 | E3 |
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

#### E3 — `claude_local` heartbeat adapter

**Files:** `internal/heartbeat/claude_adapter.go`, `internal/heartbeat/llm_client.go`, `internal/heartbeat/claude_adapter_test.go`

Tasks:
- Add `LLMClient` interface (`Do(req *http.Request) (*http.Response, error)`) in `internal/heartbeat/llm_client.go`.
- Add `ClaudeAdapter` implementing `Adapter`; constructor: `NewClaudeAdapter(apiKey, model string, client LLMClient)`.
- `Run()`: calls Anthropic Messages API with the issue title/body as user prompt; returns response text as `Summary`.
- Register `"claude_local"` in the adapter registry in `app.go` when `ANTHROPIC_API_KEY` env var is set.
- Unit tests using `mockLLMClient` (defined in `_test.go`): success, API error (→ `RunResult` with error status), empty response.

Acceptance: with `ANTHROPIC_API_KEY` set, heartbeat calls Claude; `go test ./internal/heartbeat/...` passes without a real key.

#### E4 — `heartbeat_runs` extended fields (upstream sync HI-1)

**Files:** `internal/store/migrations/0008_heartbeat_runs_ext.sql`, `internal/domain/heartbeat.go`

Tasks:
- Migration (all nullable/defaulted):
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
- Add nullable fields to `domain.HeartbeatRun`; update `scanHeartbeatRun()`.
- Existing tests must stay green (no API changes needed yet).

Acceptance: `make test` ✅; GET run response includes new fields (null by default).

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
- Migration: `secrets(id, company_id, name, value_encrypted TEXT, created_at, updated_at)`.  
  Store AES-GCM ciphertext (nonce + ciphertext + tag, base64-encoded) keyed from `config.SecretKey`.  
  **Do not use XOR or plaintext fallback.** If `config.SecretKey` is missing, write endpoints fail closed; startup emits a warning.
- CRUD: `GET /api/secrets?companyId=`, `POST /api/secrets`, `GET /api/secrets/{id}`, `PATCH /api/secrets/{id}`, `DELETE /api/secrets/{id}`.
- `GET` responses omit the value field (`{"id","name","createdAt"}`); `POST` returns value once.
- Unit tests: create, list (no values), get (no value), update, delete, 404, encrypt/decrypt round-trip, tampered ciphertext rejection, missing-key rejection.

Acceptance: `POST /api/secrets` → 201 with value; `GET /api/secrets` → list without values.

#### F2 — Instance settings table + API

**Files:** `internal/store/migrations/0011_instance_settings.sql`, `internal/domain/setting.go`, `internal/settings/service.go`, `internal/api/settings/handler.go`

Tasks:
- Migration: `instance_settings(key TEXT PRIMARY KEY, value TEXT, updated_at TEXT)`.
- `GET /api/instance-settings` → map of all settings.
- `PATCH /api/instance-settings` → merge-update settings.
- Seed defaults at startup: `deployment_mode=local_trusted`, `allowed_origins=localhost`.
- Unit tests: get defaults, patch, get updated.

Acceptance: `GET /api/instance-settings` returns `{"deployment_mode":"local_trusted",...}`.

#### F3 — `env` CLI subcommand

**Files:** `internal/cli/env.go`

Tasks:
- `paperclip-go env list --company <id>` — calls `GET /api/secrets`, pretty-prints names.
- `paperclip-go env set KEY VALUE --company <id>` — calls `POST /api/secrets`.
- `paperclip-go env get KEY --company <id>` — resolves by name, calls `GET /api/secrets/{id}`.
- Uses `internal/cli/client.go` (remote HTTP) by default; `--db` flag for direct DB access.

Acceptance: `paperclip-go env set FOO bar --company acme` creates secret; `paperclip-go env list` shows `FOO`.

#### F4 — `db:backup` CLI command

**Files:** `internal/cli/dbbackup.go`

Tasks:
- `paperclip-go db:backup [--out path]` — copies SQLite file to `<data_dir>/backups/YYYY-MM-DD_HH-MM-SS.db`.
- Uses `VACUUM INTO` SQL for a clean online copy.
- Prints the backup path on success.

Acceptance: `paperclip-go db:backup` creates a `.db` file in the backups dir.

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

| Item | Severity | Location | Effort |
|------|----------|----------|--------|
| Structured logging | LOW-MED | `internal/api/{activity,issues,agents}/handler.go` | 20 min |
| Unbounded `ListByEntity()` pagination | MEDIUM | `internal/activity/log.go` | 15 min |
| `MaxBytesReader` boilerplate (8 sites) | LOW | `internal/api/*/handler.go` | 20 min |
| Response wrapping inconsistency | LOW | GET returns `{items}`, POST returns raw object | 30 min |
| Handler unit tests missing | MEDIUM | agents, issues, companies packages | 1–2 h |
| Cross-tenant isolation at route level | MEDIUM | DELETE/PATCH/state endpoints | Phase F+ |
| State machine RBAC | MEDIUM | pause/resume/terminate handlers | Phase F+ |

---

## Deferred (explicit non-goals)

- BetterAuth / RBAC / board-claim flow
- Embedded Postgres
- Plugin host / external adapter processes
- Full Drizzle-schema parity (`goals`, `projects`, `costs`, `budgets`)
- Data sharing with the TS instance
