---
phase: 05-federation-admin-api
reviewed: 2026-06-27T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - db/routes/federation_admin.go
  - db/routes/federation_admin_test.go
  - db/main.go
findings:
  critical: 2
  warning: 5
  info: 3
  total: 10
status: issues_found
---

# Phase 05: Code Review Report

**Reviewed:** 2026-06-27
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

This review covers the federation admin API implementation: six handlers (`FederationDiscover`, `FederationFollow`, `FederationApprove`, `FederationReject`, `FederationDisconnect`, `FederationPeers`) plus their DB helpers and unit tests.

The authentication guard pattern is correctly applied as the first statement in all six handlers. The direction-aware disconnect logic is sound. The safeurl client is used for all outbound NodeInfo fetches. The hook-based delivery model (no direct federation delivery from handlers) is observed throughout.

Two critical issues were found: a NodeInfo response-URL host-hop that can probe arbitrary public hosts via the admin's SSRF-safe client, and an "already connected" filter that includes `rejected` follow records and permanently blocks reconnection to previously-rejected instances. Five warnings cover missing HTTP status checks, a misleading error collapse, a missing self-follow guard in `FederationFollow`, silent swallowing of real DB errors in `FederationPeers`, and a double-fetch TOCTOU in `FederationApprove`/`FederationReject`.

---

## Critical Issues

### CR-01: NodeInfo 2.1 `href` allows host-hop to arbitrary public hosts

**File:** `db/routes/federation_admin.go:91-133` and `631`

**Issue:** `fetchNodeInfo21URL` fetches the JRD from `{scheme}://{host}/.well-known/nodeinfo` (reconstructed from the admin-supplied URL — safe). It then extracts the `href` field from the attacker-controlled response body and passes that raw URL directly to `client.Do(niReq)` as the NodeInfo 2.1 endpoint. The `href` is not validated to share the same host as the original URL. A malicious or compromised instance can return:

```json
{"links":[{"rel":"http://nodeinfo.diaspora.software/ns/schema/2.1","href":"https://internal-monitoring.corp/path"}]}
```

The safeurl client will then make a request to `internal-monitoring.corp` (or any other public host on ports 80/443). While private IP ranges are blocked by `safeurl`, this is still a Server-Side Request Forgery (SSRF) variant: the admin's request for instance `friendly.example.com` causes the server to send a credentialed GET to an unrelated public host. This can be used to probe third-party hosts and bypass per-host rate limiting.

**Fix:** After extracting `nodeInfoURL`, validate that its host matches the original host before fetching:

```go
func fetchNodeInfo21URL(client httpDoer, rawURL string) (string, error) {
    u, err := url.Parse(rawURL)
    if err != nil || u.Scheme == "" || u.Host == "" {
        return "", fmt.Errorf("unreachable: invalid URL")
    }

    jrdURL := fmt.Sprintf("%s://%s/.well-known/nodeinfo", u.Scheme, u.Host)
    req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, jrdURL, nil)
    // ... existing code ...
    href, err := pickNodeInfo21Href(jrd.Links)
    if err != nil {
        return "", err
    }

    // ADDED: validate href stays on the same host as the original request
    hrefParsed, err := url.Parse(href)
    if err != nil || !strings.EqualFold(hrefParsed.Host, u.Host) {
        return "", fmt.Errorf("not a Wanderer instance: NodeInfo href host mismatch")
    }
    return href, nil
}
```

---

### CR-02: "already connected" check includes `rejected` follows — permanently blocks reconnection

**File:** `db/routes/federation_admin.go:674-681`

**Issue:** The `FederationDiscover` handler queries for an existing follow record with no status filter:

```go
"(follower={:l} && followee={:r}) || (follower={:r} && followee={:l})",
```

A `rejected` follow record matches this query. Consequences:

1. Remote instance A sends a Follow to local instance B. Admin rejects it (`FederationReject` → status = `rejected`).
2. Admin later tries `POST /federation/discover` to initiate an outbound follow to A.
3. The existing rejected record is found → handler returns `{"error": "already connected"}` (HTTP 400).
4. The admin has no path to establish federation with instance A without manually deleting the rejected record from the database.

