---
phase: 02-follow-lifecycle
plan: "02"
subsystem: federation
tags: [activitypub, federation, follow-lifecycle, instance-actor, inbox]
dependency_graph:
  requires: [01-01-SUMMARY.md]
  provides: [InstanceInboxHandler, ProcessFollowActivity-instance-branch]
  affects: [db/federation/follow.go, db/federation/instance.go, db/main.go, web/src/routes/api/v1/activitypub/instance/inbox/+server.ts]
tech_stack:
  added: []
  patterns:
    - X-Forwarded-Path SvelteKit proxy (mirrors existing user inbox pattern)
    - actor_type branch in ProcessFollowActivity (filter on followee, not actor)
    - early-return guard for instance follows (no Accept sent back)
key_files:
  created:
    - db/federation/instance_inbox_test.go
    - web/src/routes/api/v1/activitypub/instance/inbox/+server.ts
  modified:
    - db/federation/follow.go
    - db/federation/instance.go
    - db/main.go
decisions:
  - "Filter actor_type on the followee (object), not the actor — remote actors fetched via GetActorByIRI do not have actor_type populated locally; the local instance actor always has actor_type=instance"
  - "Early return nil after saving pending follow + persisting incoming Follow activity — prevents Accept delivery and notification for instance follows"
  - "Store incoming Follow to activitypub_activities (actor=sender IRI, object=instance IRI) so Plan 03 can reconstruct it for Accept/Reject without re-fetching"
  - "InstanceInboxHandler dispatches only Follow/Accept/Undo — Create/Update/Delete/Announce/Like are intentionally excluded (D-03)"
  - "PocketBase json-type fields return raw JSON literal from GetString — use strings.Trim in tests to compare unquoted IRI value"
metrics:
  duration: "~45 minutes"
  completed: "2026-06-25"
  tasks_completed: 2
  files_changed: 5
---

# Phase 02 Plan 02: Instance Inbox + Follow Pending Branch Summary

Implemented the receiving half of the instance Follow lifecycle: a remote instance can POST an HTTP-signed Follow to the local instance inbox, the signature is verified, and the Follow is stored as `pending` (requiring admin approval) instead of being auto-accepted — with the incoming Follow persisted to `activitypub_activities` for Plan 03 Accept/Reject reconstruction.

## What Was Built

**Task 1 (TDD): Actor-type branch in ProcessFollowActivity**

Modified `ProcessFollowActivity` in `db/federation/follow.go` to branch on the followee's `actor_type`. When a remote actor Follows the local instance actor (`actor_type="instance"` and `is_local=true` on the `object`):

1. Stores the follows record with `status="pending"` (no auto-accept)
2. Persists an `activitypub_activities` record of `type="Follow"` (actor=sender IRI, object=instance actor IRI) for Plan 03 Accept/Reject reconstruction
3. Returns `nil` immediately — the existing Accept delivery and notification code does NOT run

The person-actor auto-accept path is byte-for-byte unchanged.

Tests added to `db/federation/instance_inbox_test.go`:
- `TestProcessFollowInstanceActorSetsPending` — follows record has status=pending, no Accept activity created
- `TestProcessFollowInstanceFollowStored` — activitypub_activities Follow record created with correct actor/object
- `TestProcessFollowPersonActorAutoAccepts` — person-actor follow status=accepted (regression guard)

**Task 2: InstanceInboxHandler + SvelteKit proxy + main.go wiring**

Added `InstanceInboxHandler` to `db/federation/instance.go`:
- Reads body, unmarshals into `pub.Activity`
- Reconstructs inbox IRI from `X-Forwarded-Path` header (set by SvelteKit proxy)
- Looks up sender actor locally; fetches remotely via `GetActorByIRI` on cache miss
- Verifies HTTP signature via `util.VerifySignature`; returns 401 on failure
- Dispatches Follow/Accept/Undo only (D-03); all other activity types silently ignored

Registered route in `db/main.go`:
```
se.Router.POST("/activitypub/instance/inbox", federation.InstanceInboxHandler)
```

