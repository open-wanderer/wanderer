# Architecture Patterns: Instance-Level ActivityPub Federation

**Domain:** ActivityPub instance actor federation for Wanderer
**Researched:** 2026-06-22
**Confidence:** HIGH — based on direct codebase analysis, no speculation

---

## Existing Architecture (Baseline)

Understanding what already exists is essential before specifying what changes.

### Actor Model

`activitypub_actors` is a single flat PocketBase collection that stores **both local and remote actors** under the same schema. Local actors have `is_local = true` and a non-null `user` relation to the `users` collection. Remote actors have `is_local = false` and `user = null`. There is currently **no `actor_type` column** — all actors are implicitly `Person`.

Key fields relevant to instance actors:

| Field | Type | Notes |
|-------|------|-------|
| `id` | text (15 chars) | PocketBase record ID |
| `iri` | url | Unique index — the canonical ActivityPub ID |
| `preferred_username` | text | Handle, e.g. `"instance"` |
| `domain` | text | e.g. `"wanderer.example.com"` |
| `inbox` | url | Where inbound activities are delivered |
| `outbox` | url | Where outgoing activities are listed |
| `followers` | url | Followers collection URL |
| `following` | url | Following collection URL |
| `is_local` | bool | `true` for actors on this instance |
| `public_key` | text | PEM-encoded RSA public key |
| `private_key` | text | Encrypted RSA private key (AES via `POCKETBASE_ENCRYPTION_KEY`) |
| `user` | relation → users | `null` for the instance actor |

### Follow Model

`follows` collection: `follower` (activitypub_actors.id) → `followee` (activitypub_actors.id) + `status` (`pending`/`accepted`). The same collection is used for user-to-user follows and will be used for instance-to-instance follows — no schema change needed for the relationship itself.

### Fanout: `followerInboxes()`

`db/federation/activity.go:31` — single JOIN query:

```go
app.DB().
    Select("aa.inbox").
    From("follows f").
    InnerJoin("activitypub_actors aa", dbx.NewExp("f.follower = aa.id")).
    Where(dbx.NewExp("f.followee = {:followee} AND f.status = 'accepted' AND aa.inbox != ''", ...)).
    Rows()
```

The `followee` is always the **content author's actor ID**. This is the single chokepoint for all outbound delivery.

### Inbox Processing

The SvelteKit route `/api/v1/activitypub/user/[handle]/inbox` (POST) receives inbound activities. It:
1. Forwards the raw body + all HTTP headers (including `X-Forwarded-Path`) to the Go backend at `/activitypub/activity/process`.
2. The Go handler (`db/routes/activitypub.go:66`) resolves the `recipient` actor by matching `inbox` URL to `activitypub_actors.inbox`.
3. Verifies the HTTP signature of the sending actor.
4. Dispatches to `ProcessFollowActivity`, `ProcessAcceptActivity`, `ProcessCreateOrUpdateActivity`, `ProcessDeleteActivity`, etc.

`ProcessCreateOrUpdateActivity` receives both `actor` (the sender) and `recipient` (the local actor whose inbox received it) because the feed-insertion functions (`InsertIntoFeed`) need the recipient's ID.

### Hook Registration Pattern

`db/main.go:setupEventHandlers()` registers all PocketBase lifecycle hooks. Trail fanout hooks are `OnRecordAfterCreateSuccess("trails")` and `OnRecordAfterUpdateSuccess("trails")`. These fire after a record is durably saved, which is correct — no activity is broadcast before the DB write commits.

---

## What Needs to Change

### 1. Schema: Add `actor_type` Column to `activitypub_actors`

The current schema has no column to distinguish `Person` actors from `Application` (instance) actors. The instance actor needs to be discoverable as an `Application` type both internally (for routing decisions) and externally (for ActivityPub compliance).

**Change:** Add a `actor_type` text field (default `"Person"`) to `activitypub_actors`. The instance actor will be created with `actor_type = "Application"`. This is a non-breaking additive migration — all existing actors implicitly remain `Person`.

**Why not a separate collection:** The instance actor participates in the exact same `follows` JOIN that already drives all fanout. Putting it in a separate collection would require duplicating or rewriting `followerInboxes()`, `CreateFollowActivity()`, `ProcessFollowActivity()`, and `PostActivity()`. There is no benefit — it is just overhead.

### 2. Instance Actor Creation: `initData()` at Startup

The instance actor must exist before any hook fires. The correct place is `initData()` in `db/main.go`, which already initializes categories and plugins at serve time. A new `initInstanceActor(app)` function should run there.

