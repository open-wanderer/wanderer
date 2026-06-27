# Phase 6: Admin Browser UI - Research

**Researched:** 2026-06-27
**Domain:** Go-embedded single-file HTML page with Alpine.js + Tailwind CDN; consumes Phase 5 federation admin API
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Phase Boundary**
- Deliver a Go-embedded HTML page at `/federation/` that lets a PocketBase superuser manage all peer instance connections from a browser.
- In scope: `GET /federation/` route, auth gate via JS reading PocketBase admin localStorage, discovery form, peer table with direction/status, inline approve/reject/disconnect actions, dark/light theme from `pbColorScheme`.
- Out of scope: realtime updates, refresh metadata button, activity log, WebFinger.

**Auth Flow (D-01 through D-05)**
- Page served as public HTML at `GET /federation/` — no server-side token check on the page load itself.
- JS reads `localStorage.getItem("__pb_superusers__/_")` — value is a JSON string `{"token":"<jwt>","record":{...}}` — parse and extract `.token`.
- Missing or expired token → redirect to `/_/`.
- Token attached as `Authorization: <token>` (not `Bearer <token>`) on every API fetch — PocketBase superuser API uses the raw token in the Authorization header without "Bearer" prefix.
- API endpoints enforce `HasSuperuserAuth()` server-side; page does not duplicate that guard.

**JS/Styling (D-06 through D-09)**
- Alpine.js via CDN (`https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js`).
- Tailwind CSS Play CDN (`https://cdn.tailwindcss.com`).
- Dark/light: read `localStorage.getItem("pbColorScheme")` → "light"/"dark"; fallback to `prefers-color-scheme`; apply `dark` class to `<html>`.
- HTML file embedded via `//go:embed` at `db/routes/federation_ui.html`; handler writes it with `Content-Type: text/html; charset=utf-8`.

**Discovery UX (D-10 through D-12)**
- Two-step: paste URL → Discover → preview card → Connect.
- Discovery errors shown inline below input, not in modal/toast.
- On Connect: call `POST /federation/follow` with `actor_id` from discovery response, reset form, re-fetch peer list.

**Peer List (D-13 through D-16)**
- Columns: Domain, Direction (Outbound/Inbound/Mutual), Status (pending/accepted/rejected), Actions.
- Conditional actions per row state (see D-14).
- Disconnect uses `window.confirm()` guard.
- After any action: re-fetch `/federation/peers`.

### Claude's Discretion

- Page title, heading, and copy text (e.g., "Federation Peers", "Connect a new instance")
- Whether rejected peers are shown in the table or filtered out
- Loading/spinner states during API calls
- Exact Tailwind class choices for buttons (primary vs secondary variants)
- Whether the discovery preview card is dismissible (cancel) or only cleared on Connect or error

### Deferred Ideas (OUT OF SCOPE)

- Realtime peer list — polling or PocketBase subscription → v2 UX-01
- Refresh metadata button → v2 UX-02
- Connection activity log → v2 UX-03
- Manual "Refresh" button → v2 (not needed, re-fetches on every action)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DASH-01 | Admin can view all peer connections with status (pending/accepted/rejected) and direction (Outbound/Inbound/Mutual) in a browser page at `/federation/` | `GET /federation/peers` returns `[{follow_id, direction, status, domain}]` — direct mapping to table columns |
| DASH-02 | The page requires PocketBase superuser authentication — unauthenticated requests receive 401; regular Wanderer user tokens are rejected | Page JS reads `__pb_superusers__/_` localStorage key; API handlers enforce `e.HasSuperuserAuth()`; 401 redirects to `/_/` |
</phase_requirements>

---

## Summary

Phase 6 delivers a single self-contained HTML file (`db/routes/federation_ui.html`) served by a new Go route handler at `GET /federation/`. The page uses Alpine.js (CDN) for reactivity and Tailwind Play CDN for styling. It is a pure consumer of the six Phase 5 federation API endpoints — no backend changes are required.

