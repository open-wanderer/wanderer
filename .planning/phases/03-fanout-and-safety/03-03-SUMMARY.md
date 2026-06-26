---
phase: 03-fanout-and-safety
plan: "03"
subsystem: federation
tags: [activitypub, go, pocketbase, fanout, delete-auth, instance-inbox, safe-02, sync-03, d-10]

requires:
  - phase: 03-fanout-and-safety/03-01
    provides: instanceFollowerInboxes(app core.App) helper in db/federation/activity.go
  - phase: 03-fanout-and-safety/03-02
    provides: Create-only dedup guards, comment privacy gate, Create*Activity fanout injection

provides:
  - SAFE-02: processDeleteTrailActivity enforces trail author ownership before delete
  - SYNC-03 sending side: instance follower fanout in all 4 Delete*Activity functions
  - D-10: InstanceInboxHandler dispatches Create/Update/Delete to federation processors
  - delete_test.go with SAFE-02 ownership tests
  - instance_inbox_test.go extended with D-10 dispatch test

affects:
  - Phase 4 (NodeInfo): no overlap; delete/inbox dispatch is fully independent

tech-stack:
  added: []
  patterns:
    - "SAFE-02 ownership guard: trail.GetString(\"author\") != actor.Id before app.Delete (mirrors processDeleteCommentActivity)"
    - "Fanout inject append: instanceInboxes, err := instanceFollowerInboxes(app); recipients = append(recipients, instanceInboxes...) in Delete*Activity"
    - "CreateCommentDeleteActivity restructure: recipients slice; gate trail-author inbox on !is_local; always append instanceFollowerInboxes"
    - "D-10 switch dispatch: case pub.CreateType: fallthrough; case pub.UpdateType: ProcessCreateOrUpdateActivity; case pub.DeleteType: ProcessDeleteActivity"
    - "Feed seed in positive delete test: seed feed row before calling processDeleteTrailActivity so DeleteFromFeed succeeds"

key-files:
  created:
    - db/federation/delete_test.go
  modified:
    - db/federation/delete.go
    - db/federation/instance.go
    - db/federation/instance_inbox_test.go

key-decisions:
  - "SAFE-02 uses record-id comparison (actor.Id) not IRI — trail.GetString(\"author\") is a 15-char PocketBase record id, confirmed by RESEARCH.md Pitfall 1 and processDeleteCommentActivity pattern"
  - "CreateCommentDeleteActivity early-return removed: instance followers always receive comment deletes; trail-author inbox only added when trail author is remote"
  - "Positive SAFE-02 test seeds a feed row to satisfy DeleteFromFeed (which propagates sql.ErrNoRows if no feed entry exists)"
  - "D-10 dispatch test uses ProcessCreateOrUpdateActivity directly (same call the new switch case makes) — no signed HTTP request construction needed"

patterns-established:
  - "Feed row seeding in delete tests: required because DeleteFromFeed returns sql.ErrNoRows (not a no-op) when no feed entry exists"
  - "Place struct initialization: pub.Place{Type: pub.PlaceType, Latitude: ..., Longitude: ..., Name: ...} — Place is a flat struct, no Parent/Object embedding"

requirements-completed: [SYNC-01, SYNC-02, SYNC-03, SAFE-02]

duration: 12min
completed: "2026-06-26"
---

# Phase 3, Plan 03: Delete-Side Safety + Instance Inbox Dispatch Summary

**processDeleteTrailActivity gains actor ownership guard (SAFE-02); all 4 Delete*Activity functions fan out to instance followers (SYNC-03); InstanceInboxHandler dispatches Create/Update/Delete to federation processors (D-10); 3 new tests pass**

## Performance

- **Duration:** 12 min
- **Started:** 2026-06-26T09:10:00Z
- **Completed:** 2026-06-26T09:22:00Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- **Task 1 (delete.go):**
  - `processDeleteTrailActivity` signature changed from `(app core.App, activity pub.Activity)` to `(app core.App, actor *core.Record, activity pub.Activity)`
  - Ownership guard added: `trail.GetString("author") != actor.Id` returns `fmt.Errorf("actor is not trail author")` before any delete (SAFE-02)
  - `ProcessDeleteActivity` call site updated to pass `actor` (SAFE-02 call site fix)
  - `instanceFollowerInboxes(app)` append added to `CreateTrailDeleteActivity`, `CreateSummitLogDeleteActivity`, `CreateListDeleteActivity` — 3 calls (SYNC-03)
  - `CreateCommentDeleteActivity` restructured: removed `if commentTrailAuthor.GetBool("is_local") { return nil }` early-return; replaced literal `[]string{to + "/inbox"}` with a `recipients` slice; gates trail-author inbox on `!commentTrailAuthor.GetBool("is_local")`; always appends `instanceFollowerInboxes` — 1 call (SYNC-03, total = 4 calls)

- **Task 2 (instance.go):**
  - Added `case pub.CreateType: fallthrough; case pub.UpdateType: ProcessCreateOrUpdateActivity(e.App, actor, recipient, activity)`
  - Added `case pub.DeleteType: ProcessDeleteActivity(e.App, actor, activity)`
  - Both use already-resolved `actor` (remote sender) and `recipient` (local instance actor) variables
  - Existing Follow/Accept/Undo cases and `default:` case unchanged

- **Task 3 (tests):**
  - `delete_test.go`: `TestProcessDeleteTrailActivityOwnershipCheck` — non-author Delete returns non-nil error AND trail row survives
  - `delete_test.go`: `TestProcessDeleteTrailActivitySucceeds` — author Delete returns nil AND trail row is removed
  - `instance_inbox_test.go`: `TestInstanceInboxDispatchesCreateActivity` — Create activity dispatched to ProcessCreateOrUpdateActivity creates trail row
  - Full federation test suite: 16/16 tests pass