This is a data-correctness bug: `rejected` does not mean "connected." Only `pending` and `accepted` follow records represent active or in-progress connection attempts.

**Fix:** Add a status filter to exclude rejected records:

```go
_, followErr := e.App.FindFirstRecordByFilter(
    "follows",
    "(follower={:l} && followee={:r} && status!='rejected') || (follower={:r} && followee={:l} && status!='rejected')",
    dbx.Params{"l": localActor.Id, "r": existingActor.Id},
)
```

---

## Warnings

### WR-01: HTTP status code not checked on JRD or NodeInfo responses

**File:** `db/routes/federation_admin.go:120-133` (`fetchNodeInfo21URL`) and `635-648` (`FederationDiscover`)

**Issue:** Neither the JRD fetch (`fetchNodeInfo21URL`) nor the NodeInfo 2.1 payload fetch (in `FederationDiscover`) check `resp.StatusCode` before attempting to decode the body. A server returning HTTP 404 or 500 with a JSON body — intentionally or coincidentally — will be parsed and accepted as a valid NodeInfo response. This can cause false positives: a non-Wanderer server returning an error page that happens to be valid JSON with `software.name == "wanderer"` would pass the identity check.

**Fix:**

```go
// In fetchNodeInfo21URL, after client.Do:
if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("unreachable: JRD returned HTTP %d", resp.StatusCode)
}

// In FederationDiscover, after client.Do for niResp:
if niResp.StatusCode != http.StatusOK {
    return e.JSON(http.StatusBadRequest, map[string]any{"error": "unreachable"})
}
```

---

### WR-02: `FederationFollow` lacks a self-follow guard

**File:** `db/routes/federation_admin.go:211-248`

**Issue:** `FederationFollow` accepts any `actor_id` that resolves to a valid `activitypub_actors` record. If an admin submits the local instance actor's own ID as `actor_id`, `createOutboundFollow` will create a follows record where `follower == followee == localActor.Id`. The UNIQUE constraint `(follower, followee)` in the migration allows this first insert. The resulting record represents the local instance following itself, which will cause the `InstanceFollowCreateHandler` to attempt to deliver a `Follow` activity to the local inbox — a degenerate case that could produce unexpected behavior in the federation hooks.

`FederationDiscover` guards against this via `util.IsLocalIRI`, but that guard is bypassed when the admin calls `/federation/follow` directly.

**Fix:** Add an explicit guard in `FederationFollow` (or in `createOutboundFollow`) before saving:

```go
// In FederationFollow, after loading localActor:
if localActor.Id == remoteActor.Id {
    return e.BadRequestError("cannot follow local instance actor", nil)
}
```

---

### WR-03: `FederationApprove`/`FederationReject` collapse all `setFollowStatus` errors to "not an inbound follow"

**File:** `db/routes/federation_admin.go:554-555` and `593-594`

**Issue:** `setFollowStatus` can fail for at least three distinct reasons:
1. The follow record was deleted between the initial `FindRecordById` in the handler (step 3) and the second `FindRecordById` inside `setFollowStatus` (step 5) — returns `"follow not found"`.
2. The follow's followee is not the local actor — returns `"not an inbound follow"`.
3. `app.Save` fails — returns a wrapped DB error.

All three are caught by a single `e.BadRequestError("not an inbound follow", nil)`. This means:
- A record deleted by a concurrent request produces a misleading 400 "not an inbound follow" instead of a 404.
- A DB failure produces the same misleading message.

The record is also loaded twice (once in the handler, once inside `setFollowStatus`), which is redundant.

**Fix:** Distinguish error types, or pass the already-loaded record into `setFollowStatus` to avoid the redundant re-fetch and allow proper error classification:

```go
// Option A: pass the record directly (avoids double-fetch and allows proper error handling)
func setFollowStatus(app core.App, follow *core.Record, status, localID string) error {
    if follow.GetString("followee") != localID {
        return fmt.Errorf("not an inbound follow")
    }
    follow.Set("status", status)
    return app.Save(follow)
}
```

---

### WR-04: `FederationPeers` silently swallows real DB errors

**File:** `db/routes/federation_admin.go:444-461`

