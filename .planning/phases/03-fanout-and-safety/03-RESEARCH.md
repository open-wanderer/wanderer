# Phase 3: Fanout and Safety - Research

**Researched:** 2026-06-25
**Domain:** Go/PocketBase ActivityPub federation — content fanout, dedup, privacy gate, delete authorization
**Confidence:** HIGH (all findings verified by direct codebase inspection)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Fanout Injection**
- D-01: Add `instanceFollowerInboxes(app core.App) ([]string, error)` in `db/federation/activity.go`. Computes instance actor IRI as `os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"`, fetches via `FindFirstRecordByData("activitypub_actors", "iri", iri)`, calls `followerInboxes(app, instanceActor.Id)`.
- D-02: If actor not found (`sql.ErrNoRows`), return `(nil, nil)` — empty slice, no error.
- D-03: Each of the 4 outgoing `Create*Activity` functions and 4 outgoing `Delete*Activity` functions calls `instanceFollowerInboxes(app)` and appends to recipients before `PostActivity`.

**Broadcast-Loop Deduplication (SAFE-01)**
- D-04: Dedup inside each `processCreateOrUpdate*` function. If content record with incoming object IRI already exists: return `nil` silently.
- D-05: No changes to `activitypub_activities` for dedup. Content records (their `iri` field) are the dedup store.

**Delete Authorization Fix (SAFE-02)**
- D-06: Change `processDeleteTrailActivity` signature from `(app core.App, activity pub.Activity)` to `(app core.App, actor *core.Record, activity pub.Activity)`. Update call site in `ProcessDeleteActivity`.
- D-07: Add ownership check at top of `processDeleteTrailActivity`: compare `actor.Id` against `trail.GetString("author")`. If not equal: return `fmt.Errorf(...)`. (See SAFE-02 Correction note below.)

**Comment Privacy Gate (SAFE-03)**
- D-08: Add parent trail `public` check at top of `CreateCommentActivity` (whole-function gate). Fetch the comment's parent trail and check `trail.GetBool("public")`. If false: return `nil`.
- D-09: `CreateTrailActivity`, `CreateSummitLogActivity`, `CreateListActivity` already have public gates. No changes needed.

**Instance Inbox — Incoming Content Activities**
- D-10: Extend `InstanceInboxHandler` switch to dispatch `pub.CreateType` and `pub.UpdateType` to `ProcessCreateOrUpdateActivity`, and `pub.DeleteType` to `ProcessDeleteActivity`.

### Claude's Discretion
- Error message text for SAFE-02 unauthorized delete (content of `fmt.Errorf` string)
- Whether `instanceFollowerInboxes` de-duplicates against user-level inboxes

### Deferred Ideas (OUT OF SCOPE)
- Public→private trail propagation (v2 VIS-01)
- Private→public trail propagation (v2 VIS-02)
- De-duplicating instance follower inboxes against user-level follower inboxes
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SYNC-01 | When a public trail, summit_log, list, or comment is created, accepted instance actor followers receive a `Create` activity in addition to existing user-level fanout | D-01/D-02/D-03: `instanceFollowerInboxes` helper + append in all 4 `Create*Activity` functions |
| SYNC-02 | When a public trail, summit_log, list, or comment is updated, an `Update` activity is delivered to all accepted instance actor followers | Same mechanism as SYNC-01 — `Create*Activity` functions handle both Create and Update via `typ` param |
| SYNC-03 | When a trail, summit_log, list, or comment is deleted, a `Delete` activity is delivered to all accepted instance actor followers; receiving instances remove the local cached copy | D-03: append in all 4 `CreateTrail/Comment/SummitLog/ListDeleteActivity` functions; D-10: `InstanceInboxHandler` dispatches incoming Delete to `ProcessDeleteActivity` |
| SAFE-01 | Incoming activity IRI is checked before dispatch; duplicate activities are silently dropped | D-04/D-05: `FindFirstRecordByData` check at top of each `processCreateOrUpdate*` function |
| SAFE-02 | `processDeleteTrailActivity` verifies the deleting actor is the trail's original author before removing the local copy | D-06/D-07: signature change + `trail.GetString("author") != actor.Id` guard |
| SAFE-03 | Outgoing fanout checks `is_public = true` before including any record | D-08: whole-function gate added to `CreateCommentActivity`; existing gates in the other 3 Create functions already satisfy this for trail/summitLog/list |
</phase_requirements>

