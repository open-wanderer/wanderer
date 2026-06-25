---
phase: 02-follow-lifecycle
reviewed: 2026-06-25T00:00:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - db/migrations/1782290001_add_rejected_to_follows_status.go
  - db/federation/instance.go
  - db/federation/follow.go
  - db/federation/instance_inbox_test.go
  - db/main.go
  - web/src/routes/api/v1/activitypub/instance/inbox/+server.ts
  - db/hooks/follow.go
  - db/hooks/follow_test.go
findings:
  critical: 5
  warning: 4
  info: 2
  total: 11
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-06-25T00:00:00Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Phase 02 adds the follow lifecycle for instance-level actors: a migration for the `rejected`
status, an `InstanceInboxHandler` that verifies HTTP signatures and dispatches Follow/Accept/Undo,
`ProcessFollowActivity` branching for instance actors (pending, no auto-accept), and three
AfterSuccess hooks for admin-driven accept/reject/undo delivery.

The core branching logic and signature verification path are structurally sound. However the
hook registration introduces two independent double-send bugs (duplicate Follow activity on
admin-initiated follows; duplicate Undo on deletions), the inbound-follow save spuriously
triggers `InstanceFollowCreateHandler` in the wrong direction, `ProcessAcceptActivity` will
panic on a malformed Accept, and the inbox handler masks all processing errors behind HTTP 200.
All five of these are blockers before the phase ships.

---

## Critical Issues

### CR-01: Double Follow activity sent when admin creates an instance follow via API

**File:** `db/hooks/follow.go:12-19` and `db/main.go:127,130`

**Issue:** Two hooks are registered for the `follows` collection on creation:

1. `OnRecordCreateRequest` → `CreateFollowHandler` — calls `federation.CreateFollowActivity`
   unconditionally after `e.Next()` (no instance-follow guard).
2. `OnRecordAfterCreateSuccess` → `InstanceFollowCreateHandler` — also calls
   `federation.CreateFollowActivity` for instance follows.

When an admin creates an instance follow record via the PocketBase API, both hooks fire in
sequence, causing two `Follow` activities to be delivered to the remote inbox. The remote
instance will create a duplicate pending entry.

`CreateFollowHandler` was written for user-level follows and never had reason to guard against
instance follows; it predates this phase. The new `InstanceFollowCreateHandler` was added
without disabling `CreateFollowHandler` for instance follows.

**Fix:** Add an `isInstanceFollow` guard (or an equivalent `is_local` follower check) at the
top of `CreateFollowHandler` to skip instance follows and delegate entirely to
`InstanceFollowCreateHandler`:

```go
func CreateFollowHandler() func(e *core.RecordRequestEvent) error {
    return func(e *core.RecordRequestEvent) error {
        if err := e.Next(); err != nil {
            return err
        }
        if isInstanceFollow(e.App, e.Record) {
            return nil // InstanceFollowCreateHandler (AfterCreateSuccess) handles this
        }
        federation.CreateFollowActivity(e.App, e.Record)
        return nil
    }
}
```

---

### CR-02: Double Undo activity sent when admin deletes an instance follow via API

**File:** `db/hooks/follow.go:21-27` and `db/main.go:128,132`

**Issue:** Mirrors CR-01 for deletions. Two hooks are registered:

1. `OnRecordDeleteRequest` → `DeleteFollowHandler` — calls `federation.CreateUnfollowActivity`
   unconditionally (no instance-follow guard).
2. `OnRecordAfterDeleteSuccess` → `InstanceFollowDeleteHandler` — also calls
   `federation.CreateUnfollowActivity` for instance follows.

Both fire when the admin deletes an instance follow via the API, producing two `Undo{Follow}`
activities to the remote inbox.

**Fix:** Add an `isInstanceFollow` guard in `DeleteFollowHandler` to skip instance follows:

```go
func DeleteFollowHandler() func(e *core.RecordRequestEvent) error {
    return func(e *core.RecordRequestEvent) error {
        if isInstanceFollow(e.App, e.Record) {
            return e.Next() // InstanceFollowDeleteHandler (AfterDeleteSuccess) handles this
        }
        federation.CreateUnfollowActivity(e.App, e.Record)
        return e.Next()
    }
}
```

---

### CR-03: InstanceFollowCreateHandler fires on inbound follows saved by ProcessFollowActivity

**File:** `db/hooks/follow.go:68-103` and `db/federation/follow.go:84-113`

**Issue:** `OnRecordAfterCreateSuccess` fires for every successful record save, including
programmatic `app.Save()` calls made inside request handlers. When `ProcessFollowActivity`
stores an inbound instance follow (remote → local instance actor), it calls `app.Save(followRecord)`.
This triggers `InstanceFollowCreateHandler`.

