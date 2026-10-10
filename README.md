# TYSONCLOUD

> Push a Docker image, get a running service with a public domain + TLS, managed Postgres, and persistent volumes without managing servers.

TYSONCLOUD is a self-hosted deploy platform inspired by Railway and Render. Create a project, deploy containerized services and CloudNativePG-backed databases, attach volumes, and expose custom domains all through a single API and dashboard.

Live: **https://tysoncloud.tysonjenkins.dev**

---

## Features

- **Projects & Services** - Group deployments into projects; each service is a Kubernetes `Deployment` + `Service` driven by a user-provided Docker image.
- **Managed Postgres** - One-click databases via [CloudNativePG](https://cloudnative-pg.io/), scoped to a project.
- **Persistent Volumes** - Attach storage to services.
- **Custom Domains** - Automatic TLS via [cert-manager](https://cert-manager.io/) + [NGINX Gateway Fabric](https://github.com/nginx/nginx-gateway-fabric) / Gateway API; Cloudflare DNS integration.
- **Auth** - [Clerk](https://clerk.com/) on both frontend and backend (header-based auth via `clerk-sdk-go`).
- **WebSocket build logs** - Stream deploy logs to the dashboard.

---

## Architecture

```
                ┌─────────────┐
                │   Frontend  │  Vite + React + TanStack Router + Clerk
                │  (port 3000)│
                └──────┬──────┘
                       │  VITE_API_URL
                       ▼
                ┌─────────────┐      ┌──────────┐      ┌──────────────┐
                │   Backend   │─────▶│ Supabase │      │  Kubernetes  │
                │ (Go / :8080)│      │ (Postgres│      │  (CNPG, GW   │
                │  gorilla/mux│◀─────│  + PostgREST)   │   API, cert- │
                └─────────────┘      └──────────┘      │   manager)   │
                       │                               └──────────────┘
                       └────────── Cloudflare (DNS) ──────────┘
```

**Repo layout:**

```
.
├── backend/            Go API (cmd/deploy, server/, store/, kubernetes/, deploy/, db/, config/)
│   ├── cmd/deploy/     Entrypoint + Dockerfile (distroless)
│   ├── server/         HTTP handlers, routing, Clerk + CORS middleware
│   ├── store/          Supabase/PostgREST persistence layer
│   └── kubernetes/     k8s client deployments, services, Gateway API routes
├── frontend/           TanStack Router SPA (Vite, React 19, Tailwind CSS 4, Clerk)
│   └── src/routes/     File-based routes → src/routeTree.gen.ts (generated)
├── infra/
│   ├── k8s/            Cluster manifests (CNPG, MetalLB, NGINX Gateway, cert-manager, tc-system)
│   └── caddy/          Caddy reverse-proxy config
├── docker-compose.yaml Local dev stack (backend + frontend)
└── .github/workflows/  CI build & push images to GHCR
```

---

## Tech Stack

| Layer    | Stack |
|----------|-------|
| Frontend | React 19, TanStack Router, TanStack Query, Vite 6, Tailwind CSS 4, Clerk React |
| Backend  | Go 1.26, gorilla/mux, gorilla/websocket, clerk-sdk-go, Supabase (PostgREST), client-go / Gateway API, Cloudflare Go SDK |
| Infra    | Kubernetes, CloudNativePG, MetalLB, NGINX Gateway Fabric, cert-manager, Caddy, GHCR, GitHub Actions |

---

## Prerequisites

- Go 1.26+
- Node.js 18+ with [pnpm](https://pnpm.io/) (or npm)
- Docker & Docker Compose (for local stack)
- A Supabase project (Postgres + PostgREST)
- A Clerk application (publishable + secret key)
- A Kubernetes cluster + `~/.kube/config` (for full deploy; not required for API-only local dev)
- Cloudflare account (for custom domains / DNS)

---

## Quick Start

### 1. Clone

```bash
git clone https://github.com/timmyjinks/tysoncloud.git
cd tysoncloud
```

### 2. Environment

Create a root `.env` (used by `docker-compose.yaml` and the backend):

```bash
# Backend / shared
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_API_KEY=<supabase-service-role-key>
CLERK_API_KEY=sk_test_...

# Frontend (also create frontend/.env.local for local Vite dev)
VITE_CLERK_PUBLISHABLE_KEY=pk_test_...
VITE_API_URL=http://localhost:8080
```

See `frontend/.env.example` for the frontend variables.

Backend config defaults (`backend/config/config.go:30`): `ADDR=:8080`, `CLUSTER_IP=192.168.0.18`, `KUBECONFIG=~/.kube/config`. `ALLOWED_ORIGINS` defaults to the production frontends only; set `ALLOWED_ORIGINS=http://localhost:3000` for local dev (Docker Compose already does).

### 3. Run with Docker Compose

```bash
docker compose up --build
```

- Frontend: http://localhost:3000
- Backend: http://localhost:8080

### 4. Run services individually

**Backend:**

```bash
cd backend
go mod download
go run ./cmd/deploy
```

**Frontend:**

```bash
cd frontend
pnpm install
cp .env.example .env.local   # fill in VITE_CLERK_PUBLISHABLE_KEY, VITE_API_URL
pnpm dev                     # http://localhost:3000
pnpm build && pnpm preview   # production preview
```

`pnpm dev` / `pnpm build` generate `src/routeTree.gen.ts` from `src/routes/` via the TanStack Router Vite plugin do not edit it by hand.

---

## API Overview

All routes except `GET /projects/{id}/services/{id}/logs` require a `Authorization: Bearer <Clerk JWT>` header (enforced by `clerk-sdk-go` in `backend/server/route.go:10`).

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/projects` | List / create projects |
| GET/PUT/DELETE | `/projects/{project_id}` | Read / update / delete project |
| POST | `/projects/{project_id}/config` | Project config |
| GET/POST | `/projects/{project_id}/services` | List / create services |
| GET/PUT/DELETE | `/projects/{project_id}/services/{service_id}` | Service CRUD |
| GET | `/services/{service_id}` | Get service |
| DELETE | `/projects/{project_id}/services` | Bulk delete services |
| GET | `/projects/{project_id}/services/{service_id}/logs` | WebSocket build logs |
| GET | `/services/{service_id}/volumes` | Get volume |
| POST/DELETE | `/projects/{project_id}/services/{service_id}/volumes` | Create / delete volume |
| GET/POST | `/projects/{project_id}/databases` | List / create databases |
| GET/PUT/DELETE | `/projects/{project_id}/databases/{database_id}` | Database CRUD |
| GET | `/databases/{database_id}` | Get database |
| DELETE | `/projects/{project_id}/databases` | Bulk delete databases |

---

## Deployment

CI is defined in `.github/workflows/docker.yaml` on pushes to `main` (and manual dispatch) it builds and pushes multi-arch images to `ghcr.io/timmyjinks/tysoncloud-backend` and `ghcr.io/timmyjinks/tysoncloud-frontend` (tags via `docker/metadata-action`).

Kubernetes manifests live in `infra/k8s/` (per-component overlays for `cert-manager`, `cnpg-system`, `metallb-system`, `nginx-gateway`, `tc-system`, and cluster definitions in `infra/k8s/clusters/`).

---

## Contributing

Issues and pull requests are welcome. For local development, please run `pnpm lint` (frontend) and `go vet ./...` (backend) before submitting.

---

## License

No license file is currently present. All rights reserved unless a license is added.