---

## Summary

Phase 3 makes four targeted, additive changes to the federation package. All code is in `db/federation/` — no hooks files, no route registrations, and no schema migrations are required.

The outgoing fanout change (SYNC-01/02/03 sending side) is a pure injection: a new `instanceFollowerInboxes` helper is added to `activity.go`, and each of the 8 outgoing activity functions appends its result to the recipients slice before calling `PostActivity`. The incoming side (SYNC-01/02/03 receiving side) is a three-line switch extension in `InstanceInboxHandler`.

The safety constraints bundle three fixes: a `FindFirstRecordByData` dedup guard at the top of each `processCreateOrUpdate*` function (SAFE-01); a signature change and actor-ID comparison in `processDeleteTrailActivity` (SAFE-02, modeled exactly after `processDeleteCommentActivity`); and a parent-trail `public` check at the top of `CreateCommentActivity` (SAFE-03, modeled after `CreateSummitLogActivity`).

**Primary recommendation:** Implement in three waves: (1) `instanceFollowerInboxes` helper + fanout injection in all 8 outgoing functions; (2) SAFE-01 dedup guards in all 4 `processCreateOrUpdate*` functions; (3) SAFE-02 signature fix + SAFE-03 comment gate + D-10 inbox extension.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Outgoing content fanout to instance followers | Backend (Go federation) | — | `Create*Activity`/`Delete*Activity` functions own recipient computation; hooks call them |
| Incoming content dispatch (instance inbox) | Backend (Go federation) | — | `InstanceInboxHandler` switch extension; reuses existing processors |
| Broadcast-loop dedup (SAFE-01) | Backend (Go federation) | — | Checked inside `processCreateOrUpdate*` functions before any DB write |
| Privacy gate (SAFE-03) | Backend (Go federation) | — | Whole-function gate at top of outgoing activity functions |
| Delete authorization (SAFE-02) | Backend (Go federation) | — | Ownership check in `processDeleteTrailActivity` before `app.Delete` |

---

## Standard Stack

No new external packages are introduced. This phase uses only existing imports already present in `db/federation/`:

| Already imported | Purpose in this phase |
|------------------|-----------------------|
| `os` | `os.Getenv("ORIGIN")` for instance IRI construction |
| `database/sql` | `sql.ErrNoRows` detection in D-02 and dedup guards |
| `errors` | `errors.Is(err, sql.ErrNoRows)` |
| `fmt` | Error formatting |
| `github.com/pocketbase/pocketbase/core` | `core.App`, `*core.Record` |
| `pub "github.com/go-ap/activitypub"` | `pub.CreateType`, `pub.UpdateType`, `pub.DeleteType` |

`[VERIFIED: codebase]` — confirmed by reading imports in `activity.go`, `create.go`, `delete.go`, `instance.go`.

---

## Package Legitimacy Audit

No new packages are installed in this phase. This section is not applicable.

---

## Architecture Patterns

### System Architecture Diagram

```
Hooks (trails/comments/summit_logs/list)
    |
    v
Create*Activity / Delete*Activity (db/federation/create.go, delete.go)
    |
    |--- followerInboxes(app, authorActor.Id) → user-level inboxes
    |--- instanceFollowerInboxes(app)         → instance follower inboxes  [NEW]
    |
    v
PostActivity(app, actor, activity, merged recipients) → HTTP delivery (goroutine)

Remote instance POST /api/v1/activitypub/instance/inbox
    |
    v
InstanceInboxHandler (db/federation/instance.go)
    |--- Follow/Accept/Undo → existing handlers
    |--- Create/Update [NEW D-10] → ProcessCreateOrUpdateActivity
    |--- Delete      [NEW D-10] → ProcessDeleteActivity
              |
              v
         processCreateOrUpdate*Activity
              |--- dedup check: FindFirstRecordByData → return nil if exists [NEW D-04]
              |--- insert/update content record
         processDeleteTrailActivity
              |--- ownership check: trail.GetString("author") != actor.Id [NEW SAFE-02]
              |--- app.Delete(trail)
```

### Recommended Project Structure

