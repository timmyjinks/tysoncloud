# TYSONCLOUD frontend

TanStack Router SPA (client-only, Vite) + Clerk auth, replacing the old
SvelteKit/htmx frontend. Talks to the existing Go backend once it's updated
for Clerk (see `notes/backend-todo.md`).

## Design direction

A blend of three references, applied as one consistent system rather than
switched per page:

- **Zed** — restraint, precision, an editor's calm.
- **TigerBeetle** — technical confidence, monospace-as-default, real data
  over decoration.
- **Railway** — the deploy-platform information architecture (project →
  service/database cards, status at a glance).

Tokens live in `src/app.css`:
- Background `#08090b`, surfaces `#111214`/`#17181b`, hairline borders.
- One brand accent, violet `#6e56cf` — used for interactive state only.
- Status colors (green/amber/red) reserved strictly for real infra state
  (running/building/failed), never decorative.
- Type: **JetBrains Mono** for headlines, IDs, ports, domains, env vars —
  **IBM Plex Sans** for paragraph copy.
- Signature motif: the "terminal strip" — a monospace prompt-style header on
  every resource card (`$ svc-web-01`), tying the UI back to the containers
  it manages.

## Getting started

```bash
pnpm install
cp .env.example .env.local   # fill in VITE_CLERK_PUBLISHABLE_KEY, VITE_API_URL
pnpm dev
```

`pnpm dev` / `pnpm build` run the TanStack Router Vite plugin, which
generates `src/routeTree.gen.ts` from the files in `src/routes/` — it's
gitignored and shouldn't be hand-edited.

## Structure

```
src/
  routes/            file-based routes (see implementation plan §5)
  lib/api/           fetch client + typed query/mutation hooks per resource
  components/        navbar, resource-card (terminal-strip), delete dialog
  components/ui/     shadcn-style primitives, restyled to the token system
  app.css            design tokens + global styles
```

## Clerk integration notes

- `sign-in`/`sign-up` are directories, not single files: a `route.tsx`
  layout (owns the auth guard + the `redirect` search param), an
  `index.tsx` (bare `/sign-in`), and a `$.tsx` splat (`/sign-in/*`) that
  catches Clerk's own sub-steps — `factor-one`, `reset-password`,
  `sso-callback`, etc. Clerk reads the URL itself to know which step to
  render, so the splat just needs to mount the same `<SignIn/>`.
- The router only mounts inside `<ClerkLoaded>` (see `main.tsx`). Mounting
  it earlier means `useAuth().isSignedIn` is briefly `false` while Clerk
  is still checking the session, which sends a signed-in user to
  `/sign-in`, which then bounces them right back — a redirect loop.
- `ClerkProvider` gets `routerPush`/`routerReplace` wired to
  `router.navigate` so Clerk's internal navigations use the SPA router
  instead of falling back to full `window.location` reloads.
- If `/dashboard` or `/projects/*` redirect you to sign-in, they pass
  `?redirect=<original path>`; sign-in/sign-up read that back out
  (`safeRedirectTarget` in `lib/safe-redirect.ts`, which also guards
  against an open-redirect via a crafted `redirect` param) and send you
  back there after auth instead of always landing on `/dashboard`.

## Known gaps / next steps

This scaffold covers the SPA itself. Still open, per the implementation plan:

- **Backend**: Clerk auth middleware, service-role Supabase client, CORS
  middleware, `ServiceResponse` field-completeness fix, `/logs` route
  cleanup — none of that is in this repo.
- **`user_id` column type**: if `projects.user_id` (and any RPC `p_user_id`
  params) are still Postgres `uuid`, they need to move to `text` before
  Clerk's string user IDs will write/read correctly. Needs the actual
  schema/RPC SQL to do precisely.
- **Volumes**: `useVolume` treats a 404 from `GET /services/{id}/volumes`
  as "no volume attached" — confirm that's actually what the backend
  returns for a service with none, rather than some other error shape.
- **Websocket build logs**: intentionally deferred (per the plan) — forms
  currently just show a pending/loading state on the submit button via
  React Query's `isPending`, no live log stream.
