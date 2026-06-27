# Phase 5: Federation Admin API - Context

**Gathered:** 2026-06-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement six Go route handlers for all peer management operations — discover, follow, approve, reject, disconnect, and list peers — as a JSON API served from the Go/PocketBase backend, behind PocketBase superuser auth. No HTML/UI; that is Phase 6. All handlers only write DB records; ActivityPub delivery remains exclusively in hooks (SAFE-07).

**In scope (Phase 5):**
- `POST /federation/discover` — fetch remote NodeInfo + actor, return preview card + actor ID
- `POST /federation/follow` — create outbound follows record (hook fires Follow delivery)
- `POST /federation/approve/:id` — update inbound follow to accepted (hook fires Accept delivery)
- `POST /federation/reject/:id` — update inbound follow to rejected (hook fires Reject delivery)
- `POST /federation/disconnect/:id` — direction-aware: outbound → set deleted/undone (hook fires Undo), inbound-only → set rejected (hook fires Reject)
- `GET /federation/peers` — list all peer connections with follow_id, direction, status, domain

**Out of scope (Phase 5):**
- HTML dashboard page (`/federation/`) — Phase 6
- WebFinger instance actor resolution — v2 DISC-03
- Realtime subscription for peer status updates — v2 UX-01

</domain>

<decisions>
## Implementation Decisions

### Discovery → Follow Coupling

- **D-01:** `POST /federation/discover` is a required first step before connecting. It fetches the remote NodeInfo (via `/.well-known/nodeinfo` discovery doc → NodeInfo 2.1 endpoint) using the SSRF-safe client, verifies `software.name == "wanderer"`, then calls `GetActorByIRI` to store or refresh the remote actor. Returns `{ actor_id, domain, version, user_count, trail_count }`.
- **D-02:** `POST /federation/follow` accepts `{ actor_id }` — the actor ID returned by discovery. If no `activitypub_actors` record exists for that ID, returns `400 Bad Request`. Handler creates the `follows` record with status "pending"; the existing `onAfterSave` hook fires `CreateFollowActivity`. No HTTP calls in the follow handler.
- **D-03:** Discovery must bypass the 2-hour actor cache: clear `last_fetched` on the actor record before calling `GetActorByIRI` so a fresh fetch is performed even if the remote was previously cached.

### Peer List

- **D-04:** `GET /federation/peers` returns a lightweight JSON array per peer connection. Each entry: `{ follow_id, direction, status, domain }`. `direction` is `"outbound"` (local instance follows remote), `"inbound"` (remote follows local), or `"mutual"` (both accepted). The handler queries all follows records involving the local instance actor, groups by remote domain, and collapses two-direction pairs into a single `"mutual"` entry.
- **D-05:** Mutual detection: two `follows` records with status "accepted" where one has local actor as follower and the other has local actor as followee (same remote domain) → collapsed to one entry with `direction="mutual"`. The `follow_id` in a mutual entry refers to the outbound record (used by disconnect).

### Discovery — Remote Identity Verification

- **D-06:** Remote instance is confirmed as Wanderer by checking `software.name == "wanderer"` in the NodeInfo 2.1 response. Any other value (Mastodon, Misskey, etc.) returns a clear error per DISC-02.
- **D-07:** "Instance name" in the preview card is the hostname extracted from the admin-supplied URL (e.g., `"trails.example.com"`). No display name field exists in NodeInfo 2.1; the domain is unambiguous and always available.
- **D-08:** Discovery data flow: admin URL → `/.well-known/nodeinfo` (discover doc) → follow the `links[].href` for `"http://nodeinfo.diaspora.software/ns/schema/2.1"` → fetch NodeInfo 2.1 → extract `software.version`, `usage.users.total`, `usage.localPosts` → call `GetActorByIRI` with the instance actor IRI. All fetches use SSRF-safe client with ≤10s timeout.

### SSRF and Auth (locked from prior phases)

- **D-09:** All admin-supplied URLs are fetched via `util.FetchPublicURL` (wraps `safeurl.Client` with private IP blocking). This is the existing SSRF-safe client in `db/util/safe_fetch.go`. Timeout is already configured; researcher should verify or override to ≤10s per SAFE-06.
- **D-10:** All six endpoints require a valid PocketBase superuser token. Non-superuser Wanderer tokens receive 401. Verification via `e.Auth != nil && e.Auth.Collection().Name == "_superusers"` (researcher to confirm exact PocketBase v0.26 API for superuser detection, and whether a middleware or inline check is preferred).
- **D-11:** Handlers only call `app.Save()` / `app.Delete()`. ActivityPub delivery (Follow, Accept, Reject, Undo) fires exclusively from the existing `onAfterSave`/`onAfterDelete` hooks in `db/hooks/` — no direct calls to federation functions from route handlers.

### Claude's Discretion

- Error response format: use `e.JSON(http.StatusXxx, map[string]any{"error": "..."})` — consistent with all existing handlers
- Route registration: new file `db/routes/federation_admin.go` following the `nodeinfo.go` pattern
- Whether to use a shared superuser-check middleware or inline auth guard per handler
- DISC-02 error messages: "unreachable", "not a Wanderer instance", "already connected", "resolves to local instance" — Claude picks the exact strings

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and Roadmap
- `.planning/REQUIREMENTS.md` §"v1.1 Requirements" — DISC-01, DISC-02, CONN-01–04, SAFE-05, SAFE-06, SAFE-07 (exact requirement text for all Phase 5 requirements)
- `.planning/ROADMAP.md` §"Phase 5: Federation Admin API" — success criteria (5 items); these are the acceptance gates

