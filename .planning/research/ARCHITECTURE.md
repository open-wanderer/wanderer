# Architecture Research: Federation Connect UI (v1.1)

**Domain:** Admin UI for ActivityPub instance federation management
**Researched:** 2026-06-27
**Confidence:** HIGH — based on direct codebase analysis and PocketBase 0.38.0 source inspection

---

## Context

This document supersedes the v1.0 ARCHITECTURE.md (which covered instance actor creation and fanout). v1.0 is fully shipped. This document focuses exclusively on v1.1: an admin-only UI for managing peer instance connections.

---

## Answered: The Four Key Questions

### 1. How Does PocketBase 0.38 Expose Superuser Auth?

**Confirmed mechanism** (from PocketBase 0.38.0 source at `core/event_request.go:77` and `core/record_model_superusers.go:116`):

```go
// core/event_request.go
func (e *RequestEvent) HasSuperuserAuth() bool {
    return e.Auth != nil && e.Auth.IsSuperuser()
}

// core/record_model_superusers.go
func (m *Record) IsSuperuser() bool {
    return m.Collection().Name == CollectionNameSuperusers  // "_superusers"
}
```

`e.Auth` is populated automatically by PocketBase's built-in auth-loading middleware (registered globally for all routes). It parses the `Authorization: Bearer <token>` header, validates the JWT, and populates `e.Auth` with the matching `_superusers` record if it is a superuser token.

**No manual token parsing is needed.** A custom Go route guard is simply:

```go
func FederationAdminHandler(e *core.RequestEvent) error {
    if !e.HasSuperuserAuth() {
        return apis.NewUnauthorizedError("superuser authentication required", nil)
    }
    // handler logic
}
```

This pattern is already used in `db/routes/plugin_system.go:15` and `db/hooks/plugin_instances.go:22`. It is established, tested, and correct.

PocketBase also provides a first-class middleware for this: `apis.RequireSuperuserAuth()`, which can be bound per-route:

```go
se.Router.GET("/federation/peers", routes.FederationPeersList).
    Bind(apis.RequireSuperuserAuth())
```

Either approach (inline guard or bound middleware) works. The inline guard matches existing codebase style.

**The superuser's JWT is obtained from the PocketBase admin panel login** (`/_/` — the PocketBase dashboard). The admin uses that token in API calls. The UI (whichever option is chosen) makes requests with `Authorization: Bearer <superuser_token>`.

### 2. Remote Actor Fetch Flow

The complete flow already exists in `db/federation/actor.go`. No new capability is needed — only a new API handler that calls the existing function.

```
Admin pastes URL: "https://peer.example.com"
    ↓
UI derives instance actor IRI:
    Append "/api/v1/activitypub/instance" → "https://peer.example.com/api/v1/activitypub/instance"
    ↓
POST /federation/actors/fetch  { iri: "https://peer.example.com/api/v1/activitypub/instance" }
    ↓
Go handler calls: federation.GetActorByIRI(app, ctx, iri, false)
    ↓
GetActorByIRI (actor.go:98):
    1. Checks activitypub_actors WHERE iri = <iri>
    2. If cache miss: creates a stub record with is_local=false, iri=<iri>
    3. Calls assembleActor() → calls fetchRemoteActor()
    ↓
fetchRemoteActor (actor.go:287):
    1. HTTP GET <iri> with Accept: application/ld+json
    2. Decodes response body into pub.Actor struct
    3. Validates required fields (ID, inbox, outbox, public key)
    4. Returns populated *pub.Actor
    ↓
assembleActor() sets all fields on the DB record and calls app.Save()
    → activitypub_actors record created with actor_type unset (remote actors don't have it)
    ↓
Handler returns the saved record (id, iri, domain, inbox, preferred_username)
```

The UI stores the returned actor record ID. This ID is then used as `followee` when creating the follow record.

**Important:** `fetchRemoteActor` tries to sign the outbound request if a user actor with a private key is available in context. For the admin UI flow, no signing actor is in context, so the request goes unsigned. This is fine — Wanderer does not enforce authorized-fetch mode (WebFinger/authorized-fetch is deferred to v1.2).

