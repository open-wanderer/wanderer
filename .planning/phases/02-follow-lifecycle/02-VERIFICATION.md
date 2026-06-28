---
phase: 02-follow-lifecycle
verified: 2026-06-26T00:00:00Z
status: complete
score: 5/5 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/5
  gaps_closed:
    - "CR-01: CreateFollowHandler now has isInstanceFollow early-return guard before calling federation.CreateFollowActivity"
    - "CR-02: DeleteFollowHandler now has isInstanceFollow early-return guard before calling federation.CreateUnfollowActivity"
    - "CR-03: InstanceFollowCreateHandler now uses isOutboundInstanceFollow (follower-directional) instead of isInstanceFollow (either-direction)"
    - "CR-04: ProcessAcceptActivity now uses comma-ok type assertion on activity.Object — returns descriptive error instead of panicking on IRI-only Accept objects"
    - "CR-05: InstanceInboxHandler now returns e.BadRequestError per case on failure, has a default case rejecting unknown types with 400, and returns {success:true} on success"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Confirm that in the deployment environment, POST /api/v1/activitypub/instance/inbox is routed to the SvelteKit app the same way POST /api/v1/activitypub/user/{handle}/inbox is routed. Specifically verify that the SvelteKit adapter-node/nginx/caddy config does NOT strip or rewrite X-Forwarded-Path before it reaches the Go handler."
    expected: "Remote instances can POST HTTP-signed activities to the inbox IRI; the Go handler reconstructs the signed path correctly and verifies the signature."
    why_human: "Proxy config is outside the codebase (docker-compose / nginx / caddy / deployment config); cannot be verified by grep or code inspection."
    result: VERIFIED
    evidence: "UAT 2026-06-26 — Follow lifecycle tests 2.1–2.5 all passed against local Caddy setup (wanderer-a.mac.lan → localhost:5173 → PocketBase:8090). HTTP-signed Follow/Accept/Reject/Undo activities were successfully processed through the proxy chain, confirming X-Forwarded-Path is preserved end-to-end."
---

# Phase 02: Follow Lifecycle Verification Report (Re-verification)