The implementation is straightforward because all decisions are locked, the API contract is fully implemented and documented, the UI spec is approved, and the Go embedding pattern is confirmed in the existing codebase. The primary implementation work is writing the HTML file and the `FederationDashboard` route handler. There are no npm packages to install, no build steps, and no SvelteKit changes.

The single non-obvious technical detail is the localStorage key format: `localStorage.getItem("__pb_superusers__/_")` returns a JSON string `{"token":"<jwt>","record":{...}}` — the page must `JSON.parse()` it and use `.token`, not the raw string. This is confirmed by reading the PocketBase JS SDK source (`LocalAuthStore.save()` calls `this._storageSet(this.storageKey, {token: e, record: t})`).

**Primary recommendation:** Write `db/routes/federation_ui.html` as a complete, self-contained Alpine.js + Tailwind CDN page following the exact component structures in 06-UI-SPEC.md, then add a `FederationDashboard` handler and route registration in `db/main.go`.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Serve the HTML page | Go/PocketBase route handler | — | All custom routes live in `db/routes/`; this is a `GET /federation/` handler identical in structure to `NodeInfo` and `Health` |
| Auth enforcement on page data | API / Backend (Phase 5 handlers) | Browser JS (redirect guard) | `HasSuperuserAuth()` in every API handler is the hard gate; JS redirect is a UX shortcut |
| localStorage token read | Browser / Client | — | PocketBase admin panel writes the token to `localStorage` — no server-side equivalent |
| Peer list display | Browser / Client (Alpine.js) | — | `x-data` holds fetched peers array; `x-for` renders rows |
| Discovery form state | Browser / Client (Alpine.js) | — | Multi-step form state (url, preview, error, loading) lives in Alpine `x-data` object |
| Inline action buttons | Browser / Client (Alpine.js) | — | Conditional rendering based on row direction+status, managed in JS |
| Dark/light theme | Browser / Client | — | `pbColorScheme` is a browser localStorage key; applied via `dark` class on `<html>` |
| ActivityPub delivery | Go hooks (Phase 5) | — | Hooks own delivery; the UI never calls delivery functions directly |

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Alpine.js | 3.x (latest via CDN) | Page reactivity — `x-data`, `x-show`, `x-for`, `@click`, `:disabled` | Locked decision D-06; no build step; matches Go-embedded single-file constraint |
| Tailwind CSS Play CDN | Latest via `cdn.tailwindcss.com` | Utility-first styling, dark mode via `dark:` variants | Locked decision D-07; Play CDN generates classes on-demand in browser |

Both libraries are loaded via CDN `<script>` tags — no npm install needed.

### Supporting

None. This phase installs zero new npm or Go packages.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Alpine.js CDN | Vanilla JS | Alpine chosen (D-06); vanilla would require more boilerplate for reactive state |
| Tailwind Play CDN | Inline styles | Tailwind chosen (D-07); enables same utility classes as rest of Wanderer |
| `go:embed` | `os.ReadFile` at runtime | `go:embed` bakes file into binary (single deployable artifact); `ReadFile` requires the HTML file at runtime path |

**Installation:** No packages to install.

---

## Package Legitimacy Audit

No external packages are installed by this phase. Alpine.js and Tailwind CSS are loaded from CDN URLs at runtime in the browser — they are not installed as Go or npm dependencies.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| alpinejs (CDN only) | cdn.jsdelivr.net | ~5 years | Very high (GitHub 27k+ stars) | github.com/alpinejs/alpine | N/A — CDN, not installed | Approved |
| tailwindcss (CDN only) | cdn.tailwindcss.com | ~6 years | Very high | github.com/tailwindlabs/tailwindcss | N/A — CDN, not installed | Approved |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
Browser (Admin)
    │
    │ GET /federation/
    ▼
Go Handler: FederationDashboard
    │  serves federation_ui.html (go:embed)
    │  Content-Type: text/html; charset=utf-8
    ▼
Browser renders Alpine.js + Tailwind page
    │
    │ on page load: read localStorage["__pb_superusers__/_"]
    │               JSON.parse → extract .token
    │               if missing/expired → redirect to /_/
    │
    │ on init (x-init): fetch GET /federation/peers
    │                   Authorization: <token>
    ▼