No new files or directories are created. All changes are surgical edits within existing files:

```
db/federation/
├── activity.go      ← ADD instanceFollowerInboxes() helper (D-01/D-02)
├── create.go        ← EDIT 4 Create*Activity (fanout D-03) + 4 processCreateOrUpdate* (dedup D-04) + CreateCommentActivity gate (D-08)
├── delete.go        ← EDIT 4 Delete*Activity (fanout D-03) + processDeleteTrailActivity sig+guard (D-06/D-07) + ProcessDeleteActivity call site (D-06)
└── instance.go      ← EDIT InstanceInboxHandler switch (D-10)
```

### Pattern 1: instanceFollowerInboxes (D-01/D-02)

**What:** Mirrors `followerInboxes` but for the instance actor. Returns `(nil, nil)` on `sql.ErrNoRows` to be startup-safe.

**Example structure (modeled on existing `followerInboxes`):**

```go
// Source: db/federation/activity.go (existing followerInboxes as reference)
func instanceFollowerInboxes(app core.App) ([]string, error) {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return nil, fmt.Errorf("ORIGIN not set")
    }
    iri := origin + "/api/v1/activitypub/instance"
    instanceActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil  // D-02: safe startup behavior
        }
        return nil, err
    }
    return followerInboxes(app, instanceActor.Id)
}
```

### Pattern 2: Fanout Injection in Create*Activity (D-03)

**What:** Append `instanceFollowerInboxes` result to the existing recipients slice before `PostActivity`. Additive — does not disturb existing recipient computation.

**Example from `CreateTrailActivity` (line 91-97 of create.go):**

```go
// Source: db/federation/create.go (current pattern to extend)
inboxes, err := followerInboxes(app, trailAuthor.Id)
if err != nil {
    return err
}
recipients := append(mentions, inboxes...)
// NEW — append instance follower inboxes:
instanceInboxes, err := instanceFollowerInboxes(app)
if err != nil {
    return err
}
recipients = append(recipients, instanceInboxes...)
return PostActivity(app, trailAuthor, activity, recipients)
```

Note: `PostActivity` already deduplicates recipients via `slices.Sort` + `slices.Compact` (activity.go line 101-102), so adding duplicate URLs is safe.

**Critical: `CreateCommentActivity` currently has NO public gate (D-08 gap)**. The function on line 100 has no `if !comment... GetBool("public")` guard, unlike the other three. The gate must check the parent trail's `public` field (identical to `CreateSummitLogActivity` lines 203-206).

### Pattern 3: Dedup Guard in processCreateOrUpdate* (D-04/SAFE-01)

**What:** Check for existing content record by IRI before inserting. Return `nil` silently on duplicate. Modeled on `ProcessFollowActivity`'s idempotency guard (follow.go lines 85-95).

**For `processCreateOrUpdateTrailActivity`:**

The function calls `util.TrailFromActivity` which already calls `app.FindFirstRecordByData("trails", "iri", iri)` internally (activitypub.go line 159). If the trail exists, it sets `needs_full_sync = true` and returns the existing record — it does NOT return nil for duplicates. Therefore the dedup guard for trails must be added as an explicit check BEFORE `util.TrailFromActivity` is called, or by inspecting the returned record's state.

**Simpler approach consistent with D-04:** Add the check at the top of each `processCreateOrUpdate*` function:

```go
// Source: db/federation/follow.go lines 85-95 (idempotency pattern)
existing, err := app.FindFirstRecordByData("trails", "iri", activity.Object.GetID().String())
if err == nil && existing != nil {
    return nil  // already have this content — silent dedup
}
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    return err
}
// ... proceed to insert
```

**For `processCreateOrUpdateCommentActivity`:** The function already does `app.FindFirstRecordByData("comments", "iri", ...)` (create.go line 513). If found, it falls to an upsert path. The dedup guard should return `nil` when the record is found AND `activity.Type == pub.CreateType` (to allow legitimate updates through).

**For `processCreateOrUpdateSummitLogActivity`:** Same pattern — `app.FindFirstRecordByData("summit_logs", "iri", ...)` at line 613. Dedup on Create only.

**For `processCreateOrUpdateListActivity`:** Calls `util.ListFromActivity` which may similarly upsert. Explicit dedup guard at the top.

