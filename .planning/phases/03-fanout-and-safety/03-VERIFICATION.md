---
phase: 03-fanout-and-safety
verified: 2026-06-26T09:26:32Z
status: passed
score: 5/5 must-haves verified
overrides_applied: 0
---

# Phase 3: Fanout and Safety Verification Report

**Phase Goal:** Public content created, updated, or deleted on this instance is automatically delivered to all accepted instance-actor followers, with broadcast-loop prevention, privacy enforcement, and correct delete authorization in place
**Verified:** 2026-06-26T09:26:32Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | When a public trail, summit_log, list, or comment is created/updated, accepted instance actor followers receive a Create/Update activity in addition to existing user-level fanout | VERIFIED | `instanceFollowerInboxes(app)` appended in all 4 `Create*Activity` functions (create.go lines 99-103, 193-198, 350-355, 420-425); `grep -c "instanceFollowerInboxes(app)" create.go` = 4 |
| 2 | When a public trail, summit_log, list, or comment is deleted, accepted instance actor followers receive the corresponding Delete activity | VERIFIED | `instanceFollowerInboxes(app)` appended in all 4 `Delete*Activity` functions (delete.go lines 73-78, 131-135, 206-211, 270-275); `grep -c "instanceFollowerInboxes(app)" delete.go` = 4 |
| 3 | A duplicate incoming Create activity (object IRI already in content collection) is silently dropped — no duplicate record is created | VERIFIED | SAFE-01 `if activity.Type == pub.CreateType` guard at top of all 4 `processCreateOrUpdate*` functions; `grep -c "SAFE-01" create.go` = 4; `TestProcessCreateOrUpdateTrailActivityDedupOnCreate` passes |
| 4 | A trail with `is_public = false` (or a comment on a private trail) is never included in outgoing fanout | VERIFIED | Existing gate in `CreateTrailActivity` (line 23), new SAFE-03 gate in `CreateCommentActivity` (line 125, before commentTrailAuthor fetch), existing gate in `CreateSummitLogActivity` (line 222), existing gate in `CreateListActivity` (line 376); `TestCreateCommentActivityPrivateTrailReturnsNil` passes |
| 5 | A Delete{Trail} from a non-author remote actor is rejected (non-nil error, trail record retained); an authorized Delete succeeds and removes the record | VERIFIED | `processDeleteTrailActivity` has new `actor *core.Record` parameter and `trail.GetString("author") != actor.Id` ownership guard (delete.go lines 322-334); call site updated to pass `actor`; `TestProcessDeleteTrailActivityOwnershipCheck` and `TestProcessDeleteTrailActivitySucceeds` both pass |

**Score:** 5/5 truths verified

### Note on ROADMAP SC-3 vs. Implementation