Inside that handler, `isInstanceFollow` checks both the follower and the followee. The followee
is the local instance actor, so `isInstanceFollow` returns `true`. The handler then calls
`federation.CreateFollowActivity(e.App, e.Record)` where `e.Record` has `follower = remote_actor`.
`CreateFollowActivity` calls `PostActivity(app, followerActor, ...)`, which attempts to decrypt
the remote actor's `private_key` — a field we do not hold. `security.Decrypt("", key)` fails,
logging a silent error. The net effect is a spurious fire-and-forget goroutine per inbound
instance follow, and an incorrect follow activity is attempted as if we were sending on behalf
of the remote actor.

**Fix:** `InstanceFollowCreateHandler` must only fire when the _follower_ is the local instance
actor (i.e., an admin-initiated outbound follow). Check `followerActor.GetBool("is_local")` or
compare the follower's IRI against `instanceIRI`:

```go
func InstanceFollowCreateHandler() func(e *core.RecordEvent) error {
    return func(e *core.RecordEvent) error {
        instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
        followerActor, err := e.App.FindRecordById("activitypub_actors", e.Record.GetString("follower"))
        if err != nil || followerActor.GetString("iri") != instanceIRI {
            return e.Next() // Not an outbound instance follow; skip
        }
        // ... existing outbound logic
    }
}
```

The same directional check should be applied to `InstanceFollowDeleteHandler` if the local
instance is not always the follower in deletion scenarios.

---

### CR-04: Unsafe type assertion in ProcessAcceptActivity panics on malformed Accept

**File:** `db/federation/follow.go:293`

**Issue:**

```go
followActivity := activity.Object.(*pub.Activity)
```

This is an unguarded type assertion. If the remote instance sends an `Accept` whose `object`
field is an IRI string or any type other than `*pub.Activity` (which is common in real-world
ActivityPub implementations that inline only the activity ID), the assertion panics, taking
down the request goroutine. The panic is not recovered inside `InstanceInboxHandler`, so it
propagates as an unhandled panic.

The same pattern exists in `processUnfollowActivity` (`undo.go:147`) and
`processUnlikeActivity` (`undo.go:171`), but those were pre-existing. `ProcessAcceptActivity`
is now newly reachable via the instance inbox route registered in this phase.

**Fix:** Use the comma-ok form and return a descriptive error:

```go
followActivity, ok := activity.Object.(*pub.Activity)
if !ok {
    return fmt.Errorf("ProcessAcceptActivity: object is not *pub.Activity, got %T", activity.Object)
}
```

---

### CR-05: InstanceInboxHandler returns HTTP 200 OK when processing fails (error masking)

**File:** `db/federation/instance.go:169-179`

**Issue:**

```go
var procErr error
switch activity.Type {
case pub.FollowType:
    procErr = ProcessFollowActivity(e.App, actor, activity)
case pub.AcceptType:
    procErr = ProcessAcceptActivity(e.App, actor, activity)
case pub.UndoType:
    procErr = ProcessUndoActivity(e.App, actor, activity)
}

return e.JSON(http.StatusOK, procErr)
```

When `procErr` is non-nil, `e.JSON(http.StatusOK, procErr)` serializes the Go `error` interface
to JSON. Standard Go errors do not implement `json.Marshaler`, so they serialize to `null`.
The remote caller receives `HTTP 200` with body `null` and has no way to detect the failure.

Additionally, when `activity.Type` is none of the three handled types, `procErr` stays `nil`
and the handler silently accepts the unknown activity with `200 null` — there is no `default`
case to reject unsupported types.

**Fix:** Return a proper error response when processing fails, and add a default 400 for
unrecognised activity types:

```go
switch activity.Type {
case pub.FollowType:
    if err := ProcessFollowActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Follow processing failed", err)
    }
case pub.AcceptType:
    if err := ProcessAcceptActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Accept processing failed", err)
    }
case pub.UndoType:
    if err := ProcessUndoActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Undo processing failed", err)
    }
default:
    return e.BadRequestError("Unsupported activity type", nil)
}
return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
```

---

## Warnings

### WR-01: instance_inbox_test.go follows schema is missing the "rejected" status value

**File:** `db/federation/instance_inbox_test.go:82`

**Issue:** The inline follows collection schema used in `newInboxTestApp` declares:

```json
"values": ["pending", "accepted"]
```

The migration in this phase adds `"rejected"` as a valid status. `follow_test.go` correctly
includes `"rejected"`. If any future test in `instance_inbox_test.go` exercises a code path
that sets `status = "rejected"` (e.g., after Plan 03 integrates reject delivery), PocketBase
will reject the save with a validation error, causing the test to fail with a confusing
message.

**Fix:** Add `"rejected"` to the values array in `instance_inbox_test.go:82`:

```json
"values": ["pending", "accepted", "rejected"]
```

---

### WR-02: `isInstanceFollow` silently treats ORIGIN="" as a valid IRI prefix

**File:** `db/hooks/follow.go:34`

**Issue:**

```go
instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
```

If `ORIGIN` is unset (empty string), `instanceIRI` becomes `"/api/v1/activitypub/instance"`.
Any actor whose IRI happens to be that relative path would match, though this is unlikely in
practice. More dangerously, the function silently succeeds with a wrong value rather than
returning an error. This means all three instance hooks will mis-fire or not fire at all for
the wrong record when the server is misconfigured, producing a confusing failure mode.

**Fix:** Return `false` immediately when `ORIGIN` is empty, mirroring `InitInstanceActor` and
`InstanceInboxHandler`:

```go
func isInstanceFollow(app core.App, follow *core.Record) bool {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return false
    }
    instanceIRI := origin + "/api/v1/activitypub/instance"
    // ...
}
```

---

### WR-03: `if (success === false)` in SvelteKit proxy is dead code — 401 is thrown, not returned false

**File:** `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts:53-55`

**Issue:**

```typescript
const success = await event.locals.pb.send("/activitypub/instance/inbox", { ... });

if (success === false) {
    return json("Invalid header signature", { status: 400 });
}
```

PocketBase's SDK `send()` throws a `ClientResponseError` on non-2xx responses; it never
returns `false`. When Go's `InstanceInboxHandler` returns 401 (invalid signature), the SDK
throws, control jumps to `catch (e)`, and `handleError(e)` is called. The `success === false`
branch is never reachable. The error response message `"Invalid header signature"` is therefore
also unreachable.

This was copied from the existing user inbox handler (`user/[handle]/inbox/+server.ts`) where
the same dead check exists. Both should be removed for clarity.

**Fix:** Remove the dead check; rely on the `catch` block's `handleError` for all non-2xx
responses:

```typescript
await event.locals.pb.send("/activitypub/instance/inbox", {
    method: "POST",
    fetch: event.fetch,
    headers: originalHeaders,
    body: JSON.stringify(activity)
});

const headers = new Headers();
headers.append("Content-Type", "application/activity+json");
return json("", { status: 200, headers });
```

---

### WR-04: `CreateFollowHandler` and `DeleteFollowHandler` silently discard federation errors

**File:** `db/hooks/follow.go:15,23`

**Issue:**

```go
// CreateFollowHandler:
federation.CreateFollowActivity(e.App, e.Record)  // return value ignored

// DeleteFollowHandler:
federation.CreateUnfollowActivity(e.App, e.Record)  // return value ignored
```

Both functions return `error` but the callers discard it. A federation delivery failure
produces no log entry and no propagated error. This makes diagnosing federation problems
difficult in production.

**Fix:** Log errors consistently with the pattern used by the new instance handlers:

```go
if err := federation.CreateFollowActivity(e.App, e.Record); err != nil {
    e.App.Logger().Error(fmt.Sprintf("follow create: CreateFollowActivity failed: %v", err))
}
```

---

## Info

### IN-01: Identical follow-activity reload logic duplicated in CreateAcceptFollowActivity and CreateRejectFollowActivity

**File:** `db/federation/follow.go:187-201` and `db/federation/follow.go:249-263`

**Issue:** The code to reload the original incoming Follow activity from `activitypub_activities`
(filter by actor/object/type, reconstruct `pub.Activity`) is copy-pasted verbatim in both
functions. Any bug fix or schema change must be applied in two places.

**Fix:** Extract to a private helper, e.g.:

```go
func loadPendingFollowActivity(app core.App, followerIRI, followeeIRI string) (*pub.Activity, error) {
    rec, err := app.FindFirstRecordByFilter("activitypub_activities",
        "actor={:actor}&&object={:object}&&type={:type}",
        dbx.Params{"actor": followerIRI, "object": followeeIRI, "type": string(pub.FollowType)},
    )
    if err != nil {
        return nil, err
    }
    a := pub.FollowNew(pub.IRI(rec.GetString("iri")), pub.IRI(followeeIRI))
    a.Actor = pub.IRI(rec.GetString("actor"))
    return a, nil
}
```

---

### IN-02: Commented-out SyncOutbox call is dead code

**File:** `db/federation/follow.go:310-314`

**Issue:**

```go
// err = util.SyncOutbox(app, actor)
// if err != nil {
//     return err
// }
```

This block has been commented out and should either be removed or tracked as a known
incomplete feature in a TODO comment with a ticket reference.

**Fix:** Remove the commented-out block, or replace with:

```go
// TODO: sync outbox after remote accept (tracked in issue #XXX)
```

---

_Reviewed: 2026-06-25T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