### Pattern 4: Delete Authorization Fix (SAFE-02)

**What:** Add `actor *core.Record` param to `processDeleteTrailActivity` and check ownership. Exact pattern from `processDeleteCommentActivity` (delete.go lines 317-319) and `processDeleteSummitLogActivity` (lines 336-338).

**Critical finding:** `trail.GetString("author")` returns a **PocketBase record ID** (15-char alphanumeric), NOT an IRI. The `author` field in the `trails` collection is a `type: "relation"` to `activitypub_actors` (confirmed in migration snapshot at line 678). `actor.Id` is also the record ID. The correct comparison is:

```go
// Source: db/federation/delete.go:317 (processDeleteCommentActivity — verified pattern)
if trail.GetString("author") != actor.Id {
    return fmt.Errorf("actor is not trail author")
}
```

**CONTEXT.md D-07 wording correction:** D-07 says "compare `actor.GetString("iri")` against `trail.GetString("author")`" — this is WRONG. `trail.GetString("author")` returns the actor's **record ID**, not IRI. The correct comparison mirrors the comment and summit_log delete functions: `trail.GetString("author") != actor.Id`. The planner must implement the record-ID comparison, not an IRI comparison.

**Updated call site in `ProcessDeleteActivity` (delete.go line 277):**

```go
// Current:
case strings.Contains(object, "trail"):
    err = processDeleteTrailActivity(app, activity)
// New:
case strings.Contains(object, "trail"):
    err = processDeleteTrailActivity(app, actor, activity)
```

### Pattern 5: InstanceInboxHandler Extension (D-10)

**What:** Add Create/Update/Delete cases to the existing switch block. The `recipient` variable is already available in `InstanceInboxHandler` (line 138: `recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "inbox", inbox)`) — this is the instance actor record.

**Current switch block (instance.go lines 172-187):**

```go
switch activity.Type {
case pub.FollowType:
    if err := ProcessFollowActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Follow activity", err)
    }
case pub.AcceptType:
    if err := ProcessAcceptActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Accept activity", err)
    }
case pub.UndoType:
    if err := ProcessUndoActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Undo activity", err)
    }
default:
    return e.BadRequestError("Unsupported activity type", nil)
}
```

**Extension (insert before `default:`):**

```go
case pub.CreateType:
    fallthrough
case pub.UpdateType:
    if err := ProcessCreateOrUpdateActivity(e.App, actor, recipient, activity); err != nil {
        return e.BadRequestError("Failed to process Create/Update activity", err)
    }
case pub.DeleteType:
    if err := ProcessDeleteActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Delete activity", err)
    }
```

**`recipient` param:** `ProcessCreateOrUpdateActivity` expects `recipient *core.Record` which is used by `processCreateOrUpdateTrailActivity` and `processCreateOrUpdateListActivity` to call `util.InsertIntoFeed(app, recipient.Id, ...)`. In the user-level inbox, `recipient` is the local user actor. In the instance inbox, `recipient` is the local instance actor. The instance actor is a valid `*core.Record` with an `.Id` field — `InsertIntoFeed` will use it. This is the correct behavior: instance-federated content appears in the instance actor's feed, which can be consumed by the discovery/listing layer.

### Anti-Patterns to Avoid

- **IRI comparison for ownership:** Do NOT compare `actor.GetString("iri")` against `trail.GetString("author")` — the `author` field stores a record ID, not an IRI. This would always return "unauthorized" for remote actors.
- **Dedup at inbox handler level:** D-04 explicitly places dedup inside `processCreateOrUpdate*` functions, not in the switch statement. This preserves per-content-type flexibility.
- **Erroring on sql.ErrNoRows in instanceFollowerInboxes:** D-02 requires `(nil, nil)` — an error return would propagate and break outgoing fanout if the instance actor hasn't been seeded yet.
- **Adding a `ctx` parameter to `CreateListActivity`:** `CreateListActivity` currently has no `ctx` parameter (unlike the other three Create functions). Do NOT add one — the function uses no mention resolution that requires context.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Recipient deduplication | Custom dedup loop over recipients slice | `PostActivity` already calls `slices.Sort` + `slices.Compact` | activity.go lines 101-102 |
| HTTP signing | Custom RSA signing | `httpsig.NewSigner` + `signer.SignRequest` | Already wired in `PostActivity` |
| Inbox URL lookup | Custom JOIN query | `followerInboxes(app, actorId)` | Existing helper in activity.go line 31 |
| Activity dispatch | Custom type routing | `ProcessCreateOrUpdateActivity`, `ProcessDeleteActivity` | Already implemented in create.go:412, delete.go:266 |

