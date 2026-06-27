# Phase 5: Federation Admin API - Research

**Researched:** 2026-06-27
**Domain:** Go/PocketBase REST API — federation peer management
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** `POST /federation/discover` is a required first step. Fetches `/.well-known/nodeinfo` then the NodeInfo 2.1 endpoint using the SSRF-safe client. Verifies `software.name == "wanderer"`. Returns `{ actor_id, domain, version, user_count, trail_count }`.
- **D-02:** `POST /federation/follow` accepts `{ actor_id }`. Returns 400 if no `activitypub_actors` record exists for that ID. Handler creates `follows` record with status "pending"; hook fires `CreateFollowActivity`. No HTTP calls in the handler.
- **D-03:** Discovery must bypass the 2-hour actor cache: clear `last_fetched` on the actor record before calling `GetActorByIRI`.
- **D-04:** `GET /federation/peers` returns `[{ follow_id, direction, status, domain }]`. Direction is `"outbound"`, `"inbound"`, or `"mutual"`. Handler queries all follows records involving the local instance actor, groups by remote domain, collapses two-direction pairs into a single `"mutual"` entry.
- **D-05:** Mutual detection: two `follows` records with status "accepted" where local actor is follower in one and followee in the other (same remote domain) → collapsed to `direction="mutual"`, `follow_id` = outbound record ID.
- **D-06:** Wanderer identity check: `software.name == "wanderer"` exact string match in NodeInfo 2.1.
- **D-07:** "Instance name" in the preview card is the hostname from the admin-supplied URL.
- **D-08:** Discovery data flow: admin URL → `/.well-known/nodeinfo` (JRD) → follow `links[].href` for rel `"http://nodeinfo.diaspora.software/ns/schema/2.1"` → NodeInfo 2.1 payload → `GetActorByIRI` for instance actor IRI. All fetches: SSRF-safe client, ≤10s timeout.
- **D-09:** All admin-supplied URLs fetched via `util.FetchPublicURL` (wraps `safeurl.Client` with private IP blocking). **Note:** existing `FetchPublicURL` uses 60s timeout — discovery handler MUST create its own 10s safeurl client wrapper (SAFE-06).
- **D-10:** All six endpoints require valid PocketBase superuser token. Verified via `e.HasSuperuserAuth()` (existing method, confirmed in `db/routes/plugin_system.go`). Non-superuser tokens receive 401.
- **D-11:** Handlers only call `app.Save()` / `app.Delete()`. ActivityPub delivery fires exclusively from `onAfterSave`/`onAfterDelete` hooks (SAFE-07).

### Claude's Discretion

- Error response format: `e.JSON(http.StatusXxx, map[string]any{"error": "..."})` — consistent with all existing handlers
- Route registration: new file `db/routes/federation_admin.go` following the `nodeinfo.go` pattern
- Whether to use a shared superuser-check middleware or inline auth guard per handler
- DISC-02 error messages: "unreachable", "not a Wanderer instance", "already connected", "resolves to local instance" — Claude picks the exact strings

### Deferred Ideas (OUT OF SCOPE)

- Mutual disconnect behavior (both Undo + Reject in one operation)
- Storing NodeInfo preview data in the actor record for later display without re-fetching (v2 UX-02)
- Real-time status updates via PocketBase subscription (v2 UX-01)
- WebFinger endpoint for instance actor (v2 DISC-03)
- HTML dashboard page at `/federation/` (Phase 6)
- Back-follow UI guard (Phase 6)
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DISC-01 | Admin can paste a remote URL and receive a preview card (name, version, user/trail count) before any Follow is sent | D-01, D-06, D-07, D-08 locked; NodeInfo 2.1 schema documented in codebase |
| DISC-02 | Discovery returns clear error when remote is unreachable, not Wanderer, already connected, or resolves to local instance | D-06, SAFE-05 confirmed via `util.IsLocalIRI`; four error cases identified |
| CONN-01 | Admin can initiate outgoing Follow from the UI; appears as "Outbound / Pending" without PocketBase panel | D-02 locked; `follows` collection schema confirmed; hook chain confirmed |
| CONN-02 | Admin can approve inbound pending Follow; Accept{Follow} delivered; moves to "Inbound / Accepted" | Hook `InstanceFollowUpdateHandler` already fires on status change to "accepted" — handler just calls `app.Save()` |
| CONN-03 | Admin can reject inbound pending Follow; Reject{Follow} delivered; connection removed | Hook `InstanceFollowUpdateHandler` fires on status change to "rejected" — handler sets status |
| CONN-04 | Admin can disconnect from any peer; direction-aware: outbound → Undo{Follow}, inbound-only → Reject{Follow} | Disconnect handler: `app.Delete()` triggers `InstanceFollowDeleteHandler` → `CreateUnfollowActivity` for outbound; status update to "rejected" for inbound-only |
| SAFE-05 | Discovery rejects URLs resolving to the local instance actor | `util.IsLocalIRI()` already exists for this check — confirmed in `db/util/activitypub.go:124` |
| SAFE-06 | All outbound HTTP to admin-supplied URLs uses SSRF-safe client with ≤10s timeout | `FetchPublicURL` exists but uses 60s timeout — must create a 10s wrapper for discovery |
| SAFE-07 | API handlers only write DB records; ActivityPub delivery exclusively via hooks | Confirmed: hooks registered in `db/main.go:132–134`; handlers must never call federation functions directly |
</phase_requirements>

