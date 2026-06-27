# Phase 6: Admin Browser UI - Context

**Gathered:** 2026-06-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver a Go-embedded HTML page at `/federation/` that lets a PocketBase superuser manage all peer instance connections from a browser — paste a URL, discover the remote instance, connect (outbound Follow), view all peers with status/direction, approve or reject inbound pending follows, and disconnect — without curl or the PocketBase admin panel.

**In scope (Phase 6):**
- `GET /federation/` — serves the self-contained HTML page (Alpine.js + Tailwind CDN)
- Auth gate: unauthenticated requests redirect to `/_/`; authenticated page reads superuser JWT from PocketBase admin localStorage key, attaches it as Bearer on all API calls
- Discovery form: paste URL → POST /federation/discover → preview card (domain, version, user/trail counts) → Connect button
- Peer table: lists all peers from GET /federation/peers with direction (Outbound/Inbound/Mutual) and status badges; inline action buttons per row
- Inbound-pending actions: Approve (POST /approve/:id) + Reject (POST /reject/:id)
- Accepted peer action: Disconnect (POST /disconnect/:id) with browser confirm() guard
- Post-action: peer list refreshes automatically (re-fetch /federation/peers)
- Theme: respects `pbColorScheme` localStorage key (`"light"`/`"dark"`); falls back to `prefers-color-scheme`

**Out of scope (Phase 6):**
- Realtime peer list updates via PocketBase subscription (v2 UX-01)
- Refresh peer metadata button (v2 UX-02)
- Connection activity log (v2 UX-03)
- Email notification for inbound follows (v2 UX-04)
- WebFinger resolution (v2 DISC-03)

</domain>

<decisions>
## Implementation Decisions

### Auth Flow

- **D-01:** The page is served as a public HTML document at `GET /federation/`. The Go handler serves the HTML without requiring a token on the initial page load (the JS handles auth).
- **D-02:** On page load, JavaScript reads `localStorage.getItem("__pb_superusers__/_")` — the key PocketBase admin panel (`/_/`) stores its superuser JWT under (`new LocalAuthStore("__pb_superusers__" + currentPath)` where `currentPath` is `/_` for a root-hosted instance).
- **D-03:** If the key is missing or the token is expired, the page immediately redirects to `/_/` (PocketBase admin login). After the admin logs in at `/_/`, they navigate back to `/federation/` manually.
- **D-04:** If a valid token is found, it is attached as `Authorization: <token>` on every `fetch()` call to the six federation API endpoints. Token is never written anywhere by the `/federation/` page itself — it only reads what `/_/` already stored.
- **D-05:** The existing API endpoints already enforce `HasSuperuserAuth()` and return 401 for non-superuser tokens — the page relies on that; it does not add a second layer.

### JS / Styling Approach

- **D-06:** The page uses **Alpine.js** (loaded via CDN) for reactivity — `x-data`, `x-bind`, `x-on`, `x-show`, `@click`. No build step; the HTML is a single self-contained file embedded in the Go binary.
- **D-07:** Styling uses **Tailwind CSS Play CDN** — generates utility classes on-demand in the browser. Same Tailwind classes and color tokens as the rest of Wanderer (e.g., `bg-[#242734]`, `text-white`, standard spacing/rounding/shadow utilities).
- **D-08:** Dark/light theme is determined at page load:
  1. Read `localStorage.getItem("pbColorScheme")` — values are `"light"` or `"dark"`.
  2. If missing, use `window.matchMedia("(prefers-color-scheme: dark)").matches`.
  3. Apply `dark` class to `<html>` so Tailwind `dark:` variants activate correctly.
- **D-09:** The HTML file is embedded in the Go binary via `//go:embed` directive pointing to a separate `db/routes/federation_ui.html` file. The Go handler reads and serves it with `Content-Type: text/html; charset=utf-8`.

### Discovery UX Flow

- **D-10:** Two-step discovery: admin pastes URL → clicks "Discover" → page calls `POST /federation/discover` → on success, shows a **preview card** (domain, Wanderer version, user count, trail count) and a **Connect** button. Admin reviews the preview before committing.
- **D-11:** If discovery fails, show an inline error below the URL input (not a modal or toast) with the error message from the API response (`{"error":"..."}` field).
- **D-12:** On "Connect" click, page calls `POST /federation/follow` with the `actor_id` from the discovery response. On success, the discovery form resets and the peer list is refreshed immediately (re-fetch `/federation/peers`). The new peer appears as "Outbound / Pending" in the table.

### Peer List