---

## Exact Function Signatures (Verified from Source)

### Create*Activity Functions (SYNC-01/02/03 outgoing)

All 4 reside in `db/federation/create.go`. Each handles both Create and Update via the `typ` parameter.

| Function | Signature | Line | Public Gate? |
|----------|-----------|------|-------------|
| `CreateTrailActivity` | `(app core.App, ctx context.Context, trail *core.Record, typ pub.ActivityVocabularyType) error` | 21 | YES — `trail.GetBool("public")` line 22 |
| `CreateCommentActivity` | `(app core.App, ctx context.Context, comment *core.Record, typ pub.ActivityVocabularyType) error` | 100 | NO — **must be added (D-08/SAFE-03)** |
| `CreateSummitLogActivity` | `(app core.App, ctx context.Context, summitLog *core.Record, typ pub.ActivityVocabularyType) error` | 185 | YES — parent trail `GetBool("public")` line 203 |
| `CreateListActivity` | `(app core.App, list *core.Record, typ pub.ActivityVocabularyType) error` | 349 | YES — `list.GetBool("public")` line 350 |

**Note:** `CreateListActivity` takes no `ctx context.Context` parameter — it does no mention resolution.

### Delete*Activity Functions (SYNC-03 outgoing)

All 4 reside in `db/federation/delete.go`.

| Function | Signature | Line | Gets instanceFollowerInboxes? |
|----------|-----------|------|-----------------------------|
| `CreateTrailDeleteActivity` | `(app core.App, r *core.Record) error` | 16 | Must add |
| `CreateCommentDeleteActivity` | `(app core.App, client meilisearch.ServiceManager, r *core.Record) error` | 76 | Must add |
| `CreateSummitLogDeleteActivity` | `(app core.App, r *core.Record) error` | 138 | Must add |
| `CreateListDeleteActivity` | `(app core.App, r *core.Record) error` | 209 | Must add |

**Note:** `CreateCommentDeleteActivity` currently only sends to the trail author's inbox (`PostActivity(app, author, activity, []string{to + "/inbox"})`). For instance fanout, append `instanceFollowerInboxes` to this recipients slice.

### ProcessDeleteActivity and processDeleteTrailActivity

**Current:**
```go
// delete.go:266
func ProcessDeleteActivity(app core.App, actor *core.Record, activity pub.Activity) error

// delete.go:293 — called from ProcessDeleteActivity:277
func processDeleteTrailActivity(app core.App, activity pub.Activity) error
```

**After D-06:**
```go
// delete.go:293 — NEW signature
func processDeleteTrailActivity(app core.App, actor *core.Record, activity pub.Activity) error
```

Call site update at delete.go:277:
```go
// Before:
err = processDeleteTrailActivity(app, activity)
// After:
err = processDeleteTrailActivity(app, actor, activity)
```

### ProcessCreateOrUpdateActivity (SYNC-01/02/03 receiving side)

```go
// create.go:412 — unchanged signature
func ProcessCreateOrUpdateActivity(app core.App, actor *core.Record, recipient *core.Record, activity pub.Activity) error
```

`recipient` is used in two sub-functions:
- `processCreateOrUpdateTrailActivity` (line 439): `util.InsertIntoFeed(app, recipient.Id, actor.Id, trail.Id, util.TrailFeed)`
- `processCreateOrUpdateListActivity` (line 759): `util.InsertIntoFeed(app, recipient.Id, actor.Id, list.Id, util.ListFeed)`

When called from `InstanceInboxHandler`, `recipient` will be the local instance actor record (already available in `InstanceInboxHandler` as variable `recipient` at line 138).

---

## Common Pitfalls

### Pitfall 1: SAFE-02 Author Field Stores Record ID, Not IRI