### Existing Federation Code
- `db/federation/follow.go` — `CreateFollowActivity`, `CreateAcceptFollowActivity`, `CreateRejectFollowActivity`, `ProcessRejectActivity`, `ProcessAcceptActivity` — all follow lifecycle functions already implemented; handlers must NOT call these directly (SAFE-07), only `app.Save()` triggers them via hooks
- `db/federation/actor.go` — `GetActorByIRI` — used by discovery to fetch/cache remote actor; discovery must clear `last_fetched` before calling to bypass cache
- `db/federation/instance.go` — instance actor lookup; researcher should confirm how local instance actor IRI is obtained

### Existing Route Patterns
- `db/routes/nodeinfo.go` — canonical new route file pattern (one file per logical group, registered in `registerRoutes()`)
- `db/routes/activitypub.go` — handler pattern: `e.JSON(http.StatusOK, ...)`, `e.App` for DB access, `e.Auth` for auth record
- `db/routes/health.go` — minimal handler pattern
- `db/main.go` `registerRoutes()` — where to add new `se.Router.POST/GET(...)` calls

### SSRF and Utilities
- `db/util/safe_fetch.go` — `FetchPublicURL(ctx, rawURL, maxBytes)` — SSRF-safe HTTP client using `safeurl` library; use this for all admin-supplied URL fetches (NodeInfo, NodeInfo discovery doc). Verify timeout is ≤10s or create a tighter-timeout wrapper per SAFE-06.

### NodeInfo Protocol
- NodeInfo 2.1 schema: `http://nodeinfo.diaspora.software/ns/schema/2.1` (the `links[].rel` value)
- Discovery doc: `/.well-known/nodeinfo` returns `{ links: [{ rel: "...", href: "..." }] }` — follow `href` to get the actual NodeInfo 2.1 payload

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `util.FetchPublicURL(ctx, rawURL, maxBytes)` — `db/util/safe_fetch.go` — SSRF-safe HTTP; use for all remote fetches in discovery
- `GetActorByIRI(app, ctx, iri, follows)` — `db/federation/actor.go` — fetches remote actor and caches locally; discovery calls this after NodeInfo fetch
- `e.JSON(http.StatusOK, map[string]any{...})` — handler response pattern used throughout `db/routes/`
- `CreateFollowActivity`, `CreateAcceptFollowActivity`, `CreateRejectFollowActivity` — `db/federation/follow.go` — called by hooks, NOT by handlers
- `security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)` — ID generation pattern in `db/federation/follow.go`

### Established Patterns
- Route file per logical group in `db/routes/` — new file `db/routes/federation_admin.go`
- All route handlers registered in `registerRoutes()` in `db/main.go`
- Auth check via `e.Auth` (PocketBase sets this from Bearer token); superuser check via collection name
- `app.FindFirstRecordByFilter(collection, filter, params)` for follow record lookups
- `dbx.Params{"key": value}` for parameterized queries — used extensively in `db/federation/follow.go`
- `e.NotFoundError(...)` and returning `err` for error responses — consistent with `db/routes/activitypub.go`

### Integration Points
- `db/main.go` `registerRoutes()` — add 6 new route registrations
- `db/hooks/` — existing hooks fire on `follows` collection `afterSave`/`afterCreate` — handlers must only call `app.Save()` to stay within SAFE-07
- `db/federation/actor.go` `GetActorByIRI` — discovery calls this; researcher must verify how to clear `last_fetched` to bypass cache
- `activitypub_actors` collection — discovery stores remote actor here; `follows` collection — all connect/approve/reject/disconnect operations write here

### Direction Computation (for GET /federation/peers)
- Mutual detection requires querying the `follows` collection twice (or with OR condition): find where `follower = local_instance_actor_id` AND find where `followee = local_instance_actor_id`. Group by remote domain. If both directions are "accepted", collapse to "mutual".
- Local instance actor ID must be fetched at handler startup: `app.FindFirstRecordByFilter("activitypub_actors", "actor_type={:t} && is_local={:l}", ...)`.

</code_context>

<specifics>
## Specific Ideas

- Two-step UI flow is intentional: discover (shows preview, user confirms) → connect (sends Follow). The `actor_id` returned by discovery is the cursor that links the two steps.
- Instance name in the preview card is the domain name (hostname only), not a friendly display name — acknowledged and accepted for the admin-facing context.
- Wanderer identity check: `software.name == "wanderer"` (exact string match) in NodeInfo 2.1 response.
- NodeInfo fetch sequence: `GET /.well-known/nodeinfo` → follow `links[0].href` (or the href for the 2.1 rel) → `GET {href}` for the payload. Both fetches use the SSRF-safe client.

</specifics>

<deferred>
## Deferred Ideas

- Mutual disconnect behavior (both Undo + Reject in one operation) — not discussed; researcher should reference CONN-04 requirements. Direction-aware disconnect means outbound record gets undone; inbound record gets rejected separately if needed.
- Storing NodeInfo preview data (version, counts) in the actor record for later display without re-fetching — v2 UX-02 (refresh peer metadata button)
- Real-time status updates via PocketBase follows subscription — v2 UX-01
- WebFinger endpoint for instance actor — v2 DISC-03

</deferred>

---

*Phase: 05-federation-admin-api*
*Context gathered: 2026-06-27*
