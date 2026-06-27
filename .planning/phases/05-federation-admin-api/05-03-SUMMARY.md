---
phase: 05-federation-admin-api
plan: "03"
subsystem: federation
tags: [federation, admin-api, disconnect, peers, activitypub, tdd, route-wiring]
dependency_graph:
  requires:
    - findLocalInstanceActor (05-01)
    - FederationDiscover (05-01)
    - FederationFollow, FederationApprove, FederationReject (05-02)
    - createOutboundFollow, setFollowStatus (05-02)
  provides:
    - disconnectAction (pure routing helper)
    - FederationDisconnect handler (POST /federation/disconnect/:id)
    - followInput (lightweight value type for pure testing)
    - buildPeerEntries (pure fold helper)
    - FederationPeers handler (GET /federation/peers)
    - Six se.Router registrations in db/main.go
  affects:
    - db/routes/federation_admin.go (modified)
    - db/routes/federation_admin_test.go (modified)
    - db/main.go (modified)
tech_stack:
  added: []
  patterns:
    - Pure helper extraction pattern: disconnectAction(followerID, localID) and buildPeerEntries(outbound, inbound, localID, domainOf) tested without DB access
    - followInput value type adapts core.Record slices so the folding helper stays pure
    - domainOf closure pattern: handler supplies DB-backed closure; helper receives interface
    - Insertion-sort for deterministic peer list output (peer lists typically small)
    - TDD red/green cycle (RED commit 76778430, GREEN commit b84487cb)
key_files:
  created: []
  modified:
    - db/routes/federation_admin.go
    - db/routes/federation_admin_test.go
    - db/main.go
decisions:
  - disconnectAction extracted as pure helper so direction routing is independently unit-tested without constructing core.RequestEvent
  - buildPeerEntries accepts []followInput (lightweight struct) instead of []*core.Record so the folding logic is pure/testable; the handler adapts records via a toInputs() closure
  - mutual collapse only fires when BOTH sides have status="accepted" (not pending or rejected), matching the semantic intent of D-05
  - domainOf closure pattern keeps buildPeerEntries free of App dependency; handler supplies FindRecordById-backed closure
  - followInput defined in federation_admin.go (same package) so it is accessible from both the implementation and test file without duplication
  - All Create*Activity strings kept out of comments to satisfy SAFE-07 grep acceptance check (same pattern as Plans 01/02)
metrics:
  duration: "5m"
  completed: "2026-06-27T15:34:36Z"
  tasks_completed: 3
  files_created: 0
  files_modified: 3
---

# Phase 05 Plan 03: Disconnect Handler, Peer List, and Route Wiring Summary

**One-liner:** Direction-aware disconnect (outbound→delete/Undo, inbound-only→rejected/Reject) and domain-collapsed peer list, with all six federation admin endpoints wired in db/main.go.

## What Was Built

`db/routes/federation_admin.go` — two new exported handlers, two pure helpers, and a value type added:

- **`disconnectAction(followerID, localID string) string`** — returns "delete" when followerID == localID (outbound), "reject" otherwise (inbound-only). Pure function with no side effects; extracted for unit-testability (T-05-10).
- **`FederationDisconnect(e *core.RequestEvent) error`** — POST /federation/disconnect/:id. Superuser guard first. Loads follow record, looks up local instance actor, calls disconnectAction to determine direction, then either calls `e.App.Delete(follow)` (outbound, so the after-delete hook fires Undo{Follow}) or sets `follow.Set("status", "rejected")` + `e.App.Save(follow)` (inbound-only, so the after-update hook fires Reject{Follow}). Never calls federation delivery functions directly (SAFE-07, CONN-04, T-05-10, T-05-11, T-05-13).
- **`followInput`** — lightweight value struct `{ID, Follower, Followee, Status}` that decouples the buildPeerEntries fold from core.Record, enabling pure unit tests.
- **`buildPeerEntries(outbound, inbound []followInput, localID string, domainOf func(string) string) []peerEntry`** — folds two follow slices into a deduplicated []peerEntry keyed by remote domain. When an accepted outbound and an accepted inbound for the same domain exist, collapses to direction="mutual" keeping follow_id = outbound record id (D-04, D-05). Returns slice sorted by domain for deterministic output.
- **`FederationPeers(e *core.RequestEvent) error`** — GET /federation/peers. Superuser guard first. Queries outbound follows (follower=localActor.Id) and inbound follows (followee=localActor.Id) via FindRecordsByFilter. Adapts records to []followInput. Supplies a domainOf closure backed by FindRecordById on activitypub_actors. Returns JSON array from buildPeerEntries (DISC-01, D-04, D-05, T-05-12, SAFE-07).

`db/routes/federation_admin_test.go` — 5 new tests added:

- `TestDisconnectActionOutbound` — followerID == localID → "delete"
- `TestDisconnectActionInbound` — followerID != localID → "reject"
- `TestBuildPeerEntriesOutboundOnly` — one outbound accepted → direction="outbound"
- `TestBuildPeerEntriesInboundOnly` — one inbound pending → direction="inbound"
- `TestBuildPeerEntriesMutual` — accepted outbound + accepted inbound same domain → direction="mutual", follow_id == outbound id

`db/main.go` — six federation admin routes added to `registerRoutes()`:

- `se.Router.POST("/federation/discover", routes.FederationDiscover)`
- `se.Router.POST("/federation/follow", routes.FederationFollow)`
- `se.Router.POST("/federation/approve/{id}", routes.FederationApprove)`
- `se.Router.POST("/federation/reject/{id}", routes.FederationReject)`
- `se.Router.POST("/federation/disconnect/{id}", routes.FederationDisconnect)`
- `se.Router.GET("/federation/peers", routes.FederationPeers)`

## TDD Gate Compliance

- RED commit: `76778430` — failing tests for disconnectAction, buildPeerEntries (undefined symbols)
- GREEN commit: `b84487cb` — implementation passes all 5 new tests; full suite passes

## Requirements Satisfied

| Requirement | Status | Evidence |
|-------------|--------|----------|
| CONN-04: direction-aware disconnect | Satisfied | disconnectAction routes to delete (Undo) vs rejected (Reject); TestDisconnectActionOutbound, TestDisconnectActionInbound |
| DISC-01 (peer-list slice): GET /federation/peers returns [{ follow_id, direction, status, domain }] | Satisfied | FederationPeers + buildPeerEntries; peerEntry json tags verified |
| SAFE-07: no federation.Create*Activity calls | Satisfied | `grep -c 'Create.*Activity' db/routes/federation_admin.go` returns 0 |
| D-04: mutual pairs collapsed per domain | Satisfied | buildPeerEntries keyed by domain; TestBuildPeerEntriesMutual |
| D-05: mutual follow_id = outbound record id | Satisfied | buildPeerEntries keeps outbound ID on mutual merge; TestBuildPeerEntriesMutual asserts follow_id == outbound id |
| D-10: non-superuser 401 on disconnect and peers | Satisfied | e.HasSuperuserAuth() is first statement in FederationDisconnect and FederationPeers |
| T-05-10: wrong-direction Undo on inbound disconnect | Mitigated | inbound-only path uses Save(status=rejected), never Delete; test verifies via disconnectAction |
| T-05-11: elevation of privilege on disconnect | Mitigated | superuser guard first in FederationDisconnect |
| T-05-12: peer enumeration by non-admin | Mitigated | superuser guard first in FederationPeers |
| T-05-13: direct ActivityPub delivery from handlers | Mitigated | SAFE-07 grep returns 0 |
| All six endpoints registered | Satisfied | grep -c 'routes.Federation' db/main.go returns 6; go build ./... exits 0 |

## Verification Results

```
cd db && go build ./...                                                        → PASS
cd db && go test ./routes/ -count=1                                            → ok pocketbase/routes (0.812s)
grep -c 'Create.*Activity' db/routes/federation_admin.go                       → 0 (SAFE-07)
grep -c 'routes.Federation' db/main.go                                         → 6 (all six wired)
grep -c '/federation/discover|...' db/main.go                                  → 6
go test -run 'TestDisconnectAction|TestBuildPeerEntries' -count=1 -v           → 5/5 PASS
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Design] followInput defined in production file, not test file**
- **Found during:** Task 1 GREEN — the test file references `followInput` and `buildPeerEntries` expects `[]followInput`. Since both live in package `routes`, defining `followInput` in `federation_admin.go` (the impl file) avoids a duplicate-type compile error when the test file also tries to define it.
- **Issue:** The RED test commit defined `followInput` in the test file. Moving to impl file means the test file can reference it without redeclaration.
- **Fix:** Removed `followInput` struct from the test file; kept definition in `federation_admin.go`. No functional change — same package visibility.
- **Files modified:** `db/routes/federation_admin_test.go`, `db/routes/federation_admin.go`
- **Commit:** `b84487cb`

## Known Stubs

None. All six handlers are fully implemented and wired. No placeholder data or TODO paths exist in any handler.

## Threat Surface Scan

No new threat surface beyond what is already in the plan's threat model:
- T-05-10: direction guard via disconnectAction — mitigated
- T-05-11/T-05-12: superuser guards on both new handlers — mitigated
- T-05-13: SAFE-07 grep returns 0 — mitigated

## Commits

| Commit | Message |
|--------|---------|
| `76778430` | test(05-03): add failing tests for disconnectAction, buildPeerEntries (TDD RED) |
| `b84487cb` | feat(05-03): implement FederationDisconnect, FederationPeers, buildPeerEntries (TDD GREEN) |
| `b90bcb44` | feat(05-03): register all six federation admin routes in db/main.go |

## Self-Check: PASSED

| Item | Status |
|------|--------|
| `db/routes/federation_admin.go` contains `func FederationDisconnect` | FOUND |
| `db/routes/federation_admin.go` contains `func disconnectAction` | FOUND |
| `db/routes/federation_admin.go` contains `func buildPeerEntries` | FOUND |
| `db/routes/federation_admin.go` contains `func FederationPeers` | FOUND |
| `db/routes/federation_admin.go` contains `type peerEntry struct` | FOUND (Plan 01) |
| `db/routes/federation_admin.go` contains `type followInput struct` | FOUND |
| `db/main.go` contains 6 `/federation/` route registrations | FOUND |
| `db/main.go` contains 6 `routes.Federation` references | FOUND |
| Commit `76778430` (test RED) | FOUND |
| Commit `b84487cb` (feat GREEN) | FOUND |
| Commit `b90bcb44` (feat route wiring) | FOUND |
| `go build ./...` exits 0 | PASSED |
| `go test ./routes/ -count=1` exits 0 | PASSED |
| `grep -c 'Create.*Activity' db/routes/federation_admin.go` = 0 | PASSED |