ROADMAP Success Criterion 3 states: "IRI exists in `activitypub_activities`". The actual implementation checks the IRI in content collections (`trails`, `comments`, `summit_logs`, `lists`) — NOT in `activitypub_activities`. This is explicitly documented as intentional in CONTEXT.md D-04/D-05: "Content records themselves serve as the dedup store." The behavioral goal (silent dedup of duplicate Creates) is fully achieved. The ROADMAP wording is imprecise about the mechanism but accurate about the behavior.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/federation/activity.go` | `instanceFollowerInboxes` helper | VERIFIED | Function exists at line 60; correct signature; `errors.Is(err, sql.ErrNoRows)` guard; delegates to `followerInboxes`; imports `database/sql` and `errors` added |
| `db/federation/activity_test.go` | Unit tests for instanceFollowerInboxes | VERIFIED | 3 tests: `ReturnsAcceptedFollowers`, `NoInstanceActor`, `OnlyAcceptedFollowers`; all pass |
| `db/federation/create.go` | 4x instance fanout + SAFE-03 gate + 4x SAFE-01 dedup guards | VERIFIED | 4 `instanceFollowerInboxes` calls; `!commentTrail.GetBool("public")` gate at line 125; 4 `if activity.Type == pub.CreateType` dedup guards |
| `db/federation/create_test.go` | Tests for SAFE-01 and SAFE-03 | VERIFIED | 3 tests: `DedupOnCreate`, `AllowsUpdate`, `PrivateTrailReturnsNil`; all pass |
| `db/federation/delete.go` | processDeleteTrailActivity actor param + ownership check + 4x instance fanout + CreateCommentDeleteActivity restructure | VERIFIED | New signature `(app core.App, actor *core.Record, activity pub.Activity)`; `trail.GetString("author") != actor.Id` guard; 4 `instanceFollowerInboxes` calls; `CreateCommentDeleteActivity` uses `recipients` slice, no early-return for local trail author |
| `db/federation/instance.go` | Create/Update/Delete dispatch cases in InstanceInboxHandler | VERIFIED | `case pub.CreateType: fallthrough; case pub.UpdateType` dispatches to `ProcessCreateOrUpdateActivity`; `case pub.DeleteType` dispatches to `ProcessDeleteActivity`; existing Follow/Accept/Undo and `default:` cases unchanged |
| `db/federation/delete_test.go` | SAFE-02 ownership tests | VERIFIED | `TestProcessDeleteTrailActivityOwnershipCheck` and `TestProcessDeleteTrailActivitySucceeds`; both pass |
| `db/federation/instance_inbox_test.go` | D-10 dispatch test | VERIFIED | `TestInstanceInboxDispatchesCreateActivity` added; passes |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `instanceFollowerInboxes` | `activitypub_actors` (iri lookup) | `FindFirstRecordByData("activitypub_actors", "iri", ...)` | WIRED | activity.go line 66 |
| `instanceFollowerInboxes` | `followerInboxes` | delegates on success | WIRED | activity.go line 73: `return followerInboxes(app, instanceActor.Id)` |
| `Create*Activity` recipients | `instanceFollowerInboxes` | append before PostActivity | WIRED | 4 calls confirmed in create.go; 4 calls confirmed in delete.go |
| `CreateCommentActivity` | parent trail public field | whole-function gate | WIRED | create.go line 120-127: fetches trail, checks `GetBool("public")` |
| `processCreateOrUpdate*` | content collection iri lookup | SAFE-01 Create-only dedup guard | WIRED | trails/comments/summit_logs/lists checked before any DB write |
| `processDeleteTrailActivity` | trail author ownership | record-id comparison | WIRED | delete.go line 332: `trail.GetString("author") != actor.Id` |
| `ProcessDeleteActivity` | `processDeleteTrailActivity` | actor passed through | WIRED | delete.go line 306: `err = processDeleteTrailActivity(app, actor, activity)` |
| `InstanceInboxHandler switch` | `ProcessCreateOrUpdateActivity` / `ProcessDeleteActivity` | Create/Update/Delete dispatch cases | WIRED | instance.go lines 185-194; `fallthrough` for Create→Update; separate Delete case |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./federation/` | `cd db && go build ./federation/` | exit 0, no output | PASS |
| `go vet ./federation/` | `cd db && go vet ./federation/` | exit 0, no output | PASS |
| Full test suite (16 tests) | `cd db && go test ./federation/ -count=1` | 16/16 PASS, 1.679s | PASS |
| instanceFollowerInboxes tests (3) | `go test -run TestInstanceFollowerInboxes -count=1` | 3/3 PASS | PASS |
| SAFE-01 dedup tests | `go test -run TestProcessCreateOrUpdateTrailActivity -count=1` | 2/2 PASS | PASS |
| SAFE-03 privacy gate test | `go test -run TestCreateCommentActivityPrivateTrailReturnsNil -count=1` | 1/1 PASS | PASS |
| SAFE-02 ownership tests | `go test -run TestProcessDeleteTrailActivity -count=1` | 2/2 PASS | PASS |
| D-10 dispatch test | `go test -run TestInstanceInboxDispatchesCreateActivity -count=1` | 1/1 PASS | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| SYNC-01 | 03-01, 03-02, 03-03 | Public content created delivers Create to instance followers | SATISFIED | `instanceFollowerInboxes` in all 4 Create*Activity + D-10 incoming Create dispatch |
| SYNC-02 | 03-01, 03-02, 03-03 | Public content updated delivers Update to instance followers | SATISFIED | Same 4 Create*Activity functions handle Update type; incoming Update dispatched via D-10 fallthrough |
| SYNC-03 | 03-01, 03-03 | Public content deleted delivers Delete to instance followers | SATISFIED | `instanceFollowerInboxes` in all 4 Delete*Activity; CreateCommentDeleteActivity restructured to always reach instance followers |
| SAFE-01 | 03-02 | Duplicate incoming activities silently dropped | SATISFIED | SAFE-01 dedup guards in all 4 processCreateOrUpdate* functions; TestProcessCreateOrUpdateTrailActivityDedupOnCreate passes |
| SAFE-02 | 03-03 | Delete{Trail} only applied if actor is trail author | SATISFIED | processDeleteTrailActivity signature fixed; `trail.GetString("author") != actor.Id` guard; both ownership tests pass |
| SAFE-03 | 03-02 | is_public=false records never in fanout | SATISFIED | Gates in all 4 Create*Activity; new SAFE-03 gate added for CreateCommentActivity; TestCreateCommentActivityPrivateTrailReturnsNil passes |

### Anti-Patterns Found

No blockers or warnings found. Files scanned: `activity.go`, `activity_test.go`, `create.go`, `create_test.go`, `delete.go`, `delete_test.go`, `instance.go`, `instance_inbox_test.go`. No TBD/FIXME/XXX markers. No placeholder implementations. No stub patterns.

### Human Verification Required

None. All phase deliverables are verifiable programmatically. The test suite provides behavioral coverage for all 6 requirements. No visual, real-time, or external-service behaviors were introduced in this phase.

### Gaps Summary

No gaps. All 5 observable truths are VERIFIED. All 6 requirement IDs (SYNC-01, SYNC-02, SYNC-03, SAFE-01, SAFE-02, SAFE-03) are satisfied with code and passing tests. The package builds and vets clean. The full 16-test suite passes.

---

_Verified: 2026-06-26T09:26:32Z_
_Verifier: Claude (gsd-verifier)_