**Phase Goal:** Deliver the instance Follow lifecycle — an administrator can initiate, accept/reject, and disconnect a peer-to-peer Follow relationship between two Wanderer instances, with the correct ActivityPub activities (Follow, Accept, Reject, Undo) delivered to the remote instance at each step.
**Verified:** 2026-06-25T16:00:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (Plan 02-04 closed CR-01 through CR-05)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | POST `{ORIGIN}/api/v1/activitypub/instance/inbox` accepts HTTP-signed activities from authenticated remote actors and routes them to the correct processor | VERIFIED | `InstanceInboxHandler` in `db/federation/instance.go` calls `util.VerifySignature`, returns 401 on invalid signature, dispatches Follow/Accept/Undo. Route registered at `db/main.go:190`. SvelteKit proxy at `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts` sets `X-Forwarded-Path`. 7/7 federation tests pass including 3 ProcessFollow tests. |
| 2 | Incoming Follow from an `Application`-type actor is stored as `pending` in `follows` and NOT auto-accepted | VERIFIED | `ProcessFollowActivity` (follow.go:91-113) branches on `object.GetString("actor_type") == "instance" && object.GetBool("is_local")`, sets status="pending", persists incoming Follow to `activitypub_activities`, returns nil before Accept delivery. `TestProcessFollowInstanceActorSetsPending` and `TestProcessFollowInstanceFollowStored` pass. Person-actor auto-accept path unchanged (`TestProcessFollowPersonActorAutoAccepts` passes). |
| 3 | When admin sets pending instance follow to `accepted`, an Accept{Follow} is delivered to the remote instance's inbox (exactly once) | VERIFIED | `CreateAcceptFollowActivity` (follow.go:169-227) reloads original Follow from `activitypub_activities`, wraps in `pub.AcceptNew`, posts to followerActor's inbox. `InstanceFollowUpdateHandler` (hooks/follow.go:136-158) dispatches to this on status transition to "accepted" via `instanceFollowAction`. **CR-01 fix confirmed:** `CreateFollowHandler` has `isInstanceFollow` guard at line 16 — user-level Request hook no-ops for instance follows, leaving the AfterSuccess handler as single delivery path. |
| 4 | When admin sets pending instance follow to `rejected`, a Reject{Follow} is delivered to the remote instance's inbox (exactly once) | VERIFIED | `CreateRejectFollowActivity` (follow.go:232-289) uses `pub.RejectNew`. `InstanceFollowUpdateHandler` dispatches on status transition to "rejected". Migration `1782290001_add_rejected_to_follows_status.go` adds "rejected" to follows.status select values (idempotent up, tolerant down). Test harness includes "rejected" in status select values. |
| 5 | When admin deletes an instance follow record, an Undo{Follow} is delivered to the remote instance's inbox (exactly once); user-level follows are unaffected | VERIFIED | `InstanceFollowDeleteHandler` (hooks/follow.go:163-176) calls `federation.CreateUnfollowActivity`. **CR-02 fix confirmed:** `DeleteFollowHandler` has `isInstanceFollow` guard at line 30 — no-ops for instance follows. **CR-03 fix confirmed:** `InstanceFollowCreateHandler` (line 97) uses `isOutboundInstanceFollow` (checks only follower IRI), not `isInstanceFollow` — inbound follow regression covered by `TestIsOutboundInstanceFollowFalseWhenInstanceIsFollowee`. 9/9 hooks tests pass. |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/migrations/1782290001_add_rejected_to_follows_status.go` | Migration adding "rejected" to follows.status select field | VERIFIED | Exists; references collection id `8obn1ukumze565i`; appends "rejected" idempotently in up-function; removes in down-function. `go build ./...` passes. |
| `db/federation/instance.go` | InstanceInboxHandler — HTTP-signed inbox routing Follow/Accept/Undo | VERIFIED | `func InstanceInboxHandler(e *core.RequestEvent) error` at line 117. Calls `util.VerifySignature`, returns 401 on failure. **CR-05 fix confirmed:** no `procErr` variable, per-case `e.BadRequestError` returns, `default` case returns `e.BadRequestError("Unsupported activity type", nil)`, success returns `e.JSON(http.StatusOK, map[string]bool{"success": true})`. |
| `db/federation/follow.go` | Actor-type branch + CreateAcceptFollowActivity + CreateRejectFollowActivity + panic-safe ProcessAcceptActivity | VERIFIED | Branch at line 91; `CreateAcceptFollowActivity` (line 169) and `CreateRejectFollowActivity` (line 232) present. **CR-04 fix confirmed:** `ProcessAcceptActivity` (line 295) uses comma-ok: `followActivity, ok := activity.Object.(*pub.Activity)` with early error return; `TestProcessAcceptActivityIRIOnlyObjectReturnsError` passes without panic. |
| `db/hooks/follow.go` | CR-01/CR-02 guards + isOutboundInstanceFollow + InstanceFollowCreate/Update/Delete handlers | VERIFIED | `isInstanceFollow` guard at line 16 in `CreateFollowHandler`, line 30 in `DeleteFollowHandler`. `isOutboundInstanceFollow` (line 44) loads only follower actor. `InstanceFollowCreateHandler` opens with `if !isOutboundInstanceFollow(...)` (line 97). All three AfterSuccess handlers present. `isInstanceFollow` retains EITHER-direction semantics for Update/Delete. |
| `db/main.go` | All three AfterSuccess hook registrations + existing Request hooks preserved | VERIFIED | Lines 130-132: `OnRecordAfterCreateSuccess`, `OnRecordAfterUpdateSuccess`, `OnRecordAfterDeleteSuccess` for "follows". Lines 127-128: original `OnRecordCreateRequest` and `OnRecordDeleteRequest` retained. Route at line 190. |
| `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts` | SvelteKit proxy setting X-Forwarded-Path | VERIFIED | Exports `POST`, sets `originalHeaders['X-Forwarded-Path'] = event.url.pathname` (line 44), forwards to `/activitypub/instance/inbox` via `event.locals.pb.send`. |
| `db/federation/instance_inbox_test.go` | Tests for ProcessFollowActivity instance/person branches | VERIFIED | 3 tests pass: `TestProcessFollowInstanceActorSetsPending`, `TestProcessFollowInstanceFollowStored`, `TestProcessFollowPersonActorAutoAccepts`. |
| `db/federation/follow_accept_test.go` | CR-04 panic-safety regression test | VERIFIED | `TestProcessAcceptActivityIRIOnlyObjectReturnsError` — passes nil app and IRI-only Accept object; asserts non-nil error with message containing "ProcessAcceptActivity: object is not *pub.Activity"; no panic. |
| `db/hooks/follow_test.go` | isInstanceFollow, instanceFollowAction, isOutboundInstanceFollow tests | VERIFIED | 9/9 tests pass: 3 for `isInstanceFollow` (via followee, via follower, false), 3 for `instanceFollowAction` (no-op, accept, reject), 3 new for `isOutboundInstanceFollow` including the CR-03 inbound regression case. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `web/.../instance/inbox/+server.ts` | PocketBase route `/activitypub/instance/inbox` | `event.locals.pb.send` with `X-Forwarded-Path` | WIRED | `X-Forwarded-Path` set at line 44; `pb.send` at line 46 |
| `db/main.go` | `federation.InstanceInboxHandler` | `se.Router.POST` registration | WIRED | Line 190 |
| `db/main.go` | `hooks.InstanceFollowCreateHandler` | `OnRecordAfterCreateSuccess("follows").BindFunc` | WIRED | Line 130 |
| `db/main.go` | `hooks.InstanceFollowUpdateHandler` | `OnRecordAfterUpdateSuccess("follows").BindFunc` | WIRED | Line 131 |
| `db/main.go` | `hooks.InstanceFollowDeleteHandler` | `OnRecordAfterDeleteSuccess("follows").BindFunc` | WIRED | Line 132 |
| `db/hooks/follow.go CreateFollowHandler` | `isInstanceFollow` guard (CR-01) | early return before `CreateFollowActivity` | WIRED | Line 16: `if isInstanceFollow(e.App, e.Record) { return e.Next() }` |
| `db/hooks/follow.go DeleteFollowHandler` | `isInstanceFollow` guard (CR-02) | early return before `CreateUnfollowActivity` | WIRED | Line 30: `if isInstanceFollow(e.App, e.Record) { return e.Next() }` |
| `db/hooks/follow.go InstanceFollowCreateHandler` | directional guard (CR-03) | `isOutboundInstanceFollow` — follower IRI only | WIRED | Line 97: `if !isOutboundInstanceFollow(e.App, e.Record)` |
| `db/federation/follow.go ProcessAcceptActivity` | comma-ok type assertion (CR-04) | guard before any DB call | WIRED | Line 295: `followActivity, ok := activity.Object.(*pub.Activity)` with early error return |
| `db/federation/instance.go InstanceInboxHandler` | per-case error propagation (CR-05) | `e.BadRequestError` per case + default | WIRED | Lines 174-187: each case returns `e.BadRequestError` on failure; `default` at line 186 |
| `db/hooks/follow.go InstanceFollowUpdateHandler` | `federation.CreateAcceptFollowActivity` | status-change dispatch via `instanceFollowAction` | WIRED | Lines 146-151 |
| `db/hooks/follow.go InstanceFollowUpdateHandler` | `federation.CreateRejectFollowActivity` | status-change dispatch via `instanceFollowAction` | WIRED | Lines 152-155 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` compiles without error | `cd db && go build ./...` | exit 0 | PASS |
| `go vet ./hooks/ ./federation/ ./migrations/ .` | vet check | exit 0 | PASS |
| ProcessFollowActivity instance branch tests | `go test ./federation/ -run TestProcessFollow -count=1` | 3/3 PASS | PASS |
| CR-04 panic-safety test | `go test ./federation/ -run TestProcessAccept -count=1` | 1/1 PASS | PASS |
| Full federation test suite | `go test ./federation/ -count=1` | 7/7 PASS | PASS |
| CR-03 directional tests | `go test ./hooks/ -count=1` | 9/9 PASS (includes 3 new isOutboundInstanceFollow tests) | PASS |
| Full hooks test suite | `go test ./hooks/ -count=1` | 9/9 PASS | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|---------|
| INST-03 | 02-02, 02-04 | POST inbox accepts HTTP-signed activities | SATISFIED | `InstanceInboxHandler` verifies signature (401 on failure); route registered; CR-05 fix: honest 400 on processing failure; CR-04 fix: no panic on IRI-only Accept |
| FLCL-01 | 02-03, 02-04 | Admin-initiated outgoing Follow (exactly once) | SATISFIED | `InstanceFollowCreateHandler` with `isOutboundInstanceFollow` guard (CR-03 fix); `CreateFollowHandler` skips for instance follows (CR-01 fix) — single delivery path confirmed |
| FLCL-02 | 02-02 | Incoming Application-type Follow → pending | SATISFIED | `ProcessFollowActivity` instance branch sets status="pending", persists Follow to `activitypub_activities`; 3 tests verify behavior |
| FLCL-03 | 02-03 | Accept{Follow} delivered on admin approval | SATISFIED | `CreateAcceptFollowActivity` reloads original Follow, wraps in `pub.AcceptNew`, posts to remote inbox; `InstanceFollowUpdateHandler` dispatches on "accepted" |
| FLCL-04 | 02-01, 02-03 | Reject{Follow} delivered on admin rejection | SATISFIED | Migration adds "rejected" to status field; `CreateRejectFollowActivity` uses `pub.RejectNew`; wired in update handler |
| FLCL-05 | 02-03, 02-04 | Undo{Follow} sent on admin disconnect (exactly once) | SATISFIED | `InstanceFollowDeleteHandler` calls `CreateUnfollowActivity`; `DeleteFollowHandler` skips for instance follows (CR-02 fix) — single delivery path confirmed |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts` | 53-55 | `if (success === false)` — PocketBase SDK throws on non-2xx, never returns false | WARNING | Dead code; non-blocking; does not affect functionality |
| `db/federation/follow.go` | 316-319 | Commented-out `SyncOutbox` block | INFO | Dead code; no ticket reference; non-blocking |

