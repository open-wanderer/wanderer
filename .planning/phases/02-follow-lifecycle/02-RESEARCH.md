# Phase 2: Follow Lifecycle - Research

**Researched:** 2026-06-25
**Domain:** ActivityPub Follow lifecycle in PocketBase/Go — inbox routing, hook-driven AP activity delivery, actor-type branching
**Confidence:** HIGH

## Summary

Phase 2 wires together the instance actor's inbox endpoint and the complete Follow state machine. Every answer to the six key research questions was found directly in the existing codebase — no new patterns or libraries are needed. The pattern is already present: routes call `util.VerifySignature()` after fetching the sender actor, then dispatch by activity type; PocketBase lifecycle hooks fire via `OnRecordAfterCreateSuccess`/`OnRecordAfterUpdateSuccess`/`OnRecordAfterDeleteSuccess` on the `follows` collection; and `PostActivity()` already handles all outgoing AP activity delivery with HTTP signing. The entire phase is a set of additive changes on top of the existing machinery.

Two important schema gaps surfaced during research that require migration work: the `follows.status` select field currently lists only `["pending", "accepted"]` — `"rejected"` is missing and must be added before the Reject path can work. In addition, the `actor_type` select field was added by the Phase 1 migration, but the value used in `InitInstanceActor` is `"instance"`, not `"Application"` — this is intentional (matches the select field values `["person", "instance"]`) but the hook filter must use the string `"instance"` to identify the local instance actor, not the ActivityPub type string `"Application"`.

The Undo path for instance-level follows has a subtle difference from user-level unfollows: the existing `DeleteFollowHandler` fires on `OnRecordDeleteRequest` (HTTP request context) but for admin-initiated deletes the correct hook is `OnRecordAfterDeleteSuccess` (or `OnRecordDeleteRequest` also fires for admin panel deletes — confirmed by PocketBase v0.38.0 docs). Both hook families are available; the plan should use the `AfterSuccess` variants for consistency with the pattern used for trails, trail_like, and lists.

**Primary recommendation:** Add the instance inbox route to `instance.go`; add three new hook functions to `hooks/follow.go` (instance create/update/delete) registered via `OnRecordAfterCreateSuccess`/`OnRecordAfterUpdateSuccess`/`OnRecordAfterDeleteSuccess`; extend `ProcessFollowActivity()` in `federation/follow.go` with an actor-type branch; add the `"rejected"` value to the `follows.status` migration.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Admin initiates an outgoing Follow by creating a record directly in the `follows` collection via PocketBase's native admin CRUD UI. A Go `OnRecordAfterCreate` hook fires, fetches the remote actor's ActivityPub JSON (to discover their inbox URL), and delivers the `Follow` activity to the remote inbox.
- **D-02:** No custom admin UI is built in this phase. A custom admin panel for federation management is deferred to v2 (ADMIN-01). For v1, admin uses PocketBase's native collection record form to supply the remote actor IRI.
- **D-03:** The instance inbox endpoint (`POST /api/v1/activitypub/instance/inbox`) gets a **dedicated handler in `db/federation/instance.go`**. It does NOT route through the existing user-inbox `ProcessActivity()` dispatcher. This keeps instance-actor concerns isolated from user-level federation, consistent with Phase 1's `instance.go` isolation pattern.
- **D-04:** HTTP signature verification on the instance inbox reuses whatever signature-check function is already implemented for user inboxes. No new verification logic is introduced.
- **D-05:** `ProcessFollowActivity()` is extended with an actor-type branch: if the sending actor's `actor_type` is `Application`, the follow record is stored with `status = "pending"` and the function returns without sending an `Accept`. If the actor type is `Person` (existing user-level follow), the existing auto-accept logic runs unchanged. This is the minimal change that satisfies FLCL-02 without affecting user federation.
- **D-06:** `OnRecordAfterUpdate` hook on the `follows` collection: when `status` changes to `accepted`, deliver `Accept{Follow}` to the remote instance's inbox; when `status` changes to `rejected`, deliver `Reject{Follow}`.
- **D-07:** `OnRecordAfterDelete` hook on the `follows` collection: when an instance follow record is deleted (admin removes it in PocketBase admin), deliver `Undo{Follow}` to the remote instance's inbox.
- **D-08:** `Reject{Follow}` delivery is **mandatory** (not optional). The remote instance's UI cannot recover if it never receives the rejection. This was flagged in STATE.md and must not be skipped.

### Claude's Discretion

- Researcher determines which existing function handles HTTP signature verification and how to call it from the instance inbox handler.
- Researcher determines which PocketBase lifecycle hook type fires reliably after a `follows` record is created/updated/deleted (likely `OnRecordAfterCreate`, `OnRecordAfterUpdate`, `OnRecordAfterDelete`).
- Researcher identifies whether a `FetchActor()` utility already exists in `db/federation/` for fetching remote actor JSON, or whether one needs to be written.
- Hook filtering: researcher must ensure the Accept/Reject/Undo hooks only fire for instance-level follows (where the followee or follower is the instance actor), not for user-level follows.

### Deferred Ideas (OUT OF SCOPE)

