---
phase: 03-fanout-and-safety
fixed_at: 2026-06-26T00:00:00Z
review_path: .planning/phases/03-fanout-and-safety/03-REVIEW.md
iteration: 1
findings_in_scope: 9
fixed: 9
skipped: 0
status: all_fixed
---

# Phase 03: Code Review Fix Report

**Fixed at:** 2026-06-26
**Source review:** .planning/phases/03-fanout-and-safety/03-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 9 (4 Critical, 5 Warning)
- Fixed: 9
- Skipped: 0

## Fixed Issues

### CR-01: `processDeleteListActivity` has no actor ownership check

**Files modified:** `db/federation/delete.go`
**Commit:** `74a200e2`
**Applied fix:** Added an explicit `list.GetString("author") != actor.Id` ownership guard inside `processDeleteListActivity`, matching the pattern used by `processDeleteTrailActivity`, `processDeleteCommentActivity`, and `processDeleteSummitLogActivity`. Returns `fmt.Errorf("actor is not list author")` on mismatch.

---

### CR-02: Slice bounds panic in summit-log tag parsing when content string is empty

**Files modified:** `db/federation/create.go`
**Commit:** `1e0229f6`
**Applied fix:** Added `if len(content) == 0 { continue }` guard before the `content[:len(content)-1]` slice expression in `processCreateOrUpdateSummitLogActivity`. Extracted the slice result into a `numeric` variable for clarity. Prevents index-out-of-range panic on empty or malformed tag content from remote instances.

---

### CR-03: Activity record saved after `PostActivity` in `CreateSummitLogActivity` and `CreateListActivity`

**Files modified:** `db/federation/create.go`
**Commit:** `21ab10c8`
**Applied fix:** Moved the `record` construction and `app.Save(record)` call to before the `PostActivity` dispatch in both `CreateSummitLogActivity` and `CreateListActivity`. This matches the pattern in `CreateTrailActivity` and ensures the outbox record always exists before delivery is attempted, eliminating the ghost-delivery inconsistency on save failure.

---

### CR-04: `CreateCommentActivity` unconditionally adds local trail author inbox to recipients

**Files modified:** `db/federation/create.go`
**Commit:** `1d31488b`
**Applied fix:** Wrapped the `recipients = append(recipients, commentTrailAuthor.GetString("inbox"))` line inside `if !commentTrailAuthor.GetBool("is_local")`, matching the identical guard in `CreateSummitLogActivity` (line 346) and `CreateCommentDeleteActivity`. Local actors now receive the activity through local event hooks only, not spurious HTTP self-delivery.

---

### WR-01: Delete handlers do not tolerate `sql.ErrNoRows` on duplicate delivery

**Files modified:** `db/federation/delete.go`
**Commit:** `9496f610`
**Applied fix:** Added `errors.Is(err, sql.ErrNoRows)` idempotency guard to all four inbound delete handlers (`processDeleteTrailActivity`, `processDeleteCommentActivity`, `processDeleteSummitLogActivity`, `processDeleteListActivity`). Returns `nil` when the record is already absent, treating duplicate Delete deliveries as success. Added `"database/sql"` and `"errors"` to the import block.

---

### WR-02: `req.Header.Add("Host", req.Host)` adds an empty Host header

**Files modified:** `db/federation/activity.go`
**Commit:** `327158ae`
**Applied fix:** Replaced `req.Header.Add("Host", req.Host)` with `req.Host = req.URL.Host`. Setting the struct field (not the header map) ensures `httpsig` reads the correct hostname when building the `Host` header signing component. `net/http` derives the wire `Host` header from `req.Host` at transport time, so no explicit header entry is needed.

---

### WR-03: `generateInstanceKeyPair` is duplicated from `db/util/activitypub.go`

**Files modified:** `db/util/activitypub.go`, `db/federation/instance.go`
**Commit:** `570fc698`
**Applied fix:** Exported `GenerateRSAKeyPair` from `db/util/activitypub.go` (kept the existing unexported `generateKeyPair` as a thin wrapper for backward compatibility). Removed the local `generateInstanceKeyPair` function from `instance.go` and replaced the call site with `util.GenerateRSAKeyPair()`. Removed the now-unused `"crypto/rand"` and `"crypto/rsa"` imports from `instance.go`.

---

### WR-04: `CreateCommentActivity` does not call `followerInboxes` for comment author

**Files modified:** `db/federation/create.go`
**Commit:** `1d5008ec`
**Applied fix:** Added `followerInboxes(app, commentAuthor.Id)` after the trail author recipient guard and before the `cc` ItemCollection is built. Results are appended to `recipients`, matching the pattern in `CreateTrailActivity` and `CreateSummitLogActivity`. Comment author's followers on remote instances now receive the activity.

---

### WR-05: `InstanceInboxHandler` doc comment says "only Follow/Accept/Undo" but also handles Create/Update/Delete

**Files modified:** `db/federation/instance.go`
**Commit:** `c6b23403`
**Applied fix:** Updated the function-level doc comment to accurately state that the handler dispatches "Follow/Accept/Undo for the follow lifecycle and Create/Update/Delete for content synchronization (SYNC-01)." Removed the outdated claim that the inbox is "isolated to the follow lifecycle."

---

_Fixed: 2026-06-26_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
