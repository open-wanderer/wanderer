---
phase: 03-fanout-and-safety
reviewed: 2026-06-26T00:00:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - db/federation/activity.go
  - db/federation/activity_test.go
  - db/federation/create.go
  - db/federation/create_test.go
  - db/federation/delete.go
  - db/federation/delete_test.go
  - db/federation/instance.go
  - db/federation/instance_inbox_test.go
findings:
  critical: 4
  warning: 5
  info: 2
  total: 11
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-06-26
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

This phase implements instance-level fanout (D-03) and privacy safety gates (SAFE-01 through SAFE-03). The core mechanics are mostly correct: the `instanceFollowerInboxes` plumbing works, the SAFE-01 dedup guard fires only on Create, and the SAFE-03 gate for private-trail comments is in the right place. However, four critical defects remain: the `processDeleteListActivity` handler performs no authorization check (anyone can delete any list), the tag-parsing slice expression can panic on empty or single-byte content strings, `CreateCommentActivity` unconditionally posts to the local trail-author inbox (self-delivery noise and potential loop), and both `CreateSummitLogActivity` and `CreateListActivity` call `PostActivity` before persisting the activity record so a network-triggered panic could leave a ghost delivery with no record. Additionally, the duplicate code path for `generateInstanceKeyPair` and several missing error-handling patterns constitute notable warnings.

## Critical Issues

### CR-01: `processDeleteListActivity` has no actor ownership check — any remote actor can delete any list

**File:** `db/federation/delete.go:378-392`
**Issue:** `processDeleteTrailActivity` (line 332), `processDeleteCommentActivity` (line 352), and `processDeleteSummitLogActivity` (line 371) all verify that the requesting actor is the record's author before deleting. `processDeleteListActivity` skips this check entirely: it looks up the list by IRI and immediately deletes it (`util.DeleteFromFeed` + `app.Delete`). A malicious remote instance that knows any list's IRI can send a Delete activity to permanently remove it from the receiving instance, bypassing ownership.

**Fix:**
```go
func processDeleteListActivity(app core.App, actor *core.Record, activity pub.Activity) error {
    object := activity.Object.GetID().String()
    list, err := app.FindFirstRecordByData("lists", "iri", object)
    if err != nil {
        return err
    }

    // SAFE-02 parity: enforce list author ownership before deletion.
    if list.GetString("author") != actor.Id {
        return fmt.Errorf("actor is not list author")
    }

    err = util.DeleteFromFeed(app, list.Id)
    if err != nil {
        return err
    }

    return app.Delete(list)
}
```

---

### CR-02: Slice bounds panic in summit-log tag parsing when content string is empty

**File:** `db/federation/create.go:710-716`
**Issue:** When parsing inbound summit-log numeric tags, the code strips the trailing unit character with `content[:len(content)-1]` before calling `strconv.ParseFloat`. If `content` is an empty string (e.g., a malformed or missing tag value from a remote instance), `len(content)-1` evaluates to `-1`, causing an immediate index-out-of-range panic. Because `processCreateOrUpdateSummitLogActivity` is called from `ProcessCreateOrUpdateActivity`, which is called from `InstanceInboxHandler` inside `case pub.CreateType: / case pub.UpdateType:`, this panic would propagate to the PocketBase request goroutine and crash the handler for that request. The `recover()` in `PostActivity` does not guard this code path.

**Fix:**
```go
for _, tag := range tags.Collection() {
    tagObj, err := pub.ToObject(tag)
    if err != nil {
        continue
    }
    content := tagObj.Content.First().Value.String()
    if len(content) == 0 {
        continue // guard against empty content
    }
    numeric := content[:len(content)-1] // strip unit suffix
    switch tagObj.Name.First().Value.String() {
    case "elevation_gain":
        elevation_gain, err = strconv.ParseFloat(numeric, 64)
    case "elevation_loss":
        elevation_loss, err = strconv.ParseFloat(numeric, 64)
    case "duration":
        duration, err = strconv.ParseFloat(numeric, 64)
    case "distance":
        distance, err = strconv.ParseFloat(numeric, 64)
    }
    if err != nil {
        continue
    }
}
```

---

### CR-03: Activity record saved after `PostActivity` in `CreateSummitLogActivity` and `CreateListActivity` — record is lost on save failure

