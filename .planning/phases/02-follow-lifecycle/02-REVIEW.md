---
phase: 02-follow-lifecycle
reviewed: 2026-06-25T15:43:54Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - db/federation/follow.go
  - db/federation/instance.go
  - db/federation/follow_accept_test.go
  - db/federation/instance_inbox_test.go
  - db/hooks/follow.go
  - db/hooks/follow_test.go
  - db/main.go
  - db/migrations/1782290001_add_rejected_to_follows_status.go
  - web/src/routes/api/v1/activitypub/instance/inbox/+server.ts
findings:
  critical: 3
  warning: 4
  info: 2
  total: 9
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-06-25T15:43:54Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

This phase implements the follow lifecycle for instance-level ActivityPub federation: pending-state inbound follows, admin-driven Accept/Reject delivery, outbound Follow creation, and the HTTP inbox handler. The overall architecture is sound and the happy paths are tested. However three blockers were found that can cause silent data corruption or incorrect federation behavior in production: a discarded error in `CreateFollowHandler` that masks DB failures while still firing federation delivery, a non-idempotent replay path in `ProcessFollowActivity` that will crash on duplicate Follow requests, and an `Accept` delivery order inversion in `CreateAcceptFollowActivity` that persists the DB record before successfully delivering the HTTP message, leaving stale `accepted` rows on network failure.

---

## Critical Issues

### CR-01: `CreateFollowHandler` discards `e.Next()` error, firing federation delivery after a failed DB save

**File:** `db/hooks/follow.go:19`
**Issue:** In the non-instance branch of `CreateFollowHandler`, `e.Next()` is called without capturing its return value. If any downstream hook (or the PocketBase core record-save itself) returns an error, that error is silently dropped and the function returns `nil`. This means `CreateFollowActivity` is called and the outgoing Follow activity is dispatched to the remote inbox even though the local `follows` record was never committed. The remote instance receives a Follow but the local DB has no record of it — the follow relationship is permanently asymmetric and cannot be accepted, rejected, or undone via any admin action.

The contrast with `DeleteFollowHandler` (line 35: `return e.Next()`) shows the correct pattern is already used one function away.

**Fix:**
```go
func CreateFollowHandler() func(e *core.RecordRequestEvent) error {
    return func(e *core.RecordRequestEvent) error {
        if isInstanceFollow(e.App, e.Record) {
            return e.Next()
        }
        // Capture the error from e.Next() so we do not dispatch federation
        // delivery when the underlying DB save failed.
        if err := e.Next(); err != nil {
            return err
        }
        federation.CreateFollowActivity(e.App, e.Record)
        return nil
    }
}
```

---

### CR-02: `ProcessFollowActivity` has no idempotency guard — replayed Follow from same remote actor crashes with DB unique-constraint error

**File:** `db/federation/follow.go:79-113`
**Issue:** When a remote instance sends a Follow activity directed at the local instance actor, `ProcessFollowActivity` unconditionally creates a new `follows` record (lines 79-113). The `follows` table has a unique composite index on `(follower, followee)` (migration `1748002661_updated_follows.go`). If the same remote instance replays the Follow (network retry, duplicate delivery — both are normal in ActivityPub), `app.Save(followRecord)` on line 93 will return a unique-constraint violation. The error propagates up to `InstanceInboxHandler`, which returns a 400 to the remote caller. The remote caller is likely to retry indefinitely.

The same path also creates a second `activitypub_activities` record for the Follow (lines 98-110), which then causes `CreateAcceptFollowActivity` / `CreateRejectFollowActivity` to use `FindFirstRecordByFilter` — a non-deterministic match when duplicates exist.

**Fix:** Check for an existing follow record before inserting, and return early (with optional 200) if one already exists:
```go
// Idempotency: if a follow record already exists for this pair, skip creation.
existing, err := app.FindFirstRecordByFilter(
    "follows",
    "follower={:follower} && followee={:followee}",
    dbx.Params{"follower": actor.Id, "followee": object.Id},
)
if err == nil && existing != nil {
    return nil // already recorded; treat as duplicate delivery
}
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    return err
}
// proceed with new record creation...
```

---

### CR-03: `CreateAcceptFollowActivity` persists the Accept DB record before delivering it — leaves stale `accepted` status on network failure