---

## Summary

Phase 5 implements six Go route handlers for the federation admin API: discover, follow, approve, reject, disconnect, and list peers. All logic is additive to the existing federation infrastructure — no existing files need modification beyond registering new routes in `registerRoutes()` in `db/main.go`.

The most complex handler is `POST /federation/discover`, which performs a two-request NodeInfo fetch sequence, verifies Wanderer identity, bypasses the actor cache, and runs SAFE-05/SAFE-06 checks. Every other handler is a 10-30 line wrapper that validates the superuser token, reads request body, performs a DB write, and returns JSON — the hooks own all ActivityPub delivery.

The key cross-cutting concern for implementation is the **timeout gap in SAFE-06**: `util.FetchPublicURL` (the existing SSRF-safe client) is configured with a 60-second timeout, exceeding the ≤10s requirement. The discovery handler must construct its own `safeurl.Client` with a 10-second timeout rather than delegating to `FetchPublicURL`.

**Primary recommendation:** Create `db/routes/federation_admin.go` with all six handlers; add six route registrations to `db/main.go:registerRoutes()`; no other files require modification.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Discovery (NodeInfo fetch, identity check) | API / Backend (Go) | — | Admin-only; SSRF safety requires server-side fetch; no client-side involvement |
| Follow creation | API / Backend (Go) | — | Handler writes DB; hook owns delivery; no SvelteKit layer involved |
| Approve / Reject inbound follow | API / Backend (Go) | — | Handler updates status; hook fires Accept/Reject; DB-only operation |
| Disconnect (direction-aware) | API / Backend (Go) | — | Handler must read direction from DB before deciding delete vs status-update |
| Peer list aggregation | API / Backend (Go) | — | Mutual detection requires two-query fold against `follows` collection |
| Auth enforcement | API / Backend (Go) | — | `e.HasSuperuserAuth()` runs inside PocketBase request pipeline; no frontend auth |
| ActivityPub delivery | Hooks layer | — | Exclusively in `db/hooks/follow.go`; handlers MUST NOT call federation functions |

---

## Standard Stack

No new external packages are required. Phase 5 uses only existing Go dependencies already in the module.

### Core (already imported)

| Library | Purpose | Confirmed In |
|---------|---------|-------------|
| `github.com/pocketbase/pocketbase/core` | `RequestEvent`, `App`, `Record` — all route handlers | `db/routes/*.go` — every route file |
| `github.com/pocketbase/dbx` | `dbx.Params{}` for parameterized queries | `db/federation/follow.go`, `db/routes/activitypub.go` |
| `github.com/doyensec/safeurl` | SSRF-safe HTTP client | `db/util/safe_fetch.go:15` |
| `net/http` | Status codes, request handling | all route files |
| `encoding/json` | JSON decode of NodeInfo responses | `db/routes/nodeinfo.go:5` |
| `os` | `os.Getenv("ORIGIN")` | `db/federation/instance.go`, `db/hooks/follow.go` |
| `net/url` | URL parsing for hostname extraction | `db/federation/actor.go`, `db/util/activitypub.go` |
| `time` | 10s timeout context, timestamps | `db/util/safe_fetch.go` |