### 3. New Go API Handlers Required

Six new handlers in a new file `db/routes/federation_admin.go`:

| Handler | Method | Path | Purpose |
|---------|--------|------|---------|
| `FederationPeersList` | GET | `/federation/peers` | List all instance follows with status and direction |
| `FederationActorFetch` | POST | `/federation/actors/fetch` | Fetch remote actor by IRI, create local record |
| `FederationFollowCreate` | POST | `/federation/follows` | Create a follows record (initiates Follow activity via existing hook) |
| `FederationFollowApprove` | PATCH | `/federation/follows/{id}/approve` | Set follow status = "accepted" (triggers Accept via existing hook) |
| `FederationFollowReject` | PATCH | `/federation/follows/{id}/reject` | Set follow status = "rejected" (triggers Reject via existing hook) |
| `FederationFollowDelete` | DELETE | `/federation/follows/{id}` | Delete follow record (triggers Undo via existing hook) |

All six require `e.HasSuperuserAuth()` at the top. All federation activity delivery is handled by the existing hooks — the handlers only manipulate DB records.

`FederationPeersList` must query from both directions. A peer can appear as:
- The **followee** when this instance initiated the connection (outbound follow, local instance = follower)
- The **follower** when the peer initiated the connection (inbound follow, local instance = followee)
- Both directions (mutual follow after both sides accepted)

The query joins `follows` with `activitypub_actors` twice (follower actor + followee actor) and filters for records where either side is the local instance actor.

### 4. Integration With Existing Hooks in db/hooks/follow.go

The existing hooks are the delivery mechanism. The admin UI handlers only write DB records; hooks fire automatically and handle all ActivityPub delivery. No hook changes are needed for v1.1.

| Admin Action | DB Write | Hook Fires | Delivery |
|---|---|---|---|
| Click "Connect" | Create `follows` record (follower=instanceActor, followee=remoteActor, status="pending") | `InstanceFollowCreateHandler` (OnRecordAfterCreateSuccess) | `CreateFollowActivity` — POST Follow to remote inbox |
| Click "Approve" | PATCH `follows.status = "accepted"` | `InstanceFollowUpdateHandler` (OnRecordAfterUpdateSuccess) | `CreateAcceptFollowActivity` — POST Accept to remote inbox |
| Click "Reject" | PATCH `follows.status = "rejected"` | `InstanceFollowUpdateHandler` (OnRecordAfterUpdateSuccess) | `CreateRejectFollowActivity` — POST Reject to remote inbox |
| Click "Disconnect" | DELETE `follows` record | `InstanceFollowDeleteHandler` (OnRecordAfterDeleteSuccess) | `CreateUnfollowActivity` — POST Undo{Follow} to remote inbox |

The handler only modifies the DB record and returns the updated state. The hook fires asynchronously after the DB write commits.

**One edge case:** `FederationFollowApprove` updates a follow where the **local instance is the followee** (approving an inbound request). `InstanceFollowUpdateHandler` checks `!isOutboundInstanceFollow()` before acting — this correctly fires for inbound-follow approval. The handler only needs to set `status = "accepted"` and call `app.Save()`.

---

## Architecture Decision: Option A — Custom Go Routes (Chosen)

### Why Option A wins

**Option A (custom Go route, HTML served by PocketBase):**
- `e.HasSuperuserAuth()` is already proven in the codebase — zero uncertainty on auth mechanism
- No SvelteKit changes required (consistent with the v1.0 constraint and the existing CLAUDE.md constraint)
- The admin UI is an infrequent operation (connecting instances is a one-time setup). A minimal HTML page is sufficient.
- The Go binary already has access to all required data (activitypub_actors, follows) — no additional proxying needed
- The PocketBase superuser JWT is the natural credential — same token the admin already has from `/_/`

**Option B (SvelteKit settings page) is rejected because:**
- PocketBase superusers (`_superusers` collection) have no corresponding `users` record. They cannot authenticate to SvelteKit's `event.locals.pb` as a regular user. Adding `is_admin: bool` to users would require creating a separate "wanderer admin" account separate from the PocketBase superuser — two admin identities for one role is confusing and increases attack surface.