**File:** `db/federation/create.go:357-372` (summit log), `db/federation/create.go:427-442` (list)
**Issue:** In both `CreateSummitLogActivity` and `CreateListActivity`, `PostActivity` is called before `app.Save(record)`. `PostActivity` returns immediately (it fires a goroutine) so the call always returns `nil` and the error check on line 358 / line 428 never actually fires. However, if `app.Save(record)` subsequently fails, the activity has already been dispatched to remote inboxes but no local record exists — the outbox is inconsistent and the activity cannot be referenced or retracted. By contrast, `CreateTrailActivity` and `CreateCommentActivity` correctly save the record before dispatching.

**Fix:** Move the `record` construction and `app.Save(record)` call above the `PostActivity` call in both functions, consistent with the pattern in `CreateTrailActivity` (lines 77-105).

```go
// Save the activity record first, then dispatch
record := core.NewRecord(collection)
record.Set("id", recordId)
// ... all Set calls ...
if err := app.Save(record); err != nil {
    return err
}

return PostActivity(app, summitLogAuthor, activity, recipients)
```

---

### CR-04: `CreateCommentActivity` unconditionally adds the local trail author's inbox to `recipients`, causing self-delivery

**File:** `db/federation/create.go:152`
**Issue:** Line 152 appends `commentTrailAuthor.GetString("inbox")` to `recipients` regardless of whether the trail author is local or remote. When the trail author is local, this causes the comment activity to be HTTP-POSTed to the local instance's own ActivityPub inbox, which (1) generates spurious HTTP traffic, (2) can trigger double-processing of the comment, and (3) may cause a follow-on error if the local inbox requires auth not present in this server-to-server call. `CreateSummitLogActivity` (line 346-348) and `CreateCommentDeleteActivity` (lines 127-130 in delete.go) both correctly guard the same pattern with `if !commentTrailAuthor.GetBool("is_local")`. This function does not.

**Fix:**
```go
// Only notify the trail author if they are remote (local actors receive the
// activity through local event hooks, not via HTTP delivery).
if !commentTrailAuthor.GetBool("is_local") {
    recipients = append(recipients, commentTrailAuthor.GetString("inbox"))
}
```

## Warnings

### WR-01: `processDeleteCommentActivity`, `processDeleteSummitLogActivity`, and `processDeleteTrailActivity` do not tolerate `sql.ErrNoRows` — Delete races cause errors

**File:** `db/federation/delete.go:347-350`, `db/federation/delete.go:366-369`, `db/federation/delete.go:325-328`
**Issue:** All three inbound delete handlers call `app.FindFirstRecordByData` and return any error, including `sql.ErrNoRows`, which occurs when the record was already deleted (duplicate delivery, race condition). In contrast, the dedup guard in `processCreateOrUpdateTrailActivity` (line 471-476) explicitly ignores `sql.ErrNoRows` as a known-normal case. A duplicate Delete activity (possible in federated environments with redelivery) will return a non-nil error and log a failure even though the desired end-state is already achieved.

**Fix:** In each handler, treat `sql.ErrNoRows` as a success (idempotent delete):
```go
trail, err := app.FindFirstRecordByData("trails", "iri", object)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        return nil // already deleted — idempotent
    }
    return err
}
```

---

### WR-02: `req.Header.Add("Host", req.Host)` in `PostActivity` adds an empty Host header

**File:** `db/federation/activity.go:150`
**Issue:** When `http.NewRequest` is called with a full URL (as on line 143), `req.Host` is the empty string. Go's `net/http` client correctly derives the `Host` header from the URL at transport time, so setting it explicitly to `""` via `Header.Add` is at best redundant. However, some HTTP signature implementations include `Host` in the signed headers list (line 96 lists `"Host"` as a required header). If the `httpsig` library reads `req.Header.Get("Host")` rather than `req.URL.Host` when building the signature base string, it will sign an empty value while the actual wire `Host` header carries the correct hostname — causing the remote receiver's signature verification to fail. The correct pattern is to set `req.Host = req.URL.Host` (on the struct field, not the header) before signing.

**Fix:**
```go
// Set req.Host explicitly so httpsig reads the correct value when building
// the Host header signing component.
req.Host = req.URL.Host
// Remove the redundant explicit header — net/http derives it from req.Host.
// req.Header.Add("Host", req.Host)  <-- remove this line
```

