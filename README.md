> [!WARNING]
> **This is an experimental fork of [paperclipai/paperclip](https://github.com/paperclipai/paperclip).**
>
> Goals:
> 1. Get rid of npm/TS/node.js
> 2. Use Go std. lib where possible to have minimal dependencies
> 3. See how easy it is to let an agent refactor the project
>
> This fork is a work in progress. Expect breaking changes, incomplete features, and rough edges.
> For the original project, see [README.orig.md](README.orig.md) or the upstream repo.

## Quick start

```sh
# Build the binary
make build

# Initialize config + data directory (~/.paperclip-go/)
./bin/paperclip-go init

# Check your setup
./bin/paperclip-go doctor

# Start the server (listens on 127.0.0.1:3200)
./bin/paperclip-go serve
```

## What works today

The Go binary covers the first four phases of the [implementation plan](PLAN.md):

### HTTP API (server running on `:3200`)

```sh
# Health
curl localhost:3200/api/health

# Companies
curl -s -XPOST localhost:3200/api/companies \
     -H 'content-type: application/json' \
     -d '{"name":"Acme","shortname":"acme"}'
curl -s localhost:3200/api/companies
curl -s localhost:3200/api/companies/<id>

# Agents
curl -s -XPOST localhost:3200/api/agents \
     -H 'content-type: application/json' \
     -d '{"companyId":"<cid>","shortname":"ceo","displayName":"CEO","role":"CEO"}'
curl -s 'localhost:3200/api/agents?companyId=<cid>'
curl -s localhost:3200/api/agents/<id>
curl -s localhost:3200/api/agents/me -H 'X-Agent-Id: <id>'

# Activity log
curl -s 'localhost:3200/api/activity?companyId=<cid>'
```

### CLI (in-process, no server needed)

```sh
./bin/paperclip-go company create --name "Acme" --shortname acme
./bin/paperclip-go company list

./bin/paperclip-go agent create --company acme --shortname ceo \
    --display-name "CEO" --role CEO
./bin/paperclip-go agent list --company acme
./bin/paperclip-go agent list          # all agents across companies
```

### Not yet implemented

Issues, comments, issue checkout/release, heartbeat runs, skills loader, UI
serving, and stub endpoints for approvals/costs/goals/plugins are still
in progress (Phases 5–8 of the plan).

## Development

```sh
make build   # compile to bin/paperclip-go
make test    # go test ./...
make lint    # go vet ./...
```
