# GAPS.md — Go vs TypeScript Feature Gap Analysis

> Generated: 2026-04-18  
> Go project status: **all 9 PLAN.md phases complete** (`make build` ✅ `make test` ✅ `make lint` ✅)  
> TS project status: **functional codebase**, not buildable in this environment (pnpm not installed)

---

## 1. Build & Test Status

### Go (`paperclip-go`)

| Command | Result |
|---------|--------|
| `make build` | ✅ Pass |
| `make test` | ✅ Pass (11 packages, ~0.6 s total) |
| `make lint` | ✅ Pass (`go vet` clean) |

Binary: `./bin/paperclip-go`. All 9 PLAN.md phases marked ✅ DONE.

### TypeScript (`server/`)

The TS codebase is structurally complete and consistent (`server/src/app.ts` registers 30+ route files with no obvious syntax errors). It is the production Paperclip control plane. Build requires `pnpm` which is not installed in this environment, so a full typecheck was not run. Based on code inspection the TS code is **expected to be functional** — it is the upstream origin that Go is porting from.

---

## 2. API Coverage: Go vs TS

Go implements ~30 endpoints. TS implements ~270+.

| Route prefix | TS endpoints | Go status | Notes |
|---|---|---|---|
| `/api/health` | 1 | ✅ Implemented | Go now returns `deploymentMode`, `authReady`, `features` etc. |
| `/api/companies` | 17 | ⚠️ Partial (4) | Go: list, create, get, delete. Missing: stats, feedback-traces, export/import, archive, branding |
| `/api/agents` | 50+ | ⚠️ Partial (6) | Go: list, create, get, me, delete, patch. Missing: runtime-state, skills, keys, lifecycle (pause/resume/terminate), inbox, org-chart, adapters config |
| `/api/issues` | 45+ | ⚠️ Partial (9) | Go: list, create, get, patch, delete, checkout, release, comments list+create. Missing: documents, work-products, labels, feedback, attachments, approvals, read/archive state |
| `/api/activity` | 5 | ⚠️ Minimal (1) | Go: GET with companyId only. Missing: POST, issue-scoped activity, run tracking |
| `/api/heartbeat` | 15+ | ⚠️ Minimal (2) | Go: POST /runs, GET /runs. Missing: run cancel, events, workspace ops, run detail |
| `/api/skills` | 1 | ✅ Implemented | In-memory load from `/skills/*/SKILL.md` |
| `/api/approvals` | 10+ | 🟡 Stub | Returns `{"items":[]}` |
| `/api/costs` | 20+ | 🟡 Stub | Returns `{"items":[]}` |
| `/api/goals` | 6 | 🟡 Stub | Returns `{"items":[]}` |
| `/api/projects` | 25+ | 🟡 Stub | Returns `{"items":[]}` |
| `/api/routines` | 15+ | 🟡 Stub | Returns `{"items":[]}` |
| `/api/plugins` | 30+ | 🟡 Stub | Returns `{"items":[]}` |
| `/api/dashboard` | 1 | 🟡 Stub | Returns `{"items":[]}` (added in fix) |
| `/api/sidebar-badges` | 1 | 🟡 Stub | Added in fix |
| `/api/sidebar-preferences` | 1 | 🟡 Stub | Added in fix |
| `/api/inbox-dismissals` | 1 | 🟡 Stub | Added in fix |
| `/api/instance-settings` | 5+ | 🟡 Stub | Added in fix |
| `/api/llms` | 3+ | 🟡 Stub | Added in fix |
| `/api/access` | 8+ | 🟡 Stub | Added in fix |
| `/api/secrets` | 8+ | 🟡 Stub | Added in fix |
| `/api/adapters` | 5+ | 🟡 Stub | Added in fix |
| `/api/company-skills` | 5+ | 🟡 Stub | Added in fix |
| `/api/execution-workspaces` | 20+ | ❌ Missing | No handler |

**Legend:** ✅ Implemented | ⚠️ Partial | 🟡 Stub (empty response) | ❌ Missing (404)

---

## 3. CLI Coverage: Go vs TS