**Pattern:** Idempotent upsert — check for an existing record with `actor_type = "Application" AND is_local = true` before creating. If found, skip. This ensures the actor survives restarts without accumulating duplicates.

**Why not a migration:** Migrations run once and cannot access runtime environment variables like `ORIGIN`. The instance actor's `iri`, `inbox`, and `outbox` URLs are all derived from `ORIGIN`. `initData()` already handles this pattern (see `initCategories()` which checks for existing records before creating). `ActorFromUser()` in `db/util/activitypub.go` is the reference implementation for key generation and actor field population — `initInstanceActor()` should follow the same pattern, setting `user = null` and `actor_type = "Application"`.

**IRI convention:** `{ORIGIN}/api/v1/activitypub/instance` — distinguishable from user actors at `{ORIGIN}/api/v1/activitypub/user/{handle}`.

### 3. Fanout: Extend `followerInboxes()` — Do Not Create a Separate Pass

When Instance A creates a public trail, Instance B needs to receive a `Create` activity. Instance B's instance actor will be a follower of the **trail author's actor** (because that is what `followerInboxes()` queries) **only if** Instance B sent a user-level follow. This is not sufficient for instance-level sync.

The correct model: when Instance B's instance actor follows Instance A's instance actor and the follow is accepted, Instance B expects to receive activities for **all public content** on Instance A — not just content from actors that Instance B individually follows.

**The gap:** `followerInboxes(app, trailAuthor.Id)` queries followers of the trail author, not followers of the instance actor. The instance actor at Instance B is a follower of Instance A's instance actor, not of every individual trail author.

**Solution:** Extend the fanout with a second inbox set for instance-actor followers. In each `CreateTrailActivity`, `CreateCommentActivity`, `CreateSummitLogActivity`, and `CreateListActivity` function:

```go
// Existing: user-level followers of the content author
inboxes, err := followerInboxes(app, contentAuthor.Id)

// New: instance-level followers (followers of this instance's instance actor)
instanceActor, err := findLocalInstanceActor(app)
if err == nil {
    instanceInboxes, err := followerInboxes(app, instanceActor.Id)
    if err == nil {
        inboxes = append(inboxes, instanceInboxes...)
    }
}
recipients := append(mentions, inboxes...)
```

`PostActivity()` already deduplicates recipients (`slices.Compact` after `slices.Sort`) so no inbox will receive duplicate deliveries even if a user-level follow and an instance-level follow both resolve to the same inbox URL.

**Why not a separate delivery pass:** `PostActivity()` is already concurrent (goroutine per inbox, semaphore-bounded at 5). Merging the two inbox sets into one call is simpler, avoids double-goroutine overhead, and lets deduplication work across both sets.

**The signing actor:** `PostActivity()` signs with the actor record passed as its second argument. For instance-level fanout deliveries, this should be the **content author's actor**, not the instance actor. The activity's `actor` field in the JSON body is already the content author's IRI. Changing the signing key to the instance actor would create a mismatch between the `actor` field and the signing key, which remote signature verification would reject.

### 4. Instance Actor's Inbox: Requires a New SvelteKit Route

The current inbox route is `/api/v1/activitypub/user/[handle]/inbox`. This path pattern (`user/`) won't work for an instance actor at `instance/`. A new route is needed:

`web/src/routes/api/v1/activitypub/instance/inbox/+server.ts`

This route is structurally identical to the user inbox handler — it forwards to `/activitypub/activity/process` with `X-Forwarded-Path` set. The Go backend already resolves the recipient by matching `activitypub_actors.inbox` to the forwarded path, so no changes to the Go route handler are needed, **as long as** the instance actor's `inbox` field is set to `{ORIGIN}/api/v1/activitypub/instance/inbox`.

### 5. Incoming Activities to the Instance Actor Inbox: Existing Handlers Work With One Constraint

When Instance B (follower) receives a `Create` activity delivered to the instance actor inbox, the Go handler calls `ProcessCreateOrUpdateActivity(app, actor, recipient, activity)` where `recipient` is the instance actor.

The issue: `processCreateOrUpdateTrailActivity` and `processCreateOrUpdateListActivity` call `InsertIntoFeed(app, recipient.Id, ...)`, which inserts a feed entry for the recipient actor. The instance actor has no user and no personal feed. Inserting feed records for it is harmless but meaningless, and `InsertIntoFeed` may fail if it tries to dereference the `user` relation.

**Solution:** Add an `actor_type` check at the top of `ProcessCreateOrUpdateActivity`:

