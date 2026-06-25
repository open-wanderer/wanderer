---
phase: 02-follow-lifecycle
verified: 2026-06-25T00:00:00Z
status: gaps_found
score: 3/5 must-haves verified
overrides_applied: 0
gaps:
  - truth: "When an admin creates a follows record where the follower is the local instance actor, a Follow activity is delivered to the remote instance's inbox — and ONLY once"
    status: failed
    reason: "CR-01: CreateFollowHandler (OnRecordCreateRequest) has no isInstanceFollow guard and calls federation.CreateFollowActivity unconditionally. InstanceFollowCreateHandler (OnRecordAfterCreateSuccess) also calls federation.CreateFollowActivity for instance follows. Both fire when an admin creates an instance follow via the API, causing two Follow activities delivered to the remote inbox."
    artifacts:
      - path: "db/hooks/follow.go"
        issue: "CreateFollowHandler (lines 12-19) calls federation.CreateFollowActivity with no isInstanceFollow guard. DeleteFollowHandler (lines 21-27) calls federation.CreateUnfollowActivity with no isInstanceFollow guard (CR-02 mirror)."
    missing:
      - "Add isInstanceFollow guard to CreateFollowHandler: if isInstanceFollow(e.App, e.Record) { return nil } before calling federation.CreateFollowActivity"
      - "Add isInstanceFollow guard to DeleteFollowHandler: if isInstanceFollow(e.App, e.Record) { return e.Next() } before calling federation.CreateUnfollowActivity"

  - truth: "InstanceFollowCreateHandler fires only for admin-initiated outbound follows (where the local instance actor is the follower), not for inbound follows saved by ProcessFollowActivity"
    status: failed
    reason: "CR-03: isInstanceFollow returns true when EITHER follower OR followee is the instance actor. When ProcessFollowActivity saves an inbound follow (remote actor = follower, local instance actor = followee), OnRecordAfterCreateSuccess fires and isInstanceFollow returns true. InstanceFollowCreateHandler then calls federation.CreateFollowActivity with follower=remote actor, attempting to sign and post as the remote actor (which has no private_key locally). No directional guard (follower == instance IRI) exists in InstanceFollowCreateHandler."
    artifacts:
      - path: "db/hooks/follow.go"
        issue: "InstanceFollowCreateHandler (lines 68-103) calls isInstanceFollow which matches on followee=instance actor, not just follower=instance actor. No check that followerActor.GetString('iri') == instanceIRI before proceeding."
    missing:
      - "Add directional guard in InstanceFollowCreateHandler: load followerActor and verify followerActor.GetString('iri') == instanceIRI before proceeding to CreateFollowActivity"

  - truth: "ProcessAcceptActivity safely handles an Accept whose object field is an IRI string (not a *pub.Activity struct) without panicking"
    status: failed
    reason: "CR-04: Line 293 of db/federation/follow.go performs an unguarded type assertion: followActivity := activity.Object.(*pub.Activity). If the remote sends an Accept with an IRI-only object (common in real-world ActivityPub implementations), this panics. ProcessAcceptActivity is reachable via the instance inbox registered in this phase."
    artifacts:
      - path: "db/federation/follow.go"
        issue: "Line 293: followActivity := activity.Object.(*pub.Activity) — unguarded type assertion; panics on IRI-only Accept object"
    missing:
      - "Replace unguarded assertion with comma-ok form: followActivity, ok := activity.Object.(*pub.Activity); if !ok { return fmt.Errorf(\"ProcessAcceptActivity: object is not *pub.Activity, got %T\", activity.Object) }"

  - truth: "InstanceInboxHandler returns a non-2xx error status to the remote caller when activity processing fails, and rejects unrecognized activity types"
    status: failed
    reason: "CR-05: InstanceInboxHandler always calls e.JSON(http.StatusOK, procErr). When procErr is non-nil, Go errors serialize to null in JSON — the remote caller receives HTTP 200 null and cannot detect the failure. Additionally there is no default case in the switch, so unrecognized activity types are silently accepted with 200 null."
    artifacts:
      - path: "db/federation/instance.go"
        issue: "Lines 169-179: switch with no default case; return e.JSON(http.StatusOK, procErr) returns 200 even on processing error"
    missing:
      - "Replace procErr pattern with per-case error return: if err := ProcessFollowActivity(...); err != nil { return e.BadRequestError(...) }"
      - "Add default case: return e.BadRequestError('Unsupported activity type', nil)"
