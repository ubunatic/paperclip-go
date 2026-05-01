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

## Who is this for?

A single developer running Paperclip locally or in a VM — no Node.js, no npm, one self-contained Go binary.

The assumption is a **trusted single-user environment**: no authentication, no multi-tenancy enforcement, no cloud deployment. If community interest grows, auth and multi-user features can be added later.

## Quick start

```sh
make build
./bin/paperclip-go serve
```

See [PLAN.md](PLAN.md) for implementation status and next steps.