[VERIFIED: codebase] — all packages present in existing Go source files.

### No New Packages Required

Phase 5 is pure Go using existing module dependencies. No `go get` calls needed.

---

## Package Legitimacy Audit

> No new external packages are introduced in this phase. All dependencies are existing imports confirmed in the current module.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
Admin Client (curl / Phase 6 UI)
  │  Bearer: <superuser JWT>
  ▼
POST /federation/discover
  │  1. e.HasSuperuserAuth() → 401 if false
  │  2. GET /.well-known/nodeinfo (safeurl, 10s)
  │  3. Follow link → GET NodeInfo 2.1 (safeurl, 10s)
  │  4. Verify software.name == "wanderer"
  │  5. util.IsLocalIRI(actorIRI) → 400 if self
  │  6. Already-connected check → 400 if exists
  │  7. Clear last_fetched on actor; GetActorByIRI (refreshes cache)
  │  8. Return { actor_id, domain, version, user_count, trail_count }
  ▼
POST /federation/follow  { actor_id }
  │  1. HasSuperuserAuth() → 401
  │  2. Find activitypub_actors by actor_id → 400 if missing
  │  3. Find local instance actor by actor_type=instance, is_local=true
  │  4. app.Save(follows{follower: local, followee: remote, status: pending})
  │      └─ triggers OnRecordAfterCreateSuccess("follows")
  │              └─ InstanceFollowCreateHandler → CreateFollowActivity
  │  5. Return { follow_id, status: "pending" }
  ▼
POST /federation/approve/:id
  │  1. HasSuperuserAuth() → 401
  │  2. FindRecordById("follows", id) → 404 if missing
  │  3. Verify record is inbound (followee = local instance actor)
  │  4. follow.Set("status", "accepted"); app.Save(follow)
  │      └─ triggers OnRecordAfterUpdateSuccess("follows")
  │              └─ InstanceFollowUpdateHandler → CreateAcceptFollowActivity
  │  5. Return { follow_id, status: "accepted" }
  ▼
POST /federation/reject/:id
  │  1. HasSuperuserAuth() → 401
  │  2. FindRecordById("follows", id) → 404 if missing
  │  3. Verify record is inbound pending
  │  4. follow.Set("status", "rejected"); app.Save(follow)
  │      └─ InstanceFollowUpdateHandler → CreateRejectFollowActivity
  │  5. Return { follow_id, status: "rejected" }
  ▼
POST /federation/disconnect/:id
  │  1. HasSuperuserAuth() → 401
  │  2. FindRecordById("follows", id) → 404 if missing
  │  3. Determine direction: is local instance the follower?
  │       outbound → app.Delete(follow)
  │           └─ InstanceFollowDeleteHandler → CreateUnfollowActivity (Undo)
  │       inbound-only → follow.Set("status","rejected"); app.Save(follow)
  │           └─ InstanceFollowUpdateHandler → CreateRejectFollowActivity
  │  4. Return 200 OK
  ▼
GET /federation/peers
  │  1. HasSuperuserAuth() → 401
  │  2. Find local instance actor (actor_type=instance, is_local=true)
  │  3. Query follows where follower=local (outbound records)
  │  4. Query follows where followee=local (inbound records)
  │  5. Group by remote domain; collapse mutual pairs
  │  6. Return [{ follow_id, direction, status, domain }]
```

### Recommended Project Structure

```
db/routes/
├── federation_admin.go    # NEW — all 6 admin handlers
├── nodeinfo.go            # existing pattern to follow
├── activitypub.go         # existing auth pattern reference
└── health.go              # minimal handler pattern reference
```

### Pattern 1: Superuser Auth Guard

**What:** `e.HasSuperuserAuth()` is the correct PocketBase v0.26 method for checking if a request carries a valid superuser (admin) token. It is already used in `db/routes/plugin_system.go`.

**When to use:** First line of every federation admin handler before any other logic.

**Example:**
```go
// Source: db/routes/plugin_system.go:15 (existing codebase)
func FederationDiscover(e *core.RequestEvent) error {
    if !e.HasSuperuserAuth() {
        return e.UnauthorizedError("superuser authentication required", nil)
    }
    // ... rest of handler
}
```

[VERIFIED: codebase] — `e.HasSuperuserAuth()` confirmed at `db/routes/plugin_system.go:15,27`.

### Pattern 2: SSRF-Safe Discovery Client (10s Timeout)

**What:** `util.FetchPublicURL` has a 60-second timeout, violating SAFE-06. The discovery handler must create its own `safeurl.Client` with a 10-second timeout.

**When to use:** `POST /federation/discover` for both NodeInfo fetches.

**Example:**
```go
// Source: db/util/safe_fetch.go:48 (adapted for 10s timeout per SAFE-06)
import "github.com/doyensec/safeurl"