No BLOCKER anti-patterns remain. All five CR blockers from the previous verification are resolved.

### Human Verification Required

#### 1. Reverse Proxy Routing for Instance Inbox

**Test:** Confirm that in the deployment environment, `POST /api/v1/activitypub/instance/inbox` is routed to the SvelteKit app the same way `POST /api/v1/activitypub/user/{handle}/inbox` is routed. Specifically verify that the SvelteKit adapter-node/nginx/caddy config does NOT strip or rewrite `X-Forwarded-Path` before it reaches the Go handler.
**Expected:** Remote instances can POST HTTP-signed activities to the inbox IRI; the Go handler reconstructs the signed path from the header correctly and verifies the HTTP signature.
**Why human:** Proxy configuration is outside the codebase (docker-compose / nginx / caddy / deployment manifests); cannot be verified by code inspection or grep. This item was deferred from Plan 02-02's `<human-check>` block and remains unverifiable programmatically.

## Gaps Summary

No actionable gaps remain. All five CR blockers from the initial verification are confirmed closed in the codebase:

- **CR-01 (closed):** `CreateFollowHandler` at line 16 of `db/hooks/follow.go` has `if isInstanceFollow(e.App, e.Record) { return e.Next() }` before the `federation.CreateFollowActivity` call.
- **CR-02 (closed):** `DeleteFollowHandler` at line 30 has the matching guard before `federation.CreateUnfollowActivity`.
- **CR-03 (closed):** `InstanceFollowCreateHandler` uses `isOutboundInstanceFollow` (line 97), which loads only the follower actor and compares its IRI to the instance IRI — the inbound-follow case (instance is followee) correctly returns false. `TestIsOutboundInstanceFollowFalseWhenInstanceIsFollowee` is a passing regression test.
- **CR-04 (closed):** `ProcessAcceptActivity` at line 295 uses comma-ok assertion `followActivity, ok := activity.Object.(*pub.Activity)` and returns a descriptive error when `ok` is false. `TestProcessAcceptActivityIRIOnlyObjectReturnsError` passes without panic.
- **CR-05 (closed):** `InstanceInboxHandler` no longer uses `procErr`; each case returns `e.BadRequestError` on error; a `default` case returns `e.BadRequestError("Unsupported activity type", nil)`; success returns `{"success": true}`.

The single remaining `human_needed` item (reverse proxy routing for the instance inbox) is a deployment verification that cannot be resolved by code inspection. It was present in the previous verification and is not caused by any code defect.

---

_Verified: 2026-06-25T16:00:00Z_
_Verifier: Claude (gsd-verifier)_