**Option C (static HTML in pb_public/) is rejected because:**
- No `pb_public/` directory exists in the current codebase and PocketBase does not serve from one by default — it would need to be configured.
- Auth checking would still require a Go route to validate the superuser token, making it not simpler than Option A.
- The static file lives outside the binary, complicating deployment.

### Selected Implementation: Option A with a dedicated Go route group

A single new file `db/routes/federation_admin.go` exposes a JSON API. The UI is served from a minimal HTML page embedded in the binary via `go:embed` or served inline from a Go template.

**Recommended sub-option: JSON API only, no HTML served by Go.** The admin calls the JSON API directly from the browser's PocketBase admin panel or from curl. This is the simplest possible implementation for v1.1:

- No HTML/CSS to write or maintain
- The admin authenticates once in `/_/` and uses the superuser JWT in API calls
- Endpoints return JSON: peer list, actor fetch result, follow operation result
- A thin HTML page can be added in v1.2 if desired

If a UI page is required in v1.1 scope: serve a single minimal HTML file from Go using `http.ServeContent` or by returning `e.HTML(200, htmlString)`. This file uses the PocketBase JS SDK (loaded from CDN) to authenticate and call the federation admin endpoints.

---

## System Overview: What Changes for v1.1

```
┌─────────────────────────────────────────────────────────────────┐
│                    Existing (v1.0, unchanged)                    │
│  SvelteKit (/api/v1/activitypub/instance/inbox) → Go backend    │
│  Go hooks: InstanceFollowCreate/Update/Delete → federation.*    │
│  federation.*: PostActivity, ProcessFollow/Accept/Reject/Undo   │
└─────────────────────────────────────────────────────────────────┘
         ↑ no changes
┌─────────────────────────────────────────────────────────────────┐
│                         New in v1.1                              │
│                                                                  │
│  Admin Browser                                                   │
│      │  Authorization: Bearer <superuser_jwt>                   │
│      ↓                                                           │
│  [NEW] Go route group: /federation/*                            │
│      │  e.HasSuperuserAuth() guard                              │
│      │                                                           │
│      ├── GET  /federation/peers           → list peer follows    │
│      ├── POST /federation/actors/fetch    → GetActorByIRI()     │
│      ├── POST /federation/follows         → create follows row   │
│      ├── PATCH /federation/follows/{id}/approve → set accepted  │
│      ├── PATCH /federation/follows/{id}/reject  → set rejected  │
│      └── DELETE /federation/follows/{id} → delete follows row   │
│                                                                  │
│  Existing hooks fire automatically after each DB write above     │
└─────────────────────────────────────────────────────────────────┘
```

---

## Recommended Project Structure

New and modified files only (everything else unchanged):

```
db/
├── routes/
│   ├── federation_admin.go        # NEW: 6 handler functions
│   └── (existing files unchanged)
└── main.go                        # MODIFIED: register /federation/* routes
```

No migrations needed. No new collections. No SvelteKit changes. No hook changes.

---

## Detailed Component Design

### `db/routes/federation_admin.go`

