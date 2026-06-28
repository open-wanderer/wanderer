# Phase 6: Admin Browser UI - Pattern Map

**Mapped:** 2026-06-27
**Files analyzed:** 3 (2 new, 1 modified)
**Analogs found:** 3 / 3

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `db/routes/federation_ui.go` | handler | request-response (HTML) | `db/routes/nodeinfo.go` `NodeInfo21` (lines 23-36) | exact — same raw-bytes write pattern with custom Content-Type |
| `db/routes/federation_ui.html` | view | request-response (Alpine.js + fetch) | `db/routes/federation_admin.go` (API shapes) + UI-SPEC.md (component inventory) | role-match — no existing single-file HTML page; API contract confirmed from analog |
| `db/main.go` | config/route registry | — | `db/main.go` lines 210-216 (existing federation block) | exact — insert one `se.Router.GET` line into the established federation comment block |

---

## Pattern Assignments

### `db/routes/federation_ui.go` (handler, request-response)

**Analog:** `db/routes/nodeinfo.go` — `NodeInfo21` function

**Imports pattern** (`nodeinfo.go` lines 1-11 and `federation_admin.go` lines 1-18 for package context):
```go
package routes

import (
    _ "embed"     // REQUIRED: blank import activates go:embed even for []byte targets
    "net/http"

    "github.com/pocketbase/pocketbase/core"
)
```

**Embed directive + variable** (new — no existing `//go:embed` in `db/` yet, but pattern is standard Go):
```go
//go:embed federation_ui.html
var federationUIHTML []byte
```

**Core handler pattern** (modeled on `nodeinfo.go` lines 23-36):
```go
// NodeInfo21 from nodeinfo.go lines 23-36 — exact template to copy:
func NodeInfo21(e *core.RequestEvent) error {
    // ... build payload ...
    e.Response.Header().Set("Content-Type", `application/json; profile="..."`)
    e.Response.WriteHeader(http.StatusOK)
    _, err = e.Response.Write(b)
    return err
}

// FederationDashboard follows this same structure:
func FederationDashboard(e *core.RequestEvent) error {
    e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
    e.Response.WriteHeader(http.StatusOK)
    _, err := e.Response.Write(federationUIHTML)
    return err
}
```

**No auth guard on this handler** — confirmed by CONTEXT.md D-01: the page is served public; auth is enforced in JS. Contrast with all six federation API handlers that begin with:
```go
// federation_admin.go lines 215-217 (FederationFollow pattern, repeated on all six handlers):
if !e.HasSuperuserAuth() {
    return e.UnauthorizedError("superuser authentication required", nil)
}
```
`FederationDashboard` intentionally omits this block.

**Optional security header** (from RESEARCH.md security domain — Clickjacking mitigation):
```go
e.Response.Header().Set("X-Frame-Options", "DENY")
```

---

### `db/routes/federation_ui.html` (view, Alpine.js SPA)

**Analog:** `db/routes/federation_admin.go` — API response shapes (for JS `fetch` calls); `06-UI-SPEC.md` — full component inventory and color palette.

**No close analog exists in the codebase** — this is the first Go-embedded single-file HTML page. All structural patterns are drawn from UI-SPEC.md and the RESEARCH.md code examples.

#### HTML skeleton (from RESEARCH.md "HTML Page Skeleton"):
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Federation Peers</title>

  <!-- 1. Theme detection: synchronous IIFE in <head> — must run before <body> renders -->
  <script>
    (function() {
      var s = localStorage.getItem('pbColorScheme');
      var d = window.matchMedia('(prefers-color-scheme: dark)').matches;
      if (s === 'dark' || (!s && d)) {
        document.documentElement.classList.add('dark');
      }
    })();
  </script>

  <!-- 2. Tailwind dark mode config: MUST precede Play CDN script (Pitfall 6) -->
  <script>tailwind.config = { darkMode: 'class' }</script>
  <script src="https://cdn.tailwindcss.com"></script>

  <!-- 3. Alpine app function: MUST precede Alpine CDN defer script (Pitfall 3) -->
  <script>
    function federationApp() { return { /* state */ }; }
  </script>
  <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
</head>
<body x-data="federationApp()" x-init="init()"
      class="min-h-screen bg-white dark:bg-[#12141c] text-gray-900 dark:text-gray-100">