**File:** `db/federation/follow.go:214-226`
**Issue:** `CreateAcceptFollowActivity` saves the Accept activity record to the database at line 222 (`app.Save(record)`), and only then calls `PostActivity` at line 226. `PostActivity` runs delivery asynchronously in a goroutine and its errors are silently swallowed (the function always returns `nil` from `CreateAcceptFollowActivity`'s perspective). However, `InstanceFollowUpdateHandler` calls this function after the follow record has already been set to `status=accepted` by the PocketBase update path. If the HTTP delivery fails, the local DB shows `accepted` and an Accept activity record exists, but the remote instance never received the Accept and still considers the follow `pending`. Subsequent administrative actions (re-reject, re-accept) will not fix this because `instanceFollowAction` returns `""` for same-status transitions.

The same structural issue exists in `CreateRejectFollowActivity` (lines 284-288).

**Fix:** Move the DB record creation to after a successful delivery signal, or — since `PostActivity` is fire-and-forget — at minimum log the outstanding delivery attempt using a pending-delivery queue or explicit retry mechanism. A lighter immediate fix is to document the at-most-once guarantee and add a compensating re-send path accessible to the admin. The minimum required change is to not treat a missing DB record as evidence of non-delivery when re-accepting a follow that previously failed delivery:

```go
// After PostActivity returns (goroutine is already launched at this point),
// wrap the DB save in the error path so the record is only stored when
// the goroutine is at least enqueued (PostActivity never returns a delivery error today,
// but the save should still be last to match the intent):
if err = PostActivity(app, followeeActor, acceptActivity, []string{followerActor.GetString("inbox")}); err != nil {
    return err
}
// Only persist after delivery is enqueued.
return app.Save(record)
```

---

## Warnings

### WR-01: `InstanceInboxHandler` compares `sql.ErrNoRows` with `==` instead of `errors.Is`

**File:** `db/federation/instance.go:146`
**Issue:** The actor cache-miss check on line 146 uses `err == sql.ErrNoRows`. PocketBase wraps its not-found errors: the underlying SQLite driver or PocketBase itself may wrap `sql.ErrNoRows` inside another error value. Using direct equality comparison instead of `errors.Is(err, sql.ErrNoRows)` means the cache-miss branch is silently skipped for wrapped errors, causing the handler to return the wrapped DB error as a 500 instead of attempting the remote actor fetch.

The `InitInstanceActor` function in the same file (line 60) correctly uses `errors.Is(err, sql.ErrNoRows)`, making the inconsistency within the same file.

**Fix:**
```go
if errors.Is(err, sql.ErrNoRows) {
    // remote fetch ...
}
```

---

### WR-02: `ProcessFollowActivity` sends a notification to a person actor even for instance-actor follows

**File:** `db/federation/follow.go:152-162`
**Issue:** After the instance-actor early-return guard at lines 91-113, the code falls through to the notification dispatch at lines 152-162. For the person-actor path, `object` is the local person actor record and the notification is sent to that person — correct. However, if the instance-actor branch is somehow not taken (e.g., the `actor_type` field is absent or empty for the local instance actor, which can happen if `InitInstanceActor` was never called or the collection schema is inconsistent), the code continues past the guard and sends a notification to the instance actor record. `util.SendNotification` expects a user-linked actor; calling it with an instance actor record may panic or produce a silent error.

More importantly, the notification (lines 153-161) includes `actor.GetString("preferred_username")` and `actor.GetString("domain")` from the remote actor. For remote instance actors, `preferred_username` is always `"instance"` (set by the remote `InitInstanceActor`), so the notification text would display `@instance@remotedomain` as a follower — a confusing UX for person-actor follows from instances.

**Fix:** Guard the notification block so it only fires for the person-actor path:
```go
// Only send follow notifications for user-level follows (not instance actors).
if object.GetString("actor_type") != "instance" {
    notification := util.Notification{ ... }
    return util.SendNotification(app, notification, object)
}
return nil
```

---

### WR-03: `InstanceFollowUpdateHandler` fires for both directions of an instance follow — can incorrectly send Accept/Reject for an outbound follow that the local instance initiated

**File:** `db/hooks/follow.go:138`
**Issue:** `InstanceFollowUpdateHandler` calls `isInstanceFollow` (which returns true when *either* follower or followee is the local instance actor) and then unconditionally calls `CreateAcceptFollowActivity` or `CreateRejectFollowActivity`. `CreateAcceptFollowActivity` / `CreateRejectFollowActivity` look up the persisted Follow activity by `actor={followerActor.iri} && object={followeeActor.iri}`. For an *outbound* follow (local instance is the follower), this query looks for a Follow activity where `actor` is the local instance IRI — but the persisted activity in that case is the outgoing Follow *from* the local instance, not an incoming Follow *to* the local instance. Depending on the activity records present, the filter may match the wrong record or return no rows, causing a misleading error from `FindFirstRecordByFilter`.

The correct guard for Accept/Reject delivery is `isOutboundInstanceFollow` returning `false` (i.e., the local instance is the *followee*) before allowing the Accept/Reject delivery.

**Fix:**
```go
func InstanceFollowUpdateHandler() func(e *core.RecordEvent) error {
    return func(e *core.RecordEvent) error {
        // Accept/Reject only makes sense when the local instance is the FOLLOWEE
        // (i.e., a remote instance's inbound Follow is being approved/denied).
        if !isInstanceFollow(e.App, e.Record) || isOutboundInstanceFollow(e.App, e.Record) {
            return e.Next()
        }
        // ... rest unchanged
```

---

### WR-04: TypeScript inbox handler returns empty string body with `Content-Type: application/activity+json` on success, masking the Go handler's JSON response

**File:** `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts:57-61`
**Issue:** On success, the SvelteKit handler calls `json("", { status: 200, headers })` which serializes an empty string `""` as the JSON body and sets `Content-Type: application/activity+json`. The Go handler already returns `{"success": true}` (instance.go:189), but `event.locals.pb.send()` is called without streaming the response body back — the Go body is discarded. The response that reaches the remote ActivityPub client is `""` (a JSON string), not a JSON object. Strict ActivityPub implementations that validate the response body will fail.

The success check `if (success === false)` on line 53 is also fragile: `pb.send()` throws on non-2xx status rather than returning `false`, so this check is dead code and any 400/401 from the Go handler will be re-thrown to the outer `catch`, returning a 500 instead of forwarding the Go handler's 4xx.

**Fix:**
```typescript
const response = await event.locals.pb.send("/activitypub/instance/inbox", {
    method: "POST",
    fetch: event.fetch,
    headers: originalHeaders,
    body: JSON.stringify(activity)
});

const headers = new Headers();
headers.append("Content-Type", "application/activity+json");
return json(response, { status: 200, headers });
```
And remove the dead `if (success === false)` branch.

---

## Info

### IN-01: `instance_inbox_test.go` follows collection schema omits `"rejected"` status value — diverges from `follow_test.go`

**File:** `db/federation/instance_inbox_test.go:82`
**Issue:** The `follows` collection schema embedded in `newInboxTestApp` (line 82) only lists `["pending","accepted"]` as valid status values. `follow_test.go`'s `newFollowTestApp` (line 80) correctly includes `["pending","accepted","rejected"]` per the Plan 02 migration. Any test in `instance_inbox_test.go` that attempts to set `status="rejected"` will fail with a validation error from PocketBase. No current test exercises the rejected path in that file, so this is latent but will silently break future tests added to that file.

**Fix:** Update `instance_inbox_test.go` line 82:
```json
"values":["pending","accepted","rejected"]
```

---

### IN-02: `initData` return value is ignored in `main.go`

**File:** `db/main.go:220-232`
**Issue:** `initData` is declared with return type `error` (line 220) but the call site on line 161 (`initData(se.App, client)`) discards the return value. This is a Go vet warning (`errcheck`). Because the returned error is always `nil` in the current implementation (errors are logged internally), this has no runtime impact today, but a future change that returns a real error from `initData` will be silently ignored.

**Fix:** Either change `initData` to return nothing (preferred, since all errors are already logged internally), or capture the error at the call site:
```go
if err := initData(se.App, client); err != nil {
    app.Logger().Error(fmt.Sprintf("initData failed: %v", err))
}
```

---

_Reviewed: 2026-06-25T15:43:54Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