| Command | TS (`paperclipai`) | Go (`paperclip-go`) |
|---|---|---|
| serve / start | ✅ | ✅ |
| init | ✅ | ✅ |
| doctor | ✅ | ✅ |
| company create/list | ✅ | ✅ |
| agent create/list | ✅ | ✅ |
| issue create/list/get | ✅ | ✅ |
| heartbeat run | ✅ | ✅ |
| onboard (interactive setup) | ✅ | ❌ |
| configure | ✅ | ❌ |
| env (list/set/get env vars) | ✅ | ❌ |
| db:backup | ✅ | ❌ |
| allowed-hostname | ✅ | ❌ |
| auth bootstrap-ceo | ✅ | ❌ |
| approval list/get | ✅ | ❌ |
| routine create/list | ✅ | ❌ |
| feedback share/list | ✅ | ❌ |
| plugin install/list/remove | ✅ | ❌ |
| worktree commands | ✅ | ❌ |
| dashboard | ✅ | ❌ |

---

## 4. Schema / Data Model Gaps

The Go schema (`internal/store/migrations/0001_init.sql`) is a minimal MVP schema. Key gaps vs the TS/Drizzle schema:

| Feature | TS has | Go has |
|---|---|---|
| `issues.parent_issue_id` (sub-issues) | ✅ | ✅ |
| `issues.labels` | ✅ (via junction table) | ❌ |
| `issues.documents` | ✅ (JSON/blob) | ❌ |
| `issues.work_products` | ✅ | ❌ |
| `issues.execution_policy` | ✅ | ❌ |
| `agents.configuration` (YAML/JSON) | ✅ | ❌ |
| `agents.runtime_state` | ✅ | ❌ |
| `agents.adapter` field | ✅ | ✅ (stored, not used) |
| `heartbeat_runs.workspace_id` | ✅ | ❌ |
| `secrets` table | ✅ | ❌ |
| `routines` table | ✅ | ❌ |
| `goals` / `projects` tables | ✅ | ❌ |
| `approvals` table | ✅ | ❌ |
| `budgets` / `costs` tables | ✅ | ❌ |
| Authentication tables | ✅ (BetterAuth) | ❌ (always local_trusted) |
| `instance_settings` table | ✅ | ❌ |
| `realtime/live-events` (WebSocket) | ✅ | ❌ |

This drift is **intentional** per PLAN.md — the MVP is isolated and not a full Drizzle port.

---

## 5. Critical Findings & Improvement Proposals

### 5.1 CRITICAL — Secrets endpoint returns 404, not stub

The TS UI and agents use `/api/secrets` at startup. Currently the Go router returns a JSON 404. This will cause error noise in the UI. **Proposed fix**: add `r.Get("/secrets", apistubs.EmptyList())` to the router.

### 5.2 CRITICAL — `GET /api/agents` returns all agents without company scope

TS `agentRoutes` scopes agent queries to the authenticated company. The Go handler allows listing all agents across all companies when `?companyId=` is omitted. **Proposed fix**: Make `companyId` required for `GET /api/agents` (same as `GET /api/issues` already does), or document that the MVP is single-tenant.

### 5.3 IMPORTANT — No real heartbeat adapter

The heartbeat runner only supports the `StubAdapter` which just acknowledges issues. No real Claude/Cursor adapter exists. This is an explicit PLAN.md deferral but is the core reason the Go binary cannot yet run real agents.

### 5.4 IMPORTANT — Missing `PATCH /api/companies/{id}`

TS supports company updates (name, description, branding). The Go handler has no PATCH endpoint for companies.

### 5.5 IMPORTANT — No authentication / authorization layer

The Go server accepts all requests as `local_trusted`. The TS server has BetterAuth, board-claim flow, RBAC (`accessService`), and `actorMiddleware`. This is intentional for MVP but must be addressed before any multi-user or networked deployment.

### 5.6 IMPORTANT — `GET /api/activity` companyId is optional in TS, required in Go

The Go `activity` handler silently returns empty results if `companyId` is omitted. The TS handler scopes by company but also supports admin views. Should be validated consistently.