</body>
</html>
```

#### Alpine `x-data` state object (from RESEARCH.md "Pattern 3"):
```javascript
function federationApp() {
    return {
        token: null,
        peers: [],
        loading: false,
        discoverUrl: '',
        discoverLoading: false,
        discoverError: '',
        preview: null,
        rowErrors: {},    // keyed by follow_id for per-row error messages

        init() {
            const stored = localStorage.getItem('__pb_superusers__/_');
            if (!stored) { window.location.replace('/_/'); return; }
            try {
                const parsed = JSON.parse(stored);
                this.token = parsed.token;       // NOT the raw string — must JSON.parse
            } catch (_) { window.location.replace('/_/'); return; }
            if (this.isTokenExpired(this.token)) { window.location.replace('/_/'); return; }
            this.loadPeers();
        },

        isTokenExpired(token) {
            try {
                const payload = JSON.parse(atob(token.split('.')[1]));
                return payload.exp && Date.now() / 1000 > payload.exp;
            } catch (_) { return true; }
        },

        async apiFetch(path, opts = {}) {
            const res = await fetch(path, {
                ...opts,
                headers: { 'Authorization': this.token, ...(opts.headers || {}) },
                // NOTE: Authorization: <token> — no "Bearer" prefix (PocketBase format)
            });
            if (res.status === 401) { window.location.replace('/_/'); return null; }
            return res;
        },

        async loadPeers() {
            this.loading = true;
            try {
                const res = await this.apiFetch('/federation/peers');
                if (res) this.peers = await res.json();
            } finally { this.loading = false; }
        },
    };
}
```

#### API request/response shapes (from `federation_admin.go` — verified):

```javascript
// POST /federation/discover
// Request:  { "url": "https://other.example.com" }
// Response: { "actor_id": "...", "domain": "...", "version": "...", "user_count": N, "trail_count": N }
// Error:    { "error": "..." } with HTTP 400

// POST /federation/follow
// Request:  { "actor_id": "<activitypub_actors record id>" }
// Response: { "follow_id": "...", "status": "pending" }

// GET /federation/peers
// Response: [{ "follow_id": "...", "direction": "outbound"|"inbound"|"mutual", "status": "pending"|"accepted"|"rejected", "domain": "..." }]

// POST /federation/approve/:id  → { "follow_id": "...", "status": "accepted" }
// POST /federation/reject/:id   → { "follow_id": "...", "status": "rejected" }
// POST /federation/disconnect/:id → { "status": "ok" }
```

Note: `user_count` and `trail_count` are the field names in the discover response (from `federation_admin.go` line 738-743), NOT `users`/`trails`. The UI-SPEC.md preview card uses `preview.users` / `preview.trails` — the JS must map `user_count` → `users` and `trail_count` → `trails` when assigning to `this.preview`, or reference the correct field names.

#### Conditional action buttons per row (from RESEARCH.md "Pattern 5"):
```html
<template x-for="peer in peers" :key="peer.follow_id">
  <tr :class="peer.status === 'rejected' ? 'opacity-60' : ''">
    <td x-text="peer.domain"></td>
    <td><!-- direction badge --></td>
    <td><!-- status badge --></td>
    <td class="text-right">
      <template x-if="peer.status === 'pending' && peer.direction === 'inbound'">
        <div class="flex gap-2 justify-end">
          <button @click="approve(peer)" :disabled="loading">Approve</button>
          <button @click="reject(peer)"  :disabled="loading">Reject</button>
        </div>
      </template>
      <template x-if="peer.status === 'accepted'">
        <button @click="disconnect(peer)" :disabled="loading">Disconnect</button>
      </template>
      <template x-if="peer.status === 'pending' && peer.direction === 'outbound'">
        <span class="text-gray-400 text-sm">Pending...</span>
      </template>
    </td>
  </tr>
</template>
```

Use `x-if` inside `<template>` (not `x-show` on `<tr>`) — some browsers mishandle `display:none` on table rows (RESEARCH.md anti-patterns).

#### Disconnect confirmation (from CONTEXT.md D-15 + RESEARCH.md Pitfall 5):
```javascript
async disconnect(peer) {
    if (!confirm(`Disconnect from ${peer.domain}? This will send an Undo{Follow} or Reject{Follow} to the remote instance.`)) return;
    const res = await this.apiFetch(`/federation/disconnect/${peer.follow_id}`, { method: 'POST' });
    if (res && res.ok) await this.loadPeers();
}
```
Template literal required — string concatenation with `peer.domain` not yet evaluated would show "undefined" (Pitfall 5).

#### Loading spinner (from UI-SPEC.md "Loading State"):
```html
<span class="inline-block w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"
      aria-hidden="true"></span>
