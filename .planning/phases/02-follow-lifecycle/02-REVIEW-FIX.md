---
phase: 02-follow-lifecycle
fixed_at: 2026-06-25T16:10:00Z
review_path: .planning/phases/02-follow-lifecycle/02-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 02: Code Review Fix Report

**Fixed at:** 2026-06-25T16:10:00Z
**Source review:** .planning/phases/02-follow-lifecycle/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (3 Critical, 4 Warning; Info findings excluded per fix_scope)
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: `CreateFollowHandler` discards `e.Next()` error

**Files modified:** `db/hooks/follow.go`
**Commit:** `0368d9b2`
**Applied fix:** Replaced the bare `e.Next()` call with `if err := e.Next(); err != nil { return err }` before calling `federation.CreateFollowActivity`. This ensures federation delivery is never dispatched when the underlying DB save has failed.

---

### CR-02: `ProcessFollowActivity` has no idempotency guard

**Files modified:** `db/federation/follow.go`
**Commit:** `721415ae`
**Applied fix:** Added an idempotency check at the start of the `!actor.GetBool("is_local")` block using `FindFirstRecordByFilter` on `(follower, followee)`. If an existing follow record is found, the function returns `nil` immediately (duplicate delivery). Added `database/sql` and `errors` imports to support `errors.Is(existErr, sql.ErrNoRows)` for distinguishing not-found from real DB errors.

---

### CR-03: `CreateAcceptFollowActivity` persists DB record before delivering

**Files modified:** `db/federation/follow.go`
**Commit:** `5eb75fab`
**Applied fix:** Swapped the `app.Save(record)` and `PostActivity(...)` calls in both `CreateAcceptFollowActivity` and `CreateRejectFollowActivity`. Delivery is now attempted first; the activity record is only persisted if `PostActivity` returns without error. This prevents stale `accepted`/`rejected` DB state when HTTP delivery fails.

---

### WR-01: `InstanceInboxHandler` uses `==` instead of `errors.Is` for `sql.ErrNoRows`

**Files modified:** `db/federation/instance.go`
**Commit:** `32c01ffa`
**Applied fix:** Changed `if err == sql.ErrNoRows` to `if errors.Is(err, sql.ErrNoRows)` on line 146. Both `errors` and `database/sql` were already imported in the file. Now consistent with `InitInstanceActor` in the same file.

---

### WR-02: `ProcessFollowActivity` sends notification for instance-actor follows

**Files modified:** `db/federation/follow.go`
**Commit:** `397b3f69`
**Applied fix:** Added an explicit guard before the notification block: `if object.GetString("actor_type") == "instance" { return nil }`. This prevents `util.SendNotification` from being called with an instance-actor record if the early-return guard above was not taken (e.g., `actor_type` field absent).

---

### WR-03: `InstanceFollowUpdateHandler` fires for both directions of an instance follow

**Files modified:** `db/hooks/follow.go`
**Commit:** `b194bde4`
**Applied fix:** Changed the guard from `if !isInstanceFollow(...)` to `if !isInstanceFollow(...) || isOutboundInstanceFollow(...)`. When the local instance is the FOLLOWER (outbound follow), the handler now passes through without dispatching Accept/Reject. Accept/Reject delivery only runs when the local instance is the FOLLOWEE approving or denying an inbound follow.

---

### WR-04: TypeScript inbox handler returns empty string body

**Files modified:** `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts`
**Commit:** `a120b21d`
**Applied fix:** Renamed `success` to `response` and passed `response` to `json(response, ...)` instead of `json("", ...)`. Removed the dead `if (success === false)` branch (pb.send throws on non-2xx; it never returns false). Remote ActivityPub clients now receive the Go handler's `{"success":true}` JSON object.

---

_Fixed: 2026-06-25T16:10:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