```go
package routes

import (
    "context"
    "net/http"
    "os"
    "pocketbase/federation"

    "github.com/pocketbase/pocketbase/apis"
    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/dbx"
)

// FederationPeersList returns all instance-level follows (both directions),
// joining the remote actor record to include domain, status, and direction.
func FederationPeersList(e *core.RequestEvent) error {
    if !e.HasSuperuserAuth() {
        return apis.NewUnauthorizedError("superuser authentication required", nil)
    }
    instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
    instanceActor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", instanceIRI)
    if err != nil {
        return err
    }
    // Query follows where follower OR followee is the local instance actor
    follows, err := e.App.FindRecordsByFilter(
        "follows",
        "follower={:id} || followee={:id}",
        "", 0, 0,
        dbx.Params{"id": instanceActor.Id},
    )
    if err != nil {
        return err
    }
    // Enrich each record with remote actor fields (domain, iri, status, direction)
    // ... build response slice
    return e.JSON(http.StatusOK, map[string]any{"items": follows})
}

// FederationActorFetch fetches the remote instance actor by IRI and
// persists/refreshes it in activitypub_actors. Returns the actor record.
func FederationActorFetch(e *core.RequestEvent) error {
    if !e.HasSuperuserAuth() {
        return apis.NewUnauthorizedError("superuser authentication required", nil)
    }
    var body struct {
        IRI string `json:"iri"`
    }
    if err := e.BindBody(&body); err != nil {
        return apis.NewBadRequestError("invalid request body", err)
    }
    if body.IRI == "" {
        return apis.NewBadRequestError("iri is required", nil)
    }
    ctx := context.Background()
    actor, err := federation.GetActorByIRI(e.App, ctx, body.IRI, false)
    if err != nil {
        return apis.NewBadRequestError("failed to fetch remote actor", err)
    }
    return e.JSON(http.StatusOK, map[string]any{"actor": actor})
}

// FederationFollowCreate creates a follows record with the local instance actor
// as follower. The InstanceFollowCreateHandler hook fires after the save and
// delivers the Follow activity to the remote inbox.
func FederationFollowCreate(e *core.RequestEvent) error {
    if !e.HasSuperuserAuth() {
        return apis.NewUnauthorizedError("superuser authentication required", nil)
    }
    var body struct {
        FolloweeActorID string `json:"followee_actor_id"`
    }
    if err := e.BindBody(&body); err != nil {
        return apis.NewBadRequestError("invalid request body", err)
    }
    instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
    instanceActor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", instanceIRI)
    if err != nil {
        return apis.NewNotFoundError("local instance actor not found", err)
    }
    collection, err := e.App.FindCollectionByNameOrId("follows")
    if err != nil {
        return err
    }
    follow := core.NewRecord(collection)
    follow.Set("follower", instanceActor.Id)
    follow.Set("followee", body.FolloweeActorID)
    follow.Set("status", "pending")
    if err := e.App.Save(follow); err != nil {
        return err
    }
    return e.JSON(http.StatusCreated, map[string]any{"follow": follow})
}
```

`FederationFollowApprove`, `FederationFollowReject`, and `FederationFollowDelete` follow the same pattern: load the record, set the field (or delete), call `e.App.Save()` / `e.App.Delete()`, return JSON. The hooks handle all delivery.

### Route Registration in `db/main.go`

```go
func registerRoutes(se *core.ServeEvent, client meilisearch.ServiceManager) {
    // ... existing routes ...

    // Federation admin API (superuser-only)
    se.Router.GET("/federation/peers", routes.FederationPeersList)
    se.Router.POST("/federation/actors/fetch", routes.FederationActorFetch)
    se.Router.POST("/federation/follows", routes.FederationFollowCreate)
    se.Router.PATCH("/federation/follows/{id}/approve", routes.FederationFollowApprove)
    se.Router.PATCH("/federation/follows/{id}/reject", routes.FederationFollowReject)
    se.Router.DELETE("/federation/follows/{id}", routes.FederationFollowDelete)
}
```

---

## Data Flow: Admin Connects to a Peer Instance

```
Admin Browser
    │
    │  Step 1: Fetch remote actor
    │  POST /federation/actors/fetch  { "iri": "https://peer.example.com/api/v1/activitypub/instance" }
    │  Authorization: Bearer <superuser_jwt>
    ↓
Go: FederationActorFetch
    → e.HasSuperuserAuth() ✓
    → federation.GetActorByIRI(app, ctx, iri, false)
        → HTTP GET https://peer.example.com/api/v1/activitypub/instance
        → Parse JSON-LD actor response
        → Validate: id, inbox, outbox, publicKey present
        → Upsert activitypub_actors record
    → Return { actor: { id, iri, domain, inbox, preferred_username, ... } }
    │
    │  Step 2: Initiate Follow
    │  POST /federation/follows  { "followee_actor_id": "<actor.id from step 1>" }
    ↓
Go: FederationFollowCreate
    → e.HasSuperuserAuth() ✓
    → Resolve local instance actor (iri = ORIGIN + "/api/v1/activitypub/instance")
    → Create follows record: follower=localInstanceActor.Id, followee=remoteActor.Id, status="pending"
    → app.Save(follow)
    ↓
Hook: InstanceFollowCreateHandler (OnRecordAfterCreateSuccess)
    → isOutboundInstanceFollow() = true (local instance is follower)
    → federation.CreateFollowActivity(app, follow)
        → POST Follow activity to peer.example.com/api/v1/activitypub/instance/inbox
        → Signed with local instance actor's private key
    │
    │  Step 3: Admin on peer instance approves (via their UI or PB admin)
    │  Their InstanceFollowUpdateHandler fires → Accept delivered back
    │
    │  ProcessAcceptActivity on this instance:
    │  → follows.status updated to "accepted"
    │
    │  Step 4: GET /federation/peers shows peer with status="accepted", direction="outbound"
```