```go
func ProcessCreateOrUpdateActivity(app core.App, actor *core.Record, recipient *core.Record, activity pub.Activity) error {
    isInstanceRecipient := recipient.GetString("actor_type") == "Application"
    // ... dispatch as before, but pass isInstanceRecipient to sub-handlers
}
```

Sub-handlers skip `InsertIntoFeed` when the recipient is the instance actor. All other processing (upsert trail record, upsert comment record, etc.) is unchanged. The content itself is stored on the receiving instance — only the feed insertion is skipped.

### 6. Follow Lifecycle: Manual Admin Action, Not Auto-Accept

Current behavior: `ProcessFollowActivity()` auto-accepts all incoming follows from remote actors. Instance-to-instance follows must **not** be auto-accepted. The admin on each side must explicitly approve.

**Change:** In `ProcessFollowActivity()`, check the `actor_type` of the incoming actor. If `actor_type == "Application"` (i.e., an instance actor is requesting a follow), create the follow record with `status = "pending"` and do **not** send an `Accept` back. The existing `follows` collection is already used by the admin UI in PocketBase — the admin can manually set `status = "accepted"` through the PocketBase admin panel, which triggers `ProcessAcceptActivity` (if the accept is federated back) or requires a custom hook.

More precisely: the admin's action of setting `status = "accepted"` on the `follows` record should trigger an `OnRecordAfterUpdateSuccess("follows")` hook that calls `federation.CreateAcceptActivity()` to send the Accept back to the requesting instance. This hook does not currently exist and must be added.

**Outgoing follow initiation:** The admin creates a `follows` record in PocketBase admin UI where `follower = instanceActor.Id` and `followee = remoteInstanceActorId`. The existing `CreateFollowHandler()` hook fires `OnRecordCreateRequest("follows")` and calls `federation.CreateFollowActivity()` — this already works, because `CreateFollowActivity` looks up both actors by ID and posts the Follow activity to the followee's inbox.

---

## Component Boundaries

| Component | New or Extended | Responsibility | File |
|-----------|----------------|----------------|------|
| **Instance Actor Init** | New | Idempotent upsert of instance actor at startup | `db/main.go:initInstanceActor()` |
| **Instance Actor Util** | New | `ActorFromInstance()` function (mirrors `ActorFromUser()`) | `db/util/activitypub.go` |
| **Schema Migration** | New | Add `actor_type` column (default `"Person"`) | New migration file |
| **Instance Inbox Route** | New | Forward POSTs at `/api/v1/activitypub/instance/inbox` to Go backend | `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts` |
| **Instance Actor GET Route** | New | Serve the instance actor's JSON-LD representation | `web/src/routes/api/v1/activitypub/instance/+server.ts` |
| **Webfinger Extension** | Extended | Handle `acct:instance@domain` resource in webfinger response | `web/src/routes/.well-known/webfinger/+server.ts` |
| **Fanout Extension** | Extended | Add instance-actor follower inboxes to all four content-type fanout functions | `db/federation/create.go`, `db/federation/delete.go` |
| **Follow Accept Hook** | New | Fire `CreateAcceptActivity` when admin sets `follows.status = "accepted"` for instance follows | `db/hooks/follow.go` |
| **Inbox Processing Guard** | Extended | Skip `InsertIntoFeed` when recipient is instance actor | `db/federation/create.go:ProcessCreateOrUpdateActivity` |
| **Follow Processing Guard** | Extended | Set `status = "pending"` (not auto-accept) for incoming instance-actor follows | `db/federation/follow.go:ProcessFollowActivity` |

---

## Data Flow: Trail Creation on Instance A Reaching Instance B

