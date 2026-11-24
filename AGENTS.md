# AGENTS.md

Guidance for AI coding agents working in this repo. See `README.md` for the product overview, architecture diagram and API table.

## What this is

TYSONCLOUD is a self-hosted deploy platform. A user pushes a Docker image or connects a GitHub repo and gets a running Kubernetes service with a public domain and TLS, managed Postgres (CloudNativePG) and persistent volumes.

## Layout

| Path | What lives there |
|------|------------------|
| `backend/` | Go API (module `github.com/timmyjinks/tysoncloud`) |
| `backend/cmd/deploy/` | Entrypoint and distroless Dockerfile |
| `backend/server/` | HTTP handlers, routing (`route.go`), Clerk auth, authz, CORS, rate limiting, websockets, GitHub webhooks |
| `backend/store/` | Supabase/PostgREST persistence, one file per table |
| `backend/kubernetes/` | client-go wrappers: deployments, services, HTTPRoutes, quotas, network policy, volumes, CNPG databases |
| `backend/deploy/` | Orchestrates store + kubernetes to deploy projects, services, databases and volumes |
| `backend/github/` | GitHub App integration and image builds (BuildKit) |
| `backend/config/` | Env-var config and defaults |
| `frontend/` | TanStack Start/Router SPA (React 19, Vite, Tailwind 4, TanStack Query, Clerk) |
| `frontend/src/routes/` | File-based routes; `src/routeTree.gen.ts` is generated, never edit it by hand |
| `frontend/src/lib/api/` | Backend API client |
| `supabase/migrations/` | SQL migrations, the source of truth for the database schema |
| `infra/k8s/` | Cluster manifests (cert-manager, CNPG, MetalLB, NGINX Gateway Fabric, `tc-system`) |
| `infra/caddy/` | Caddy reverse-proxy config |

## Commands

Backend (run from `backend/`):

```bash
go vet ./...
go test -race -count=1 ./...   # same as CI
go run ./cmd/deploy            # API on :8080
```

Frontend (run from `frontend/`, pnpm only):

```bash
pnpm install
pnpm lint
pnpm build    # tsc -b && vite build
pnpm dev      # http://localhost:3000
```

Full local stack, including BuildKit and a local registry: `docker compose up --build` from the repo root. It needs a root `.env` (see `README.md`).

Migrations: `supabase db start` applies everything in `supabase/migrations/` to a fresh database. CI does the same, so a new migration has to apply cleanly from scratch.

## Conventions

- **Worktrees:** the repo is checked out as `main/` plus one sibling worktree per change. Never edit `main/` directly. Make each change on its own branch and worktree, cut from the latest `origin/main`.
- **Stacked PRs:** use the `gh stack` extension for stacks. Use one worktree per stack, not one per branch.
- **Commits:** conventional commits (`feat:`, `fix:`, `refactor:`, `chore:`, `fix(security):` and so on). Commit, push or open PRs only when asked.
- **Image tags:** `chore: update image tags ... [skip ci]` commits are made by CI. Don't bump image tags in `infra/` by hand.
- **CI:** keep workflows simple, with a plain `on: push` trigger and no path or branch filters.
- **Auth:** every route except the logs websocket goes through Clerk header auth. Project-scoped routes must also go through `RequireProjectOwner` (`backend/server/authz.go`). Handlers rely on that check before they touch the Kubernetes namespace named by `{project_id}`.
- **Schema changes:** add a new timestamped file in `supabase/migrations/`, never edit an existing one. Update the matching struct in `backend/store/` to match.
- **Domains:** the agreed direction is a `domains` table, with each service table referencing it through a plain `domain_id` foreign key. Today the schema still has `public_domains`.
- **Errors:** return errors through `writeError` (`backend/server/errors.go`) with a public message plus the internal error. The internal error is logged, and on 5xx a cause from `extractCause` is added to the message, so don't wrap secrets or internal details into errors that can reach a 5xx.
- **Tests:** put Go tests next to the code they cover (`*_test.go`). Run the race-enabled test command before you call backend work done.