- **Custom admin UI for federation management** — Admin wants a proper UI to initiate follows and view connection status. This is v2 scope (ADMIN-01 in REQUIREMENTS.md). For v1, PocketBase native admin CRUD is sufficient.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INST-03 | POST `{ORIGIN}/api/v1/activitypub/instance/inbox` accepts HTTP-signed activities from authenticated remote actors | Signature verification: `util.VerifySignature(app, req, publicKeyPem)` — same function used in `ActivitypubActivityProcess`. Route registration pattern: `se.Router.POST("/activitypub/instance/inbox", ...)` in `registerRoutes()`. |
| FLCL-01 | Admin can initiate an outgoing Follow to a remote instance actor by supplying the remote IRI in PocketBase admin; local instance actor sends a Follow activity | `OnRecordAfterCreateSuccess("follows")` hook — fires after admin creates a follows record. Must fetch remote actor via `GetActorByIRI()` to get inbox URL, then call `PostActivity()` using the local instance actor. Requires a migration to add `"rejected"` value to `follows.status`. |
| FLCL-02 | Incoming Follow from an Application-type actor creates a `pending` follow record (not auto-accepted) | Branch in `ProcessFollowActivity()`: check `actor.GetString("actor_type") == "instance"`. If true, save with `status = "pending"` and return. Existing `person` path unchanged. |
| FLCL-03 | When admin sets pending instance follow to `accepted`, deliver `Accept{Follow}` to remote instance's inbox | `OnRecordAfterUpdateSuccess("follows")` hook. Detect status change: `e.Record.Original().GetString("status") != "accepted" && e.Record.GetString("status") == "accepted"`. Then call `PostActivity()` with `pub.AcceptNew(...)` signed as the instance actor. |
| FLCL-04 | When admin sets pending instance follow to `rejected`, deliver `Reject{Follow}` to remote instance's inbox | Same hook as FLCL-03. Branch when new status is `"rejected"`. Requires `"rejected"` added to the `follows.status` select field via migration. Use `pub.RejectNew(id, followActivity)` from `go-ap`. |
| FLCL-05 | Admin can unfollow a peer instance; `Undo{Follow}` activity is sent to the remote | `OnRecordAfterDeleteSuccess("follows")` hook. Check that the deleted record involves the instance actor. Use existing `CreateUnfollowActivity()` pattern or equivalent in `instance.go`. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Instance inbox HTTP handler | API / Backend (`instance.go`) | — | Instance-actor concerns isolated from user federation per D-03 |
| HTTP signature verification | API / Backend (`util.VerifySignature`) | — | Already implemented; reused per D-04 |
| Follow record creation (outgoing) | Database hook (`OnRecordAfterCreateSuccess`) | AP delivery (`PostActivity`) | Admin drives via native PocketBase UI; hook reacts |
| Actor-type branch in ProcessFollowActivity | API / Backend (`federation/follow.go`) | — | Single function owns incoming follow dispatch |
| Accept/Reject delivery | Database hook (`OnRecordAfterUpdateSuccess`) | AP delivery (`PostActivity`) | Status change detected via `Record.Original()` |
| Undo delivery | Database hook (`OnRecordAfterDeleteSuccess`) | AP delivery (`PostActivity`) | Record deletion triggers unfollow activity |
| follows.status schema | Database / Storage (migration) | — | `"rejected"` value missing from select field; requires migration |

## Standard Stack

### Core

No new external packages are introduced in this phase. All building blocks are already in the project.

| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| `util.VerifySignature()` | `db/util/activitypub.go:615` | HTTP signature verification for inbox endpoints | Already used in `ActivitypubActivityProcess` |
| `federation.PostActivity()` | `db/federation/activity.go:55` | Deliver signed outgoing AP activities | Used for all existing outgoing activities (Follow, Accept, Undo, Like) |
| `federation.GetActorByIRI()` | `db/federation/actor.go:98` | Fetch-or-cache remote actor JSON, returning `*core.Record` | Used by existing user inbox handler |
| `util.GetSafeActorContext()` | `db/util/network.go:173` | Build context carrying actor identity for rate limiting and signing | Required by `GetActorByIRI` |
| `pub.AcceptNew()` | `go-ap/activitypub` | Construct Accept activity wrapping original Follow | Already used in `ProcessFollowActivity` |
| `pub.RejectNew()` | `go-ap/activitypub` | Construct Reject activity wrapping original Follow | Available in go-ap (type alias of `Activity` with `RejectType`) |
| `pub.UndoNew()` | `go-ap/activitypub` | Construct Undo activity wrapping Follow | Already used in `CreateUnfollowActivity` |
| `e.Record.Original()` | PocketBase v0.38.0 `core/record_model.go:618` | Returns a Record snapshot of the pre-update values | Used in update hook to detect status transition |

### Package Legitimacy Audit

No new packages are installed in this phase. All dependencies (`github.com/go-ap/activitypub`, `github.com/go-fed/httpsig`, `github.com/pocketbase/pocketbase`) are already in `go.mod` and are proven production dependencies.

| Package | Registry | Status | Disposition |
|---------|----------|--------|-------------|
| `github.com/go-ap/activitypub` | Go modules | Already in use | Approved (in go.mod) |
| `github.com/go-fed/httpsig` | Go modules | Already in use | Approved (in go.mod) |
| `github.com/pocketbase/pocketbase` | Go modules | Already in use | Approved (in go.mod) |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
Admin (PocketBase UI)
    |
    | creates/updates/deletes follows record
    v
