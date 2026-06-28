---
phase: 02-follow-lifecycle
plan: 04
subsystem: federation
tags: [activitypub, go, pocketbase, follow-lifecycle, gap-closure]

# Dependency graph
requires:
  - phase: 02-follow-lifecycle/02-03
    provides: InstanceFollowCreateHandler, InstanceFollowDeleteHandler, InstanceFollowUpdateHandler, InstanceInboxHandler, ProcessAcceptActivity

provides:
  - isOutboundInstanceFollow helper (follower-directional check, CR-03 fix)
  - CR-01 guard: CreateFollowHandler skips user-level delivery for instance follows
  - CR-02 guard: DeleteFollowHandler skips user-level delivery for instance follows
  - CR-03 fix: InstanceFollowCreateHandler uses isOutboundInstanceFollow instead of isInstanceFollow
  - CR-04 fix: ProcessAcceptActivity uses comma-ok assertion — no panic on IRI-only Accept objects
  - CR-05 fix: InstanceInboxHandler returns 400 on processing failures and on unrecognized activity types
  - Regression tests covering CR-03 inbound case and CR-04 panic-safety

affects: [phase-03, phase-04, federation-fanout]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Directional guard pattern: isOutboundInstanceFollow checks only the follower actor IRI; isInstanceFollow retains EITHER-direction semantics for update/delete handlers"
    - "Comma-ok type assertion before any DB call for safe handling of IRI-only AP objects"
    - "Per-case error propagation in inbox dispatch: BadRequestError on failure, success body on 2xx"

key-files:
  created:
    - db/federation/follow_accept_test.go
  modified:
    - db/hooks/follow.go
    - db/hooks/follow_test.go
    - db/federation/follow.go
    - db/federation/instance.go

key-decisions:
  - "isOutboundInstanceFollow is a separate helper from isInstanceFollow — the EITHER-direction semantics of isInstanceFollow are intentionally preserved for InstanceFollowUpdateHandler and InstanceFollowDeleteHandler (inbound follows legitimately need Accept/Reject/Undo on the followee side)"
  - "InstanceInboxHandler success body is map[string]bool{success:true} not nil-error JSON (Go errors serialize to null, masking failures from remote callers)"
  - "CR-04 guard placed as first statement in ProcessAcceptActivity so it returns before any DB call — a nil app is safe to pass in unit tests"

patterns-established:
  - "Direction matters for hook dispatch: use a follower-only check in create handlers, EITHER-direction check in update/delete handlers"
  - "Comma-ok on ActivityPub type assertions: IRI-only object fields are common in real-world AP implementations; never assert without ok guard"

requirements-completed: [INST-03, FLCL-01, FLCL-02, FLCL-03, FLCL-04, FLCL-05]

# Metrics
duration: 3min
completed: 2026-06-25
---

# Phase 02 Plan 04: Gap Closure (CR-01 through CR-05) Summary

**Five correctness defects in Phase 02 follow-lifecycle code fixed: double-delivery guards on two hooks, directional guard preventing inbound-follow mis-firing, panic-safe type assertion on Accept objects, and honest 400 error propagation in the instance inbox.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-06-25T15:34:41Z
- **Completed:** 2026-06-25T15:37:44Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- CR-01/CR-02: `CreateFollowHandler` and `DeleteFollowHandler` now skip user-level ActivityPub delivery for instance follows via `isInstanceFollow` early-return guards; the AfterSuccess handlers (InstanceFollowCreate/DeleteHandler) are the single delivery path
- CR-03: Added `isOutboundInstanceFollow` helper that loads only the follower actor and compares its IRI to the local instance IRI; `InstanceFollowCreateHandler` now uses this instead of `isInstanceFollow`, preventing it from firing on inbound follows saved by `ProcessFollowActivity`
- CR-04: `ProcessAcceptActivity` now uses comma-ok type assertion on `activity.Object` and returns a descriptive error instead of panicking on IRI-only Accept objects (common in real ActivityPub implementations)
- CR-05: `InstanceInboxHandler` now propagates processing errors as 400 responses per case, has a `default` case rejecting unknown activity types with 400, and returns `{"success":true}` on success instead of serialized-nil JSON
- All 9 hooks tests and all 7 federation tests pass; `go build ./...` and `go vet ./...` pass

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: CR-03 directional tests** - `7d96f8fe` (test)
2. **Task 1 GREEN: CR-01/CR-02/CR-03 guards in hooks/follow.go** - `4e9a6cfa` (feat)
3. **Task 2 RED: CR-04 panic-safety test** - `281df160` (test)
4. **Task 2 GREEN: CR-04 comma-ok assertion in follow.go** - `8243bf41` (feat)
5. **Task 3: CR-05 error propagation in instance.go** - `3c7f9ee8` (feat)

