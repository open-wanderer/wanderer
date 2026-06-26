---
plan: 02-03
phase: 02-follow-lifecycle
status: complete
completed: 2026-06-25
tasks_total: 3
tasks_complete: 3
self_check: PASSED
---

## Summary

Delivered the admin-driven half of the instance Follow lifecycle. An admin operating PocketBase's native UI can now initiate an outgoing Follow, approve or reject an incoming pending Follow, and disconnect from a peer — with the correct ActivityPub activity delivered to the remote instance at each step.

## What Was Built

### Task 1: Accept/Reject delivery helpers (federation/follow.go)

Added two exported functions:

- `CreateAcceptFollowActivity(app core.App, follow *core.Record) error` — reloads the original incoming Follow from `activitypub_activities`, wraps it in `pub.AcceptNew`, persists an Accept activity record, and delivers it to the remote sender's inbox signed as the instance actor.
- `CreateRejectFollowActivity(app core.App, follow *core.Record) error` — identical shape using `pub.RejectNew`; mandatory per D-08.

Both use `FindFirstRecordByFilter("activitypub_activities", "actor={:actor}&&object={:object}&&type={:type}", ...)` to reconstruct the original Follow, ensuring correctness without trusting attacker-controlled input.

### Task 2: Instance follow hooks + isInstanceFollow filter (hooks/follow.go + hooks/follow_test.go)

Added to `db/hooks/follow.go`:

- `isInstanceFollow(app, follow)` — checks whether follower or followee IRI matches `ORIGIN + "/api/v1/activitypub/instance"`; returns false for all user-level follows (guards Pitfall 4).
- `instanceFollowAction(oldStatus, newStatus)` — pure helper mapping status transitions to `"accept"` / `"reject"` / `""` for unit-testability.
- `InstanceFollowCreateHandler()` — `AfterSuccess` handler that dispatches `CreateFollowActivity`; includes remote-actor auto-fetch via `GetActorByIRI` so admin only needs to supply the remote IRI.
- `InstanceFollowUpdateHandler()` — dispatches Accept or Reject based on `e.Record.Original()` vs current status; no-ops when status unchanged.
- `InstanceFollowDeleteHandler()` — dispatches `CreateUnfollowActivity`.

All three use `AfterSuccess` hooks (not `Request` hooks) as required by D-06/D-07 to fire on admin-panel operations.

Created `db/hooks/follow_test.go` with 4 tests covering `isInstanceFollow` (true via followee, true via follower, false for person-only) and `instanceFollowAction` status-change gate.

### Task 3: Hook registration in main.go

Added three `AfterSuccess` registrations in `setupEventHandlers`:
```go
app.OnRecordAfterCreateSuccess("follows").BindFunc(hooks.InstanceFollowCreateHandler())
app.OnRecordAfterUpdateSuccess("follows").BindFunc(hooks.InstanceFollowUpdateHandler())
app.OnRecordAfterDeleteSuccess("follows").BindFunc(hooks.InstanceFollowDeleteHandler())
```
The existing `OnRecordCreateRequest("follows")` and `OnRecordDeleteRequest("follows")` registrations are unchanged.

## Verification

- `go build ./...` — passes
- `go vet ./hooks/ ./federation/` — passes
- `go test ./hooks/ -run TestInstanceFollow -count=1` — passes (4 tests)
- `go test ./federation/ -count=1` — passes (Plan 02 tests still green)
- All 3 AfterSuccess registrations present in main.go; both Request registrations retained

## Key Files

### Created
- `db/hooks/follow_test.go` — unit tests for isInstanceFollow and instanceFollowAction

### Modified
- `db/federation/follow.go` — added CreateAcceptFollowActivity, CreateRejectFollowActivity
- `db/hooks/follow.go` — added isInstanceFollow, instanceFollowAction, three AfterSuccess handlers
- `db/main.go` — registered three AfterSuccess follows hooks

## Deviations

None. Plan executed as specified.

## Requirements Addressed

- FLCL-01: Admin-initiated outgoing Follow via OnRecordAfterCreateSuccess hook
- FLCL-03: Accept{Follow} delivered when admin sets status=accepted
- FLCL-04: Reject{Follow} delivered when admin sets status=rejected (D-08 mandatory path)
- FLCL-05: Undo{Follow} delivered when admin deletes instance follow record