---

## Data Flow: Admin Approves Inbound Follow Request

```
Peer instance admin initiated a Follow → arrives at our InstanceInboxHandler
    → ProcessFollowActivity creates follows record: status="pending",
      follower=peerInstanceActor, followee=localInstanceActor
    │
    │  Admin sees pending request via GET /federation/peers
    │  (filter: followee=localInstanceActor.Id AND status="pending")
    │
    │  Admin clicks Approve:
    │  PATCH /federation/follows/{id}/approve
    ↓
Go: FederationFollowApprove
    → e.HasSuperuserAuth() ✓
    → Load follows record by {id}
    → follow.Set("status", "accepted")
    → app.Save(follow)
    ↓
Hook: InstanceFollowUpdateHandler (OnRecordAfterUpdateSuccess)
    → isInstanceFollow() = true, isOutboundInstanceFollow() = false
    → instanceFollowAction("pending", "accepted") = "accept"
    → federation.CreateAcceptFollowActivity(app, follow)
        → Reloads original Follow activity from activitypub_activities
        → POST Accept{Follow} to peer's inbox
```

---

## Peer List Query Design

`FederationPeersList` must show every connected or pending peer with:
- Remote actor: domain, iri, preferred_username
- Follow direction: "outbound" (we follow them) / "inbound" (they follow us) / "mutual" (both)
- Status: pending / accepted / rejected

The query fetches all follows records where either `follower` or `followee` is the local instance actor, then joins with `activitypub_actors` to get remote actor details. Building the direction label requires checking whether the local instance actor is the follower or followee.

A single `FindRecordsByFilter` with the composite OR filter is sufficient. PocketBase's filter DSL supports `||`. The result set is expected to be small (tens of records at most).

Group by remote actor domain to merge outbound+inbound follows for the same peer into a "mutual" entry when both exist.

---

## Build Order (Dependencies)

No step has dependencies on any other — all six handlers operate on existing collections (activitypub_actors, follows) that already exist in production.

### Phase 1 — API layer (no dependencies)
Add `db/routes/federation_admin.go` with all six handlers. Register routes in `db/main.go`. Each handler can be written and tested independently.

**Testable after this phase:** All six endpoints return correct responses when called with a valid superuser JWT. Existing hook tests remain green (no hook changes).

### Phase 2 — Peer list enrichment (depends only on Phase 1)
The list endpoint initially returns raw follow records. Enrich with remote actor fields (join activitypub_actors). Add direction labeling and mutual-follow detection.

### Phase 3 — UI (depends on Phase 1 and 2)
If a browser UI is in scope: add a minimal HTML page served from Go via `e.String(200, "text/html", htmlContent)` or `go:embed`. The page calls the Phase 1+2 JSON API. Auth: the page reads the superuser JWT from localStorage (same key as the PocketBase admin panel) or prompts the user to paste it.

**Alternative for v1.1:** Skip Phase 3 entirely and document the API for curl/Postman use. The admin uses the PocketBase admin panel to trigger the endpoints. Phase 3 becomes v1.2 scope.

---

## Anti-Patterns

### Implementing federation delivery in the API handlers directly