```

---

### `db/main.go` (route registry modification)

**Analog:** `db/main.go` lines 210-216 — existing federation block

**Insertion point** (add after line 216, before the closing `}`):
```go
// Federation admin endpoints (Plan 03)
se.Router.POST("/federation/discover", routes.FederationDiscover)
se.Router.POST("/federation/follow", routes.FederationFollow)
se.Router.POST("/federation/approve/{id}", routes.FederationApprove)
se.Router.POST("/federation/reject/{id}", routes.FederationReject)
se.Router.POST("/federation/disconnect/{id}", routes.FederationDisconnect)
se.Router.GET("/federation/peers", routes.FederationPeers)
// ADD THIS LINE (Phase 6):
se.Router.GET("/federation/", routes.FederationDashboard)
```

Trailing slash on `/federation/` is load-bearing — Echo-based router does exact path matching; `/federation/` will not conflict with `/federation/peers` or `/federation/discover`.

---

## Shared Patterns

### Raw-bytes HTTP response with custom Content-Type
**Source:** `db/routes/nodeinfo.go` lines 23-36 (`NodeInfo21`)
**Apply to:** `FederationDashboard` in `federation_ui.go`
```go
e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
e.Response.WriteHeader(http.StatusOK)
_, err := e.Response.Write(federationUIHTML)
return err
```

### Auth guard pattern (for reference — NOT used on `FederationDashboard`)
**Source:** `db/routes/federation_admin.go` lines 215-217 (repeated on all six handlers)
**Apply to:** All six existing handlers already have it; `FederationDashboard` explicitly omits it per D-01
```go
if !e.HasSuperuserAuth() {
    return e.UnauthorizedError("superuser authentication required", nil)
}
```

### PocketBase admin localStorage token format
**Source:** RESEARCH.md (confirmed from PocketBase v0.38.0 `ui/src/pb.js` + `pocketbase.es.js`)
**Apply to:** `federationApp().init()` in `federation_ui.html`
```javascript
// localStorage key: "__pb_superusers__/_"
// Value format: JSON string {"token":"<jwt>","record":{...}}
// MUST JSON.parse() and extract .token — do NOT use the raw string value
const parsed = JSON.parse(localStorage.getItem('__pb_superusers__/_'));
const token = parsed.token;
// Authorization header: NO "Bearer" prefix — PocketBase expects raw JWT
fetch('/federation/peers', { headers: { 'Authorization': token } });
```

### Color palette (dark/light)
**Source:** `06-UI-SPEC.md` §Color
**Apply to:** All elements in `federation_ui.html`

| Role | Light | Dark |
|------|-------|------|
| Page background | `bg-white` | `dark:bg-[#12141c]` |
| Section/card background | `bg-gray-50` | `dark:bg-[#1e2028]` |
| Header strip + primary buttons | `bg-[#242734]` | same |
| Destructive (Reject, Disconnect) | `bg-red-500` | same |
| Border | `border-gray-300` | `dark:border-[#373c50]` |
| Text primary | `text-gray-900` | `dark:text-gray-100` |
| Text secondary | `text-gray-500` | `dark:text-gray-400` |

### Status badge colors
**Source:** `06-UI-SPEC.md` §Color
```
pending:  bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200
accepted: bg-green-100  text-green-800  dark:bg-green-900  dark:text-green-200
rejected: bg-gray-100   text-gray-500   dark:bg-gray-800   dark:text-gray-400
```

### Direction badge colors
**Source:** `06-UI-SPEC.md` §Color
```
Outbound: bg-blue-100   text-blue-800   dark:bg-blue-900   dark:text-blue-200
Inbound:  bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200
Mutual:   bg-teal-100   text-teal-800   dark:bg-teal-900   dark:text-teal-200
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `db/routes/federation_ui.html` | view | request-response (Alpine.js) | No existing Go-embedded single-file HTML page in the codebase. The `db/templates/mail/notification.html` uses PocketBase's template registry (`LoadFiles`) not `//go:embed` into `[]byte`. All patterns sourced from 06-UI-SPEC.md and RESEARCH.md code examples. |

---

## Critical Anti-Patterns (from RESEARCH.md)

1. **Wrong localStorage value used raw** — `localStorage.getItem("__pb_superusers__/_")` returns a JSON string, not a JWT. Must `JSON.parse()` and extract `.token`.
2. **`Bearer` prefix on Authorization header** — PocketBase superuser API expects `Authorization: <token>` (raw JWT). No "Bearer" prefix.
3. **Theme detection in `x-init`** — too late; causes theme flash. Must run as synchronous IIFE in `<head>`.
4. **`tailwind.config = { darkMode: 'class' }` missing** — `dark:` variants have no effect in Play CDN without this config before the CDN script tag.
5. **`_ "embed"` import omitted** — Go compile error. Required even when embedding into `[]byte`.
6. **Alpine CDN before app function** — `defer` causes Alpine to process DOM before inline script defines `federationApp`. App function must be in `<head>` script before the `<script defer>` Alpine tag.
7. **`x-show` on `<tr>`** — use `<template x-if>` for conditional table row content instead.
8. **`discover` response field names** — API returns `user_count` / `trail_count`, not `users` / `trails`. Map correctly when assigning to `this.preview`.

---

## Metadata

**Analog search scope:** `db/routes/`, `db/util/`, `db/main.go`
**Files scanned:** `nodeinfo.go`, `federation_admin.go`, `email_templates.go`, `main.go` (lines 200-233)
**No `//go:embed` exists yet in `db/`** — `federation_ui.go` will establish the first use of this pattern in the module
**Pattern extraction date:** 2026-06-27
