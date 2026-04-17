# Internal Packages

This directory contains all non-public Go packages for the paperclip-go control plane.

## Package Guide

| Package | Purpose |
|---------|---------|
| `activity` | Activity logging service for tracking agent/company actions |
| `agents` | Agent entity service with lifecycle management |
| `api` | HTTP API router, middleware, and endpoint handlers |
| `api/activity` | HTTP endpoint for activity logs |
| `api/agents` | HTTP endpoint for agent CRUD operations |
| `api/companies` | HTTP endpoint for company CRUD operations |
| `api/health` | HTTP health check endpoint |
| `api/issues` | HTTP endpoint for issue management and comments |
| `api/skills` | HTTP endpoint for listing loaded skills |
| `cli` | Cobra CLI commands (for future expansion) |
| `comments` | Comments service for managing issue comments |
| `companies` | Company entity service |
| `config` | Configuration loading and environment variable handling |
| `domain` | Pure data types: Agent, Skill, Issue, Company, Comment, Activity |
| `ids` | UUID generation utilities |
| `issues` | Issues service with checkout/release workflow |
| `respond` | HTTP response helpers (JSON serialization) |
| `skills` | Skills loader: parses SKILL.md files from filesystem |
| `store` | SQLite store with transaction support |
| `testutil` | Test utilities and server spawning |

## Architecture Notes

- **Domain types** are immutable data structures in `domain/`
- **Services** implement business logic, stored in their own packages (e.g., `agents/`, `issues/`)
- **API handlers** are in `api/` subdirectories and return responses via the `respond` helper
- **Skills loader** walks the filesystem looking for `SKILL.md` files with YAML frontmatter
- **Store** provides database access via a transactional interface