**What goes wrong:** Implementing `processDeleteTrailActivity` ownership check as `actor.GetString("iri") != trail.GetString("author")` — this always returns "unauthorized" for every remote actor because `trail.author` is a record ID.

**Why it happens:** CONTEXT.md D-07 describes the check in IRI terms. The actual DB schema stores `author` as a `relation` field containing a 15-char record ID (`pbc_1295301207` collection).

**How to avoid:** Use `trail.GetString("author") != actor.Id` — identical to `processDeleteCommentActivity` (line 317) and `processDeleteSummitLogActivity` (line 336).

**Warning signs:** Authorization check always fails in integration testing regardless of actor.

### Pitfall 2: CreateCommentActivity Has No Public Gate (SAFE-03)

**What goes wrong:** Assuming D-09 ("existing gates already in place") covers comments. It does not. `CreateCommentActivity` has no `public` check at lines 100-183.

**Why it happens:** The function does not check `trail.GetBool("public")` before building and sending the activity.

**How to avoid:** Add at the top of `CreateCommentActivity` (after fetching `commentTrail`):
```go
if !commentTrail.GetBool("public") {
    return nil
}
```
This is the same gate pattern as `CreateSummitLogActivity` lines 203-206.

### Pitfall 3: CreateCommentDeleteActivity Uses a Fixed Recipient Slice

**What goes wrong:** `CreateCommentDeleteActivity` calls `PostActivity(app, author, activity, []string{to + "/inbox"})` with a literal one-element slice. Appending `instanceFollowerInboxes` to a `[]string{to + "/inbox"}` literal requires constructing the slice as a variable first.

**How to avoid:** Refactor to:
```go
recipients := []string{to + "/inbox"}
instanceInboxes, err := instanceFollowerInboxes(app)
if err != nil { return err }
recipients = append(recipients, instanceInboxes...)
return PostActivity(app, author, activity, recipients)
```

### Pitfall 4: SAFE-01 Dedup for Trails — TrailFromActivity Already Has Upsert Logic

**What goes wrong:** Assuming that because `processCreateOrUpdateTrailActivity` calls `util.TrailFromActivity`, which internally checks for existing records, no explicit dedup guard is needed. In fact `TrailFromActivity` on an existing record sets `needs_full_sync = true` and saves the record — it does NOT return nil.

**How to avoid:** Add an explicit dedup check at the top of `processCreateOrUpdateTrailActivity` before the `util.TrailFromActivity` call, keyed on `activity.Object.GetID().String()` against the `trails.iri` field:
```go
objectIRI := activity.Object.GetID().String()
existing, err := app.FindFirstRecordByData("trails", "iri", objectIRI)
if err == nil && existing != nil {
    return nil  // already have this trail — silent dedup (D-04)
}
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    return err
}
```

### Pitfall 5: Update Activities Must Pass Through Dedup (D-04 Scope)

**What goes wrong:** Applying the dedup guard to `pub.UpdateType` as well as `pub.CreateType`, blocking legitimate updates.

**How to avoid:** D-04 says "check whether a content record with the incoming object IRI already exists — if found: return nil silently." This means both Create AND Update from the same IRI are deduplicated. This is intentional for the broadcast-loop case (where the same activity is delivered twice), but an Update from a new content version will have the same IRI and would be dropped. The CONTEXT.md explicitly accepts this trade-off ("idempotent delivery — remote gets 200, no retry triggered"). Implement as specified — deduplicate regardless of activity type.

### Pitfall 6: ProcessDeleteActivity Already Guards for is_local (No Change Needed)

**What goes wrong:** Adding a redundant `is_local` check inside `processDeleteTrailActivity`.

**Why:** `ProcessDeleteActivity` already returns early on line 267-270:
```go
if actor.GetBool("is_local") {
    return nil
}
```
The sub-functions never execute for local actors. Do not add a duplicate guard.

---

## Code Examples

### Full fanout injection for CreateTrailActivity (representative example)

```go
// Source: db/federation/create.go — extends existing lines 91-97
inboxes, err := followerInboxes(app, trailAuthor.Id)
if err != nil {
    return err
}
recipients := append(mentions, inboxes...)

instanceInboxes, err := instanceFollowerInboxes(app)
if err != nil {
    return err
}
recipients = append(recipients, instanceInboxes...)

return PostActivity(app, trailAuthor, activity, recipients)
```