**What people do:** Call `federation.CreateFollowActivity()` directly from `FederationFollowCreate`.
**Why it's wrong:** The hook already does this. Double-calling delivers two Follow activities to the remote inbox. The hook fires unconditionally after `app.Save()`.
**Do this instead:** The handler only calls `app.Save()`. The hook handles delivery. This is the pattern used by every other route in the codebase.

### Parsing the superuser JWT manually in the handler

**What people do:** Read `Authorization` header, parse the JWT, verify against a secret.
**Why it's wrong:** PocketBase's built-in auth middleware already does this and populates `e.Auth`. Manual parsing duplicates logic and risks bugs.
**Do this instead:** Check `e.HasSuperuserAuth()`. If false, return 401. Done.

### Exposing `/federation/*` through the SvelteKit proxy

**What people do:** Add a SvelteKit API route that proxies to `/federation/*` on the Go backend.
**Why it's wrong:** SvelteKit has no superuser concept. The proxy would either strip the Authorization header or forward it blindly. If it strips it, auth fails. If it forwards blindly, there's no value added.
**Do this instead:** The admin calls the Go backend's `/federation/*` routes directly (or via the PocketBase JS SDK which targets the backend URL directly). SvelteKit is not in this path.

### Using `e.App.FindAuthRecordByToken()` to validate the superuser token

**What people do:** In the handler, call `app.FindAuthRecordByToken(token, "superusers")` to verify the JWT.
**Why it's wrong:** Unnecessary — `e.Auth` is already populated by the middleware before the handler runs. If `e.Auth` is non-nil and `e.Auth.IsSuperuser()` is true, the token has already been validated.
**Do this instead:** `e.HasSuperuserAuth()`.

---

## Integration Points

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `federation_admin.go` handlers → `db/federation/actor.go` | Direct Go call: `GetActorByIRI()` | Used only by FederationActorFetch |
| `federation_admin.go` handlers → PocketBase DB | `e.App.Save()`, `e.App.Delete()`, `e.App.FindRecordsByFilter()` | Standard PocketBase CRUD |
| PocketBase hooks → `db/federation/follow.go` | `InstanceFollowCreate/Update/DeleteHandler` fires after any `follows` write | No code change needed — hooks see the write regardless of who initiates it |
| Admin browser → Go `/federation/*` | HTTP with `Authorization: Bearer <superuser_jwt>` | PocketBase admin panel issues this JWT on login |

### External Boundaries

| Service | Integration | Notes |
|---------|-------------|-------|
| Remote Wanderer instance | `FederationActorFetch`: HTTP GET to remote actor IRI | Uses `util.SafeHTTPClient()` (SSRF protection via doyensec/safeurl) |
| Remote Wanderer instance inbox | Via existing hooks — no new outbound HTTP in the admin handlers | Delivery is async, fire-and-forget |

---

## Sources

- `db/routes/plugin_system.go:14-27` — existing `e.HasSuperuserAuth()` pattern in a custom route
- `db/hooks/plugin_instances.go:22,37` — `e.HasSuperuserAuth()` in hook context
- `db/hooks/follow.go` — `InstanceFollowCreate/Update/DeleteHandler` confirmed to fire on all saves/deletes
- `db/federation/actor.go:98-116` — `GetActorByIRI()` for remote actor fetch
- `db/federation/follow.go:193-255` — `CreateAcceptFollowActivity`, `CreateRejectFollowActivity`
- `db/federation/follow.go:320-337` — `ProcessRejectActivity`
- `github.com/pocketbase/pocketbase@v0.38.0/core/event_request.go:77-80` — `HasSuperuserAuth()` implementation
- `github.com/pocketbase/pocketbase@v0.38.0/core/record_model_superusers.go:116-118` — `IsSuperuser()` implementation
- `github.com/pocketbase/pocketbase@v0.38.0/apis/middlewares.go:108-113` — `RequireSuperuserAuth()` middleware
- `github.com/pocketbase/pocketbase@v0.38.0/tools/router/group.go:29-30` — route-level `Bind()` for middleware

---

*Architecture research for: Federation Connect UI (v1.1)*
*Researched: 2026-06-27*