func newDiscoveryClient() *http.Client {
    config := safeurl.GetConfigBuilder().
        SetTimeout(10 * time.Second).
        SetAllowedSchemes("http", "https").
        SetAllowedPorts(80, 443).
        EnableIPv6(true).
        AllowSendingCredentials(false).
        Build()
    return safeurl.Client(config)
}
```

[VERIFIED: codebase] — `safeurl.GetConfigBuilder()` pattern confirmed in `db/util/safe_fetch.go`.

### Pattern 3: Local Instance Actor Lookup

**What:** The local instance actor is found by filtering `activitypub_actors` for `actor_type=instance && is_local=true`. Its IRI is always `ORIGIN + "/api/v1/activitypub/instance"`.

**When to use:** `POST /federation/follow`, `GET /federation/peers`, disconnect direction check.

**Example:**
```go
// Source: db/hooks/follow.go:49 (IRI pattern), db/federation/instance.go:44 (lookup pattern)
localActor, err := e.App.FindFirstRecordByFilter(
    "activitypub_actors",
    "actor_type={:t} && is_local={:l}",
    dbx.Params{"t": "instance", "l": true},
)
```

[VERIFIED: codebase] — filter pattern used in `db/federation/instance.go:44`.

### Pattern 4: Actor Cache Bypass for Discovery

**What:** `GetActorByIRI` skips the remote fetch if `last_fetched` is within 2 hours (see `db/federation/actor.go:221`). Discovery must force a fresh fetch by clearing `last_fetched` before calling `GetActorByIRI`.

**When to use:** `POST /federation/discover` after NodeInfo verification, before returning actor data.

**Example:**
```go
// Source: db/federation/actor.go:221 (cache logic confirmed)
// Clear last_fetched to bypass the 2-hour cache
if existingActor, err := e.App.FindFirstRecordByFilter(
    "activitypub_actors", "iri={:iri}", dbx.Params{"iri": actorIRI},
); err == nil {
    existingActor.Set("last_fetched", time.Time{}) // zero time forces re-fetch
    _ = e.App.Save(existingActor)
}
ctx := context.Background()
actor, err := federation.GetActorByIRI(e.App, ctx, actorIRI, false)
```

[VERIFIED: codebase] — cache check at `db/federation/actor.go:221`: `twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour); if dbActor.GetDateTime("last_fetched").Time().After(twoHoursAgo) { return dbActor, nil }`. Clearing `last_fetched` to zero makes this condition false.

### Pattern 5: SAFE-05 Self-Follow Guard

**What:** `util.IsLocalIRI(iri string) bool` reports whether an IRI belongs to the local `ORIGIN`. Use this to reject discovery requests that point at the local instance.

**When to use:** `POST /federation/discover` after extracting the actor IRI from NodeInfo.

**Example:**
```go
// Source: db/util/activitypub.go:124
if util.IsLocalIRI(actorIRI) {
    return e.JSON(http.StatusBadRequest, map[string]any{
        "error": "resolves to local instance",
    })
}
```

[VERIFIED: codebase] — `util.IsLocalIRI` confirmed at `db/util/activitypub.go:124`.

### Pattern 6: JSON Response Format

**What:** All handlers use `e.JSON(statusCode, map[string]any{...})`. For errors, the convention is `map[string]any{"error": "message string"}`. For PocketBase-style errors, `e.UnauthorizedError`, `e.NotFoundError`, `e.BadRequestError` return structured errors.

**When to use:** All handlers.

**Example:**
```go
// Source: db/routes/activitypub.go:63 (existing codebase)
return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": nil})

