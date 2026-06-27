# Project Research Summary

**Project:** Wanderer Instance Federation Connect UI (v1.1)
**Domain:** ActivityPub admin UI -- peer instance connection management
**Researched:** 2026-06-27
**Confidence:** HIGH

## Executive Summary

v1.1 builds an admin-only UI on top of the v1.0 ActivityPub protocol layer, which is already fully shipped. The core insight from research is that the work here is almost entirely a UI and routing problem, not a protocol problem: all ActivityPub delivery is already handled by three existing hooks (InstanceFollowCreate/Update/DeleteHandler). The new admin handlers only need to write DB records correctly -- the hooks fire automatically and take care of delivery. This keeps the implementation surface small and the risk low.

The recommended approach is a custom Go route group (/federation/*) protected by e.HasSuperuserAuth(), serving a JSON API for five admin operations: list peers, discover remote actor, initiate follow, approve/reject inbound, and disconnect. This is the same pattern already used by plugin_system.go in this codebase, requires zero new Go dependencies, and makes no SvelteKit changes. The admin interacts either through a minimal embedded HTML page or via curl with a superuser JWT. The only unresolved architectural question -- see INGEST-CONFLICTS.md -- is whether the page lives as a custom Go-served page (PocketBase superuser auth) or a SvelteKit settings page (Wanderer is_admin user flag); that decision needs to be made in requirements and will drive the UI phase structure.

The top risks are all implementation-level: double-delivery of ActivityPub activities (if the route handler also calls federation functions that the hook already calls), incorrect Undo direction on disconnect for inbound-only connections, SSRF from admin-supplied URLs if util.SafeHTTPClient() is not used, and the 2-hour actor cache silently returning stale data during discovery. All seven critical pitfalls have concrete, code-level mitigations that can be turned into a pre-commit checklist.

## Key Findings

### Recommended Stack

No new Go dependencies are required. The entire implementation uses packages already in go.mod: e.HasSuperuserAuth() and apis.RequireSuperuserAuth() are available in pocketbase/pocketbase/apis at v0.38.0 (the version in use), embed.FS is standard library (Go 1.16+), and apis.Static() handles embedded file serving. The PocketBase JS SDK at v0.26.8 is already in web/package.json and can be used from a CDN in a standalone HTML page. If the SvelteKit path is chosen, it reuses 100% of the existing Vite/Svelte toolchain with only a new page and a migration.

The PocketBase UI extension API is explicitly ruled out -- the PocketBase maintainer marked it not production-safe (discussion #7612), and it will break on the next PocketBase upgrade.

**Core technologies:**
- e.HasSuperuserAuth() / apis.RequireSuperuserAuth(): route auth guard -- already proven in db/routes/plugin_system.go, zero uncertainty
- embed.FS + apis.Static(): serve embedded HTML from Go binary -- standard library, no new dependency
- PocketBase JS SDK (0.26.8): browser-side auth and API calls -- already in web/package.json
- util.SafeHTTPClient(): SSRF-safe HTTP for admin-supplied URLs -- already in codebase, must be used explicitly

### Expected Features

**Must have (table stakes -- all P1 for v1.1):**
- Paste-a-URL connect flow with server-side discovery (NodeInfo + actor fetch)
- Software identity check blocking non-Wanderer instances at preview time
- Remote instance preview card before committing to a Follow
- Inline error messages for all connect failure modes (7 distinct error codes)
- Peer dashboard with status (pending/accepted/rejected) and direction (Outbound/Inbound/Mutual)
- Approve / Reject inbound pending follows (with ActivityPub Accept{Follow} / Reject{Follow} delivery)
- Atomic disconnect that handles both directions in one DB transaction
- Admin-only access guard (mechanism TBD -- see INGEST-CONFLICTS.md)

**Should have (differentiators, P2):**
- Direction column (Outbound / Inbound / Mutual) in peer list
- Pending inbound request context (received timestamp, remote actor name, domain)
- Remote instance metadata (Wanderer version, user count, trail count) shown at connect time
- PocketBase realtime subscription on follows collection so status updates without manual refresh

**Defer (v1.2+):**
- Refresh peer metadata button
- Connection activity log (timeline of Follow/Accept/Reject/Undo events)
- Domain blocklist
- Email notification on inbound pending follow
- WebFinger instance actor (@instance@domain handle, depends on FEP-d556 stabilising)

### Architecture Approach

v1.1 is additive only. A single new file db/routes/federation_admin.go registers six handler functions on a /federation/* route group. No hooks change. No migrations are needed for the Go-route path. All federation delivery continues through the existing three hooks which fire automatically after any follows record write. The handlers only manipulate DB records and return JSON; the hook layer owns all ActivityPub activity delivery. FederationPeersList is the only query with complexity -- it needs an OR filter across both follow directions and a group-by-domain step to merge mutual follows.

**Major components:**
1. db/routes/federation_admin.go (NEW) -- six handler functions behind e.HasSuperuserAuth() guard; no delivery logic, only DB writes
2. Existing hooks (db/hooks/follow.go) -- unchanged; fire on every follows write and deliver ActivityPub activities
3. db/federation/actor.go GetActorByIRI() -- reused for remote actor discovery; cache bypass needed at discovery time
4. Admin browser UI -- minimal HTML page (embedded) or SvelteKit /settings/federation page; calls the JSON API with superuser JWT
5. db/main.go -- modified only to register the six new routes

### Critical Pitfalls

1. **Double Follow delivery** -- The Initiate Follow handler must ONLY call app.Save(followRecord). Calling federation.CreateFollowActivity() directly in the handler as well causes two Follow activities to the remote inbox. The hook owns delivery exclusively.

2. **Incorrect Undo direction on disconnect** -- Deleting an inbound follow record triggers InstanceFollowDeleteHandler which sends Undo{Follow}, but the local instance never sent that Follow -- the remote did. For inbound-only disconnects, send Reject{Follow} (status update to rejected) instead of a hard delete.

3. **SSRF from admin-supplied URL** -- Any HTTP call in the new route file that fetches a user-supplied URL must use util.SafeHTTPClient(), not http.DefaultClient or the module-level httpClient from federation/activity.go. Both compile without error but bypass SSRF protection.

4. **Stale actor cache during discovery** -- GetActorByIRI has a 2-hour TTL. The discovery endpoint must clear last_fetched to zero time before calling it so the admin always gets fresh authoritative actor data, not a cached stale record.

5. **Self-follow loop** -- If an admin pastes their own instance URL, GetActorByIRI returns the local actor. A follows record with follower == followee triggers the hook which POSTs a Follow to the local inbox, which fires ProcessFollowActivity again. Guard: compare resolved actor IRI against os.Getenv("ORIGIN") + "/api/v1/activitypub/instance" before creating any follows record.

6. **Wrong auth guard** -- e.Auth != nil checks only that a valid token was presented. Any regular Wanderer user can pass this check. All admin federation routes must use e.HasSuperuserAuth() exclusively.

7. **Missing activitypub_activities record blocks Approve** -- CreateAcceptFollowActivity looks up the original incoming Follow from activitypub_activities. If that record is missing, the approve endpoint returns 500. Return 409 with a descriptive message instead, and check for the records existence before saving the status update.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Go API Layer (federation_admin.go)
**Rationale:** All five UI flows depend on these endpoints. Building them first makes everything else testable via curl/Postman without any frontend. No dependencies on other phases.
**Delivers:** Six handler functions with superuser auth guard; all five admin operations (list, discover, follow, approve/reject, disconnect) callable from the command line.
**Addresses:** All P1 features except the browser UI
**Avoids:** Pitfalls 1, 2, 3, 5, 6, 7, 8 (all implementation-level; must be addressed here)
**Research flag:** Standard patterns -- e.HasSuperuserAuth() is already in the codebase; no additional research needed.

### Phase 2: Peer List Query and Direction Logic
**Rationale:** FederationPeersList is the most complex query in v1.1 (OR filter across both directions, mutual-follow detection, actor join). Keeping it as a separate phase ensures correct data shape before UI is built on top of it.
**Delivers:** /federation/peers returns enriched peer records with direction label (Outbound/Inbound/Mutual), status, remote actor domain, and metadata.
**Implements:** Peer list query with FindRecordsByFilter OR composite, actor join, mutual-follow grouping by domain
**Avoids:** Pitfall 10 (stale dashboard state -- query returns current DB state; realtime subscription added in Phase 3 UI)

### Phase 3: Admin UI (browser page)
**Rationale:** Depends on Phase 1 + 2 API being complete and correct. UI path is the unresolved conflict (Go-served HTML vs SvelteKit settings page) -- see INGEST-CONFLICTS.md. Phase structure is the same regardless of which path is chosen, but the implementation differs significantly.
**Delivers:** Browser-accessible admin page with all four UX flows (Connect, Approve, Reject, Disconnect), inline error messages, direction-aware peer list, and realtime status updates via PocketBase follows subscription.
**Uses:** PocketBase JS SDK (0.26.8) for auth + API calls; either embedded HTML (Go path) or SvelteKit page (SvelteKit path)
**Implements:** All P1 and P2 features
**Avoids:** Pitfalls 9 (token expiry handling), 10 (stale dashboard), UX pitfalls (feedback on every action)
**Research flag:** Needs resolution of admin auth conflict before planning (see INGEST-CONFLICTS.md). UI implementation itself follows standard patterns once the decision is made.

### Phase Ordering Rationale

- Phase 1 before Phase 2: the list endpoint is a subset of the full API layer but benefits from being built after the write endpoints are understood, because the data shape it must return is determined by what the write endpoints store.
- Phase 2 before Phase 3: the UI has no value without a correct peer list query. Building the data model first avoids frontend rework.
- All three phases are small enough to ship in a single milestone, but the phase separation ensures each phase is independently testable and reviewable.
- The admin auth decision (INGEST-CONFLICTS.md) does not block Phases 1 or 2 -- the Go API endpoints are identical regardless of which UI path is chosen. It only affects Phase 3.

### Research Flags

Phases that need deeper research during planning:
- **Phase 3 (Admin UI):** Blocked on admin auth conflict resolution. Once resolved, the chosen path (embedded HTML vs SvelteKit page) may benefit from a brief research pass on the specific integration details. The Go-embedded-HTML path is lower risk (proven pattern); the SvelteKit path needs PocketBase collection rules analysis for the is_admin guard on follows PATCH operations.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Go API Layer):** All patterns (e.HasSuperuserAuth(), app.Save(), app.FindRecordsByFilter()) are already in the codebase. The pitfall checklist in PITFALLS.md is sufficient context.
- **Phase 2 (Peer List Query):** PocketBase filter DSL is well-documented; OR composite filters are used elsewhere. The mutual-follow grouping is a straightforward in-memory reduce after the query.

## Competing Variant: Admin Auth Approach

**This is the primary unresolved conflict from research.** See .planning/research/INGEST-CONFLICTS.md for full tradeoff analysis.

**Variant A -- Custom Go Route + Superuser JWT (ARCHITECTURE.md recommendation):**
- Admin uses PocketBase superuser credentials (/_/ login)
- UI is a minimal embedded HTML page or pure API (curl/Postman)
- Zero SvelteKit changes, zero migrations
- Auth: e.HasSuperuserAuth() -- already proven in codebase
- Tradeoff: Admin must have PocketBase superuser access, not just a Wanderer account. Two separate logins for admins who use both the Wanderer web app and the federation dashboard.

**Variant B -- SvelteKit Settings Page + is_admin Flag (FEATURES.md recommendation):**
- Admin uses their regular Wanderer account with is_admin: true
- UI lives at /settings/federation alongside existing settings pages (plugins, maintenance, privacy)
- Requires: one migration (is_admin: bool on users), one new SvelteKit page, Go routes accept user token and check is_admin, PocketBase collection rules updated for follows PATCH
- Tradeoff: More invasive but consistent UX. Wanderer admin != PocketBase superuser, which is appropriate for multi-tenant or managed deployments where the instance operator is not the PocketBase DBA.

**Recommendation for requirements step:** Make this the first decision. Both paths are implementable. The choice comes down to deployment context: if the intended admin is always the PocketBase superuser (self-hosted single-operator), Variant A is simpler and lower risk. If the intended admin is a Wanderer user role (managed hosting, multiple admins), Variant B is the right model.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All APIs verified against PocketBase v0.38.0 source; all patterns already exist in this codebase |
| Features | HIGH | Derived from codebase analysis + Mastodon/Pleroma/GoToSocial source review; existing v1.0 hooks verified |
| Architecture | HIGH | Based on direct code analysis of all affected files; build order has no circular dependencies |
| Pitfalls | HIGH | All 11 pitfalls identified from direct code analysis (not inference); all have specific file + line references |

**Overall confidence:** HIGH

### Gaps to Address

- **Admin auth model** (critical): Must be resolved before Phase 3 planning. See INGEST-CONFLICTS.md. Does not block Phase 1 or 2.
- **PocketBase collection rules for follows**: Current rules likely restrict PATCH to the record owner (the user actor). Admin PATCH for approve/reject may require a rule change or a Go-side bypass. Needs verification during Phase 1 planning.
- **NodeInfo data persistence strategy**: For v1.1, re-fetching NodeInfo on every admin page load is acceptable (low-traffic admin page). For v1.2, consider persisting version/user/trail counts to activitypub_actors to avoid repeated outbound requests.
- **PocketBase realtime subscription scope**: The follows subscription must be scoped to instance-actor follows only; subscribing to all follows records would include user-to-user follows. Verify filter syntax for PocketBase realtime subscriptions during Phase 3 planning.

## Sources

### Primary (HIGH confidence)
- PocketBase v0.38.0 source -- core/event_request.go:77, core/record_model_superusers.go:116, apis/middlewares.go:108-113
- PocketBase Go Routing docs -- apis.RequireSuperuserAuth, apis.Static, route groups
- Codebase direct analysis -- db/routes/plugin_system.go, db/hooks/follow.go, db/federation/actor.go, db/federation/follow.go, db/federation/activity.go, db/util/network.go, db/main.go
- PocketBase authentication docs -- stateless JWT, no sessions, _superusers collection
- PocketBase UI extensions discussion #7612 -- maintainer explicitly not production-safe

### Secondary (MEDIUM confidence)
- Mastodon relay model (relay.rb, PR #7998, issue #14961) -- status states, approve/reject patterns, documented UX failures
- Pleroma/Akkoma admin relay API docs -- relay add/remove patterns
- GoToSocial admin settings -- domain permission UI patterns
- FEP-d556 (server-level actor WebFinger) -- confirmed exploratory/unstable, correctly deferred

### Tertiary (LOW confidence -- not needed for v1.1)
- Mastodon follow_requests API -- approve/reject pattern reference only
- Community discussion on go:embed with PocketBase -- confirmed pattern works (not relied upon for correctness)

---
*Research completed: 2026-06-27*
*Ready for roadmap: yes*