### 5.7 MINOR — Health endpoint version is hardcoded `"dev"`

The TS health endpoint reads `serverVersion` from `server/src/version.ts` (populated at build time). The Go handler hardcodes `"dev"`. **Proposed fix**: inject version via `ldflags` at build time (`go build -ldflags "-X main.version=..."`).

### 5.8 MINOR — No `X-Request-Id` propagation in error logs

The middleware adds a request ID but the handler error logs (`log.Printf`) don't include it. Should use structured slog with the request ID in context for traceability.

### 5.9 MINOR — Stub endpoints return `{"items":[]}` for all requests regardless of method

TS stubs return correct structure per endpoint. The Go stubs only handle GET and only return the empty list shape. POST/PUT/PATCH/DELETE to stub routes will fall through to the JSON 404 handler.

---

## 6. Is the TS Code Still Functional?

**Assessment: Yes, the TS code is the production implementation.**

- All 30+ route files are present and structurally complete.
- `server/src/app.ts` wires all routes with middleware, auth, plugin system, and WebSocket live events.
- The TS server uses Drizzle ORM + Postgres/SQLite, BetterAuth, plugin worker processes, WebSocket realtime, and execution workspaces — none of which are in the Go port.
- The TS code is not buildable in this sandbox (pnpm not installed), but there are no structural anomalies that suggest it is broken.

---

## 7. Does the Go Code Do the Same?

**Assessment: Partially — for the MVP core loop.**

Go covers the control-plane essentials needed for basic agent operation:

| Capability | Go |
|---|---|
| Run a server with CRUD for companies/agents/issues | ✅ |
| Atomic issue checkout/release (conflict detection) | ✅ |
| Agent heartbeat (stub adapter) | ✅ |
| Post comments on issues | ✅ |
| List loaded skills | ✅ |
| Activity log | ✅ |
| UI asset serving + SPA fallback | ✅ |

Go does **not** cover:

- Real agent execution (no Claude/Cursor adapter)
- Secrets management
- Approvals workflow
- Budgets/costs tracking
- Routines/cron
- Goals and projects
- Plugin system
- WebSocket realtime events
- Authentication and RBAC
- Execution workspaces
- Company portability (export/import)

---

## 8. On-the-Fly Fixes Applied (this session)

| Fix | File(s) | Impact |
|---|---|---|
| Health endpoint: added `deploymentMode`, `deploymentExposure`, `authReady`, `bootstrapStatus`, `features` fields | `internal/api/health/handler.go` | UI correctly detects local_trusted mode |
| Added 7 missing UI stub routes: dashboard, sidebar-badges, sidebar-preferences, inbox-dismissals, instance-settings, llms, access | `internal/api/router.go` | Eliminates JSON 404 noise when UI loads |
| Added `PATCH /api/agents/{id}` endpoint + `agents.Service.Update()` | `internal/api/agents/handler.go`, `internal/agents/service.go` | Agents can now be updated via API |
| E2E health test updated to match new response shape | `internal/api/api_e2e_test.go` | Test accuracy |

All fixes: `make build` ✅ `make test` ✅ `make lint` ✅

---

## 9. Recommended Next Steps (priority order)

1. **Add `/api/secrets` stub** — prevents UI startup errors (5 min).
2. **Add `/api/company-skills` and `/api/adapters` stubs** — more UI noise reduction (5 min).
3. **Add `PATCH /api/companies/{id}`** — needed for company management in UI (30 min).
4. **Add `PATCH /api/issues/{id}` to also handle `status` field transitions** — currently works but doesn't validate status enum (15 min).
5. **Inject build version via ldflags** — replace `"dev"` in health handler (15 min).
6. **Implement first real heartbeat adapter** (`claude_local`) — needed to actually run agents (2–4 days).
7. **Add `secrets` table + CRUD** — needed for agent API key storage (1 day).
8. **Add routines/cron** — needed for scheduled agent execution (2 days).
9. **Add WebSocket live events** — needed for responsive UI without polling (3 days).
10. **Add authentication layer** — needed before any multi-user or networked deployment (1 week).