// Error case (consistent with user_email_change.go:16):
return e.UnauthorizedError("superuser authentication required", nil)
```

[VERIFIED: codebase] — both patterns confirmed across multiple route files.

### Pattern 7: Route Registration

**What:** Routes are registered in `registerRoutes()` in `db/main.go` inside the `OnServe` handler. Pattern is `se.Router.METHOD("/path", handlers.FuncName)` or `se.Router.METHOD("/path/{id}", handlers.FuncName)`.

**When to use:** After implementing all handlers in `federation_admin.go`.

**Example:**
```go
// Source: db/main.go:172–209 (registerRoutes function)
se.Router.POST("/federation/discover", routes.FederationDiscover)
se.Router.POST("/federation/follow", routes.FederationFollow)
se.Router.POST("/federation/approve/{id}", routes.FederationApprove)
se.Router.POST("/federation/reject/{id}", routes.FederationReject)
se.Router.POST("/federation/disconnect/{id}", routes.FederationDisconnect)
se.Router.GET("/federation/peers", routes.FederationPeers)
```

[VERIFIED: codebase] — pattern confirmed in `db/main.go:172–209`.

### Anti-Patterns to Avoid

- **Calling federation functions directly from handlers:** `CreateFollowActivity`, `CreateAcceptFollowActivity`, `CreateRejectFollowActivity`, `CreateUnfollowActivity` must NEVER be called from route handlers. Hooks own all delivery (SAFE-07 — verified active at `db/main.go:132–134`).
- **Using `util.FetchPublicURL` for discovery fetches:** Its 60s timeout violates SAFE-06. Use a dedicated 10s safeurl client.
- **Using `http.DefaultClient` or `util.SafeHTTPClient` for admin-supplied URLs:** `SafeHTTPClient` applies rate limiting but does NOT block private IPs via safeurl. Use `safeurl.Client` directly for SSRF protection.
- **Skipping the "already connected" check in discover:** Two `follows` records for the same pair will cause a DB constraint violation when `POST /federation/follow` fires the hook.
- **Using `e.Auth.Collection().Name == "_superusers"`:** The codebase uses `e.HasSuperuserAuth()` — a single method that encapsulates this check. Use it instead.
- **Forgetting that `app.Delete()` on an outbound follow triggers `InstanceFollowDeleteHandler` which calls `CreateUnfollowActivity`:** For inbound-only disconnect, do NOT delete — update status to "rejected" so `CreateRejectFollowActivity` fires instead.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SSRF blocking | Custom IP blocklist | `github.com/doyensec/safeurl` (already imported) | safeurl handles private ranges, IPv6, redirect chains; hand-rolled list misses edge cases |
| Actor cache bypass | New fetch function | Clear `last_fetched` field + existing `GetActorByIRI` | Cache logic is centralized in one function; clearing the timestamp triggers re-fetch without duplicating fetch code |
| Self-follow detection | URL comparison inline | `util.IsLocalIRI()` (existing, `db/util/activitypub.go:124`) | Handles `www.` prefix stripping, scheme normalization |
| Superuser auth check | JWT decode + collection query | `e.HasSuperuserAuth()` (PocketBase built-in) | Single method, PocketBase maintains it |
| ActivityPub activity creation | Inline in handler | Hook chain (`InstanceFollowCreateHandler`, `InstanceFollowUpdateHandler`, `InstanceFollowDeleteHandler`) | Double-delivery risk if handler AND hook both call delivery functions |
| Mutual peer detection | Complex JOIN query | Two separate `FindRecordsByFilter` calls + in-memory grouping | PocketBase's `FindFirstRecordByFilter` does not support GROUP BY; two queries + Go map folding is correct pattern |

---

## Common Pitfalls

### Pitfall 1: SAFE-06 Timeout Violation

**What goes wrong:** Using `util.FetchPublicURL` for NodeInfo fetches — it uses a 60-second timeout, violating SAFE-06 (≤10s).
**Why it happens:** `FetchPublicURL` is the obvious "SSRF-safe fetch" utility but was designed for media imports, not discovery.
**How to avoid:** Create a local `safeurl.Client` with `SetTimeout(10 * time.Second)` inside `FederationDiscover`. Do not reuse `FetchPublicURL`.
**Warning signs:** SAFE-06 requirement in the phase requirements — always check timeout value against it.

### Pitfall 2: Double-Delivery via Direct Hook + Handler Call

**What goes wrong:** Handler calls `federation.CreateFollowActivity(...)` directly AND the `InstanceFollowCreateHandler` hook fires — remote instance receives two Follow activities.
**Why it happens:** Developer sees `CreateFollowActivity` exported and assumes it's the right call site.
**How to avoid:** SAFE-07 is an explicit hard rule. Handlers call only `app.Save()` and `app.Delete()`. ActivityPub delivery is exclusively hook-owned. Verify with grep: `grep -n "Create.*Activity\|CreateUnfollowActivity" db/routes/federation_admin.go` — must return no results.
**Warning signs:** `InstanceFollowCreateHandler` at `db/main.go:132` already fires `CreateFollowActivity` on every outbound instance follow create.

### Pitfall 3: Wrong Disconnect Operation for Inbound-Only Follows

**What goes wrong:** Handler calls `app.Delete(follow)` on an inbound-only follow record — `InstanceFollowDeleteHandler` fires `CreateUnfollowActivity`, which sends `Undo{Follow}` where the local instance was never the follower. The remote instance rejects or ignores it.
**Why it happens:** `app.Delete()` is the simple "remove" operation, but the Undo verb is only valid for outbound follows (where local issued the original Follow).
**How to avoid:** Check direction before deciding the operation:
- `follow.GetString("follower") == localActorID` → outbound → `app.Delete()` → hook fires Undo
- `follow.GetString("followee") == localActorID` → inbound-only → `follow.Set("status","rejected"); app.Save(follow)` → hook fires Reject
**Warning signs:** CONN-04 explicitly states "direction-aware."

### Pitfall 4: Stale Actor Cache in Discovery

**What goes wrong:** `GetActorByIRI` returns a 2-hour-old cached actor. Admin sees stale `user_count`/`trail_count` data in the preview card. Worse, an actor whose IRI changed (instance moved) is not re-fetched.
**Why it happens:** `assembleActor` in `actor.go:221` short-circuits if `last_fetched` is recent.
**How to avoid:** Clear `last_fetched` on the existing actor record (if any) before calling `GetActorByIRI`. Decision D-03 mandates this.
**Warning signs:** D-03 in CONTEXT.md.

### Pitfall 5: Already-Connected Check in Discover

**What goes wrong:** Admin calls `/discover` then `/follow` for a peer already connected → hook fires Follow activity again → remote receives duplicate. The `follows` collection has a unique index on `(follower, followee)`, so `app.Save()` returns a constraint error.
**Why it happens:** Discover is idempotent (refreshes actor) but doesn't guard follow creation.
**How to avoid:** In `POST /federation/discover`, check whether a `follows` record already exists between local instance actor and the remote actor. If yes, return 400 with error message "already connected".

### Pitfall 6: NodeInfo Discovery Link Selection

**What goes wrong:** Taking `links[0].href` from the JRD discovery doc without checking `rel` — the JRD may contain multiple links for different schema versions (2.0, 2.1).
**Why it happens:** The local `NodeInfo` function only emits one link (for 2.1), so developers assume all instances do the same.
**How to avoid:** Iterate over `links` and select the href where `rel == "http://nodeinfo.diaspora.software/ns/schema/2.1"`. Decision D-08 specifies this.

### Pitfall 7: Context Requirement for GetActorByIRI

**What goes wrong:** `federation.GetActorByIRI(app, context.Background(), iri, false)` passes a plain background context without the `"actor"` key — the rate limiter in `SafeHTTPClient` uses `"system"` as the identifier, which may conflict with other system requests.
**Why it happens:** Handlers have no request auth actor to pass, and the context pattern is designed for user actors.
**How to avoid:** For discovery, pass `context.Background()` (same pattern used in `db/hooks/follow.go:121`). The "system" rate limit bucket is appropriate for admin operations.

---

## Code Examples

### NodeInfo Fetch Sequence

```go
// Source: db/routes/nodeinfo.go (local NodeInfo implementation for reference)
// Source: db/util/safe_fetch.go:48 (safeurl config pattern)

