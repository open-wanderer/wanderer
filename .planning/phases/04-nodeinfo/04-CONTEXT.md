# Phase 4: NodeInfo - Context

**Gathered:** 2026-06-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Serve two well-known HTTP endpoints so that peer ActivityPub instances and federation tools can identify this deployment as Wanderer software and inspect live usage statistics:

1. `GET /.well-known/nodeinfo` — discovery document listing available NodeInfo schema versions with a link to the 2.1 endpoint
2. `GET /.well-known/nodeinfo/2.1` — NodeInfo 2.1 payload with `software.name: "wanderer"`, `software.version` from env, and current user/post counts

**In scope:** SAFE-04 only
**Out of scope:** WebFinger extension for instance actor discovery (v2 DISC-01), web/mobile admin UI

</domain>

<decisions>
## Implementation Decisions

### Version String

- **D-01:** `software.version` is sourced from the `WANDERER_VERSION` environment variable. If the variable is unset (local dev), fall back to `"dev"`. `web/package.json` is the single source of truth for the version — CI/Docker reads it and injects `WANDERER_VERSION` at build time so both frontend and backend report the same value.

### Post Count (localPosts)

- **D-02:** `usage.localPosts` = COUNT of rows in the `trails` collection WHERE `public = true`. Trails are the primary ActivityPub content type; private trails are excluded since they never leave the instance.

### User Count (users.total)

- **D-03:** `usage.users.total` = COUNT of all rows in the `users` collection. No filtering needed — the instance actor lives in `activitypub_actors`, not `users`, so every row in `users` is a real human account.

### Claude's Discretion

- Response content type and caching headers (standard `application/json; profile="..."` per NodeInfo spec, no cache by default or short TTL — Claude picks)
- Whether to split the two handlers into one file (`nodeinfo.go`) or two

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and Roadmap
- `.planning/REQUIREMENTS.md` §"Safety and Correctness" — SAFE-04: exact requirement text for what the NodeInfo endpoints must return
- `.planning/ROADMAP.md` §"Phase 4: NodeInfo" — success criteria (2 criteria: discovery doc + NodeInfo 2.1 payload)

### Route Registration Pattern
- `db/main.go` `registerRoutes()` — where to add the two new `GET` routes; all custom routes are registered here via `se.Router.GET(...)`
- `db/routes/health.go` — canonical minimal handler pattern: `e.JSON(http.StatusOK, map[string]any{...})`

### Database Query Pattern (for counts)
- `db/federation/activity.go` — `followerInboxes()` shows the PocketBase `RecordQuery` + `Count()` pattern for querying collections
- `db/routes/activitypub.go` — shows how `e.App` is used inside route handlers to access the database

### NodeInfo Specification
- NodeInfo 2.1 schema: `http://nodeinfo.diaspora.software/ns/schema/2.1` (the `links[].rel` value and the schema URL for `Content-Type` profile)
- Discovery doc format: `https://nodeinfo.diaspora.software/protocol.html` (the `/.well-known/nodeinfo` JSON structure with `links` array)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `e.JSON(http.StatusOK, ...)` — used in every route handler; the NodeInfo handlers use this exact pattern
- `os.Getenv("ORIGIN")` — already used throughout `db/federation/` for base URL construction; `WANDERER_VERSION` follows the same pattern
- `e.App.RecordQuery("collection").Count()` or `e.App.FindAllRecords(...)` — for querying `users` and `trails` counts; researcher should verify the exact PocketBase v0.26 count API

### Established Patterns
- One file per logical group in `db/routes/` — new file `db/routes/nodeinfo.go` with one or two exported handler functions
- Route registration in `registerRoutes()` in `db/main.go` — add two `se.Router.GET(...)` calls there
- No authentication on well-known endpoints — consistent with how `/.well-known/webfinger` is handled (public endpoint)

### Integration Points
- `registerRoutes()` in `db/main.go:168` — add `se.Router.GET("/.well-known/nodeinfo", routes.NodeInfo)` and `se.Router.GET("/.well-known/nodeinfo/2.1", routes.NodeInfo21)` here
- Researcher should verify that `se.Router` in PocketBase v0.26 supports root-level paths like `/.well-known/...` (not prefixed with `/api/v1/`); if not, an alternative registration path (e.g., `se.App.Router()`) may be needed

</code_context>

<specifics>
## Specific Ideas

- User wants `web/package.json` as the single source of truth for the version — `WANDERER_VERSION` env var is the bridge between the JS ecosystem (package.json) and the Go backend
- The version string should be consistent across both the SvelteKit frontend and the Go backend NodeInfo response

</specifics>

<deferred>
## Deferred Ideas

- WebFinger extension to resolve instance actor IRI (v2 DISC-01, already in REQUIREMENTS.md) — out of scope for this phase
- Active user count (users active in last 30/180 days for `users.activeMonth`/`users.activeHalfyear`) — NodeInfo has these fields but they're optional; not discussed, Claude may include them as 0 or omit

</deferred>

---

*Phase: 04-nodeinfo*
*Context gathered: 2026-06-26*