PocketBase follows collection
    |
    +-- OnRecordAfterCreateSuccess("follows") --------> InstanceFollowCreateHook
    |       (outgoing Follow: admin created row)              |
    |                                                         | GetActorByIRI(remoteIRI)
    |                                                         | PostActivity(instanceActor, Follow, [remote.inbox])
    |
    +-- OnRecordAfterUpdateSuccess("follows") --------> InstanceFollowUpdateHook
    |       (status changed to accepted/rejected)             |
    |                                                         | if accepted: PostActivity(instanceActor, Accept{Follow}, [remote.inbox])
    |                                                         | if rejected: PostActivity(instanceActor, Reject{Follow}, [remote.inbox])
    |
    +-- OnRecordAfterDeleteSuccess("follows") --------> InstanceFollowDeleteHook
            (admin deleted follow record)                     |
                                                             | PostActivity(instanceActor, Undo{Follow}, [remote.inbox])

Remote Instance (ActivityPub client)
    |
    | POST /api/v1/activitypub/instance/inbox (HTTP-signed)
    v
InstanceInboxHandler (db/federation/instance.go)
    |
    | 1. Read body, parse pub.Activity
    | 2. FindOrFetch sending actor by actor.Actor IRI
    | 3. util.VerifySignature(app, req, actor.public_key)
    | 4. switch activity.Type
    |       Follow  --> ProcessFollowActivity() [extended with actor_type branch]
    |       Accept  --> ProcessAcceptActivity()
    |       Undo    --> ProcessUndoActivity()
    v
follows collection (status = "pending" for Application actors)
```

### Recommended Project Structure

The phase adds to existing files only. No new files required except a migration.

```
db/
├── federation/
│   └── instance.go          # Add: InstanceInboxHandler function + route handler
├── hooks/
│   └── follow.go            # Add: instance create/update/delete handler functions
├── migrations/
│   └── XXXXXXXXXX_add_rejected_to_follows_status.go   # New: adds "rejected" to status select
└── main.go                  # Add: register new hooks + instance inbox route
```

### Pattern 1: HTTP Signature Verification (Reusing Existing)

**What:** The existing `ActivitypubActivityProcess` in `routes/activitypub.go` shows the full pattern for an authenticated AP inbox endpoint.
**When to use:** Exactly this pattern for the instance inbox handler.

```go
// Source: db/routes/activitypub.go (ActivitypubActivityProcess)
// Pattern for instance inbox handler in db/federation/instance.go:

func InstanceInboxHandler(e *core.RequestEvent) error {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return fmt.Errorf("ORIGIN not set")
    }

    body, err := io.ReadAll(e.Request.Body)
    if err != nil {
        return err
    }
    var activity pub.Activity
    if err = activity.UnmarshalJSON(body); err != nil {
        return err
    }

    // 1. Find or fetch the sending actor
    actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", activity.Actor.GetID().String())
    if err != nil {
        // Actor unknown — fetch from remote
        ctx, err := util.GetSafeActorContext(e.Request, nil /* instance actor */)
        if err != nil {
            return err
        }
        actor, err = federation.GetActorByIRI(e.App, ctx, activity.Actor.GetID().String(), false)
        if err != nil {
            return err
        }
    }

    // 2. Verify HTTP signature
    verified, err := util.VerifySignature(e.App, e.Request, actor.GetString("public_key"))
    if err != nil || !verified {
        return e.UnauthorizedError("Invalid http signature", err)
    }

    // 3. Dispatch by type (instance inbox only needs Follow, Accept, Undo)
    switch activity.Type {
    case pub.FollowType:
        err = federation.ProcessFollowActivity(e.App, actor, activity)
    case pub.AcceptType:
        err = federation.ProcessAcceptActivity(e.App, actor, activity)
    case pub.UndoType:
        err = federation.ProcessUndoActivity(e.App, actor, activity)
    }
    return e.JSON(http.StatusOK, err)
}
```

**CRITICAL NOTE:** `util.VerifySignature` reads `X-Forwarded-Path` from the request header to reconstruct the signed URL path. The instance inbox route will be at `/activitypub/instance/inbox` — the proxy must forward this header correctly, same as for user inboxes. [VERIFIED: db/util/activitypub.go:625-626]

### Pattern 2: Actor-Type Branch in ProcessFollowActivity

**What:** Extend `ProcessFollowActivity()` to detect Application-type actors and store pending instead of auto-accepting.
**When to use:** Every incoming Follow activity passes through this function.

```go
// Source: db/federation/follow.go (modified section)
func ProcessFollowActivity(app core.App, actor *core.Record, activity pub.Activity) error {
    // ... existing IRI/object lookup ...

    if !actor.GetBool("is_local") {
        followCollection, err := app.FindCollectionByNameOrId("follows")
        if err != nil {
            return err
        }
        followRecord := core.NewRecord(followCollection)
        followRecord.Set("follower", actor.Id)
        followRecord.Set("followee", object.Id)

        // NEW: instance actors require admin approval; person actors auto-accept
        if actor.GetString("actor_type") == "instance" {
            followRecord.Set("status", "pending")
            err = app.Save(followRecord)
            return err  // Return early: no Accept sent, no notification
        }

        followRecord.Set("status", "accepted")
        err = app.Save(followRecord)
        if err != nil {
            return err
        }
    }

    // existing Accept + notification logic (person path only) ...
}
```

**Key detail:** The `actor_type` value for the instance actor is `"instance"` (not `"Application"`). This is the select field value established by the Phase 1 migration `1782290000_add_actor_type_to_activitypub_actors.go` with values `["person", "instance"]`. [VERIFIED: db/migrations/1782290000_add_actor_type_to_activitypub_actors.go, db/federation/instance.go:89]

### Pattern 3: Status-Change Detection in Update Hook

**What:** PocketBase v0.38.0 provides `Record.Original()` which returns a snapshot of the record state before the current save. Use this to detect status transitions in the update hook.

```go
// Source: PocketBase v0.38.0 core/record_model.go:618
// Pattern for OnRecordAfterUpdateSuccess("follows") hook:

func InstanceFollowUpdateHandler() func(e *core.RecordEvent) error {
    return func(e *core.RecordEvent) error {
        // Filter: only handle instance-level follows
        if !isInstanceFollow(e.App, e.Record) {
            return e.Next()
        }

        newStatus := e.Record.GetString("status")
        oldStatus := e.Record.Original().GetString("status")

        if oldStatus == newStatus {
            return e.Next()  // No status change
        }

        instanceActor := getInstanceActor(e.App)  // fetch local instance actor

        switch newStatus {
        case "accepted":
            // deliver Accept{Follow}
        case "rejected":
            // deliver Reject{Follow}
        }
        return e.Next()
    }
}
```

### Pattern 4: Hook Registration in main.go

The phase adds three new hook registrations to `setupEventHandlers()`. The distinction between hook variants matters:

| PocketBase Hook | Event Type | When to Use |
|-----------------|------------|-------------|
| `OnRecordCreateRequest("follows")` | `*core.RecordRequestEvent` | Fires on HTTP API POST (existing user follow path) |
| `OnRecordDeleteRequest("follows")` | `*core.RecordRequestEvent` | Fires on HTTP API DELETE (existing user unfollow path) |
| `OnRecordAfterCreateSuccess("follows")` | `*core.RecordEvent` | Fires after any successful create (HTTP API + admin panel + direct `app.Save()`) |
| `OnRecordAfterUpdateSuccess("follows")` | `*core.RecordEvent` | Fires after any successful update (HTTP API + admin panel + direct `app.Save()`) |
| `OnRecordAfterDeleteSuccess("follows")` | `*core.RecordEvent` | Fires after any successful delete (HTTP API + admin panel + direct `app.Delete()`) |

For Phase 2, the new hooks must use the `AfterSuccess` variants because admin-initiated changes come through the PocketBase admin panel, not the HTTP API. The existing `CreateFollowHandler` and `DeleteFollowHandler` use `OnRecordCreateRequest`/`OnRecordDeleteRequest` which handle the HTTP API user-follow path — those are NOT triggered by admin panel operations. [VERIFIED: db/main.go:127-128, PocketBase v0.38.0 core/app.go:1040,1070,1100]

```go
// Source: db/main.go (setupEventHandlers pattern — new additions)
// These THREE lines are added alongside existing follows registrations:
app.OnRecordAfterCreateSuccess("follows").BindFunc(hooks.InstanceFollowCreateHandler())
app.OnRecordAfterUpdateSuccess("follows").BindFunc(hooks.InstanceFollowUpdateHandler())
app.OnRecordAfterDeleteSuccess("follows").BindFunc(hooks.InstanceFollowDeleteHandler())
```

### Pattern 5: Filtering Hook to Instance-Level Follows

**What:** The new hooks must not fire for user-level follows. The distinguishing characteristic is whether the `follower` or `followee` actor record has `actor_type == "instance"` and `is_local == true`.

```go
// Helper: determine if a follows record involves the local instance actor
func isInstanceFollow(app core.App, follow *core.Record) bool {
    // Check follower
    followerActor, err := app.FindRecordById("activitypub_actors", follow.GetString("follower"))
    if err == nil && followerActor.GetString("actor_type") == "instance" && followerActor.GetBool("is_local") {
        return true
    }
    // Check followee
    followeeActor, err := app.FindRecordById("activitypub_actors", follow.GetString("followee"))
    if err == nil && followeeActor.GetString("actor_type") == "instance" && followeeActor.GetBool("is_local") {
        return true
    }
    return false
}
```

Alternatively, look up the local instance actor IRI from the environment and compare directly:

```go
func isInstanceFollow(app core.App, follow *core.Record) bool {
    origin := os.Getenv("ORIGIN")
    instanceIRI := origin + "/api/v1/activitypub/instance"

    for _, field := range []string{"follower", "followee"} {
        actor, err := app.FindRecordById("activitypub_actors", follow.GetString(field))
        if err == nil && actor.GetString("iri") == instanceIRI {
            return true
        }
    }
    return false
}
```

The IRI-based approach is marginally more reliable since it doesn't depend on the `actor_type` field being set correctly on remote actors (which may not have `actor_type` set at all). [VERIFIED: db/federation/instance.go:48, db/migrations/1782290000_add_actor_type_to_activitypub_actors.go]

### Pattern 6: Route Registration for Instance Inbox

```go
// Source: db/main.go registerRoutes() — new line alongside existing AP routes
se.Router.POST("/activitypub/instance/inbox", federation.InstanceInboxHandler)
```

The existing routes show the pattern: `se.Router.POST("/activitypub/activity/process", routes.ActivitypubActivityProcess)`. The instance inbox mirrors this at a new path. [VERIFIED: db/main.go:185]

### Anti-Patterns to Avoid

- **Routing instance inbox through `ActivitypubActivityProcess`:** D-03 explicitly prohibits this. Sharing the handler would conflate instance-actor and user-actor concerns and make it harder to apply instance-specific filtering (e.g., restricting activity types, applying different auth requirements).
- **Using `OnRecordCreateRequest`/`OnRecordDeleteRequest` for admin-initiated actions:** These hooks only fire on HTTP API requests, not admin panel saves. Admin panel operations go through `app.Save()` / `app.Delete()` which trigger the `AfterSuccess` hooks instead.
- **Sending `Accept` from `ProcessFollowActivity` for Application actors:** The current code sends Accept unconditionally. The branch must `return` early for `actor_type == "instance"` before the Accept code runs.
- **Omitting `Reject{Follow}` delivery:** D-08: this is mandatory. A remote instance that sent a Follow and never receives Accept or Reject is stuck in a permanently pending state with no recovery path.
- **Sending `Undo{Follow}` for user-level follow deletions:** The `OnRecordAfterDeleteSuccess("follows")` hook fires for all follow deletions. Must filter with `isInstanceFollow()` before dispatching.
- **Not adding `"rejected"` to the follows.status migration:** The `status` select field currently only allows `["pending", "accepted"]`. Attempting to `Set("status", "rejected")` and save will fail PocketBase validation. A migration is required.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP signature verification | Custom sig verifier | `util.VerifySignature(app, req, publicKeyPem)` | Already implements `httpsig` correctly with `X-Forwarded-Path` header handling |
| Outgoing AP activity delivery | Custom HTTP POST | `federation.PostActivity(app, actor, activity, inboxes)` | Already handles signing, concurrency (semaphore), deduplication, error logging |
| Remote actor fetch-or-cache | Custom HTTP fetch | `federation.GetActorByIRI(app, ctx, iri, false)` | 2-hour cache, validation, upsert — handles all edge cases |
| AP activity construction | Raw JSON building | `pub.AcceptNew()`, `pub.RejectNew()`, `pub.UndoNew()`, `pub.FollowNew()` | Type-safe, serializes correctly with go-ap |
| Pre-update state access | Store old value in separate variable | `e.Record.Original().GetString("status")` | PocketBase built-in; available in all `AfterSuccess` hooks |

**Key insight:** Every building block for this phase already exists in the codebase. The risk area is hook selection (Request vs. AfterSuccess) and the missing `"rejected"` status value — these are the two things most likely to create silent failures.

## Common Pitfalls

### Pitfall 1: Wrong Hook Family for Admin Panel Operations

**What goes wrong:** `OnRecordCreateRequest` and `OnRecordDeleteRequest` never fire when the admin creates or deletes follows records via PocketBase's built-in admin UI. The outgoing Follow and Undo activities are never sent.
**Why it happens:** `Request` hooks bind to the HTTP request lifecycle of the PocketBase API endpoint. Admin panel operations bypass the HTTP API and call `app.Save()`/`app.Delete()` directly.
**How to avoid:** Use `OnRecordAfterCreateSuccess`, `OnRecordAfterUpdateSuccess`, `OnRecordAfterDeleteSuccess` for the three new follow lifecycle hooks.
**Warning signs:** Follow activity never appears in remote instance logs; no error in local Wanderer logs.

### Pitfall 2: Missing "rejected" Status Value in Migration

**What goes wrong:** Calling `followRecord.Set("status", "rejected"); app.Save(record)` fails with a PocketBase validation error because `"rejected"` is not in the select field's allowed values.
**Why it happens:** The `follows.status` field only has `["pending", "accepted"]` (verified in `1747064968_collections_snapshot.go`). `"rejected"` was never added.
**How to avoid:** Add a migration before any code that writes `status = "rejected"`.
**Warning signs:** `app.Save()` returns a non-nil error mentioning the status field or validation.

### Pitfall 3: Accept Sent to Application-Type Actors

**What goes wrong:** An incoming Follow from a remote Wanderer instance gets auto-accepted and the follow record is set to `"accepted"` — bypassing the admin approval requirement (FLCL-02).
**Why it happens:** `ProcessFollowActivity()` currently sends Accept unconditionally for all remote actors. The actor-type branch must return early before the Accept code.
**How to avoid:** In `ProcessFollowActivity()`, add the `actor_type == "instance"` check immediately after creating the pending follow record, and `return nil` before the Accept activity is constructed.
**Warning signs:** Remote instance shows Follow as accepted immediately; `follows` record has `status = "accepted"` right after the Follow arrives; admin never sees a pending approval.

### Pitfall 4: Hook Fires for User-Level Follows

**What goes wrong:** Every time a user (not admin) follows another user, the instance lifecycle hooks fire and attempt to send `Accept{Follow}` / `Reject{Follow}` / `Undo{Follow}` to the remote user's inbox using the instance actor's key — which the remote will reject as unauthorized.
**Why it happens:** `OnRecordAfterUpdateSuccess("follows")` fires on every follows record update, not just instance-level ones.
**How to avoid:** First line of every new hook: call `isInstanceFollow(e.App, e.Record)` and return early if false.
**Warning signs:** AP delivery errors in logs for user follow updates; remote user actors receiving unexpected Accept/Reject activities signed by the instance key.

### Pitfall 5: GetActorByIRI Requires Context with Actor Identity

**What goes wrong:** Calling `GetActorByIRI(app, context.Background(), iri, false)` silently skips signing the outbound actor fetch request. Remote instances running "authorized fetch" mode return 401 and the actor cannot be resolved.
**Why it happens:** `fetchRemoteActor()` checks for a `userActor` in the context (via `ctx.Value("actor")`). If context is bare, no signing occurs.
**How to avoid:** Use `util.GetSafeActorContext(nil, instanceActor)` to build a context that attaches the local instance actor as the signing identity. The `nil` for the request parameter is safe (used in `hooks/trails.go:57`).
**Warning signs:** "actor fetch failed: status 401" in logs when processing incoming follows from strict AP servers.

### Pitfall 6: X-Forwarded-Path Must Be Set for Signature Verification

**What goes wrong:** `util.VerifySignature` reads `X-Forwarded-Path` to reconstruct the signed request path. If the reverse proxy does not forward this header for the instance inbox endpoint, signature verification always fails.
**Why it happens:** The function at `db/util/activitypub.go:625` rewrites `req.URL.Path` from the `X-Forwarded-Path` header to match what the sender signed. Without it, the path is empty and the signature check fails.
**How to avoid:** The existing user inbox path (`/activitypub/activity/process`) already relies on this header. The instance inbox at `/activitypub/instance/inbox` goes through the same proxy config — verify that the Caddy/nginx/traefik config forwards `X-Forwarded-Path` for all `/activitypub/*` paths.
**Warning signs:** "Invalid http signature" errors for all incoming activities to the instance inbox.

## Code Examples

### Outgoing Follow from Admin-Created follows Record

```go
// Source: db/hooks/follow.go (existing CreateFollowHandler — instance variant follows same shape)
// In InstanceFollowCreateHandler, filter first, then use CreateFollowActivity:

func InstanceFollowCreateHandler() func(e *core.RecordEvent) error {
    return func(e *core.RecordEvent) error {
        if !isInstanceFollow(e.App, e.Record) {
            return e.Next()
        }
        // CreateFollowActivity already handles this: looks up follower+followee actors,
        // constructs a Follow activity, calls PostActivity, saves activitypub_activities record.
        // Call federation.CreateFollowActivity(e.App, e.Record) directly.
        if err := federation.CreateFollowActivity(e.App, e.Record); err != nil {
            e.App.Logger().Error(fmt.Sprintf("instance follow activity failed: %v", err))
        }
        return e.Next()
    }
}
```

**Note:** `CreateFollowActivity` in `db/federation/follow.go` looks up both actors by their record IDs in `follower`/`followee` fields, constructs `pub.FollowNew(...)`, calls `PostActivity(app, followerActor, activity, [followeeActor.inbox])`, and saves an `activitypub_activities` record. This is exactly what the outgoing instance Follow needs — it can be called directly without modification. [VERIFIED: db/federation/follow.go:16-61]

### Accept{Follow} Delivery Pattern

```go
// Source: db/federation/follow.go ProcessFollowActivity() lines 94-103 (existing Accept pattern)
// Replicate this for the Accept delivery from the update hook:

recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)
id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

// Wrap the original Follow activity as the Accept's object
// Must find the original Follow activity IRI from activitypub_activities
acceptActivity := pub.AcceptNew(pub.IRI(id), originalFollowActivity)
acceptActivity.Actor = pub.IRI(instanceActor.GetString("iri"))
PostActivity(app, instanceActor, acceptActivity, []string{remoteActor.GetString("inbox")})
```

### Reject{Follow} Delivery Pattern

```go
// Source: go-ap activity.go:704-709 (RejectNew constructor)
// pub.RejectNew is a type alias of Activity with RejectType, same signature as AcceptNew:

rejectActivity := pub.RejectNew(pub.IRI(id), originalFollowActivity)
rejectActivity.Actor = pub.IRI(instanceActor.GetString("iri"))
PostActivity(app, instanceActor, rejectActivity, []string{remoteActor.GetString("inbox")})
```

### Fetching the Local Instance Actor in a Hook

```go
// Source: db/federation/instance.go:48 (IRI construction)
// In hooks, there is no request context — fetch the instance actor by its known IRI:

origin := os.Getenv("ORIGIN")
instanceIRI := origin + "/api/v1/activitypub/instance"
instanceActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", instanceIRI)
```

## Schema Analysis

### follows Collection (current state)

**Collection ID:** `8obn1ukumze565i`

| Field | Type | Values / Notes |
|-------|------|----------------|
| `id` | text (PK) | auto-generated 15-char |
| `follower` | relation → `activitypub_actors` | required; cascade delete |
| `followee` | relation → `activitypub_actors` | required; cascade delete |
| `status` | select | `["pending", "accepted"]` — **`"rejected"` IS MISSING** |
| `created` | autodate | set on create |
| `updated` | autodate | set on create and update |

**Access rules (current):**
- `listRule`: `@request.auth.id = follower.user.id || @request.auth.id = followee.user.id`
- `createRule`: `@request.auth.id = follower.user.id`

**Gap:** The `"rejected"` value is missing from the `status` select field. A migration must add it before implementing FLCL-04.

**Note for admin-initiated outgoing Follow (FLCL-01):** When an admin creates a follows record directly, the `follower` must be the local instance actor's record ID (not a user actor). The `createRule` restricts API-level creates to `@request.auth.id = follower.user.id` — but admin panel operations bypass collection access rules entirely. The admin can create any record directly. No rule change is needed for v1. [VERIFIED: db/migrations/1747064968_collections_snapshot.go]

### activitypub_actors actor_type field

| Value | Meaning |
|-------|---------|
| `"person"` | User actor (backfilled on all existing actors by Phase 1 migration) |
| `"instance"` | Local instance actor (set by `InitInstanceActor`) |

Remote actors fetched by `GetActorByIRI`/`assembleActor` do NOT have `actor_type` set (the field is only written locally). To identify a remote Application-type actor in `ProcessFollowActivity`, the check should be against the actor's AP `Type` field from the activity payload, OR against whether the incoming actor's IRI matches the `/activitypub/instance` path pattern. The most reliable approach: check `actor.GetString("actor_type") == "instance"` — this works because `GetActorByIRI` calls `assembleActor` which calls `app.Save(dbActor)`, and when fetching a remote instance actor the `actor_type` field would be unset (empty). A remote Wanderer instance actor sent a Follow will appear with `actor_type = ""` unless we explicitly set it during `assembleActor`. This means the current approach of checking `actor_type == "instance"` on the `actor` record in `ProcessFollowActivity` will NOT reliably detect a remote Application-type actor.

**Correct filter logic for FLCL-02:** Instead of checking the `actor` record's `actor_type`, check whether the Follow activity's `object` (the local followee) is the instance actor:

```go
// In ProcessFollowActivity — check if the Follow targets the local instance actor
object, err := app.FindFirstRecordByData("activitypub_actors", "iri", activity.Object)
// ...
if object.GetString("actor_type") == "instance" && object.GetBool("is_local") {
    // This Follow is directed at our instance actor — require admin approval
    followRecord.Set("status", "pending")
    return app.Save(followRecord)
}
// Person-level follow: existing auto-accept path
```

This is more reliable than checking the sending actor's type, because the local instance actor is always known and its `actor_type` is set correctly. [VERIFIED: db/federation/instance.go:89, db/migrations/1782290000_add_actor_type_to_activitypub_actors.go]

## State of the Art

| Old Approach | Current Approach | Impact for Phase 2 |
|--------------|------------------|-------------------|
| Hardcoded Follow auto-accept | Actor-type branch determines accept/pending | Phase 2 adds this branch |
| User-only follows | Instance-level follows via same `follows` collection | Reuse; add hook filter |
| No `"rejected"` status | Add `"rejected"` via migration | Migration required |
| No instance inbox endpoint | Dedicated handler in `instance.go` | Phase 2 adds this |

**Deprecated/outdated:**
- The comment `// err = util.SyncOutbox(app, actor)` in `ProcessAcceptActivity` is commented out — this was a planned outbox sync that was deferred. Do not uncomment it.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Admin panel operations on follows records trigger `OnRecordAfterCreateSuccess`/`OnRecordAfterUpdateSuccess`/`OnRecordAfterDeleteSuccess` hooks | Architecture Patterns — Hook Registration | If admin panel bypasses these hooks too, the outgoing activities are never sent. Mitigation: verify in PocketBase 0.38.0 docs or integration test. |
| A2 | The remote Wanderer instance actor's inbox URL is correctly stored in `activitypub_actors.inbox` after `GetActorByIRI` is called | Patterns | If `assembleActor` fails to populate `inbox` from the remote actor JSON, `PostActivity` delivers to an empty URL. The `validateActorResponse` function checks for inbox presence, so fetch failure is caught. |
| A3 | `Record.Original()` returns pre-update data in `OnRecordAfterUpdateSuccess` hooks | Pattern 3 | If `Original()` returns the post-save state, status-change detection is broken. PocketBase source confirms `originalData` is the pre-save state. |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **Does PocketBase admin panel trigger `OnRecordAfterCreateSuccess`?**
   - What we know: The PocketBase admin panel uses `app.Save()` internally, not the HTTP API. The `AfterSuccess` hooks bind to the model lifecycle, not the request lifecycle.
   - What's unclear: The exact chain from admin panel save → hook invocation in PocketBase v0.38.0 should be confirmed with a small integration test.
   - Recommendation: Add a `go test` that calls `app.Save(followRecord)` directly and asserts the hook fires. This is lower risk than discovering it at runtime.

2. **How does the admin identify the remote instance actor IRI to enter in the follows record?**
   - What we know: D-01 says the admin supplies the remote actor IRI. The `followee` field is a relation to `activitypub_actors`.
   - What's unclear: The admin must either know the remote actor IRI upfront OR the system must fetch/create the remote actor record first. If the `activitypub_actors` record for the remote instance doesn't exist yet, the admin can't set the `followee` relation.
   - Recommendation: The `OnRecordAfterCreateSuccess` hook for the outgoing Follow should handle the case where the `followee` actor is not yet in the local DB — it should call `GetActorByIRI` to fetch and create it automatically. Alternatively, provide admin instructions to first fetch the remote actor via the existing actor endpoint.

3. **Must the original Follow activity IRI be stored to construct Accept/Reject objects?**
   - What we know: `pub.AcceptNew(id, originalFollow)` wraps the Follow activity. `ProcessFollowActivity` in the existing code receives the activity directly from the incoming request. For Accept/Reject from the update hook, the original Follow activity must be retrieved from `activitypub_activities` by actor + object + type.
   - What's unclear: The incoming Follow from a remote instance actor may or may not be stored in `activitypub_activities` (the existing `ProcessFollowActivity` for person actors doesn't store it — only the `Accept` is stored). A separate store-the-incoming-Follow step may be needed.
   - Recommendation: In `ProcessFollowActivity` for Application-type actors, store the incoming Follow activity in `activitypub_activities` before returning, so the update hook can look it up by `actor` + `object` + `type = "Follow"`.

## Environment Availability

This phase is purely code and migration changes within the existing Go backend. No external services beyond those already running are required.

| Dependency | Required By | Available | Notes |
|------------|------------|-----------|-------|
| Go toolchain | Compilation | ✓ | Already in use |
| PocketBase v0.38.0 | All hooks | ✓ | `go.mod` confirmed |
| `go-ap/activitypub` | AP types | ✓ | Already in `go.mod` |
| `go-fed/httpsig` | Signature | ✓ | Already in `go.mod` |
| ORIGIN env var | IRI construction | ✓ | Required at runtime; startup verifies |
| POCKETBASE_ENCRYPTION_KEY | Key decryption | ✓ | Required at runtime; startup verifies |

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase 2 |
|-----------|------------------|
| Go/PocketBase backend only | All new code goes in `db/` — no SvelteKit or Flutter changes |
| Must use standard AP types | Use `pub.AcceptNew`, `pub.RejectNew`, `pub.UndoNew` from go-ap |
| Privacy hard constraint: `is_public = false` never federated | Not directly relevant to Follow lifecycle (Phase 3 concern) |
| Online-only: dropped activities are dropped | `PostActivity` already implements fire-and-forget; no retry needed |
| No breaking changes to user-level federation | The actor-type branch in `ProcessFollowActivity` must not change behavior for `person` actors; the new hooks must filter and skip user-level follows |
| Additive only | Instance actor and follows hooks are new additions alongside existing code |

## Sources

### Primary (HIGH confidence)
- `db/routes/activitypub.go` — `ActivitypubActivityProcess` is the canonical inbox handler pattern; HTTP sig verification pattern at lines 106-109
- `db/util/activitypub.go` — `VerifySignature()` implementation at line 615; `GetSafeActorContext` at line 173
- `db/federation/activity.go` — `PostActivity()` signature and behavior at line 55
- `db/federation/follow.go` — `ProcessFollowActivity()` and `CreateFollowActivity()` full source
- `db/federation/undo.go` — `CreateUnfollowActivity()` pattern for Undo construction
- `db/federation/actor.go` — `GetActorByIRI()` at line 98
- `db/hooks/follow.go` — existing `CreateFollowHandler`/`DeleteFollowHandler` using `RecordRequestEvent`
- `db/main.go` — all hook registrations; route registration pattern
- `db/migrations/1747064968_collections_snapshot.go` — `follows` collection schema with status values `["pending", "accepted"]`
- `db/migrations/1782290000_add_actor_type_to_activitypub_actors.go` — `actor_type` select field values `["person", "instance"]`
- `db/federation/instance.go` — `InitInstanceActor()` sets `actor_type = "instance"` at line 89
- PocketBase v0.38.0 `core/app.go` — hook API: `OnRecordAfterCreateSuccess`, `OnRecordAfterUpdateSuccess`, `OnRecordAfterDeleteSuccess` at lines 1040, 1070, 1100
- PocketBase v0.38.0 `core/record_model.go:618` — `Record.Original()` method returning pre-save state
- go-ap `activitypub@v0.0.0-20250409143848` `activity.go:704` — `RejectNew()` constructor; `AcceptNew()` at line confirming type alias structure

### Secondary (MEDIUM confidence)
- `.planning/phases/02-follow-lifecycle/02-CONTEXT.md` — locked implementation decisions D-01 through D-08
- `.planning/REQUIREMENTS.md` — INST-03, FLCL-01 through FLCL-05 acceptance criteria

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all components verified in existing codebase
- Architecture: HIGH — patterns verified against existing production code
- Hook semantics: HIGH — verified against PocketBase v0.38.0 source
- Schema gaps: HIGH — `"rejected"` confirmed missing from migration snapshot
- Actor-type filter logic: MEDIUM — the recommended approach (filter on `object.actor_type`) is more reliable than the CONTEXT.md suggestion (filter on `actor.actor_type`), but the nuance should be confirmed during planning

**Research date:** 2026-06-25
**Valid until:** 2026-07-25 (stable codebase; no fast-moving dependencies)