// Step 1: Fetch JRD discovery doc
func fetchNodeInfoURL(rawURL string) (string, error) {
    u, err := url.Parse(rawURL)
    if err != nil {
        return "", fmt.Errorf("invalid URL: %w", err)
    }
    jrdURL := fmt.Sprintf("%s://%s/.well-known/nodeinfo", u.Scheme, u.Host)

    client := newDiscoveryClient() // 10s safeurl client
    req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, jrdURL, nil)
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("unreachable: %w", err)
    }
    defer resp.Body.Close()

    var jrd struct {
        Links []struct {
            Rel  string `json:"rel"`
            Href string `json:"href"`
        } `json:"links"`
    }
    if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&jrd); err != nil {
        return "", fmt.Errorf("invalid discovery document")
    }

    // Step 2: Find NodeInfo 2.1 link
    for _, link := range jrd.Links {
        if link.Rel == "http://nodeinfo.diaspora.software/ns/schema/2.1" {
            return link.Href, nil
        }
    }
    return "", fmt.Errorf("no NodeInfo 2.1 endpoint found")
}
```

### Follows Collection Create for Outbound Follow

```go
// Source: db/federation/follow.go:97–138 (ProcessFollowActivity — inbound path)
// Adapted for outbound admin-initiated follow

followCollection, err := e.App.FindCollectionByNameOrId("follows")
if err != nil {
    return err
}
followRecord := core.NewRecord(followCollection)
followRecord.Set("follower", localActor.Id)   // local instance actor
followRecord.Set("followee", remoteActor.Id)  // remote actor from discovery
followRecord.Set("status", "pending")
if err := e.App.Save(followRecord); err != nil {
    return err
}
// Hook InstanceFollowCreateHandler fires → CreateFollowActivity → HTTP delivery
```

### Peer List Query and Mutual Collapse

```go
// Source: db/federation/follow.go (FindFirstRecordByFilter patterns with dbx.Params)