## Task Commits

Each task was committed atomically:

1. **Task 1: SAFE-02 ownership check + instance fanout in Delete*Activity functions** — `35fc7b90` (feat)
2. **Task 2: Extend InstanceInboxHandler with Create/Update/Delete dispatch** — `69e79cfc` (feat)
3. **Task 3: SAFE-02 ownership tests + D-10 dispatch test** — `0f02b1e7` (test)

## Files Created/Modified

- `db/federation/delete.go` — SAFE-02 signature+guard; call site fix; 4× instanceFollowerInboxes append; CreateCommentDeleteActivity restructure
- `db/federation/instance.go` — Create/Update/Delete dispatch cases in InstanceInboxHandler switch
- `db/federation/delete_test.go` — New file: TestProcessDeleteTrailActivityOwnershipCheck, TestProcessDeleteTrailActivitySucceeds, addFeedCollection helper
- `db/federation/instance_inbox_test.go` — Added TestInstanceInboxDispatchesCreateActivity

## Decisions Made

- **SAFE-02 comparison**: `trail.GetString("author") != actor.Id` (record id, not IRI) — confirmed by RESEARCH.md Pitfall 1 and existing `processDeleteCommentActivity` pattern
- **CreateCommentDeleteActivity restructure**: removes early-return for local trail author so instance followers always receive comment delete fanout (Pitfall 3 / Open Question 2)
- **Positive delete test**: seeds a feed row before calling `processDeleteTrailActivity` because `DeleteFromFeed` propagates `sql.ErrNoRows` (not a graceful no-op) when no feed entry exists
- **D-10 dispatch test**: calls `ProcessCreateOrUpdateActivity` directly with a `pub.Place`-equipped activity object; `pub.Place` is a flat struct with no `Parent`/`Object` embedding

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] pub.Place struct literal had non-existent `Parent` field**
- **Found during:** Task 3 — IDE diagnostic immediately after writing the test
- **Issue:** `pub.Place` is a flat struct (not embedding `pub.Object`); the `Parent` field does not exist
- **Fix:** Changed to `pub.Place{Type: pub.PlaceType, Latitude: ..., Longitude: ..., Name: ...}` (all fields inline)
- **Files modified:** `db/federation/instance_inbox_test.go`
- **Commit:** Included in `0f02b1e7`

**2. [Rule 1 - Bug] Positive SAFE-02 test failed because DeleteFromFeed propagates sql.ErrNoRows**
- **Found during:** Task 3 — test run showed `delete_test.go:131: expected trail to be deleted, but record still exists`
- **Issue:** `DeleteFromFeed` calls `FindFirstRecordByData("feed","item",id)` and returns the error directly (including `sql.ErrNoRows`) — the trail delete never ran
- **Fix:** Seeded a feed record for the trail before calling `processDeleteTrailActivity` so `DeleteFromFeed` finds and deletes the entry, and `app.Delete(trail)` is reached
- **Files modified:** `db/federation/delete_test.go`
- **Commit:** Included in `0f02b1e7`

## Known Stubs

None. All changes are behavioral (ownership guard, fanout injection, inbox dispatch) with no placeholder data.

## Threat Flags

None. All threat model items from the plan's `<threat_model>` are addressed:
- T-03-06 (Tampering — unauthorized trail delete) → mitigated by SAFE-02 ownership guard
- T-03-07 (Spoofing — forged Delete) → HTTP signature (Phase 2, unchanged) + SAFE-02 ownership check
- T-03-08 (DoS — Delete storm) → signature auth + SAFE-02 rejects non-matching actors with 400 before DB delete
- T-03-09 (Information Disclosure — private trail delete leaking) → existing `if !r.GetBool("public") { return nil }` gates remain; instance fanout appended only after those gates

No new network endpoints, auth paths, file access patterns, or schema changes introduced.

## Self-Check

Files exist:
- `db/federation/delete.go` — FOUND (modified: SAFE-02 guard + 4× instanceFollowerInboxes)
- `db/federation/instance.go` — FOUND (modified: Create/Update/Delete dispatch cases)
- `db/federation/delete_test.go` — FOUND (created with 2 SAFE-02 tests + addFeedCollection)
- `db/federation/instance_inbox_test.go` — FOUND (extended with TestInstanceInboxDispatchesCreateActivity)

Commits exist:
- `35fc7b90` — feat(03-03): SAFE-02 ownership check + instance fanout in Delete*Activity functions
- `69e79cfc` — feat(03-03): extend InstanceInboxHandler with Create/Update/Delete dispatch (D-10)
- `0f02b1e7` — test(03-03): add SAFE-02 ownership tests and D-10 dispatch test

Verification:
- `go build ./federation/` — PASS
- `go vet ./federation/` — PASS
- `grep -c "instanceFollowerInboxes(app)" db/federation/delete.go` — 4
- `grep -F "func processDeleteTrailActivity(app core.App, actor *core.Record, activity pub.Activity) error"` — 1 match
- `grep -F "processDeleteTrailActivity(app, actor, activity)"` — 1 match (call site)
- `grep -c "case pub.CreateType|case pub.UpdateType|case pub.DeleteType" db/federation/instance.go` — 3
- `go test ./federation/ -run "TestProcessDeleteTrailActivity|TestInstanceInboxDispatchesCreateActivity" -count=1` — PASS (3/3)
- Full suite: PASS (16/16)

## Self-Check: PASSED

---
*Phase: 03-fanout-and-safety*
*Completed: 2026-06-26*