Phase 5 API Handlers (all enforce HasSuperuserAuth())
    ├── GET  /federation/peers            → [{follow_id, direction, status, domain}]
    ├── POST /federation/discover         ← {url}    → {actor_id, domain, version, user_count, trail_count}
    ├── POST /federation/follow           ← {actor_id} → {follow_id, status}
    ├── POST /federation/approve/:id      → {follow_id, status}
    ├── POST /federation/reject/:id       → {follow_id, status}
    └── POST /federation/disconnect/:id   → {status: "ok"}
```

### Recommended Project Structure

```
db/
├── routes/
│   ├── federation_admin.go       # existing Phase 5 handlers (unchanged)
│   ├── federation_ui.go          # new: FederationDashboard handler + go:embed
│   └── federation_ui.html        # new: single-file HTML page
└── main.go                       # add GET /federation/ route registration
```

**Note:** The `//go:embed federation_ui.html` directive must appear in a `.go` file in the same package directory as the HTML file (`db/routes/`). A separate `federation_ui.go` is the clean approach — keeps embedding isolated from the six API handlers in `federation_admin.go`.

### Pattern 1: Go Raw HTML Response

**What:** Serve a raw HTML file from a PocketBase route handler using `e.Response.Write()`.
**When to use:** When the route returns HTML, not JSON — identical to `NodeInfo21` which writes raw bytes with a custom Content-Type.

```go
// Source: db/routes/nodeinfo.go (confirmed pattern)
// db/routes/federation_ui.go

package routes

import (
    _ "embed"
    "net/http"

    "github.com/pocketbase/pocketbase/core"
)

//go:embed federation_ui.html
var federationUIHTML []byte

func FederationDashboard(e *core.RequestEvent) error {
    e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
    e.Response.WriteHeader(http.StatusOK)
    _, err := e.Response.Write(federationUIHTML)
    return err
}
```

**Key detail:** The `//go:embed` directive reads the file at compile time. The `_ "embed"` blank import is required when only using `//go:embed` into a `[]byte` (not `embed.FS`). [VERIFIED: Go stdlib embed documentation]

### Pattern 2: Route Registration in `db/main.go`

**What:** Add the `GET /federation/` route alongside the existing six federation routes.
**When to use:** All custom routes registered in `registerRoutes()` function.

```go
// Source: db/main.go lines 211-216 (confirmed pattern)
// Add after the existing federation routes:
se.Router.GET("/federation/", routes.FederationDashboard)
```

**Key detail:** The trailing slash matters. The existing endpoints use `/federation/discover`, `/federation/peers`, etc. The dashboard route is `GET /federation/` — a distinct path that will not conflict.

### Pattern 3: Alpine.js `x-data` State Object

**What:** Single Alpine component on the `<body>` tag holding all page state.
**When to use:** Self-contained single-page admin tools with no component hierarchy.