### processDeleteTrailActivity — after D-06/D-07

```go
// Source: db/federation/delete.go — replaces lines 293-307
func processDeleteTrailActivity(app core.App, actor *core.Record, activity pub.Activity) error {
    object := activity.Object.GetID().String()
    trail, err := app.FindFirstRecordByData("trails", "iri", object)
    if err != nil {
        return err
    }

    if trail.GetString("author") != actor.Id {
        return fmt.Errorf("actor is not trail author")
    }

    err = util.DeleteFromFeed(app, trail.Id)
    if err != nil {
        return err
    }

    return app.Delete(trail)
}
```

### InstanceInboxHandler extension (D-10)

```go
// Source: db/federation/instance.go — extends switch block at line 172
case pub.CreateType:
    fallthrough
case pub.UpdateType:
    if err := ProcessCreateOrUpdateActivity(e.App, actor, recipient, activity); err != nil {
        return e.BadRequestError("Failed to process Create/Update activity", err)
    }
case pub.DeleteType:
    if err := ProcessDeleteActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Delete activity", err)
    }
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `processDeleteTrailActivity(app, activity)` — no actor param | `processDeleteTrailActivity(app, actor, activity)` — actor param added | Phase 3 (D-06) | Enables SAFE-02 ownership check |
| No instance follower inboxes in fanout | `instanceFollowerInboxes` appended to all 8 outgoing functions | Phase 3 (D-01/D-03) | Enables SYNC-01/02/03 |
| Comments federated without parent trail public gate | Gate added at top of `CreateCommentActivity` | Phase 3 (D-08) | Enforces SAFE-03 for comments |

**Existing (no change needed):**
- `PostActivity`: goroutine-based fire-and-forget, HTTP signature, semaphore concurrency (max 5), recipient dedup — unchanged
- `CreateTrailActivity` public gate: line 22 — already present
- `CreateSummitLogActivity` public gate: lines 203-206 — already present
- `CreateListActivity` public gate: lines 350-352 — already present
- `processDeleteCommentActivity` actor ownership check: line 317 — already present, is the model for SAFE-02
- `processDeleteSummitLogActivity` actor ownership check: line 336 — already present

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `InsertIntoFeed` with instance actor as recipient is a valid, harmless call when processing content from remote instances via the instance inbox | D-10: InstanceInboxHandler extension | Feed entries would be attributed to the instance actor, which may or may not be desirable for UI display — low risk since instance-federated content has no direct user UI in v1 |

All other claims in this research were verified by direct codebase inspection. No user confirmation is needed.

---

## Open Questions

1. **SAFE-01 dedup scope for Update activities**
   - What we know: D-04 says "if found: return nil silently" for all activity types.
   - What's unclear: Should an incoming Update for already-known content be deduplicated (dropped) or allowed through? The existing upsert paths in `processCreateOrUpdateCommentActivity` and `processCreateOrUpdateSummitLogActivity` already check for existing records and upsert. A dedup guard returning nil would prevent updates from remote peers from applying.
   - Recommendation: Apply the dedup guard only on `pub.CreateType`. For `pub.UpdateType`, allow the existing upsert logic to run. The broadcast-loop scenario produces duplicate Creates, not duplicate Updates. Raise with user if D-04's "both types" interpretation is confirmed.

2. **`CreateCommentDeleteActivity` and instance fanout for comment deletes when trail author is local**
   - What we know: `CreateCommentDeleteActivity` line 102-104 returns `nil` if the trail author is local (`commentTrailAuthor.GetBool("is_local")`). This guard would prevent instance fanout for comments on locally-authored trails.
   - What's unclear: Should instance followers receive Delete{Comment} activities when a local user deletes a comment on a locally-authored trail?
   - Recommendation: The existing early return at line 102 needs to be restructured: send the Delete to instance followers regardless of whether the trail author is local, but still gate sending to the trail author only when the trail author is remote. This requires splitting the "send to trail author" and "send to instance followers" paths.

---

## Environment Availability

Step 2.6: SKIPPED — this phase makes no changes that require external tools, services, or runtimes beyond the existing Go/PocketBase backend.

---

## Validation Architecture

Nyquist validation is disabled (`workflow.nyquist_validation: false` in `.planning/config.json`). This section is skipped.

---

## Security Domain

`security_enforcement: true` in config. ASVS level 1 applies.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | HTTP signature verification already in `InstanceInboxHandler` (unchanged) |
| V3 Session Management | No | Stateless ActivityPub delivery |
| V4 Access Control | Yes — SAFE-02 | `trail.GetString("author") != actor.Id` ownership check |
| V5 Input Validation | Partial | Object IRI checked for known content types (`strings.Contains`) — existing pattern |
| V6 Cryptography | No | RSA HTTP signing already in `PostActivity` (unchanged) |

### Known Threat Patterns for ActivityPub Federation

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Spoofed Delete from non-author | Tampering | SAFE-02: ownership check in `processDeleteTrailActivity` |
| Broadcast amplification loop | Denial of Service | SAFE-01: dedup guard in `processCreateOrUpdate*` |
| Private content exfiltration via fanout | Information Disclosure | SAFE-03: `is_public` check gates all outgoing activities |
| Unauthorized content injection | Tampering | HTTP signature verification in `InstanceInboxHandler` (Phase 2, unchanged) |

---

## Sources

### Primary (HIGH confidence)

All findings were derived from direct source code inspection. No external documentation lookups were required as this is an extension of existing in-repo patterns.

- `db/federation/activity.go` — `followerInboxes`, `PostActivity` signatures and behavior
- `db/federation/create.go` — all 4 `Create*Activity` signatures, all 4 `processCreateOrUpdate*` implementations, `ProcessCreateOrUpdateActivity` dispatcher
- `db/federation/delete.go` — all 4 `CreateXxxDeleteActivity` signatures, `ProcessDeleteActivity`, `processDeleteTrailActivity`, `processDeleteCommentActivity`, `processDeleteSummitLogActivity`
- `db/federation/instance.go` — `InstanceInboxHandler` full implementation including `recipient` variable
- `db/federation/follow.go` — `ProcessFollowActivity` idempotency guard (reference pattern for D-04)
- `db/hooks/trails.go`, `db/hooks/comments.go`, `db/hooks/summit_logs.go`, `db/hooks/list.go` — confirmed hook signatures and which federation functions they call
- `db/migrations/1747064968_collections_snapshot.go` line 678 — `author` field in `trails` is `type: "relation"` to `activitypub_actors`
- `db/util/activitypub.go` line 254 — `record.Set("author", actor.Id)` confirms author stores record ID
- `db/util/trail_access.go` line 23 — `trail.GetString("author") == actor.Id` confirms field stores record ID
- `db/routes/activitypub.go` lines 82-122 — user-level inbox handler; how `recipient` and `actor` are constructed for `ProcessCreateOrUpdateActivity`

### Test Infrastructure

Existing test files in `db/federation/` that the planner should extend:

- `db/federation/instance_inbox_test.go` — test helpers `newInboxTestApp`, `createTestActor` are reusable; contains the activitypub_actors + follows + activitypub_activities collection setup
- `db/federation/instance_test.go` — `newTestApp` helper (actors collection only)
- `db/federation/follow_accept_test.go` — test structure example

**No test files exist for `create.go` or `delete.go`.** The planner should create:
- `db/federation/create_test.go` — tests for dedup behavior (SAFE-01) and privacy gate on comments (SAFE-03)
- `db/federation/delete_test.go` — tests for SAFE-02 unauthorized delete rejection

---

## Metadata

**Confidence breakdown:**
- Function signatures: HIGH — read directly from source files
- Author field type (record ID vs IRI): HIGH — verified against schema migration and `trail_access.go` usage
- SAFE-02 comparison operator (actor.Id vs actor IRI): HIGH — verified against `processDeleteCommentActivity` pattern
- `recipient` variable availability in InstanceInboxHandler: HIGH — read from instance.go line 138
- Dedup approach for `TrailFromActivity`: HIGH — read `TrailFromActivity` source to confirm it does not return nil on duplicate
- Comment privacy gate missing: HIGH — confirmed by absence of any public check in `CreateCommentActivity` source

**Research date:** 2026-06-25
**Valid until:** 2026-07-25 (codebase is stable; federation package not under active parallel development)