// Query outbound follows (local is follower)
outbound, _ := e.App.FindRecordsByFilter(
    "follows",
    "follower={:local}",
    "-created", -1, 0,
    dbx.Params{"local": localActor.Id},
)
// Query inbound follows (local is followee)
inbound, _ := e.App.FindRecordsByFilter(
    "follows",
    "followee={:local}",
    "-created", -1, 0,
    dbx.Params{"local": localActor.Id},
)

// Build map: domain → peerEntry; collapse mutual pairs in Go
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| Checking `e.Auth.Collection().Name == "_superusers"` | `e.HasSuperuserAuth()` (PocketBase built-in method) | Single call, less brittle if PocketBase renames internal collection |
| Inline SSRF checks | `github.com/doyensec/safeurl` builder pattern | Library handles redirect chain SSRF, IPv6-mapped IPv4, credential stripping |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `e.HasSuperuserAuth()` accepts superuser JWT tokens passed as `Authorization: Bearer <token>` by the admin client | Pattern 1 / Auth Guard | If wrong, all 6 endpoints would reject valid tokens; need to check PocketBase docs for token header |
| A2 | `FindRecordsByFilter` (plural) is available on `core.App` in PocketBase v0.26 | Peer List Query | If not available, use two `FindAllRecords` + filter loop, or use `app.DB().NewQuery()` |
| A3 | The `follows` collection does NOT have a unique index on `(follower, followee)` that prevents creating duplicate pending records via `app.Save()` — only the PocketBase API layer enforces `createRule` | Pitfall 5 | If wrong, the "already connected" check in discover is defense-in-depth only; duplicate records would be blocked by DB |

**Note on A1:** `e.HasSuperuserAuth()` usage is confirmed in the codebase. The token delivery mechanism (Bearer header) is standard PocketBase auth [ASSUMED: follows PocketBase v0.26 conventions].

---

## Open Questions

1. **`FindRecordsByFilter` vs `FindAllRecords` API**
   - What we know: `FindFirstRecordByFilter` is used throughout the codebase. `FindAllRecords` is used in `db/migrations/1747061271_migrate_follows.go:10`.
   - What's unclear: Whether the PocketBase v0.26 `core.App` interface exposes `FindRecordsByFilter` (plural) directly, or whether the peer list handler should use `FindAllRecords("follows")` + Go-level filter.
   - Recommendation: The planner should use `app.FindRecordsByFilter("follows", filter, sort, limit, offset, params)` if available, or `app.FindAllRecords("follows", dbx.HashExp{"follower": localActor.Id})` as fallback. Either works for the scale of peer connections expected.

