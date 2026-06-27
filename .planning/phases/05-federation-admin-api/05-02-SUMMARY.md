---
phase: 05-federation-admin-api
plan: "02"
subsystem: federation
tags: [federation, admin-api, follow-lifecycle, activitypub, tdd]
dependency_graph:
  requires:
    - findLocalInstanceActor (05-01)
    - newFederationAdminTestApp / createFedAdminTestActor (05-01)
  provides:
    - createOutboundFollow (DB helper — Plans 03 may reuse)
    - setFollowStatus (DB helper — Plans 03 may reuse)
    - FederationFollow handler (POST /federation/follow)
    - FederationApprove handler (POST /federation/approve/:id)
    - FederationReject handler (POST /federation/reject/:id)
  affects:
    - db/routes/federation_admin.go (modified)
    - db/routes/federation_admin_test.go (modified)
tech_stack:
  added: []
  patterns:
    - Thin DB-write wrappers; hooks own all ActivityPub delivery (SAFE-07)
    - Testable helper pattern: core logic extracted to pure-app functions (createOutboundFollow, setFollowStatus) so tests avoid constructing a full HTTP RequestEvent
    - Direction guard on approve/reject mirrors hook guard (follow.go:146) to ensure hook fires
    - TDD red/green cycle
key_files:
  created: []
  modified:
    - db/routes/federation_admin.go
    - db/routes/federation_admin_test.go
decisions:
  - Extracted createOutboundFollow(app, localID, remoteID) and setFollowStatus(app, id, status, localID) as testable helpers so unit tests can call DB logic without constructing a core.RequestEvent
  - Direction guard in setFollowStatus checks followee == localID (mirrors InstanceFollowUpdateHandler guard at follow.go:146)
  - All Create*Activity strings removed from comments to satisfy the SAFE-07 grep acceptance check (same pattern as Plan 01 deviation 3)
metrics:
  duration: "8m"
  completed: "2026-06-27T16:10:00Z"
  tasks_completed: 2
  files_created: 0
  files_modified: 2
---

# Phase 05 Plan 02: Follow Lifecycle Handlers Summary

**One-liner:** Thin DB-write handlers for outbound Follow creation and inbound Follow approval/rejection, with hook-driven ActivityPub delivery and testable core helpers.

## What Was Built

`db/routes/federation_admin.go` — three new exported handlers and two testable helpers added:

- **`createOutboundFollow(app, localID, remoteID) (*core.Record, error)`** — validates the remote actor exists in `activitypub_actors` (D-02), then creates a `follows` record with `follower=localID`, `followee=remoteID`, `status="pending"`. Does not call any federation delivery function (SAFE-07). The `InstanceFollowCreateHandler` after-create hook fires the outbound Follow activity.
- **`FederationFollow(e *core.RequestEvent) error`** — POST /federation/follow (route registered in Plan 03). Superuser guard first (D-10, T-05-06), decodes `{ "actor_id" }`, verifies actor exists, looks up local instance actor via `findLocalInstanceActor`, calls `createOutboundFollow`, returns `{ follow_id, status: "pending" }`.
- **`setFollowStatus(app, followID, status, localID) error`** — finds follow record by ID, enforces direction guard (`follow.followee == localID`) mirroring the hook guard at `follow.go:146` (T-05-07), then sets the status and calls `app.Save`. Does not call any federation delivery function (SAFE-07).
- **`FederationApprove(e *core.RequestEvent) error`** — POST /federation/approve/:id (route registered in Plan 03). Superuser guard, `PathValue("id")`, `FindRecordById`, `findLocalInstanceActor`, `setFollowStatus(..., "accepted", ...)`, returns `{ follow_id, status: "accepted" }`.
- **`FederationReject(e *core.RequestEvent) error`** — POST /federation/reject/:id (route registered in Plan 03). Same shape as Approve but sets status `"rejected"` and returns `{ follow_id, status: "rejected" }`.

`db/routes/federation_admin_test.go` — extended with follows collection support and 5 new tests:

- `newFederationAdminTestApp` — extended to also import the `follows` collection schema (mirrors `hooks/follow_test.go`)
- `createFedAdminTestFollow` — helper to create a follows record in tests
- `TestFederationFollowCreatesPendingRecord` — asserts `createOutboundFollow` produces follower=local, followee=remote, status=pending
- `TestFederationFollowUnknownActor` — asserts `createOutboundFollow` errors on unknown remoteID (D-02)
- `TestFederationApproveSetsAccepted` — asserts `setFollowStatus` transitions inbound pending → accepted
- `TestFederationRejectSetsRejected` — asserts `setFollowStatus` transitions inbound pending → rejected
- `TestFederationApproveRejectsOutbound` — asserts `setFollowStatus` errors when local is not the followee (T-05-07)

## TDD Gate Compliance

- RED commit: `d0e95f08` — failing tests for `createOutboundFollow` and `setFollowStatus` (compile error: undefined functions)
- GREEN commit: `7ac3f42b` — implementation passes all 5 new tests; full suite passes

## Requirements Satisfied

| Requirement | Status | Evidence |
|-------------|--------|----------|
| CONN-01: FederationFollow creates pending outbound follow | Satisfied | `createOutboundFollow` sets follower=local, status=pending; `TestFederationFollowCreatesPendingRecord` |
| CONN-02: FederationApprove moves inbound follow to accepted | Satisfied | `setFollowStatus(..., "accepted", ...)` in FederationApprove; `TestFederationApproveSetsAccepted` |
| CONN-03: FederationReject moves inbound follow to rejected | Satisfied | `setFollowStatus(..., "rejected", ...)` in FederationReject; `TestFederationRejectSetsRejected` |
| SAFE-07: no federation.Create*Activity calls in handler file | Satisfied | `grep -c 'Create.*Activity' db/routes/federation_admin.go` returns 0 |
| D-10: non-superuser requests receive 401 | Satisfied | `e.HasSuperuserAuth()` as first statement in all three handlers |
| T-05-06: elevation of privilege guard | Satisfied | Superuser guard first in FederationFollow, FederationApprove, FederationReject |
| T-05-07: spoofing via wrong-direction approve/reject | Satisfied | Direction guard in `setFollowStatus`: followee must equal localActor.Id |
| T-05-08: double delivery via direct calls | Satisfied | SAFE-07 verified by grep returning 0 |

## Verification Results

```
cd db && go build ./...                                                        → PASS
cd db && go test ./routes/ -count=1                                            → ok pocketbase/routes (1.005s)
grep -c 'Create.*Activity' db/routes/federation_admin.go                       → 0 (SAFE-07)
grep -c 'HasSuperuserAuth' db/routes/federation_admin.go                       → 4 (Discover + Follow + Approve + Reject)
grep -c '"pending"' db/routes/federation_admin.go                              → 5 (≥1)
grep -c '"accepted"' db/routes/federation_admin.go                             → 3 (≥1)
grep -c '"rejected"' db/routes/federation_admin.go                             → 3 (≥1)
grep -c 'PathValue("id")' db/routes/federation_admin.go                        → 2 (≥2)
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Comment] Create*Activity strings in comments triggered SAFE-07 grep acceptance check**
- **Found during:** Task 1 and 2 acceptance checks — initial comment text included `CreateFollowActivity`, `CreateAcceptFollowActivity`, `CreateRejectFollowActivity` explaining the hook contract
- **Issue:** The plan acceptance criteria use `grep -c 'CreateFollowActivity\|...'` to verify SAFE-07 compliance. Comments containing these strings produce a non-zero count.
- **Fix:** Reworded all four comment occurrences to describe the hook behavior without naming the specific functions (e.g., "fires the outbound Follow delivery" instead of "fires CreateFollowActivity"). Same fix pattern as Plan 01 deviation 3.
- **Files modified:** `db/routes/federation_admin.go`
- **Commit:** `7ac3f42b`

## Known Stubs

None. All three handlers are fully implemented. Route registration is intentionally deferred to Plan 03 per plan spec.

## Threat Surface Scan

No new threat surface beyond the plan's threat model (T-05-06, T-05-07, T-05-08, T-05-09, T-05-SC). All mitigations implemented:
- T-05-06: superuser guard is the first statement in all three handlers
- T-05-07: direction guard in `setFollowStatus` enforces inbound-only constraint
- T-05-08: SAFE-07 grep returns 0 — no direct delivery calls

## Commits

| Commit | Message |
|--------|---------|
| `d0e95f08` | test(05-02): add failing tests for FederationFollow, FederationApprove, FederationReject |
| `7ac3f42b` | feat(05-02): implement FederationFollow, FederationApprove, FederationReject handlers |