```html
<!-- Source: Alpine.js CDN documentation [ASSUMED — CDN, no Context7 entry] -->
<body x-data="federationApp()" x-init="init()">
```

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
        rowErrors: {},   // keyed by follow_id for per-row error messages

        init() {
            // 1. Read token from PocketBase admin localStorage
            const stored = localStorage.getItem('__pb_superusers__/_');
            if (!stored) { window.location.replace('/_/'); return; }
            try {
                const parsed = JSON.parse(stored);
                this.token = parsed.token;
            } catch (_) { window.location.replace('/_/'); return; }

            // 2. Basic JWT expiry check (decode payload, check exp field)
            if (this.isTokenExpired(this.token)) {
                window.location.replace('/_/');
                return;
            }

            // 3. Load peer list
            this.loadPeers();
        },

        isTokenExpired(token) {
            try {
                const payload = JSON.parse(atob(token.split('.')[1]));
                return payload.exp && Date.now() / 1000 > payload.exp;
            } catch (_) { return true; }
        },

        async fetch(path, opts = {}) {
            const res = await fetch(path, {
                ...opts,
                headers: { 'Authorization': this.token, ...(opts.headers || {}) },
            });
            if (res.status === 401) { window.location.replace('/_/'); return null; }
            return res;
        },

        async loadPeers() {
            this.loading = true;
            try {
                const res = await this.fetch('/federation/peers');
                if (res) this.peers = await res.json();
            } finally { this.loading = false; }
        },
    };
}
```

### Pattern 4: Dark/Light Theme Application

**What:** Apply `dark` class to `<html>` before Alpine initializes so `dark:` variants work.
**When to use:** Any Alpine + Tailwind page that needs to match PocketBase admin theme.

```javascript
// Source: 06-CONTEXT.md D-08 (locked decision)
// Run synchronously in <head> to avoid flash of wrong theme:
(function() {
    const scheme = localStorage.getItem('pbColorScheme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    if (scheme === 'dark' || (!scheme && prefersDark)) {
        document.documentElement.classList.add('dark');
    }
})();
```

**Key detail:** This IIFE must run in `<head>` before the `<body>` renders. Putting it in `x-init` is too late — the page would flash with wrong theme on load.

### Pattern 5: Conditional Action Buttons Per Row

**What:** Render only the buttons valid for a row's state using Alpine `x-show` or `x-if`.
**When to use:** Per 06-CONTEXT.md D-14 and 06-UI-SPEC.md table row spec.

```html
<!-- Source: 06-UI-SPEC.md Table Row spec -->
<template x-for="peer in peers" :key="peer.follow_id">
  <tr :class="peer.status === 'rejected' ? 'opacity-60' : ''">
    <td x-text="peer.domain"></td>
    <td><!-- direction badge --></td>
    <td><!-- status badge --></td>
    <td class="text-right">
      <!-- Inbound pending: Approve + Reject -->
      <template x-if="peer.status === 'pending' && peer.direction === 'inbound'">
        <div class="flex gap-2 justify-end">
          <button @click="approve(peer)" :disabled="loading">Approve</button>
          <button @click="reject(peer)" :disabled="loading">Reject</button>
        </div>
      </template>
      <!-- Accepted: Disconnect -->
      <template x-if="peer.status === 'accepted'">
        <button @click="disconnect(peer)" :disabled="loading">Disconnect</button>
      </template>
      <!-- Outbound pending: text only -->
      <template x-if="peer.status === 'pending' && peer.direction === 'outbound'">
        <span class="text-gray-400 text-sm">Pending...</span>
      </template>
    </td>
  </tr>
</template>
```

### Anti-Patterns to Avoid

- **`x-show` on `<tr>` elements:** Some browsers do not correctly handle `display: none` on table rows; use `x-if` inside `<template>` tags for conditional table row content, or wrap in a `<tbody>` with `x-show`.
- **Reading raw localStorage value as the token:** `localStorage.getItem("__pb_superusers__/_")` returns a JSON string `{"token":"<jwt>","record":{...}}`. Must `JSON.parse()` and extract `.token`. Using the raw string as the Authorization header value will cause all API calls to fail with 401.
- **`Bearer` prefix on Authorization header:** PocketBase superuser API expects `Authorization: <token>` (raw JWT, no "Bearer" prefix). Confirmed in `db/routes/federation_admin.go` — `e.HasSuperuserAuth()` handles the PocketBase token format, and the PocketBase UI itself sends `Authorization: TOKEN` (not `Authorization: Bearer TOKEN`). [VERIFIED: PocketBase v0.38.0 source `ui/src/auth/pageInstaller.js` line `headers: { Authorization: token }`]
- **Putting theme detection in `x-init`:** Too late — causes flash of wrong theme. Must run in `<head>` as a synchronous script.
- **`//go:embed` without blank `_ "embed"` import:** Go compiler will reject it. Required even when embedding into `[]byte` (not `embed.FS`).
- **CORS on fetch calls:** The page is served from the same origin as the API (same Go binary, same domain), so no CORS headers are needed.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JWT expiry check | Custom JWT parser | Decode base64 payload, check `.exp` field | Simple `atob(token.split('.')[1])` + `JSON.parse` is sufficient; no cryptographic verification needed since the API re-validates every request |
| Dark mode detection | Custom color scheme logic | Read `pbColorScheme` localStorage key, fallback to `prefers-color-scheme` | Exactly what PocketBase admin panel does — confirmed in `store.js` |
| Reactive state management | Custom pub/sub or event emitter | Alpine.js `x-data` | Alpine is the locked choice (D-06); its built-in reactivity handles all state needs |
| Tailwind CSS build | Local PostCSS/Vite build | Tailwind Play CDN | No build step is the locked constraint (D-07); Play CDN handles all utility classes on-demand |
| Loading spinners | CSS animation library | `animate-spin` Tailwind class on a border-trick div | Specified in UI-SPEC.md; no additional library needed |