- **D-13:** The peer table columns: **Domain**, **Direction** (Outbound / Inbound / Mutual), **Status** (pending / accepted / rejected), **Actions**.
- **D-14:** Inline action buttons per row — only the actions valid for that row's state are rendered:
  - `status=pending AND direction=inbound` → **Approve** + **Reject** buttons
  - `status=accepted` (any direction) → **Disconnect** button
  - `status=pending AND direction=outbound` → no action (waiting for remote to respond); optionally a disabled "Pending…" indicator
  - `status=rejected` → no action (tombstone row; may be hidden or shown greyed out)
- **D-15:** **Disconnect** shows `window.confirm("Disconnect from <domain>? This will send an Undo{Follow} or Reject{Follow} to the remote instance.")` before calling the API.
- **D-16:** After any action (Approve, Reject, Disconnect), the peer list re-fetches `/federation/peers` to show the updated state.

### Claude's Discretion

- Page title, heading, and copy text (e.g., "Federation Peers", "Connect a new instance", etc.)
- Whether rejected peers are shown in the table or filtered out
- Loading/spinner states during API calls
- Exact Tailwind class choices for buttons (primary vs secondary variants)
- Whether the discovery preview card is dismissible (cancel) or only cleared on Connect or error

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and Roadmap
- `.planning/REQUIREMENTS.md` §"v1.1 Requirements" — DASH-01, DASH-02 (acceptance gates for this phase)
- `.planning/ROADMAP.md` §"Phase 6: Admin Browser UI" — success criteria (3 items); these are the acceptance gates

### Phase 5 API (already implemented — page consumes these)
- `db/routes/federation_admin.go` — all six handler implementations: `FederationDiscover`, `FederationFollow`, `FederationApprove`, `FederationReject`, `FederationDisconnect`, `FederationPeers`; understand the request/response shapes before building the page
- `db/main.go` lines 211–216 — route registration confirming the six endpoints are live

### PocketBase Admin Auth
- `/Users/christianbeutel/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/ui/src/pb.js` line 13 — `new LocalAuthStore("__pb_superusers__" + currentPath)` — confirms the localStorage key the page must read to piggyback on the admin session

### Go Embedding Pattern
- `db/util/email_templates.go` — existing use of `html/template` in the db module; confirms the import path and template registry pattern available if Go templates are needed
- `db/templates/mail/notification.html` — existing embedded HTML file; confirms the `db/templates/` tree is a precedent for embedding HTML in the Go binary

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Six federation API endpoints at `/federation/*` (Phase 5) — fully implemented, auth-guarded, tested; the page is a pure consumer with no backend changes needed
- `db/util/safe_fetch.go` `NewSafeURLClient` — not needed by the page but confirms the SSRF pattern is already in util; page does not make outbound HTTP calls itself

### Established Patterns
- Go `//go:embed` for HTML: the project embeds `db/templates/mail/notification.html` via PocketBase's template registry; the same pattern works for a raw HTML file served directly by a handler
- PocketBase route handler pattern: `func Handler(e *core.RequestEvent) error` returning `e.JSON(...)` or writing `e.Response.Write(...)` for raw content — see `db/routes/nodeinfo.go` and `db/routes/health.go`
- Auth guard: `e.HasSuperuserAuth()` is the idiomatic check (already used on all six Phase 5 handlers); the page-serving handler does NOT need this guard — the page itself handles auth in JS

### Integration Points
- `db/main.go` `registerRoutes()` — add `se.Router.GET("/federation/", routes.FederationDashboard)` alongside the six existing federation routes
- The HTML file lives at `db/routes/federation_ui.html` and is embedded via `//go:embed federation_ui.html` in `db/routes/federation_admin.go` (or a new `db/routes/federation_ui.go`)

</code_context>

<specifics>
## Specific Ideas

- The page should visually match the PocketBase admin panel aesthetic (dark sidebar `#242734`, clean table layout) so it feels like a native extension of the admin panel, not a foreign page
- Tailwind Play CDN enables using the exact same utility classes as the Wanderer web app — same `bg-*`, `text-*`, `rounded-*` tokens without any build step
- The `pbColorScheme` localStorage key is what PocketBase uses for its own dark/light toggle; reading it makes the federation page respect the admin's existing preference without any extra UI

</specifics>

<deferred>
## Deferred Ideas

- **Realtime peer list** — polling or PocketBase subscription for live status updates → v2 UX-01
- **Refresh metadata button** — re-fetch NodeInfo for an existing peer to update version/user counts → v2 UX-02
- **Connection activity log** — timeline of Follow/Accept/Reject/Undo events per peer → v2 UX-03
- **Manual page refresh** — the current design re-fetches on every action; a manual "Refresh" button is not needed but could be added in v2

</deferred>

---

*Phase: 6-Admin Browser UI*
*Context gathered: 2026-06-27*