```
Instance A                                  Instance B
----------                                  ----------

1. Admin user creates trail via web UI
   → POST /api/v1/trail
   → SvelteKit API route saves record via PocketBase client

2. PocketBase fires OnRecordAfterCreateSuccess("trails")
   → hooks.CreateTrailHandler runs

3. CreateTrailHandler checks trail.public == true
   → calls federation.CreateTrailActivity(app, ctx, trail, CreateType)

4. CreateTrailActivity:
   a. Builds ActivityPub Create{Article} object
   b. Saves activity to activitypub_activities collection
   c. Calls followerInboxes(app, trailAuthor.Id)
      → returns inboxes of accepted followers of this specific user
   d. [NEW] Calls followerInboxes(app, instanceActor.Id)
      → returns inboxes of accepted followers of Instance A's instance actor
      → includes Instance B's instance actor inbox URL
   e. Merges + deduplicates both inbox sets
   f. Calls PostActivity(app, trailAuthor, activity, recipients)

5. PostActivity (goroutine):
   a. Signs HTTP request with trailAuthor's private key
   b. POSTs JSON-LD activity to Instance B's instance actor inbox:
      POST https://instanceb.example.com/api/v1/activitypub/instance/inbox

Instance B receives the POST:

6. SvelteKit instance inbox route
   → Adds X-Forwarded-Path: /api/v1/activitypub/instance/inbox
   → Forwards to Go backend: POST /activitypub/activity/process

7. Go ActivitypubActivityProcess handler:
   a. Parses activity body
   b. Resolves recipient = activitypub_actors WHERE inbox = '{ORIGIN}/api/v1/activitypub/instance/inbox'
      → returns Instance B's local instance actor record
   c. Resolves actor = activitypub_actors WHERE iri = activity.Actor
      → returns Instance A's trail author actor (fetched/cached from remote)
   d. Verifies HTTP signature against actor's public key
   e. Dispatches: ProcessCreateOrUpdateActivity(app, actor, recipient, activity)

8. ProcessCreateOrUpdateActivity:
   a. [NEW] Detects recipient.actor_type == "Application" → isInstanceRecipient = true
   b. Routes to processCreateOrUpdateTrailActivity(activity, app, actor, recipient, isInstanceRecipient)

9. processCreateOrUpdateTrailActivity:
   a. Calls util.TrailFromActivity() → upserts trail record locally
   b. [SKIPPED if isInstanceRecipient] InsertIntoFeed — instance actor has no personal feed
   c. Processes @mention notifications as normal
```

**Result:** The trail record from Instance A is stored on Instance B's local database. Users on Instance B can discover and view it.

---

## Data Flow: Instance B Establishing a Follow to Instance A

```
Instance B Admin                            Instance A
----------------                            ----------

1. Admin opens PocketBase admin UI
   → Creates follows record:
     follower = instanceB_actor.Id
     followee = remoteInstanceA_actor.Id  (looked up or created via GetActorByIRI)
     status = "pending"

2. OnRecordCreateRequest("follows") fires
   → hooks.CreateFollowHandler calls federation.CreateFollowActivity(app, follow)

3. CreateFollowActivity:
   a. Looks up follower = instanceB_actor, followee = instanceA_actor
   b. Builds Follow activity
   c. Calls PostActivity(app, instanceB_actor, activity, [instanceA_actor.inbox])
      Signs with Instance B's instance actor private key

4. Instance A receives Follow at /api/v1/activitypub/instance/inbox
   → Go handler dispatches: ProcessFollowActivity(app, instanceB_actor, activity)

5. ProcessFollowActivity [MODIFIED]:
   a. [NEW] Checks incoming actor.actor_type == "Application"
   b. Creates follows record with status = "pending" (does NOT auto-accept)
   c. Does NOT send Accept back immediately

6. Instance A admin views pending follows in PocketBase admin UI
   → Manually sets follows.status = "accepted"

7. OnRecordAfterUpdateSuccess("follows") fires [NEW HOOK]:
   → Checks old status was "pending", new status is "accepted"
   → Calls federation.CreateAcceptActivity(app, follow)
   → Sends Accept back to Instance B's instance actor inbox

8. Instance B receives Accept
   → ProcessAcceptActivity updates local follows.status = "accepted"

Federation is now established. Subsequent Create/Update/Delete activities
from Instance A are delivered to Instance B's instance actor inbox.
```

---

## Build Order and Dependencies

The components have hard dependencies that must be respected. Build in this order:

### Step 1 — Schema Migration (no dependencies)
Add `actor_type` text column (default `"Person"`) to `activitypub_actors`. This is a prerequisite for everything else. The migration must run before `initInstanceActor` can query by `actor_type`.

### Step 2 — Instance Actor Lifecycle (depends on Step 1)
- `db/util/activitypub.go`: Add `ActorFromInstance()` — generates RSA key pair, sets `actor_type = "Application"`, `is_local = true`, `user = null`, IRI at `{ORIGIN}/api/v1/activitypub/instance`.
- `db/main.go`: Add `initInstanceActor(app)` call inside `initData()`. Idempotent: find-or-create by `actor_type = "Application" AND is_local = true`.

### Step 3 — SvelteKit Routes (depends on Step 2, because routes serve the actor whose IRI must exist)
- Instance actor GET route: `web/src/routes/api/v1/activitypub/instance/+server.ts` — serves JSON-LD representation.
- Instance actor inbox POST route: `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts` — identical structure to user inbox handler.
- Webfinger extension: handle `acct:instance@{domain}` resource.