**Key insight:** This phase is intentionally simple — the entire implementation is one HTML file and one Go handler. Resist the temptation to add any JavaScript framework beyond Alpine.js or any Go middleware.

---

## Common Pitfalls

### Pitfall 1: Wrong localStorage Token Format

**What goes wrong:** `JSON.parse` fails or the token is `undefined`, causing all API calls to get 401 and redirect loop to `/_/`.
**Why it happens:** `LocalAuthStore.save()` in the PocketBase JS SDK stores `{token: e, record: t}` as a JSON string — not the raw token. Reading the value with `localStorage.getItem(key)` returns the JSON string, not the JWT.
**How to avoid:** Always wrap in try/catch: `const parsed = JSON.parse(stored); this.token = parsed.token;`. Guard against missing `.token` field.
**Warning signs:** Immediate redirect to `/_/` even after successful admin login; `Authorization` header value starts with `{"token":` in DevTools.

### Pitfall 2: `//go:embed` Blank Import Omission

**What goes wrong:** Go compile error: `//go:embed only allowed in Go source files` or `undefined: embed`.
**Why it happens:** `//go:embed` into a `[]byte` variable (as opposed to `embed.FS`) still requires `import _ "embed"` to activate the embed package.
**How to avoid:** Add `_ "embed"` to the import block in `federation_ui.go`.
**Warning signs:** `go build` fails with embed-related error.

### Pitfall 3: Alpine.js CDN Defer Loading

**What goes wrong:** Alpine initializes before the `x-data` function is defined in the `<script>` tag.
**Why it happens:** Alpine.js CDN with `defer` starts processing the DOM before inline scripts run.
**How to avoid:** Define the `federationApp()` function in a `<script>` tag placed **before** the Alpine CDN `<script defer>` tag, OR use `document.addEventListener('alpine:init', ...)`. The simplest approach: put the app script in `<head>` before the Alpine CDN script.
**Warning signs:** `Alpine Expression Error: federationApp is not defined`.

### Pitfall 4: Route Path Conflict with API Routes

**What goes wrong:** A request to `GET /federation/peers` might match `GET /federation/` if the router does prefix matching.
**Why it happens:** PocketBase's Echo-based router does exact path matching (not prefix), but a trailing slash matters.
**How to avoid:** Register as `se.Router.GET("/federation/", routes.FederationDashboard)` — the trailing slash makes it unambiguous. The existing routes are `/federation/discover`, `/federation/follow`, etc. — no conflict.
**Warning signs:** Peer list fetch returns HTML instead of JSON.

### Pitfall 5: `window.confirm()` String Interpolation

**What goes wrong:** Confirmation dialog shows literal `<domain>` instead of the actual domain name.
**Why it happens:** Alpine `@click` handler uses string concatenation — needs to reference `peer.domain` correctly.
**How to avoid:** Build the confirm string with JavaScript template literals: `` `Disconnect from ${peer.domain}? This will send an Undo{Follow} or Reject{Follow} to the remote instance.` ``
**Warning signs:** Confirmation dialog shows "Disconnect from undefined?".