human_verification:
  - test: "Confirm the SvelteKit proxy route POST /api/v1/activitypub/instance/inbox is served by the deployment reverse proxy identically to how POST /api/v1/activitypub/user/{handle}/inbox is routed"
    expected: "The new SvelteKit route receives HTTP-signed POST requests from remote instances; X-Forwarded-Path is set internally by the proxy (event.url.pathname) and forwarded to Go"
    why_human: "Cannot verify nginx/caddy/adapter-node proxy config programmatically from the codebase alone; requires live infra inspection or staging test"
---

# Phase 2: Follow Lifecycle Verification Report

**Phase Goal:** An admin can connect two Wanderer instances by initiating or receiving a Follow, approving or rejecting it, and disconnecting later — with correct AP activities delivered at each step
**Verified:** 2026-06-25T00:00:00Z
**Status:** gaps_found (4 blockers from code review confirmed in codebase)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (from ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | POST `{ORIGIN}/api/v1/activitypub/instance/inbox` accepts HTTP-signed activities and routes them to the activity processor | VERIFIED | `InstanceInboxHandler` in db/federation/instance.go calls `util.VerifySignature`, returns 401 on failure, dispatches to ProcessFollowActivity/ProcessAcceptActivity/ProcessUndoActivity. Route registered at main.go:190. SvelteKit proxy at web/src/routes/api/v1/activitypub/instance/inbox/+server.ts sets X-Forwarded-Path. |
| 2 | Incoming Follow from an `Application`-type actor is stored as `pending` in `follows` and NOT auto-accepted | VERIFIED | ProcessFollowActivity (follow.go:91-113) branches on `object.GetString("actor_type") == "instance" && object.GetBool("is_local")`, sets status="pending", persists the incoming Follow to activitypub_activities, returns nil before Accept delivery. Three tests pass: TestProcessFollowInstanceActorSetsPending, TestProcessFollowInstanceFollowStored, TestProcessFollowPersonActorAutoAccepts. |
| 3 | When admin sets pending instance follow to `accepted`, Accept{Follow} delivered to remote inbox | VERIFIED | CreateAcceptFollowActivity in follow.go:169-227 reloads original Follow from activitypub_activities, wraps in pub.AcceptNew, posts to followerActor's inbox. InstanceFollowUpdateHandler (hooks/follow.go:109-132) dispatches to this function on status transition to "accepted". Registered at main.go:131. |
| 4 | When admin sets pending instance follow to `rejected`, Reject{Follow} delivered to remote inbox | PARTIAL — delivery helper wired, but double-delivery bugs (CR-01/CR-02) affect create/delete paths and CR-03/CR-04/CR-05 are open blockers | CreateRejectFollowActivity exists (follow.go:229-289) using pub.RejectNew; the reject path itself is wired. However the broader lifecycle correctness is broken by the four CR items. |
| 5 | Admin-initiated Undo removes peer from follows collection and sends Undo{Follow} activity | FAILED — double-delivery | InstanceFollowDeleteHandler (hooks/follow.go:134-148) calls federation.CreateUnfollowActivity when isInstanceFollow is true. BUT DeleteFollowHandler (OnRecordDeleteRequest, lines 21-27) also calls federation.CreateUnfollowActivity with NO isInstanceFollow guard. Both fire when delete is via the API, producing two Undo{Follow} activities. |

**Score:** 3/5 truths fully verified (SC-1, SC-2, SC-3 VERIFIED; SC-4 PARTIAL; SC-5 FAILED)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/migrations/1782290001_add_rejected_to_follows_status.go` | Migration adding "rejected" to follows.status | VERIFIED | Exists, references collection id 8obn1ukumze565i, appends "rejected" idempotently in up-function, removes in down-function. `go build ./...` passes. |
| `db/federation/instance.go` | InstanceInboxHandler — signed inbox routing Follow/Accept/Undo | VERIFIED | `func InstanceInboxHandler(e *core.RequestEvent) error` exists at line 117. Calls util.VerifySignature, dispatches Follow/Accept/Undo, returns e.UnauthorizedError on signature failure. |
| `db/federation/follow.go` | Actor-type branch + CreateAcceptFollowActivity + CreateRejectFollowActivity | VERIFIED (structure) / PARTIAL (correctness) | Branch at line 91 stores instance follows as pending. CreateAcceptFollowActivity (line 169) and CreateRejectFollowActivity (line 232) exist and use pub.AcceptNew/pub.RejectNew. ProcessAcceptActivity (line 291) has unguarded type assertion (CR-04). |
| `db/hooks/follow.go` | InstanceFollowCreate/Update/Delete handlers + isInstanceFollow | VERIFIED (structure) / FAILED (correctness) | All three handlers and isInstanceFollow exist. BUT CreateFollowHandler and DeleteFollowHandler have no isInstanceFollow guard (CR-01, CR-02). InstanceFollowCreateHandler has no directional check (CR-03). |
| `db/main.go` | All three AfterSuccess hook registrations + existing Request hooks preserved | VERIFIED | Lines 130-132 register OnRecordAfterCreateSuccess, OnRecordAfterUpdateSuccess, OnRecordAfterDeleteSuccess for follows. Lines 127-128 retain original OnRecordCreateRequest and OnRecordDeleteRequest. Route se.Router.POST("/activitypub/instance/inbox", ...) at line 190. |
| `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts` | SvelteKit proxy setting X-Forwarded-Path | VERIFIED | File exists, exports POST, sets originalHeaders['X-Forwarded-Path'] = event.url.pathname, forwards to /activitypub/instance/inbox via event.locals.pb.send. Dead code `if (success === false)` present (WR-03, non-blocking). |
| `db/federation/instance_inbox_test.go` | Tests for ProcessFollowActivity instance branch | VERIFIED | 3 tests pass: TestProcessFollowInstanceActorSetsPending, TestProcessFollowInstanceFollowStored, TestProcessFollowPersonActorAutoAccepts. |
| `db/hooks/follow_test.go` | Tests for isInstanceFollow and instanceFollowAction | VERIFIED | 6 tests pass (3 for isInstanceFollow, 3 for instanceFollowAction). Note: the test suite named these differently (TestIsInstanceFollowTrueViaFollowee etc.) — only 3 run under `-run TestInstanceFollow`, but all 6 exist and pass. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| web/src/routes/api/v1/activitypub/instance/inbox/+server.ts | PocketBase route /activitypub/instance/inbox | event.locals.pb.send with X-Forwarded-Path | WIRED | X-Forwarded-Path set at line 44; pb.send at line 46 |
| db/main.go | federation.InstanceInboxHandler | se.Router.POST registration | WIRED | Line 190 |
| db/main.go | hooks.InstanceFollowCreateHandler | OnRecordAfterCreateSuccess("follows").BindFunc | WIRED | Line 130 |
| db/main.go | hooks.InstanceFollowUpdateHandler | OnRecordAfterUpdateSuccess("follows").BindFunc | WIRED | Line 131 |
| db/main.go | hooks.InstanceFollowDeleteHandler | OnRecordAfterDeleteSuccess("follows").BindFunc | WIRED | Line 132 |
| db/hooks/follow.go InstanceFollowUpdateHandler | federation.CreateAcceptFollowActivity | status-change dispatch using Record.Original() | WIRED | Lines 118-122 |
| db/hooks/follow.go InstanceFollowUpdateHandler | federation.CreateRejectFollowActivity | status-change dispatch using Record.Original() | WIRED | Lines 123-126 |
| db/hooks/follow.go CreateFollowHandler | isInstanceFollow guard | MISSING guard | NOT_WIRED — BLOCKER | CreateFollowHandler calls CreateFollowActivity unconditionally (CR-01) |
| db/hooks/follow.go DeleteFollowHandler | isInstanceFollow guard | MISSING guard | NOT_WIRED — BLOCKER | DeleteFollowHandler calls CreateUnfollowActivity unconditionally (CR-02) |
| db/hooks/follow.go InstanceFollowCreateHandler | directional check (follower == instance actor) | MISSING guard | NOT_WIRED — BLOCKER | Fires on inbound follows where followee is instance actor (CR-03) |
| db/federation/follow.go ProcessAcceptActivity | safe type assertion | unguarded assertion | PANICS — BLOCKER | Line 293: activity.Object.(*pub.Activity) panics on IRI-only object (CR-04) |
| db/federation/instance.go InstanceInboxHandler | error propagation to caller | procErr always returns 200 | BROKEN — BLOCKER | No default case; e.JSON(http.StatusOK, procErr) masks errors (CR-05) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|---------|
| INST-03 | 02-02 | POST inbox accepts HTTP-signed activities | SATISFIED | InstanceInboxHandler verifies signature; route registered |
| FLCL-01 | 02-03 | Admin-initiated outgoing Follow | BLOCKED — CR-01/CR-03 | Double-delivery when created via API (CR-01); fires on inbound follows (CR-03) |
| FLCL-02 | 02-02 | Incoming Application-type Follow → pending | SATISFIED | ProcessFollowActivity branch verified; 3 tests pass |
| FLCL-03 | 02-03 | Accept{Follow} delivered on admin approval | SATISFIED — delivery helper correct | CreateAcceptFollowActivity wired; update handler dispatches correctly |
| FLCL-04 | 02-01, 02-03 | Reject{Follow} delivered on admin rejection | SATISFIED — delivery helper correct | Migration adds "rejected"; CreateRejectFollowActivity wired |
| FLCL-05 | 02-03 | Undo{Follow} sent on admin disconnect | BLOCKED — CR-02 | Double-delivery when deleted via API (CR-02) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| db/hooks/follow.go | 12-19 | CreateFollowHandler calls federation.CreateFollowActivity with no isInstanceFollow guard | BLOCKER | Double Follow delivery for admin-created instance follows via API (CR-01) |
| db/hooks/follow.go | 21-27 | DeleteFollowHandler calls federation.CreateUnfollowActivity with no isInstanceFollow guard | BLOCKER | Double Undo delivery for admin-deleted instance follows via API (CR-02) |
| db/hooks/follow.go | 68-103 | InstanceFollowCreateHandler uses isInstanceFollow which returns true on EITHER follower or followee; no directional guard | BLOCKER | Fires on inbound follows saved by ProcessFollowActivity; attempts to sign Follow as remote actor (CR-03) |
| db/federation/follow.go | 293 | activity.Object.(*pub.Activity) unguarded type assertion | BLOCKER | Panic on IRI-only Accept object; takes down request goroutine (CR-04) |
| db/federation/instance.go | 169-179 | switch with no default; return e.JSON(http.StatusOK, procErr) on any processing error | BLOCKER | Remote caller cannot detect failures; unrecognized activity types silently accepted (CR-05) |
| web/src/routes/api/v1/activitypub/instance/inbox/+server.ts | 53-55 | if (success === false) dead code | WARNING | PocketBase SDK throws on non-2xx, never returns false (WR-03) |
| db/hooks/follow.go | 34 | instanceIRI computed with os.Getenv("ORIGIN") — no empty guard | WARNING | Misfires if ORIGIN not set (WR-02) |
| db/federation/follow.go | 310-314 | Commented-out SyncOutbox block | INFO | Dead code, no ticket reference (IN-02) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` compiles without error | `cd db && go build ./...` | exit 0 | PASS |
| `go vet ./...` passes | `cd db && go vet ./...` | exit 0 | PASS |
| ProcessFollowActivity instance branch tests | `go test ./federation/ -run TestProcessFollow -count=1` | 3/3 PASS | PASS |
| hooks isInstanceFollow and instanceFollowAction tests | `go test ./hooks/ -run TestInstanceFollow -count=1` | 3/3 PASS (note: 3 more tests exist under different names) | PASS |
| Full federation test suite | `go test ./federation/ -count=1` | 6/6 PASS | PASS |

### Human Verification Required

#### 1. Reverse Proxy Routing for Instance Inbox

**Test:** Confirm that in the deployment environment, `POST /api/v1/activitypub/instance/inbox` is routed to the SvelteKit app the same way `POST /api/v1/activitypub/user/{handle}/inbox` is routed. Specifically verify that the SvelteKit adapter-node/nginx/caddy config does NOT strip or rewrite `X-Forwarded-Path` before it reaches the Go handler.
**Expected:** Remote instances can POST HTTP-signed activities to the inbox IRI; the Go handler reconstructs the signed path correctly and verifies the signature.
**Why human:** Proxy config is outside the codebase (docker-compose / nginx / caddy / deployment config); cannot be verified by grep.

## Gaps Summary

Five blockers were identified by the code reviewer (02-REVIEW.md) and confirmed in the actual codebase:

**CR-01 and CR-02 (double-delivery):** `CreateFollowHandler` and `DeleteFollowHandler` — the pre-existing user-level hooks — have no isInstanceFollow guard. When an admin creates or deletes an instance follow via the PocketBase API (not the admin UI), both the Request-family hook AND the AfterSuccess-family hook fire, delivering two Follow/Undo activities to the remote inbox. Root cause: the new AfterSuccess handlers were added without disabling the pre-existing Request handlers for the instance-follow case. Fix is a 2-line isInstanceFollow guard at the top of each existing handler.

**CR-03 (wrong-direction fire):** `isInstanceFollow` correctly returns true when either the follower OR the followee is the local instance actor. This is the right design for the update/delete handlers. But `InstanceFollowCreateHandler` must only deliver an outgoing Follow when the local instance actor is the *follower* (admin-initiated outbound follow). When ProcessFollowActivity saves an inbound follow (remote = follower, local instance = followee), OnRecordAfterCreateSuccess fires and InstanceFollowCreateHandler mistakenly calls CreateFollowActivity with the remote actor as follower — attempting to decrypt a private key that does not exist locally. Fix is a directional check on the follower IRI.

**CR-04 (panic):** `ProcessAcceptActivity` performs an unguarded type assertion `activity.Object.(*pub.Activity)`. Many real-world ActivityPub implementations send Accept with IRI-only object fields. This panics and is now reachable via the instance inbox route. Fix is the standard comma-ok pattern.

**CR-05 (error masking):** `InstanceInboxHandler` returns `e.JSON(http.StatusOK, procErr)` regardless of whether processing succeeded. Go errors serialize to `null` in JSON, so the remote caller always sees `200 null` even when processing fails. There is also no `default` case to reject unknown activity types. Fix is per-case error propagation with a default 400.

These four gaps are all in code committed by this phase. They are behavioral correctness failures that prevent the lifecycle from operating correctly under realistic conditions. The structural wiring (routes, handlers, migrations) is all present and builds/tests pass — but the logic has four correctness defects that must be fixed before the phase goal is achieved.

---

_Verified: 2026-06-25T00:00:00Z_
_Verifier: Claude (gsd-verifier)_