---

### WR-03: `generateInstanceKeyPair` is duplicated from `db/util/activitypub.go`

**File:** `db/federation/instance.go:26-32`
**Issue:** The comment on line 25 acknowledges this is a copy from `db/util/activitypub.go`. Duplicate cryptographic helpers create maintenance risk: if the key size, algorithm, or error-handling changes in the authoritative copy, the duplicate is silently left behind with the old behavior. The function should be exported from `util` (or moved to a shared internal package) and called from both sites.

**Fix:** Export `GenerateRSAKeyPair` from `db/util/activitypub.go` and replace the local copy:
```go
// In instance.go — remove generateInstanceKeyPair and use the util version:
priv, pubKey, err := util.GenerateRSAKeyPair()
```

---

### WR-04: `CreateCommentActivity` does not call `followerInboxes` for the comment author — comment author's followers do not receive the activity

**File:** `db/federation/create.go:108-201`
**Issue:** `CreateTrailActivity` (line 92) and `CreateSummitLogActivity` (line 340) both call `followerInboxes(app, author.Id)` to ensure the author's own follower list is included in delivery. `CreateCommentActivity` builds `recipients` from mentions and the trail author only — it never calls `followerInboxes` for `commentAuthor`. As a result, the comment author's followers on remote instances never receive the comment activity, breaking comment federation for user-level followers.

**Fix:**
```go
inboxes, err := followerInboxes(app, commentAuthor.Id)
if err != nil {
    return err
}
recipients = append(recipients, inboxes...)
```
Add this after line 152, before building `cc`.

---

### WR-05: `InstanceInboxHandler` comment says "only Follow/Accept/Undo" but implementation also dispatches Create/Update/Delete

**File:** `db/federation/instance.go:112-113`, `db/federation/instance.go:185-193`
**Issue:** The function-level doc comment (line 112-113) states: "only dispatches Follow/Accept/Undo activities (D-03: the instance inbox is isolated to the follow lifecycle)." The actual `switch` block (lines 185-193) also handles `pub.CreateType`, `pub.UpdateType`, and `pub.DeleteType`. This is not a runtime bug — the extra cases were presumably added intentionally for SYNC-01 — but the stale comment will mislead future maintainers and create uncertainty about the intended inbox contract.

**Fix:** Update the doc comment to reflect the actual dispatch table:
```go
// It dispatches Follow/Accept/Undo for the follow lifecycle and
// Create/Update/Delete for content synchronization (SYNC-01).
```

## Info

### IN-01: Inconsistent `sql.ErrNoRows` comparison style in `create.go`

**File:** `db/federation/create.go:544`, `db/federation/create.go:571`, `db/federation/create.go:655`, `db/federation/create.go:683`
**Issue:** Several `sql.ErrNoRows` checks use direct equality (`err == sql.ErrNoRows`) rather than the idiomatic `errors.Is(err, sql.ErrNoRows)`. The dedup guard at lines 474 and 525 correctly uses `errors.Is`. Mixing the two styles is inconsistent and `errors.Is` is preferred because it handles wrapped errors.

**Fix:** Replace all `err == sql.ErrNoRows` with `errors.Is(err, sql.ErrNoRows)`.

---

### IN-02: `followers` field is absent from the test `activitypub_actors` schema in `instance_inbox_test.go`

**File:** `db/federation/instance_inbox_test.go:31-65` (actor collection JSON)
**Issue:** The production `activitypub_actors` collection includes a `followers` URL field (added in migration `1747061257`). The test schema in `newInboxTestApp` omits it. Tests that exercise `CreateTrailActivity`, `CreateSummitLogActivity`, or `CreateListActivity` against this test app would get an empty string from `trailAuthor.GetString("followers")`, silently producing a malformed CC field in the activity. None of the current tests in scope call those create functions, so this is currently latent, but it will cause silent CC omission if tests are added.

**Fix:** Add the `followers` field to the test actor collection JSON alongside `inbox` and `outbox`:
```json
{"exceptDomains":null,"hidden":false,"id":"url_followers_001","name":"followers","onlyDomains":null,"presentable":false,"required":false,"system":false,"type":"url"}
```
And seed it in `createTestActor`:
```go
r.Set("followers", iri+"/followers")
```

---

_Reviewed: 2026-06-26_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