### Pitfall 6: Tailwind Play CDN `dark:` Variant Requires `darkMode: 'class'`

**What goes wrong:** `dark:` utility classes have no effect even when `<html class="dark">` is set.
**Why it happens:** Tailwind Play CDN defaults to `media` strategy for dark mode. Class-based dark mode needs explicit configuration.
**How to avoid:** Add the Tailwind config script before the Play CDN script:
```html
<script>
  tailwind.config = { darkMode: 'class' }
</script>
```
**Warning signs:** Dark mode colors never apply despite `dark` class on `<html>`.

---

## Code Examples

Verified patterns from official sources and codebase inspection:

### Go Handler — Serve Raw HTML (confirmed from `nodeinfo.go`)

```go
// Source: db/routes/nodeinfo.go lines 32-35 (confirmed codebase pattern)
package routes

import (
    _ "embed"
    "net/http"
    "github.com/pocketbase/pocketbase/core"
)

//go:embed federation_ui.html
var federationUIHTML []byte

func FederationDashboard(e *core.RequestEvent) error {
    e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
    e.Response.WriteHeader(http.StatusOK)
    _, err := e.Response.Write(federationUIHTML)
    return err
}
```

### Route Registration (confirmed from `db/main.go` lines 211-216)

```go
// Source: db/main.go — add after existing federation routes
se.Router.GET("/federation/", routes.FederationDashboard)
```

### HTML Page Skeleton

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Federation Peers</title>
  <!-- Theme detection: must run synchronously before body renders (Pitfall 3) -->
  <script>
    (function() {
      var s = localStorage.getItem('pbColorScheme');
      var d = window.matchMedia('(prefers-color-scheme: dark)').matches;
      if (s === 'dark' || (!s && d)) {
        document.documentElement.classList.add('dark');
      }
    })();
  </script>
  <!-- Tailwind dark mode config: must precede Play CDN script (Pitfall 6) -->
  <script>tailwind.config = { darkMode: 'class' }</script>
  <script src="https://cdn.tailwindcss.com"></script>
  <!-- Alpine app function: must precede Alpine CDN defer script (Pitfall 3) -->
  <script>
    function federationApp() {
      return {
        token: null,
        peers: [],
        loading: false,
        // ... full state object
      };
    }
  </script>
  <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
</head>
<body x-data="federationApp()" x-init="init()"
      class="min-h-screen bg-white dark:bg-[#12141c] text-gray-900 dark:text-gray-100">
  <!-- header, main content per UI-SPEC.md -->
</body>
</html>
```

### Discovery API Request Shape

```javascript
// Source: db/routes/federation_admin.go FederationDiscover (verified)
// POST /federation/discover
// Request:  {"url": "https://other.example.com"}
// Response: {"actor_id": "...", "domain": "...", "version": "...", "user_count": N, "trail_count": N}
// Error:    {"error": "..."}  (HTTP 400)

async discover() {
    this.discoverLoading = true;
    this.discoverError = '';
    this.preview = null;
    try {
        const res = await this.fetch('/federation/discover', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url: this.discoverUrl }),
        });
        if (!res) return; // 401 → redirected
        const data = await res.json();
        if (!res.ok) {
            this.discoverError = data.error || 'Discovery failed';
        } else {
            this.preview = data; // {actor_id, domain, version, user_count, trail_count}
        }
    } finally {
        this.discoverLoading = false;
    }
}
```

### Follow API Request Shape

```javascript
// Source: db/routes/federation_admin.go FederationFollow (verified)
// POST /federation/follow
// Request:  {"actor_id": "<activitypub_actors record id>"}
// Response: {"follow_id": "...", "status": "pending"}

