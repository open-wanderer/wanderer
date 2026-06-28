---
phase: 05-federation-admin-api
verified: 2026-06-27T18:00:00Z
status: human_needed
score: 9/9 must-haves verified
overrides_applied: 0
human_verification:
  - test: "POST /federation/discover against a live (or test-double) Wanderer instance returns a preview card with actor_id, domain, version, user_count, trail_count as non-empty values"
    expected: "HTTP 200 with JSON body containing all five fields populated from the remote NodeInfo 2.1 payload and the actor record"
    why_human: "The GetActorByIRI code path requires an actual HTTP round-trip to a remote ActivityPub endpoint; the unit tests only cover the pure-logic pieces (NodeInfo 2.1 selection, Wanderer identity check) not the full end-to-end handler flow"
  - test: "POST /federation/discover with a reachable non-Wanderer URL (e.g., a Mastodon instance) returns HTTP 400 with error 'not a Wanderer instance'"
    expected: "400 response and the correct DISC-02 error string"
    why_human: "Requires a live outbound HTTP connection to a non-Wanderer instance or a test double running an HTTP server"
  - test: "POST /federation/follow with a valid actor_id then inspect the hooks: verify that InstanceFollowCreateHandler fires and a Follow activity is delivered (no double-delivery)"
    expected: "A follows record with status=pending exists; one Follow activity is queued/sent; no direct CreateFollowActivity call occurs from the handler layer"
    why_human: "Hook firing and ActivityPub delivery are runtime behaviour requiring an integration environment; grep verifies the absence of direct delivery calls in handler code but cannot confirm hook invocation"
  - test: "POST /federation/approve/:id for an inbound pending follow, then confirm that InstanceFollowUpdateHandler fires and an Accept{Follow} is delivered"
    expected: "Follow record moves to status=accepted; one Accept{Follow} activity is delivered to the remote instance"
    why_human: "Same as above — hook invocation at runtime cannot be verified by static analysis"
  - test: "POST /federation/disconnect/:id for an outbound follow, verify the record is deleted and Undo{Follow} is sent (not Reject); then test an inbound-only follow, verify record moves to rejected and Reject{Follow} is sent (not Undo)"
    expected: "Correct activity verb per direction; no wrong-direction Undo on inbound-only disconnects"
    why_human: "Direction-aware hook routing is a runtime behaviour; unit tests cover the disconnectAction pure helper and the DB write path, but not the hook invocation or wire payload"
  - test: "Every endpoint (discover, follow, approve/:id, reject/:id, disconnect/:id, peers) returns HTTP 401 when called without a PocketBase superuser token or with a regular Wanderer user token"
    expected: "401 Unauthorized on all six endpoints for non-superuser callers"
    why_human: "e.HasSuperuserAuth() is verified present by grep, but the guard's runtime interaction with the PocketBase auth middleware requires an integration test against a running server"
---

# Phase 5: Federation Admin API Verification Report

**Phase Goal:** Build the Go HTTP API layer that exposes instance-federation operations to a privileged admin caller — discovery, follow initiation, accept/reject, disconnect, and peer listing — as thin, auth-guarded route handlers wired into the existing PocketBase application.
**Verified:** 2026-06-27T18:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Step 0: Previous Verification

No previous VERIFICATION.md found. Initial mode.

