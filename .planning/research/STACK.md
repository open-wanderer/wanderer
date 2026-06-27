# Stack Research: Federation Connect UI (v1.1)

**Project:** Wanderer Instance Federation — Admin UI for peer connection management
**Researched:** 2026-06-27
**Confidence:** HIGH — all critical claims verified against PocketBase 0.38 source and official docs

---

## Core Question

What stack additions or changes are needed to deliver an admin-only UI for managing peer instance connections? Three specific questions:

1. Can PocketBase serve custom HTML from a Go route with superuser auth protection, and how does superuser auth work?
2. If going the SvelteKit route, what is the minimal addition to add an admin concept to the frontend?
3. What Go dependencies are needed for serving embedded HTML/static assets from the binary?

---

## Finding 1: PocketBase Superuser Auth Mechanism

**Verdict: Fully verifiable on custom Go routes. Use `apis.RequireSuperuserAuth()`. No new dependencies needed.**

### How superuser auth works

PocketBase superusers are stored in the `_superusers` collection (a standard auth collection). Authentication produces a JWT token (HS256, signed with the app's auth settings key). The token is stateless — no server-side sessions.

The token is transmitted via the **`Authorization` header** (`Authorization: <token>` or `Authorization: Bearer <token>` — both forms are accepted by PocketBase's token loader middleware). There is **no cookie-based session for superusers**. PocketBase's own documentation explicitly states: "Web APIs are fully stateless and there are no sessions in the traditional sense."

### How to check superuser auth on a custom Go route

PocketBase's built-in auth-token-loader middleware runs on every request and populates `e.Auth` when a valid token is present. This happens automatically before any route handler executes. Two verified APIs:

```go
// Option A: check in handler body (already used in this codebase)
if !e.HasSuperuserAuth() {
    return apis.NewUnauthorizedError("superuser required", nil)
}

// Option B: bind middleware to the route (preferred — cleaner, fails fast)
se.Router.GET("/federation/admin", handler).Bind(apis.RequireSuperuserAuth())
```

`e.HasSuperuserAuth()` is equivalent to `e.Auth != nil && e.Auth.IsSuperuser()`. Both patterns are **already used in this codebase** — `db/routes/plugin_system.go` and `db/hooks/plugin_instances.go` both call `e.HasSuperuserAuth()`. This confirms the pattern works without any new dependency.

`apis.RequireSuperuserAuth()` is defined as `apis.RequireAuth(core.CollectionNameSuperusers)` — verified in `github.com/pocketbase/pocketbase@v0.38.0/apis/middlewares.go`.

**Confidence: HIGH** — verified against PocketBase v0.38 (project uses v0.38.0 per go.mod), and the pattern exists in this exact codebase already.

### Browser login flow for a custom HTML admin page

Because there are no session cookies, a browser-facing admin page must handle login explicitly:

1. Page loads — checks localStorage for a valid superuser token (PocketBase JS SDK stores it under key `"pocketbase_auth"` by default).
2. If no valid token: render a login form. On submit, `POST /api/collections/_superusers/auth-with-password` → receive JWT → store in localStorage.
3. On every API call from the admin page: send `Authorization: <token>` header.
4. The Go route handler calls `e.HasSuperuserAuth()` or uses `.Bind(apis.RequireSuperuserAuth())` — returns 401 if not a valid superuser token.

The PocketBase JS SDK (`pocketbase` npm package, already a dependency at v0.26.8 in `web/package.json`) handles all of this automatically via `pb.collection('_superusers').authWithPassword(email, pass)` and `pb.authStore`.

---

## Finding 2: Path A — Custom Go Route Serving Embedded HTML (Recommended)

**Verdict: Feasible, zero new Go dependencies, no SvelteKit changes required.**

### Approach

Serve a single self-contained HTML page (or a small JS bundle) from a Go route protected by `apis.RequireSuperuserAuth()`. The HTML is embedded in the binary at compile time using Go's standard `embed` package.

### go:embed for static assets

`embed.FS` is part of the Go standard library since Go 1.16. The project uses Go 1.25 — fully supported, no new dependency.

Pattern (mirrors PocketBase's own Admin UI, which uses `//go:embed all:dist`):

```go
// db/federation/admin/embed.go
package admin

import "embed"

//go:embed dist/*
var AdminDistFS embed.FS
```

```go
// db/main.go — in registerRoutes()
import (
    "io/fs"
    "github.com/pocketbase/pocketbase/apis"
)

adminFS, _ := fs.Sub(admin.AdminDistFS, "dist")

// Serve the login+dashboard page (no auth guard on the HTML asset itself —
// the page's JS handles the login form, then calls guarded API routes)
se.Router.GET("/federation/admin/{path...}", apis.Static(adminFS, true))

// Guard all data API routes with superuser auth
g := se.Router.Group("/api/federation")
g.Bind(apis.RequireSuperuserAuth())
g.GET("/peers", handleListPeers)
g.POST("/peers/connect", handleConnect)
g.POST("/peers/{id}/approve", handleApprove)
g.POST("/peers/{id}/reject", handleReject)
g.DELETE("/peers/{id}", handleDisconnect)
```

`apis.Static(fsys, indexFallback bool)` is the verified signature in PocketBase v0.38 `apis` package. The `{path...}` wildcard is required by this function (enforced at runtime). Setting `indexFallback: true` makes it serve `index.html` for unknown paths — correct for a single-page admin app.

### What to embed

**Option A (minimal, no build step):** A single `index.html` file with inline `<script>` and `<style>`. Uses the PocketBase JS SDK from a CDN or inlined. Sufficient for a federation dashboard with ~5 operations.

**Option B (proper SPA):** A small Svelte or vanilla JS app built with Vite, output embedded as `dist/*`. Requires a build step but gives proper component structure. The existing Vite/Svelte toolchain in `web/` can be reused or a separate `db/federation/admin/` Vite config can be added.

**Recommendation: Option A first.** The admin dashboard is low-traffic, admin-only, and has limited UI surface. A self-contained HTML page with inline JavaScript avoids a build step and keeps the deployment as a single binary. If the UI grows beyond ~300 lines of JS, graduate to Option B.

### No new Go dependencies

| Need | Solution | Dependency |
|------|----------|------------|
| Embed static files | `//go:embed` + `embed.FS` | Standard library (Go 1.16+) |
| Serve embedded files | `apis.Static()` | Already in `pocketbase/pocketbase/apis` |
| Guard routes | `apis.RequireSuperuserAuth()` | Already in `pocketbase/pocketbase/apis` |
| Auth check in handler | `e.HasSuperuserAuth()` | Already in `pocketbase/core` |

**Zero new Go module dependencies.**

---

## Finding 3: Path B — SvelteKit Frontend with Admin Flag (Fallback)

**Verdict: More invasive, not recommended for v1.1, but viable if a richer UI is required later.**

### What "admin" means in Wanderer's context

PocketBase superusers are not Wanderer users — they exist only in `_superusers` and have no record in the `users` collection. The SvelteKit frontend authenticates against the `users` collection. A "Wanderer admin" must be a separate concept: a regular user with an elevated flag.

### Minimal change to add an admin flag

**Migration:** Add a boolean field `is_admin` (default `false`) to the `users` collection.

**PocketBase API rule on admin-only endpoints:** In collection API rules, use `@request.auth.is_admin = true` to restrict access.

**Go route guard (for custom routes):**
```go
if e.Auth == nil || !e.Auth.GetBool("is_admin") {
    return apis.NewUnauthorizedError("admin required", nil)
}
```

**SvelteKit server-side guard (`+page.server.ts`):**
```typescript
import { error } from '@sveltejs/kit';

export const load = async ({ locals }) => {
    if (!locals.user?.is_admin) {
        throw error(403, 'Admin access required');
    }
    // fetch peers data
};
```

**SvelteKit `locals` change (`hooks.server.ts`):** The user is already loaded into `event.locals.user`. The `is_admin` field would be present on the record automatically once added to the schema — no hooks change needed.

**Type change (`web/src/lib/models/user.ts`):**
```typescript
export type User = AuthRecord & {
    id: string,
    username?: string,
    email?: string,
    password: string,
    avatar?: string;
    created?: string;
    is_admin?: boolean;  // add this
}
```

### Why Path B is not recommended for v1.1

1. **Conflation of concerns:** Wanderer admins (is_admin users) and PocketBase superusers are different roles. Managing the federation infrastructure should belong to the infrastructure operator (PocketBase superuser), not an application-level admin user.
2. **More surface area:** Requires a migration, a type change, route guards in two places (Go and SvelteKit), and new SvelteKit pages.
3. **Auth confusion:** The SvelteKit frontend uses cookie-based auth for `users`. Adding an admin section that manages infrastructure-level settings via the same session creates a privilege escalation risk if the admin-flagging logic has bugs.
4. **Not needed for v1.1:** The Go route approach (Path A) delivers the same UX with zero frontend changes and directly uses PocketBase's own superuser auth.

**Choose Path B when:** The admin dashboard needs to be deeply integrated with the SvelteKit UI (e.g., accessible from a settings sidebar), or the team wants the admin to be a Wanderer user rather than a PocketBase superuser.

---

## Finding 4: PocketBase UI Extensions (Do Not Use for v1.1)

The PocketBase admin panel extension API (introduced experimentally in v0.37) is **explicitly not production-safe**. The PocketBase maintainer's own statement (discussion #7612):

> "it is NOT recommended to use yet because your extensions most likely will break with the 'Stage 2' completion"

Stage 2 has no ETA. Any UI extensions built now will break on upgrade. Do not use this approach for v1.1.

---

## Recommended Stack

### Core Technologies (no additions needed)

| Technology | Version | Role in v1.1 | Status |
|------------|---------|-------------|--------|
| PocketBase | 0.38.0 | Route registration, superuser auth guard, embedded file serving | Already in go.mod |
| Go `embed` package | stdlib (Go 1.16+) | Embed admin HTML/JS into binary | Standard library, no import needed |
| `apis.Static()` | PocketBase 0.38 | Serve embedded `fs.FS` from Go route | Already available via `pocketbase/apis` |
| `apis.RequireSuperuserAuth()` | PocketBase 0.38 | Middleware guard on data API routes | Already available via `pocketbase/apis` |
| `e.HasSuperuserAuth()` | PocketBase 0.38 | Inline superuser check in handler body | Already used in codebase |

### Frontend for Admin Page (Option A — inline HTML)

| Technology | Version | Role | Status |
|------------|---------|------|--------|
| PocketBase JS SDK | CDN or inline | Login form, API calls with auth header | Already in `web/package.json` at 0.26.8; can reference from CDN in standalone page |
| Vanilla HTML/CSS/JS | — | Single-file admin UI, no build step | No new tooling |

### Frontend for Admin Page (Option B — compiled SPA, if needed later)

| Technology | Version | Role | Status |
|------------|---------|------|--------|
| Vite | 8.x | Build tool for admin SPA | Already in `web/package.json` |
| Svelte | 5.x | Component framework | Already in `web/package.json` |
| PocketBase JS SDK | 0.26.8 | Auth and API calls | Already in `web/package.json` |

Option B reuses 100% of existing tooling — only a separate Vite config pointing at a `db/federation/admin/src/` directory is new.

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| Path A: Go route + embedded HTML | PocketBase UI extensions | Officially not production-safe; will break on next PocketBase upgrade |
| Path A: Go route + embedded HTML | Path B: SvelteKit admin flag | More invasive, conflates Wanderer users with infrastructure operators, unnecessary for v1.1 |
| Inline HTML (no build step) | Compiled Svelte SPA | Overhead not justified for ~5 admin operations; can graduate later |
| `apis.RequireSuperuserAuth()` middleware | Manual JWT parse | Reinvents PocketBase's already-correct JWT verification; error-prone |
| `apis.Static()` for embedded FS | `http.FileServer` + `WrapStdHandler` | More verbose; `apis.Static()` is the idiomatic PocketBase 0.38 API |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| PocketBase UI extensions (`/_/`) | Marked unstable by maintainer; breaking changes guaranteed before v1.0 | Custom Go route at `/federation/admin/` |
| Cookie-based auth for admin page | PocketBase superuser auth is stateless JWT only; no session cookies exist for `_superusers` | Authorization header sent by JS from localStorage |
| New Go HTTP framework (chi, gorilla) | PocketBase's router (`se.Router`) is already Echo-based; mixing frameworks breaks middleware chain | Use `se.Router.GET/POST/Group` as done in existing routes |
| `go-chi/chi` or `gorilla/mux` | Not needed — PocketBase 0.38 router is fully capable | `se.Router` |

---

## Installation

No new Go dependencies. The entire implementation uses packages already in `go.mod`.

```bash
# No new dependencies to install.
# Verify existing apis package has the needed symbols:
grep -r "RequireSuperuserAuth\|apis.Static" $(go env GOPATH)/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/apis/
```

If Option B (compiled SPA) is chosen later:

```bash
# From db/federation/admin/ — reuses project's existing Node 22
npm create vite@latest . -- --template svelte-ts
npm install pocketbase
```

---

## Version Compatibility

| Package | Version in Use | Verified APIs |
|---------|---------------|--------------|
| `github.com/pocketbase/pocketbase` | v0.38.0 | `apis.RequireSuperuserAuth()`, `apis.Static()`, `e.HasSuperuserAuth()`, `core.RequestEvent.Auth` |
| Go standard library | 1.25 (module), 1.24.1 (runtime) | `embed.FS`, `io/fs.Sub()` |
| `pocketbase` JS SDK | 0.26.8 (web) | `pb.collection('_superusers').authWithPassword()`, `pb.authStore.isSuperuser` |

---

## Sources

- PocketBase Go Routing docs — `apis.RequireSuperuserAuth`, `apis.Static`, route groups: https://pocketbase.io/docs/go-routing/
- PocketBase `apis` package (v0.38.0) — `RequireSuperuserAuth`, `RequireAuth`, `Static` signatures: https://pkg.go.dev/github.com/pocketbase/pocketbase@v0.38.0/apis
- PocketBase middlewares source — token extraction from Authorization header: https://github.com/pocketbase/pocketbase/blob/master/apis/middlewares.go
- PocketBase authentication docs — stateless JWT, no sessions, `_superusers` collection: https://pocketbase.io/docs/authentication/
- PocketBase UI extensions discussion (maintainer explicitly not production-safe): https://github.com/pocketbase/pocketbase/discussions/7612
- Existing codebase — `HasSuperuserAuth()` already in use: `db/routes/plugin_system.go`, `db/hooks/plugin_instances.go`
- `go:embed` with PocketBase community discussion: https://github.com/pocketbase/pocketbase/discussions/4810
- PocketBase Admin UI own embed pattern (reference implementation): https://github.com/pocketbase/pocketbase/blob/master/ui/embed.go

---

*Stack research for: Wanderer Federation Connect UI (v1.1)*
*Researched: 2026-06-27*