### Step 4 — Follow Lifecycle Changes (depends on Steps 2 and 3)
- `db/federation/follow.go:ProcessFollowActivity`: Guard for `actor_type == "Application"` → pending-only.
- `db/hooks/follow.go`: Add `OnRecordAfterUpdateSuccess("follows")` hook → send Accept when admin approves instance follow.
- Register the new hook in `db/main.go:setupEventHandlers()`.

### Step 5 — Fanout Extension (depends on Step 2)
- `db/federation/create.go`: Extend all four `Create*Activity` functions to also call `followerInboxes(app, localInstanceActor.Id)` and merge results.
- `db/federation/delete.go`: Same extension for all four `CreateTrailDeleteActivity`, `CreateCommentDeleteActivity`, `CreateSummitLogDeleteActivity`, `CreateListDeleteActivity`.
- Add `findLocalInstanceActor(app core.App) (*core.Record, error)` helper function (shared by both files).

### Step 6 — Inbox Processing Guard (depends on Steps 1 and 2)
- `db/federation/create.go:ProcessCreateOrUpdateActivity`: Check `recipient.GetString("actor_type") == "Application"` and skip `InsertIntoFeed` in sub-handlers.

---

## Anti-Patterns to Avoid

### Separate Collection for Instance Actors
**Why bad:** Breaks `followerInboxes()`, `CreateFollowActivity()`, `PostActivity()`, and all inbox routing. Everything depends on a single `activitypub_actors` table. The marginal benefit (slightly cleaner schema) does not justify the cascading changes.

### Auto-Accepting Instance Follows
**Why bad:** An instance follow means all public content flows to the accepting instance indefinitely. Accidental or malicious connections would be impossible to prevent without an immediate unfollow. The existing auto-accept behavior for user follows is appropriate because users choose to follow; instance connections are admin-level decisions.

### Signing Instance-Fanout Deliveries With the Instance Actor Key
**Why bad:** The `actor` field in each Create/Update/Delete activity body references the content author (e.g., `@alice@instance-a.com`). If the HTTP signature is from the instance actor, the signature key ID (`{instanceIRI}#main-key`) does not match the activity's `actor`. Remote instances will fail signature verification or reject the activity as invalid. Sign with the content author's key — the instance actor's key is only used for Follow/Accept activities where the instance actor *is* the activity actor.

### Creating a New Route for Instance-Level Delivery on the Go Backend
**Why unnecessary:** `ActivitypubActivityProcess` already resolves `recipient` from the `inbox` URL. As long as the instance actor's `inbox` field is set to the correct URL and the SvelteKit route forwards to `/activitypub/activity/process` with the right `X-Forwarded-Path`, the Go handler requires no changes for dispatch.

### Separate Fanout Pass for Instance Followers
**Why bad:** `PostActivity()` already handles concurrent delivery with a bounded semaphore. A separate goroutine pass doubles the goroutine overhead without adding value. `slices.Compact` after `slices.Sort` is already called inside `PostActivity()` to deduplicate recipients — merging the two inbox sets before passing to `PostActivity()` is the correct pattern.

---

## Scalability Considerations

| Concern | Current (user-level only) | With Instance Federation |
|---------|--------------------------|--------------------------|
| Fanout per trail create | O(N followers of author) inbox POSTs | O(N user-followers + M connected instances) |
| Instance follows | Not applicable | Bounded — admins manually approve connections |
| Duplicate deliveries | Already deduplicated by `slices.Compact` | Same — deduplication covers both sets |
| Semaphore | 5 concurrent HTTP connections per `PostActivity()` call | Same limit applies — instance inboxes are just more recipients |
| Feed table growth | Feed entries for every received trail | Instance actor has no feed entries (skipped by guard) |

Connected instances are expected to number in the tens, not thousands, for Wanderer deployments. This is not Mastodon relay scale. The existing goroutine-per-inbox model is adequate.

---

## Sources

- Direct code analysis of `db/federation/activity.go`, `create.go`, `delete.go`, `follow.go`, `actor.go`
- `db/routes/activitypub.go` — inbox dispatch and recipient resolution
- `web/src/routes/api/v1/activitypub/user/[handle]/inbox/+server.ts` — inbox forwarding pattern
- `db/main.go` — `initData()`, `setupEventHandlers()`, hook registration patterns
- `db/hooks/trails.go`, `follow.go`, `activitypub_actor.go` — hook implementation patterns
- `db/util/activitypub.go` — `ActorFromUser()` as the reference for actor creation
- `db/migrations/1747061257_created_activitypub_actors.go` — current schema fields
- `db/migrations/1780734977_updated_activitypub_actors.go` — most recent schema change
- ActivityPub spec: `Application` actor type for service/instance actors (W3C ActivityStreams vocabulary)
