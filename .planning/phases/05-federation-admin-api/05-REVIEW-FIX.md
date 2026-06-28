---
phase: 05-federation-admin-api
fixed_at: 2026-06-27T00:00:00Z
review_path: .planning/phases/05-federation-admin-api/05-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 05: Code Review Fix Report

**Fixed at:** 2026-06-27
**Source review:** .planning/phases/05-federation-admin-api/05-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (2 Critical, 5 Warning)
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: NodeInfo href host-hop SSRF

**Files modified:** `db/routes/federation_admin.go`
**Commit:** 49092e79
**Applied fix:** Added `"strings"` import. In `fetchNodeInfo21URL`, changed the direct return of `pickNodeInfo21Href(jrd.Links)` to a two-step pattern: call `pickNodeInfo21Href`, then parse the returned href and compare `hrefParsed.Host` to `u.Host` (case-insensitive via `strings.EqualFold`). If they differ, return `"not a Wanderer instance: NodeInfo href host mismatch"` instead of passing the cross-host URL to the second HTTP fetch.

---

### CR-02: "Already connected" check includes rejected follows

**Files modified:** `db/routes/federation_admin.go`
**Commit:** 44cce0aa
**Applied fix:** In `FederationDiscover`, added `&& status!='rejected'` predicate to both sides of the OR in the `FindFirstRecordByFilter` call that checks for existing follow records. Only `pending` and `accepted` records now trigger the "already connected" guard, allowing admins to re-initiate federation with a previously-rejected instance.

---

### WR-01: HTTP status code not checked before decoding responses

**Files modified:** `db/routes/federation_admin.go`
**Commit:** 86a2f689
**Applied fix:** Two locations:
1. In `fetchNodeInfo21URL`, added `if resp.StatusCode != http.StatusOK { return "", fmt.Errorf("unreachable: JRD returned HTTP %d", resp.StatusCode) }` after the JRD fetch.
2. In `FederationDiscover`, added `if niResp.StatusCode != http.StatusOK { return e.JSON(http.StatusBadRequest, ...) }` after the NodeInfo 2.1 payload fetch. Both checks precede the JSON decode calls.

---

### WR-02: FederationFollow lacks self-follow guard

**Files modified:** `db/routes/federation_admin.go`
**Commit:** 76869b70
**Applied fix:** Added an explicit self-follow guard in `FederationFollow` after loading `localActor`: `if localActor.Id == remoteActor.Id { return e.BadRequestError("cannot follow local instance actor", nil) }`. This fires before `createOutboundFollow` so no degenerate `follower==followee` record is created.

---

### WR-03: FederationApprove/FederationReject collapse errors; double-fetch of record

**Files modified:** `db/routes/federation_admin.go`, `db/routes/federation_admin_test.go`
**Commit:** ba1f1a43
**Applied fix:** Changed `setFollowStatus` signature from `(app core.App, followID, status, localID string)` to `(app core.App, follow *core.Record, status, localID string)`. The helper no longer calls `FindRecordById` internally (eliminating the double-fetch TOCTOU). Updated `FederationApprove` and `FederationReject` to pass the already-loaded `*core.Record` and to distinguish error types: `"not an inbound follow"` → 400 Bad Request; any other error (DB save failure) → 500 via `fmt.Errorf`. Updated all three test call sites in `federation_admin_test.go` to pass `follow` (the record) instead of `follow.Id`.

---

### WR-04: FederationPeers silently swallows DB errors

**Files modified:** `db/routes/federation_admin.go`
**Commit:** c7c07e17
**Applied fix:** In `FederationPeers`, replaced the `if err != nil { slice = nil }` pattern for both outbound and inbound `FindRecordsByFilter` calls with `if err != nil { return fmt.Errorf("query outbound/inbound follows: %w", err) }`. `FindRecordsByFilter` returns `[]` (not an error) when no rows match, so any error is a genuine DB failure that should surface as a 500.

---

### WR-05: Duplicate /federation/follow call hits DB UNIQUE constraint — returns HTTP 500

**Files modified:** `db/routes/federation_admin.go`
**Commit:** 9275b59d
**Applied fix:** In `createOutboundFollow`, added a pre-insert `FindFirstRecordByFilter` check for an existing `(follower, followee)` pair. If found, returns sentinel error `"follow already exists"`. In `FederationFollow`, added detection of this sentinel before the generic `fmt.Errorf` wrapper: if `err.Error() == "follow already exists"`, return `e.JSON(http.StatusConflict, ...)` with message `"already following this instance"`. This replaces the opaque 500 with a meaningful 409 Conflict.

---

_Fixed: 2026-06-27_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