## Files Created/Modified
- `db/hooks/follow.go` — added `isOutboundInstanceFollow` helper; CR-01 guard in `CreateFollowHandler`; CR-02 guard in `DeleteFollowHandler`; CR-03 directional guard in `InstanceFollowCreateHandler`
- `db/hooks/follow_test.go` — added 3 new tests: `TestIsOutboundInstanceFollowTrueWhenInstanceIsFollower`, `TestIsOutboundInstanceFollowFalseWhenInstanceIsFollowee` (CR-03 inbound regression), `TestIsOutboundInstanceFollowFalseWhenNeitherIsInstance`
- `db/federation/follow.go` — replaced unguarded `activity.Object.(*pub.Activity)` with comma-ok form and early error return in `ProcessAcceptActivity` (CR-04)
- `db/federation/follow_accept_test.go` (created) — `TestProcessAcceptActivityIRIOnlyObjectReturnsError` asserting no panic and correct error message
- `db/federation/instance.go` — removed `var procErr error` / `e.JSON(http.StatusOK, procErr)` pattern; per-case `BadRequestError` returns; `default` case; success body `{"success":true}` (CR-05)

## Decisions Made
- `isOutboundInstanceFollow` is a separate helper so `isInstanceFollow` retains its EITHER-direction semantics unchanged — Update/Delete handlers legitimately need to match on followee=instance (to Accept/Reject/Undo inbound follows)
- CR-04 guard is the first statement in `ProcessAcceptActivity` so passing nil app is safe in unit tests (returns before any DB call)
- `InstanceInboxHandler` success body is `map[string]bool{"success": true}` not a nil-error JSON value — Go errors serialize to null, so `return e.JSON(http.StatusOK, procErr)` with a nil procErr was indistinguishable from success with nil error from a processing failure

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None — all five gap fixes applied cleanly. The comma-ok fix for CR-04 was confirmed to prevent a real panic by the RED-phase test which demonstrated the panic before the fix.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All five CR blockers resolved; the Phase 02 follow lifecycle is now correct under realistic conditions
- Phase 03 (content fanout) can proceed: single-delivery for Follow/Undo, correct directional routing, panic-safe Accept handling, and honest error propagation are all in place
- No architectural changes; all existing Phase 02 tests remain green

## Self-Check: PASSED

- SUMMARY.md: FOUND at .planning/phases/02-follow-lifecycle/02-04-SUMMARY.md
- Commit 7d96f8fe (test CR-03 RED): FOUND
- Commit 4e9a6cfa (feat CR-01/02/03): FOUND
- Commit 281df160 (test CR-04 RED): FOUND
- Commit 8243bf41 (feat CR-04): FOUND
- Commit 3c7f9ee8 (feat CR-05): FOUND
- `go build ./...`: PASSED
- `go vet ./hooks/ ./federation/`: PASSED
- `go test ./hooks/ -count=1`: 9/9 PASS
- `go test ./federation/ -count=1`: 7/7 PASS

---
*Phase: 02-follow-lifecycle*
*Completed: 2026-06-25*
</content>
</invoke>