**Issue:** Both the outbound and inbound follow queries treat any error (including genuine DB failures) as "no records," setting the slice to `nil`:

```go
if err != nil {
    // Treat "no records" as empty rather than an error.
    outboundRecords = nil
}
```

`FindRecordsByFilter` returns an empty slice (not `sql.ErrNoRows`) when no matching records exist. So the comment is incorrect: a real DB error — disk full, connection failure, corrupted index — will silently return an empty peers list to the admin, hiding the fault. This makes the endpoint unreliable for ops monitoring.

**Fix:** Return a 500 on genuine DB errors:

```go
outboundRecords, err := e.App.FindRecordsByFilter(...)
if err != nil {
    return fmt.Errorf("query outbound follows: %w", err)
}
// FindRecordsByFilter returns [] not error when no rows found
```

---

### WR-05: `FederationFollow` duplicate-follow error surfaces as HTTP 500 with an internal message

**File:** `db/routes/federation_admin.go:238-241`

**Issue:** If an admin calls `POST /federation/follow` twice with the same `actor_id`, the second call hits the UNIQUE constraint `(follower, followee)` in the `follows` table. `createOutboundFollow` wraps the error as `"save follow record: <DB error>"` and `FederationFollow` propagates it as `fmt.Errorf("createOutboundFollow: %w", err)`, which PocketBase turns into an unhandled HTTP 500. The admin receives an opaque internal error rather than a meaningful 409 Conflict response.

**Fix:** Check for an existing follow before inserting, or detect the constraint violation and return a 409:

```go
// In createOutboundFollow, before creating the new record:
existing, err := app.FindFirstRecordByFilter(
    "follows",
    "follower={:f} && followee={:e}",
    dbx.Params{"f": localID, "e": remoteID},
)
if err == nil && existing != nil {
    return nil, fmt.Errorf("follow already exists")
}
```

And in `FederationFollow`, check for this specific error and return HTTP 409.

---

## Info

### IN-01: Orphaned/duplicate section comment for `FederationDiscover`

**File:** `db/routes/federation_admin.go:154-174`

**Issue:** Lines 154-174 contain a full `// FederationDiscover handler` section banner with a detailed JSDoc comment for the `FederationDiscover` function — but the actual `FederationDiscover` function is defined at line 609, under a second identical section banner at line 603. The first section banner (lines 154-174) immediately transitions into the `FederationFollow` sub-section. This is a copy-paste artifact: the `FederationDiscover` comment block was placed before `FederationFollow` instead of before its own function.

**Fix:** Remove the duplicate banner block at lines 154-156 and move the JSDoc comment (lines 158-173) to immediately precede `FederationDiscover` at line 609, or simply delete the orphaned block entirely since the function already has a near-identical comment at line 605.

---

### IN-02: `containsString` helper in test file reimplements `strings.Contains`

**File:** `db/routes/federation_admin_test.go:557-569`

**Issue:** The test file defines `containsString` and `findSubstring` helper functions that reproduce the behavior of `strings.Contains` from the standard library. The `"strings"` package is not imported. The custom implementation is correct but adds unnecessary code.

**Fix:** Import `"strings"` and replace calls to `containsString(s, substr)` with `strings.Contains(s, substr)`. Remove the two helper functions.

---

### IN-03: `TestFederationDiscoverAcceptsWanderer` and `TestFederationDiscoverRejectsNonWanderer` do not test the handler

**File:** `db/routes/federation_admin_test.go:272-305`

**Issue:** These two tests check a struct field comparison directly (`ni.Software.Name == "wanderer"`), not the `FederationDiscover` handler. The test names suggest they test the discover endpoint's identity-check behavior, but they only verify that a local variable has a specific value — they cannot catch regressions in request parsing, auth checking, HTTP response codes, or the surrounding logic. The `fetchNodeInfo21URL` function also has no unit test.

**Fix:** Consider replacing these trivial struct tests with table-driven tests over `fetchNodeInfo21URL` using `httptest.NewServer` to serve controlled JRD and NodeInfo responses, covering: status code non-200, cross-host href, missing 2.1 link, and non-Wanderer software name.

---

_Reviewed: 2026-06-27_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