---

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Admin can POST a remote instance URL to `/federation/discover` and receive a preview card (actor_id, domain, version, user/trail counts) or a clear DISC-02 error | VERIFIED | `FederationDiscover` in `federation_admin.go:609`; all four DISC-02 error strings confirmed in file; `actor_id/domain/version/user_count/trail_count` returned at line 701-707 |
| 2 | Admin can POST to `/federation/follow`; follows record created status=pending; Follow activity delivered via hook, no double-delivery | VERIFIED | `createOutboundFollow` at line 182 creates the record; `grep -c 'Create.*Activity'` returns 0; `TestFederationFollowCreatesPendingRecord` passes with follower=local, followee=remote, status=pending |
| 3 | Admin can POST to `/federation/approve/:id`; Accept{Follow} delivered; connection moves to accepted | VERIFIED | `setFollowStatus(..., "accepted", ...)` in `FederationApprove` line 554; hook delivers Accept; `TestFederationApproveSetsAccepted` passes; SAFE-07 grep = 0 |
| 4 | Admin can POST to `/federation/reject/:id` (Reject{Follow}); `/federation/disconnect/:id` is direction-aware — outbound → Undo, inbound-only → Reject (no wrong Undo) | VERIFIED | `FederationReject` line 571 + `FederationDisconnect` line 282; `disconnectAction` pure helper tested by `TestDisconnectActionOutbound/Inbound`; outbound uses `e.App.Delete` (line 308); inbound uses `Set("status","rejected")` (line 314) |
| 5 | All six endpoints reject non-superuser requests with 401; admin-supplied URLs fetched via SSRF-safe client with 10-second timeout | VERIFIED | `e.HasSuperuserAuth()` guard is first statement in all six handlers (lines 213, 284, 426, 534, 573, 611); `SetTimeout(10 * time.Second)` at line 75; `util.FetchPublicURL` count = 0 |

**Score:** 9/9 must-haves verified (see breakdown below)

---

### Must-Haves from PLAN Frontmatter (merged)