Created `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts` — a 1:1 mechanical mirror of the existing user inbox SvelteKit proxy:
- Clones all request headers
- Sets `X-Forwarded-Path = event.url.pathname` (trusted internal header)
- Forwards to `/activitypub/instance/inbox` via `event.locals.pb.send`

## Verification Results

```
cd db && go build ./...      → PASS (exit 0)
cd db && go vet ./federation/ → PASS (exit 0)
cd db && go test ./federation/ -run TestProcessFollow -count=1 → PASS (3/3 tests)
grep actor_type == "instance" in follow.go → 1 match
grep func InstanceInboxHandler in instance.go → 1 match
grep activitypub/instance/inbox in main.go → 1 match
grep X-Forwarded-Path in +server.ts → 2 matches
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Worktree branched from wrong base commit**
- **Found during:** Pre-execution worktree branch check
- **Issue:** Worktree HEAD was at `485ec53f` (main branch commit) instead of `3a4c0cbe` (feature/ap-instance-actors tip). Phase 1 artifacts (`db/federation/instance.go`, `db/federation/instance_test.go`, `db/main.go` federation import, migration, SvelteKit GET route) were absent.
- **Fix:** Used `git checkout 3a4c0cbe -- <files>` to restore phase 1 prerequisites into the worktree. Committed as `chore(02-02): restore phase-1 prerequisites` (hash `14d5bb1f`).
- **Files modified:** db/federation/instance.go, db/federation/instance_test.go, db/main.go, db/migrations/1782290000_add_actor_type_to_activitypub_actors.go, web/src/routes/api/v1/activitypub/instance/+server.ts

**2. [Rule 1 - Bug] Variable name collision in instance.go**
- **Found during:** Task 2 implementation
- **Issue:** Adding `pub "github.com/go-ap/activitypub"` import to `instance.go` caused a collision: `InitInstanceActor` used `priv, pub, err := generateInstanceKeyPair()` where `pub` was a local variable shadowing the package alias.
- **Fix:** Renamed the local variable to `pubKey` in `InitInstanceActor`.
- **Files modified:** db/federation/instance.go

**3. [Rule 1 - Bug] PocketBase JSON field returns raw JSON literal**
- **Found during:** Task 1 TDD GREEN phase test run
- **Issue:** `TestProcessFollowInstanceFollowStored` failed because `rec.GetString("object")` on a `json`-type field returns the raw JSON literal (e.g. `"https://..."` with surrounding double quotes), not the unquoted string.
- **Fix:** Added `strings.Trim(rec.GetString("object"), "\"")` in the test assertion; added `"strings"` import to test file.
- **Files modified:** db/federation/instance_inbox_test.go

## Threat Surface Scan

All threat mitigations from the plan's STRIDE register are implemented:

| Threat ID | Mitigation | Location |
|-----------|-----------|----------|
| T-02-03 | `util.VerifySignature` + `e.UnauthorizedError` on failure | InstanceInboxHandler |
| T-02-04 | `X-Forwarded-Path` set by SvelteKit layer, not accepted from client | +server.ts proxy |
| T-02-05 | instance Follow stored as `pending`, early return prevents Accept | ProcessFollowActivity |
| T-02-07 | Incoming Follow persisted to activitypub_activities | ProcessFollowActivity |

No new threat surface introduced beyond the advertised inbox endpoint.

## Commits

| Hash | Type | Description |
|------|------|-------------|
| `14d5bb1f` | chore | Restore phase-1 prerequisites from feature/ap-instance-actors |
| `48a64fa4` | test | Add failing tests for instance-actor Follow branch (RED) |
| `1ed44c4e` | feat | Branch ProcessFollowActivity on instance actor — store as pending (GREEN) |
| `bd9275a0` | fix | Add strings import and fix JSON field assertion in inbox tests |
| `c7d73b24` | feat | Add InstanceInboxHandler, register route, and SvelteKit proxy |

## Self-Check: PASSED

All created/modified files exist on disk. All task commits are present in git history.