2. **NodeInfo 2.1 schema for `usage.localPosts`**
   - What we know: The local `NodeInfo21` handler uses `usage.localPosts` mapped to public trail count. The discovery preview needs `trail_count`.
   - What's unclear: Whether all Wanderer instances expose `localPosts` (they do if running current code), and whether the count represents trails specifically.
   - Recommendation: Map `usage.localPosts` → `trail_count` in the discover response. Document in handler comment that this equals public trails per the NodeInfo 2.1 implementation.

---

## Environment Availability

> Phase 5 is purely additive Go code. No external tools, services, or CLIs beyond the existing Go module are required.

Step 2.6: SKIPPED — no external dependencies identified beyond existing Go module.

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | `e.HasSuperuserAuth()` — PocketBase superuser token required on all 6 endpoints |
| V3 Session Management | no | Stateless API; no sessions |
| V4 Access Control | yes | Superuser-only gate; no regular user access permitted |
| V5 Input Validation | yes | Admin-supplied URL validated via `url.Parse` + safeurl scheme/port restrictions; `actor_id` validated by DB lookup |
| V6 Cryptography | no | No new crypto; existing RSA signing used by hook-delivered activities |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SSRF via admin-supplied discovery URL | Elevation of Privilege | `safeurl.Client` with 10s timeout, private IP blocking, scheme restriction to http/https |
| Self-follow loop (local instance follows itself) | Denial of Service | `util.IsLocalIRI()` check in discover before creating follow |
| Duplicate Follow delivery (idempotency failure) | Denial of Service | "already connected" check in discover; idempotency guard in `ProcessFollowActivity` (line 82–96 in `db/federation/follow.go`) |
| Wrong Undo direction (inbound-only disconnect) | Spoofing | Direction check in disconnect handler: outbound → Delete+Undo; inbound-only → status update+Reject |
| Unauthorized peer management | Elevation of Privilege | `e.HasSuperuserAuth()` at top of every handler → 401 for non-superuser tokens |
| Oversized NodeInfo response (DoS via memory) | Denial of Service | `io.LimitReader` on NodeInfo responses; same pattern as `db/federation/actor.go:353` |

---

## Sources

### Primary (HIGH confidence — verified in codebase)
- `db/routes/plugin_system.go:15` — `e.HasSuperuserAuth()` method confirmed
- `db/routes/nodeinfo.go` — route file pattern (one file per group, no middleware)
- `db/routes/activitypub.go` — handler patterns: `e.JSON`, `e.Auth`, path values
- `db/main.go:132–134` — hook registrations for follows collection
- `db/main.go:172–209` — `registerRoutes()` function pattern
- `db/hooks/follow.go` — `InstanceFollowCreateHandler`, `InstanceFollowUpdateHandler`, `InstanceFollowDeleteHandler` — confirmed delivery-only roles
- `db/federation/follow.go` — `CreateFollowActivity`, `CreateAcceptFollowActivity`, `CreateRejectFollowActivity` signatures
- `db/federation/actor.go:98–116` — `GetActorByIRI` signature and cache logic at line 221
- `db/federation/instance.go:40–45` — `FindFirstRecordByData("activitypub_actors", "iri", iri)` pattern for local actor lookup
- `db/util/safe_fetch.go:37–77` — `FetchPublicURL` confirmed 60s timeout (violates SAFE-06)
- `db/util/activitypub.go:124` — `util.IsLocalIRI()` function confirmed
- `db/util/network.go:138` — `SafeHTTPClient()` pattern
- `db/migrations/1747064968_collections_snapshot.go:100–174` — `follows` collection schema (fields: follower, followee, status: pending|accepted; rejected added via later migration)
- `db/migrations/1782290001_add_rejected_to_follows_status.go` — "rejected" status value added to follows.status select field

### Secondary (MEDIUM confidence)
- NodeInfo 2.1 schema spec at `http://nodeinfo.diaspora.software/ns/schema/2.1` — verified via local implementation in `db/routes/nodeinfo.go` and CONTEXT.md D-08

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing codebase imports, no new packages
- Architecture: HIGH — all patterns verified directly in Go source files with line references
- Pitfalls: HIGH — derived from reading actual hook and federation code, not assumptions
- Auth mechanism: HIGH — `e.HasSuperuserAuth()` confirmed in codebase

**Research date:** 2026-06-27
**Valid until:** 2026-09-27 (stable Go/PocketBase stack; 90 days)