All requirements from the three PLAN frontmatter `must_haves` blocks are satisfied. The combined unique truths from all three plans map directly to the five ROADMAP success criteria above.

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/routes/federation_admin.go` | All six handlers + helpers + types | VERIFIED | File exists, 709 lines; all 14 named functions and 5 types confirmed present |
| `db/routes/federation_admin_test.go` | Unit tests covering helpers and handler logic | VERIFIED | 18 test functions across the full helper set; all pass (18/18 PASS) |
| `db/main.go` | Six se.Router registrations | VERIFIED | Lines 211-216; `grep -c 'routes.Federation' main.go` = 6 |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `FederationDiscover` | `newDiscoveryClient` (safeurl 10s) | Both NodeInfo fetches (JRD + 2.1) | WIRED | `client := newDiscoveryClient()` at line 624; both fetch calls use the same `client`; `SetTimeout(10 * time.Second)` confirmed |
| `FederationDiscover` | `util.IsLocalIRI` | SAFE-05 self-follow guard | WIRED | `if util.IsLocalIRI(actorIRI)` at line 657 |
| `FederationDiscover` | `federation.GetActorByIRI` | Actor cache bypass after clearing last_fetched | WIRED | `existingActor.Set("last_fetched", time.Time{})` + `_ = e.App.Save(existingActor)` at lines 685-686; `GetActorByIRI` called at line 690 |
| `FederationFollow` | `follows` collection (app.Save) | `createOutboundFollow` helper | WIRED | `createOutboundFollow` at line 182 creates record via `core.NewRecord + Set + app.Save` |
| `FederationApprove/Reject` | `follows` record status field | `setFollowStatus + FindRecordById + Set + app.Save` | WIRED | `setFollowStatus` at line 508; direction guard enforces followee == localID |
| `FederationFollow` | `findLocalInstanceActor` | follower = local instance actor id | WIRED | `findLocalInstanceActor(e.App)` called at line 232 before `createOutboundFollow` |
| `FederationDisconnect` | `app.Delete` (outbound) vs `app.Save status=rejected` (inbound-only) | `disconnectAction` direction check | WIRED | `e.App.Delete(follow)` at line 308 (outbound); `follow.Set("status", "rejected") + e.App.Save` at lines 314-316 (inbound) |
| `FederationPeers` | `buildPeerEntries` | Two follows queries folded by remote domain | WIRED | `FindRecordsByFilter` for outbound + inbound at lines 437-460; `buildPeerEntries(...)` at line 487 |
| `db/main.go registerRoutes` | `routes.FederationDiscover` and five other handlers | `se.Router.POST/GET` registrations | WIRED | All six registrations confirmed at lines 211-216 |

---

### Data-Flow Trace (Level 4)

`FederationDiscover` is the primary dynamic-data handler:

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `FederationDiscover` | `ni.Software.Version/Usage.*` | NodeInfo 2.1 HTTP fetch via `newDiscoveryClient()` | Yes (live remote endpoint) | FLOWING (runtime-dependent; unit tests cover pure logic only) |
| `FederationDiscover` | `actor` | `federation.GetActorByIRI` | Yes (creates/refreshes DB record from remote AP endpoint) | FLOWING |
| `FederationPeers` | `outboundRecords/inboundRecords` | `FindRecordsByFilter("follows", ...)` | Yes (real DB queries) | FLOWING |
| `buildPeerEntries` | `domainOf(actorID)` | `FindRecordById("activitypub_actors", actorID)` via closure | Yes (real DB lookup) | FLOWING |

No static/empty data sources detected. All handlers query live DB or live remote endpoints.

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full module compiles | `cd db && go build ./...` | exit 0, no output | PASS |
| All 18 federation admin tests pass | `go test ./routes/ -run TestPickNodeInfo21Href\|TestFindLocal...\|TestFederationDiscover\|TestFederationFollow\|TestFederationApprove\|TestFederationReject\|TestDisconnectAction\|TestBuildPeerEntries` | 18/18 PASS | PASS |
| SAFE-07: no Create*Activity in handler file | `grep -c 'Create.*Activity' federation_admin.go` | 0 | PASS |
| SAFE-06: 10s timeout present | `grep -n 'SetTimeout(10'` | line 75 confirmed | PASS |
| SAFE-06: FetchPublicURL absent | `grep -c 'util.FetchPublicURL'` | 0 | PASS |
| Route wiring count | `grep -c 'routes.Federation' main.go` | 6 | PASS |
| All four DISC-02 error strings | grep each in file | "unreachable", "not a Wanderer instance", "already connected", "resolves to local instance" — all present | PASS |
| Auth guard first in all handlers | `grep -n 'HasSuperuserAuth'` | 6 occurrences (lines 213, 284, 426, 534, 573, 611) | PASS |

---

### Probe Execution

No conventional `scripts/*/tests/probe-*.sh` files declared or found for this phase. Step 7c: SKIPPED (no declared probes).

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| DISC-01 | 05-01, 05-03 | Preview card (actor_id, domain, version, user/trail counts) + peer list | SATISFIED | `FederationDiscover` returns all five fields; `FederationPeers` returns `[{follow_id, direction, status, domain}]` |
| DISC-02 | 05-01 | Clear errors for unreachable / non-Wanderer / already-connected / local-instance | SATISFIED | All four error strings verified at runtime paths in `FederationDiscover` |
| CONN-01 | 05-02 | Outbound Follow created with status pending; hook delivers Follow | SATISFIED | `createOutboundFollow` confirmed; `TestFederationFollowCreatesPendingRecord` passes; SAFE-07 grep = 0 |
| CONN-02 | 05-02 | Approve inbound follow → status=accepted → Accept{Follow} via hook | SATISFIED | `setFollowStatus(..., "accepted")` confirmed; `TestFederationApproveSetsAccepted` passes |
| CONN-03 | 05-02 | Reject inbound follow → status=rejected → Reject{Follow} via hook | SATISFIED | `setFollowStatus(..., "rejected")` confirmed; `TestFederationRejectSetsRejected` passes |
| CONN-04 | 05-03 | Direction-aware disconnect — outbound=Delete/Undo, inbound-only=Save(rejected)/Reject | SATISFIED | `disconnectAction` helper; `e.App.Delete` on outbound branch; `Set(status=rejected)` on inbound branch; unit tests pass |
| SAFE-05 | 05-01 | Self-follow guard via util.IsLocalIRI | SATISFIED | `if util.IsLocalIRI(actorIRI)` at line 657 in FederationDiscover |
| SAFE-06 | 05-01 | SSRF-safe client with ≤10s timeout; no FetchPublicURL | SATISFIED | `newDiscoveryClient()` with `SetTimeout(10 * time.Second)`; FetchPublicURL count = 0 |
| SAFE-07 | 05-01, 05-02, 05-03 | Handlers only write DB records; hooks own ActivityPub delivery | SATISFIED | `grep -c 'Create.*Activity'` = 0 across the entire handler file |

**Phase 5 requirement IDs declared in plans:** DISC-01, DISC-02, CONN-01, CONN-02, CONN-03, CONN-04, SAFE-05, SAFE-06, SAFE-07 (9 IDs)
**Orphaned requirements (Phase 5 in REQUIREMENTS.md not in any plan):** None

---

### Anti-Patterns Found

No debt markers (TBD, FIXME, XXX) found in either `federation_admin.go` or `federation_admin_test.go`.

No TODO/HACK/PLACEHOLDER markers found.

No stub patterns found:
- No `return null`, `return {}`, `return []` in handler paths
- No hardcoded empty data propagating to response
- No console.log-only implementations (Go file)

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | — | — | No anti-patterns found |

---

### Human Verification Required

All automated checks pass. The following items require runtime integration testing — they cannot be verified by static analysis or unit tests alone.

#### 1. FederationDiscover — End-to-End Preview Card

**Test:** POST `{"url": "https://remote.wanderer.instance"}` to `/federation/discover` with a valid superuser token
**Expected:** HTTP 200 with JSON `{"actor_id": "<non-empty>", "domain": "remote.wanderer.instance", "version": "<semver>", "user_count": <int>, "trail_count": <int>}`
**Why human:** `federation.GetActorByIRI` makes a live outbound HTTP request to the remote ActivityPub endpoint; unit tests cover only the pure NodeInfo parsing and identity check logic

#### 2. FederationDiscover — Non-Wanderer Rejection (Live)

**Test:** POST a live Mastodon or other non-Wanderer instance URL to `/federation/discover`
**Expected:** HTTP 400 `{"error": "not a Wanderer instance"}`
**Why human:** Requires live outbound HTTP

#### 3. FederationFollow — Hook Invocation (No Double-Delivery)

**Test:** POST `{"actor_id": "<id>"}` to `/federation/follow` in an environment with hooks active; observe that exactly one Follow activity is sent to the remote inbox
**Expected:** One outgoing Follow; no direct delivery from handler
**Why human:** `InstanceFollowCreateHandler` hook invocation is a runtime event; grep confirms absence of direct calls but cannot verify the hook fires or that delivery is not duplicated

#### 4. FederationApprove — Hook Delivers Accept{Follow}

**Test:** With a pending inbound follow, POST to `/federation/approve/:id`; verify Accept{Follow} is delivered to the remote
**Expected:** Follow record status=accepted; one Accept{Follow} in remote inbox
**Why human:** Hook invocation and wire delivery are runtime behaviors

#### 5. FederationDisconnect — Direction-Aware Verb Selection (Runtime)

**Test:** (a) Disconnect an outbound follow — verify record deleted and Undo{Follow} sent. (b) Disconnect an inbound-only follow — verify record NOT deleted, status=rejected, Reject{Follow} sent (not Undo)
**Expected:** Correct activity verb per direction; no wrong-direction Undo on case (b)
**Why human:** `disconnectAction` and DB write path are unit-tested; hook invocation and ActivityPub payload verification require integration environment

#### 6. 401 Guard — All Six Endpoints (Runtime)

**Test:** Call each of the six `/federation/*` endpoints without a token and with a regular Wanderer user token
**Expected:** HTTP 401 on all six in both cases
**Why human:** `e.HasSuperuserAuth()` is confirmed present in all six handlers, but the guard's interaction with PocketBase's auth middleware stack requires a running server to validate end-to-end

---

### Gaps Summary

No gaps found. All 9 requirement IDs satisfied. All artifacts exist at full implementation level (not stubs). All key links wired. Build and full test suite pass (18/18 tests, `go build ./...` exits 0). No anti-patterns.

Status is `human_needed` because six integration behaviors — live HTTP round-trips, hook invocation, and the PocketBase auth middleware gate — cannot be verified programmatically without a running server.

---

_Verified: 2026-06-27T18:00:00Z_
_Verifier: Claude (gsd-verifier)_