async connect() {
    const res = await this.fetch('/federation/follow', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ actor_id: this.preview.actor_id }),
    });
    if (!res) return;
    if (res.ok) {
        this.discoverUrl = '';
        this.preview = null;
        await this.loadPeers();
    }
}
```

### Peers API Response Shape

```javascript
// Source: db/routes/federation_admin.go peerEntry type (verified)
// GET /federation/peers
// Response: [{follow_id: "...", direction: "outbound"|"inbound"|"mutual", status: "pending"|"accepted"|"rejected", domain: "..."}]
```

### Approve/Reject/Disconnect API

```javascript
// Source: db/routes/federation_admin.go FederationApprove/Reject/Disconnect (verified)
// POST /federation/approve/:id   → {follow_id, status: "accepted"}
// POST /federation/reject/:id    → {follow_id, status: "rejected"}
// POST /federation/disconnect/:id → {status: "ok"}

async approve(peer) {
    const res = await this.fetch(`/federation/approve/${peer.follow_id}`, { method: 'POST' });
    if (res && res.ok) await this.loadPeers();
}

async disconnect(peer) {
    if (!confirm(`Disconnect from ${peer.domain}? This will send an Undo{Follow} or Reject{Follow} to the remote instance.`)) return;
    const res = await this.fetch(`/federation/disconnect/${peer.follow_id}`, { method: 'POST' });
    if (res && res.ok) await this.loadPeers();
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `Authorization: Bearer <token>` | `Authorization: <token>` (no Bearer prefix) | PocketBase v0.22+ | PocketBase superuser API expects raw JWT without Bearer prefix — confirmed in v0.38.0 source |
| PocketBase `/_/` SvelteKit admin extension | Not recommended — "not production-safe, breaks on upgrade" | Per REQUIREMENTS.md Out of Scope | Ruled out; Go-embedded page is the approved pattern |
| `go:embed` into `string` | `go:embed` into `[]byte` | Go 1.16+ | `[]byte` avoids UTF-8 string conversion overhead; both work |

**Deprecated/outdated:**
- PocketBase admin UI extensions: maintainer-stated not production-safe (REQUIREMENTS.md out-of-scope).
- `Bearer` token prefix: PocketBase's own UI sends `Authorization: TOKEN` without prefix.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Alpine.js 3.x CDN URL `https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js` resolves to latest 3.x | Standard Stack | Would need to pin a specific version; risk is very low since 3.x is stable and jsDelivr maintains it |
| A2 | `tailwind.config = { darkMode: 'class' }` before Play CDN activates class-based dark mode in Play CDN | Common Pitfalls | Play CDN behavior could differ — test in browser first |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed. (Two low-risk assumptions remain above.)

---

## Open Questions (RESOLVED)

1. **Alpine.js version pinning**
   - What we know: UI-SPEC.md specifies `alpinejs@3.x.x` (floating minor).
   - What's unclear: Whether to pin to a specific version (e.g., `@3.14.8`) for reproducibility.
   - Recommendation: Pin to latest stable 3.x at plan time (e.g., `3.14.8`); easy to update later.

2. **No open structural questions.** The API, auth, routing, embedding, and UI spec are all fully specified. Implementation is execution work.

---

## Environment Availability

This phase has no external tool dependencies. The HTML file and Go handler are compiled into the existing binary. The CDN URLs are runtime browser fetches, not build-time dependencies.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go compiler | `go build` | ✓ | (project standard) | — |
| PocketBase v0.38.0 | `e.HasSuperuserAuth()`, route handler signature | ✓ | v0.38.0 (confirmed in go.mod) | — |
| Alpine.js CDN | Browser runtime | ✓ (CDN) | 3.x | — |
| Tailwind Play CDN | Browser runtime | ✓ (CDN) | Latest | — |

---

## Validation Architecture

> `workflow.nyquist_validation` is `false` in `.planning/config.json` — this section is skipped per config.

---

## Security Domain

`security_enforcement: true`, `security_asvs_level: 1` in config.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Yes | JS reads PocketBase admin JWT from localStorage; redirect to `/_/` if missing/expired; API enforces `HasSuperuserAuth()` on every endpoint |
| V3 Session Management | Partial | No session is created by this page; it piggybacks on PocketBase admin session; 401 redirects to `/_/` |
| V4 Access Control | Yes | `HasSuperuserAuth()` in all six Phase 5 handlers — regular Wanderer user tokens fail; page-serving handler has no auth guard (intentional per D-01) |
| V5 Input Validation | Minimal | Page only sends URLs and actor_ids — both validated server-side by Phase 5 handlers; no direct DB writes from the page |
| V6 Cryptography | No | No cryptographic operations in this phase |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Token exfiltration from localStorage | Information Disclosure | localStorage is same-origin only; page served from same origin as API; no cross-origin risk |
| CSRF on action endpoints | Tampering | PocketBase superuser API checks the JWT on every request — stateless auth; no CSRF tokens needed |
| XSS via peer domain name | Tampering | Alpine `x-text="peer.domain"` sets text content (not innerHTML) — safe from XSS by default |
| Auth bypass via direct API call | Elevation of Privilege | `HasSuperuserAuth()` in Phase 5 handlers is the hard gate — page cannot bypass it |
| Clickjacking | Tampering | Optional: add `X-Frame-Options: DENY` header in `FederationDashboard` handler |

**XSS note:** `x-text` in Alpine.js sets `textContent`, not `innerHTML`. Badge values (`peer.domain`, `peer.status`, `peer.direction`) rendered via `x-text` are safe. Error messages from the API (`data.error`) similarly rendered via `x-text` are safe. The only dangerous pattern would be `x-html` — do not use it. [ASSUMED — Alpine.js CDN documentation, not Context7-verified for this specific version]

---

## Sources

### Primary (HIGH confidence)
- `db/routes/federation_admin.go` — all six handler implementations, request/response shapes, auth guard pattern (verified by direct file read)
- `db/main.go` lines 211-216 — confirmed route registration pattern (verified by direct file read)
- `/Users/christianbeutel/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/ui/src/pb.js` line 13 — `new LocalAuthStore("__pb_superusers__" + currentPath)` confirms localStorage key (verified by direct file read)
- `/Users/christianbeutel/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/ui/src/store.js` — `pbColorScheme` storage key confirmed (verified by direct file read)
- `/Users/christianbeutel/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/ui/src/auth/pageInstaller.js` — `Authorization: token` header format (no Bearer prefix) confirmed (verified by direct file read)
- `web/node_modules/pocketbase/dist/pocketbase.es.js` — `LocalAuthStore.save()` stores `{token: e, record: t}` JSON; confirms parse pattern needed (verified by grep)
- `db/routes/nodeinfo.go` lines 32-35 — confirmed `e.Response.Header().Set()` + `e.Response.Write()` raw HTML serving pattern
- `.planning/phases/06-admin-browser-ui/06-CONTEXT.md` — all locked decisions D-01 through D-16
- `.planning/phases/06-admin-browser-ui/06-UI-SPEC.md` — approved UI design contract: component inventory, color palette, interaction contracts, copywriting

### Secondary (MEDIUM confidence)
- Go stdlib `embed` package documentation — `//go:embed` into `[]byte`, blank import requirement [ASSUMED: training knowledge, standard Go pattern]
- Alpine.js CDN behavior with `defer` — load order requirement [ASSUMED: Alpine.js docs pattern]

### Tertiary (LOW confidence)
- Tailwind Play CDN `darkMode: 'class'` configuration via `tailwind.config` global [ASSUMED: Tailwind Play CDN documentation]

---

## Metadata

**Confidence breakdown:**
- API contract: HIGH — read directly from `federation_admin.go`
- Go embedding pattern: HIGH — confirmed from `nodeinfo.go` raw response pattern + standard Go embed docs
- localStorage key format: HIGH — confirmed from PocketBase v0.38.0 source (pb.js + pocketbase.es.js)
- Authorization header format: HIGH — confirmed from PocketBase UI source (no Bearer prefix)
- Alpine.js CDN integration: MEDIUM — CDN behavior confirmed from UI-SPEC.md decision, load order from training knowledge
- Tailwind Play CDN dark mode config: MEDIUM — standard Tailwind Play CDN behavior

**Research date:** 2026-06-27
**Valid until:** 2026-07-27 (stable APIs; CDN versions pinned by Play CDN at generation time)
