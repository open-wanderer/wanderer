# Pitfalls Research

**Domain:** Admin Federation Connect UI — ActivityPub instance peer management (v1.1)
**Researched:** 2026-06-27
**Confidence:** HIGH (based on direct code analysis of db/hooks/follow.go, db/federation/follow.go, db/federation/actor.go, db/federation/activity.go, db/routes/plugin_system.go)

---

## Critical Pitfalls

### Pitfall 1: Double Follow Delivery via "Initiate Follow" API Endpoint

**What goes wrong:**
A new `/admin/federation/connect` API endpoint that directly calls `federation.CreateFollowActivity()` AND also creates a follows record in the database will fire the AP activity twice. `InstanceFollowCreateHandler` is registered on `OnRecordAfterCreateSuccess("follows")` in `main.go` (line 132). That hook calls `CreateFollowActivity` for any follows record where `isOutboundInstanceFollow` returns true. If the new API endpoint also calls `CreateFollowActivity` before or after saving the record, two Follow activities are dispatched to the remote inbox.

**Why it happens:**
The hook-based delivery pattern is invisible to code written in the new API route layer. An implementer who reads `federation.CreateFollowActivity` and calls it directly — not knowing that saving the follows record also triggers `InstanceFollowCreateHandler` — produces two deliveries. The remote instance receives two Follow activities for the same pair; some ActivityPub servers treat the second as a fresh request and reset their pending state.

**How to avoid:**
The new "initiate Follow" endpoint must ONLY create the follows record via `app.Save(followRecord)`. It must not call `CreateFollowActivity` itself. `InstanceFollowCreateHandler` fires automatically after the successful save and handles delivery. This split is already enforced in `CreateFollowHandler` (hooks/follow.go line 16-27): the request hook skips delivery for instance follows precisely because `InstanceFollowCreateHandler` owns that path. The endpoint must also pre-create the remote actor record (via `GetActorByIRI`) before saving the follows record, because `InstanceFollowCreateHandler` does a re-fetch on the already-resolved actor — if the actor record does not exist yet, the handler logs an error and exits without delivery.

**Warning signs:**
- Remote instance receives two Follow activities within milliseconds
- `activitypub_activities` table has duplicate rows with `type='Follow'` for the same actor/object pair
- Remote instance's pending follow queue shows two entries from this instance

**Phase to address:**
Phase implementing "Initiate Follow" API endpoint — before writing the route handler.

---

### Pitfall 2: "Approve/Reject" Endpoint Causes Double Delivery by Calling Activity Functions Directly

**What goes wrong:**
A new "approve inbound Follow" endpoint that calls `federation.CreateAcceptFollowActivity()` directly (to send the Accept) and then calls `app.Save(followRecord)` with `status=accepted` will cause `InstanceFollowUpdateHandler` to fire on the save and call `CreateAcceptFollowActivity` a second time — delivering a duplicate Accept to the remote instance.

**Why it happens:**
`InstanceFollowUpdateHandler` fires on every `OnRecordAfterUpdateSuccess("follows")` where `isInstanceFollow` is true and `isOutboundInstanceFollow` is false. If the new endpoint both directly delivers the Accept AND saves the status change, the hook fires again. Most ActivityPub servers tolerate duplicate Accepts, but it creates unnecessary noise and could confuse state machines on strict implementations.

**How to avoid:**
The approve/reject endpoints must ONLY update the follows record status via `app.Save(followRecord)`. `InstanceFollowUpdateHandler` owns the delivery path exclusively. Never call `CreateAcceptFollowActivity` or `CreateRejectFollowActivity` from an API route handler directly.

**Warning signs:**
- Remote instance logs show two Accept activities for the same Follow
- `activitypub_activities` table has two rows with `type='Accept'` for the same actor/object within a short time window

**Phase to address:**
Phase implementing "Approve/Reject" API endpoint.

---

### Pitfall 3: Disconnect Endpoint Sends Semantically Incorrect Undo for Inbound-Only Connections

**What goes wrong:**
A "disconnect" endpoint that deletes the follows record triggers `InstanceFollowDeleteHandler`, which calls `CreateUnfollowActivity` for any follows record where `isInstanceFollow` is true — regardless of direction. If the local instance is the FOLLOWEE (the remote instance followed us, we accepted), deleting the record sends `Undo{Follow}` signed by the local instance actor, but the Follow being undone was originally signed by the remote instance actor. The local instance cannot Undo a Follow it did not send.

**Why it happens:**
`InstanceFollowDeleteHandler` uses `isInstanceFollow` (bidirectional check) rather than `isOutboundInstanceFollow` (directional). This was intentional for the v1.0 admin-panel workflow where the admin was explicitly deleting any follow record and the semantic distinction did not matter. For a UI-initiated "Disconnect" operation, an Undo on an inbound follow confuses the remote instance.

**How to avoid:**
The disconnect endpoint must enforce direction-aware semantics. For an inbound follow (remote followed us): the correct AP signal is `Reject{Follow}` (retroactive denial), not `Undo{Follow}`. For an outbound follow (we followed remote): `Undo{Follow}` is correct. Inspect direction before deleting: soft-delete or change status to `rejected` for inbound follows (which triggers `InstanceFollowUpdateHandler` to send `Reject`) rather than hard-deleting the record (which triggers the hook with an `Undo`). The design decision between these approaches must be made deliberately before implementation.

**Warning signs:**
- Remote instance logs show `Undo{Follow}` activities where the Follow's actor IRI is the remote instance (not the local one)
- Peers continue showing the connection as "following" after disconnect because they ignored the semantically incorrect Undo

**Phase to address:**
Phase implementing "Disconnect" API endpoint — design the direction check before writing the delete handler.

---

### Pitfall 4: Remote Actor Fetch Silently Returns Stale Data Due to 2-Hour Cache

**What goes wrong:**
`GetActorByIRI` (actor.go line 98) checks `last_fetched` before making a remote HTTP call. If the actor record was fetched less than 2 hours ago, `assembleActor` returns the cached record without re-fetching (line 222-224 of actor.go). An admin who pastes a remote URL minutes after the instance actor was first cached during a previous failed connect attempt gets stale actor data silently — no error, no re-fetch, no indication the cached data may be outdated.

**Why it happens:**
The 2-hour TTL is a performance guard for normal actor profile lookups. For the discovery flow ("admin pastes URL, we fetch their actor"), the admin expects fresh authoritative data. The discovery endpoint makes a new decision based on cached data that could predate a key rotation or inbox URL change on the remote instance.

**How to avoid:**
The actor discovery endpoint should bypass the cache. Before calling `GetActorByIRI`, if an actor record exists for the given IRI, set its `last_fetched` to zero time so the cache check in `assembleActor` always misses and triggers a fresh fetch. The simpler one-liner before calling `GetActorByIRI`:

```go
if existing, err := app.FindFirstRecordByData("activitypub_actors", "iri", remoteIRI); err == nil {
    existing.Set("last_fetched", time.Time{})
    _ = app.Save(existing)
}
```

**Warning signs:**
- Actor `public_key` or `inbox` fields shown in the UI differ from what the remote instance currently publishes
- Follow delivery fails with an HTTP signature error because the stored public key is stale

**Phase to address:**
Phase implementing "Remote actor discovery" endpoint — when writing the route that accepts a URL from the admin.

---

### Pitfall 5: Self-Follow Attempt Creates a Follow Loop

**What goes wrong:**
An admin who pastes their own instance's URL (or a URL that resolves to this instance's actor IRI) causes `GetActorByIRI` to return the local instance actor record. A follows record is then created with `follower = local_instance_actor_id` and `followee = local_instance_actor_id`. `InstanceFollowCreateHandler` fires, `isOutboundInstanceFollow` returns true (follower is the instance actor), and `CreateFollowActivity` dispatches a Follow to the local instance's own inbox. The local inbox creates a second follows record with the same IDs, triggering the hook again.

**Why it happens:**
No guard prevents `follower == followee` at the API level or the DB level. The composite unique index on `(follower, followee)` in the follows collection prevents a second identical row, so the loop terminates on the second attempt — but the first iteration has already delivered a Follow activity to the local inbox, which then fires `ProcessFollowActivity` and creates an inbound follows record.

**How to avoid:**
The "Initiate Follow" endpoint must compare the resolved remote actor's IRI against `os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"` before creating any follows record. Reject with `400 Bad Request` if they match:

```go
instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
if remoteActor.GetString("iri") == instanceIRI {
    return e.BadRequestError("Cannot follow own instance actor", nil)
}
```

**Warning signs:**
- `follows` table has a row where `follower == followee`
- Instance inbox logs show a Follow activity from the local instance IRI to itself

**Phase to address:**
Phase implementing "Remote actor discovery" and "Initiate Follow" endpoints — add the IRI equality check before any follows record is created.

---

### Pitfall 6: fetchRemoteActor Accepts Non-AP JSON Silently

**What goes wrong:**
`fetchRemoteActor` (actor.go line 287) uses `json.Decode` on the response body regardless of `Content-Type`. If the admin pastes a plain URL that redirects to a REST API endpoint returning generic JSON, the decode may partially succeed and pass `validateActorResponse` — which only checks for the presence of `id`, `inbox`, `outbox`, and `publicKey` fields. A non-AP JSON response that happens to have those field names creates a malformed actor record with wrong values stored in the database.

Additionally, if the URL returns HTML (a homepage), the JSON decoder returns an error that surfaces as a cryptic 500 to the admin instead of a descriptive "this is not an ActivityPub actor" message.

**How to avoid:**
Before decoding, check that `resp.Header.Get("Content-Type")` contains `application/activity+json` or `application/ld+json`. Reject any other content type with a descriptive error at the discovery endpoint level:

```go
ct := resp.Header.Get("Content-Type")
if !strings.Contains(ct, "application/activity+json") && !strings.Contains(ct, "application/ld+json") {
    return fmt.Errorf("remote URL did not return an ActivityPub actor (Content-Type: %s)", ct)
}
```

**Warning signs:**
- Actor record created with empty `inbox` field but no error returned
- Admin UI shows "discovery succeeded" but follow delivery immediately fails

**Phase to address:**
Phase implementing "Remote actor discovery" endpoint.

---

### Pitfall 7: SSRF via Admin-Supplied URL if Not Using SafeHTTPClient

**What goes wrong:**
`fetchRemoteActor` correctly uses `util.SafeHTTPClient()`, which blocks private/loopback IPs and applies rate limiting. However, any new "discovery" endpoint written in the routes layer that uses a raw `http.Get()` or the module-level `httpClient` variable defined in `federation/activity.go` (line 29: `var httpClient = &http.Client{Timeout: 10 * time.Second}`) bypasses SSRF protection entirely. An admin could be tricked into pasting `http://192.168.1.1/admin` or `http://169.254.169.254/latest/meta-data/`.

**Why it happens:**
The safe client is in the `util` package and must be explicitly imported and called. Code written quickly in a new route file may use `http.DefaultClient` or the module-level `httpClient` variable from `federation/activity.go` without realizing the SSRF implications. Both compile without error.

**How to avoid:**
Any HTTP call in the new admin federation routes that fetches a user-supplied URL must use `util.SafeHTTPClient()`. This is already the pattern for `GetActorByIRI` and `iriFromHandle`. Add a code review checklist item: "Never use `http.DefaultClient` or `httpClient` from activity.go for user-supplied URLs."

**Warning signs:**
- New route file uses `http.Get(remoteURL)` or `httpClient.Do(req)` without the `util` prefix
- Code review misses it because the import compiles fine without `util.SafeHTTPClient`

**Phase to address:**
Phase implementing "Remote actor discovery" endpoint — enforced in code review checklist.

---

## Moderate Pitfalls

### Pitfall 8: Admin Routes Protected by `e.Auth != nil` Instead of `e.HasSuperuserAuth()`

**What goes wrong:**
New admin federation routes protected with `if e.Auth == nil { return Unauthorized }` allow any logged-in regular user to access federation management. PocketBase distinguishes `e.Auth` (any record auth — a user from any collection with a valid token) from `e.HasSuperuserAuth()` (superuser token from the `_superusers` collection). A regular user can obtain a valid PocketBase user token and call the federation admin endpoints.

**Why it happens:**
`e.Auth != nil` only checks that a valid token was presented; it does not check the identity's privilege level. The existing `PluginSystemPluginsList` route (routes/plugin_system.go line 15) demonstrates the correct pattern: `if e.Auth == nil && !e.HasSuperuserAuth()`.

**How to avoid:**
All new admin federation routes must use `e.HasSuperuserAuth()` as the sole auth gate:

```go
if !e.HasSuperuserAuth() {
    return apis.NewUnauthorizedError("superuser authentication required", nil)
}
```

If the UI uses a SvelteKit page, the SvelteKit server must verify the PocketBase superuser token before rendering or forwarding API calls — never expose superuser operations to client-side JavaScript.

**Warning signs:**
- Any regular user who knows the endpoint URL can list, modify, or delete instance peers
- API endpoint returns 200 with a regular user's auth token in integration tests

**Phase to address:**
Phase implementing admin federation API endpoints — enforced at route registration time, before any other logic in the handler.

---

### Pitfall 9: Superuser Token Expiry Causes Silent UI Failure

**What goes wrong:**
PocketBase superuser tokens have a bounded validity window (the default admin UI token is approximately 30 minutes). A long-running admin UI session will silently fail API calls once the token expires. If the UI does not handle 401 responses from the Go backend and prompt re-authentication, the admin sees a blank connection list or an ambiguous error mid-session.

**Why it happens:**
The PocketBase JS SDK auto-refreshes regular user auth tokens. Superuser tokens issued for Go custom routes must be managed explicitly. The SvelteKit `locals.pb` instance holds the token but does not automatically refresh it for superuser sessions on custom Go routes.

**How to avoid:**
The UI must intercept 401 responses from the federation admin API and either redirect to the PocketBase admin login or surface a clear "Session expired — please log in again" message. Alternatively, design the admin federation UI as a custom PocketBase admin panel extension (if PocketBase UI extensions become stable) so session management is handled by the PocketBase admin framework itself.

**Warning signs:**
- Admin UI federation list goes blank after ~30 minutes with no error message
- Browser network tab shows 401 responses on federation API calls while the admin thinks they are logged in

**Phase to address:**
Phase implementing the UI auth flow — design the token refresh/expiry strategy before building the frontend.

---

### Pitfall 10: Stale Dashboard State After Remote Accepts the Follow

**What goes wrong:**
The connection dashboard shows status (pending/accepted/rejected) derived from the follows table. If the dashboard does not poll or subscribe after initiating a Follow, the UI continues to show "pending" even after the remote instance has sent back an Accept. The remote's Accept arrives at the local instance inbox, which calls `ProcessAcceptActivity` and updates the follows record status to `accepted` — but the dashboard rendered earlier has no live subscription to that change.

**Why it happens:**
ActivityPub Accepts arrive asynchronously (seconds to hours after the Follow, depending on the remote admin). The dashboard rendered at load time does not reflect subsequent updates unless it re-fetches or subscribes to the `follows` collection via PocketBase SSE realtime.

**How to avoid:**
The dashboard should subscribe to the `follows` collection via PocketBase's realtime API (`pb.collection('follows').subscribe('*', callback)`) so status changes from incoming Accepts or Rejects are reflected immediately without manual refresh. As a fallback, poll the follows list on a short interval (every 20-30 seconds) while any connection is in "pending" status.

**Warning signs:**
- Admin initiates Connect, remote accepts, local UI still shows "Pending" indefinitely
- Admin initiates Connect twice because the UI did not update to show "Pending" after the first attempt

**Phase to address:**
Phase implementing the connection dashboard UI.

---

### Pitfall 11: `activitypub_activities` Record Missing Blocks Approve Flow

**What goes wrong:**
`CreateAcceptFollowActivity` and `CreateRejectFollowActivity` (federation/follow.go) look up the original incoming Follow activity from `activitypub_activities` by filter `actor={remote_IRI}&&object={local_IRI}&&type=Follow`. This record is created by `ProcessFollowActivity` when the remote instance's Follow arrives at the local inbox. If the admin creates a follows record through a path other than the inbox handler (e.g., a future "manually add peer" feature, or a database seed), that prerequisite record does not exist. The approve endpoint then fails with `sql.ErrNoRows` and the Accept is never delivered — surfacing as a 500 to the admin.

**Why it happens:**
The approve flow has an implicit dependency on a prior `ProcessFollowActivity` execution. This dependency is not expressed in the database schema — there is no foreign key from follows to activitypub_activities.

**How to avoid:**
The "Approve inbound follow" endpoint must verify that the corresponding `activitypub_activities` Follow record exists before calling update on the follows status. If it is missing, return `409 Conflict` with the message "No incoming Follow activity found for this peer — cannot send Accept". As a hardening step, consider creating a synthetic activity record in this case rather than failing.

**Warning signs:**
- "Approve" API call returns 500 with a `sql.ErrNoRows` error
- `activitypub_activities` has no row matching the follower/followee actor IRI pair

**Phase to address:**
Phase implementing "Approve/Reject" API endpoint.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| UI polls follows table every 10s instead of realtime subscription | Simpler to implement | Polling adds DB load; misses rapid state changes under network delay | Acceptable for MVP only if PocketBase realtime subscription is unfamiliar territory |
| Disconnect always hard-deletes the follows record (triggering Undo hook regardless of direction) | No directional logic needed | Sends semantically incorrect Undo for inbound-only follows; remote may ignore and retain connection | Never — distinguish inbound vs outbound before delete |
| Use `http.DefaultClient` in new discovery route | Shorter code, no import needed | SSRF vulnerability; bypasses rate limiter | Never |
| No self-follow check in discover endpoint | Skips one IRI comparison | Infinite Follow loop fires hooks; fills follows table | Never |
| Skip `Content-Type` check on actor fetch response | No change to `fetchRemoteActor` signature | Silently accepts non-AP JSON; creates actor records with empty inbox | Never in the production discovery path |
| Use `e.Auth != nil` instead of `e.HasSuperuserAuth()` | Shorter guard condition | Any regular user can manage instance peers | Never |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `InstanceFollowCreateHandler` hook | Calling `CreateFollowActivity` in the new route AND saving the record | Only save the follows record; the hook calls `CreateFollowActivity` |
| `InstanceFollowUpdateHandler` hook | Calling `CreateAcceptFollowActivity` in the approve route AND updating status | Only update the status; the hook calls `CreateAcceptFollowActivity` |
| `InstanceFollowDeleteHandler` hook | Calling `CreateUnfollowActivity` in the disconnect route AND deleting the record | Only delete/update the record; the hook handles delivery — but check direction first |
| `GetActorByIRI` 2-hour cache | Discovery endpoint returns stale actor without re-fetching | Clear `last_fetched` to zero before calling `GetActorByIRI` from the discovery endpoint |
| `validateActorResponse` | Trusts any JSON with id/inbox/outbox/publicKey fields | Add `Content-Type` check before decoding in `fetchRemoteActor` |
| `util.SafeHTTPClient()` | New route uses `http.DefaultClient` for user-supplied URL | Always use `util.SafeHTTPClient()` for any admin-supplied URL |
| `e.HasSuperuserAuth()` | Route uses `e.Auth != nil` as the auth guard | Use `e.HasSuperuserAuth()` — existing pattern in `PluginSystemPluginsList` |
| `activitypub_activities` Follow record | Approve endpoint called when no incoming activity record exists | Check for activity record existence; return 409 if missing |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Admin federation endpoints guarded by `e.Auth != nil` only | Any regular user can manage instance peers | Use `e.HasSuperuserAuth()` exclusively on all federation admin routes |
| User-supplied actor URL fetched via `http.DefaultClient` or `httpClient` module variable | SSRF to internal services (metadata endpoints, admin UIs, internal APIs) | Enforce `util.SafeHTTPClient()` — SSRF blocked, rate-limited, private IPs denied |
| Self-follow not guarded | Follow loop delivers activities to the local inbox; fills follows table | Compare resolved actor IRI to local instance IRI before creating follows record |
| No `Content-Type` validation on actor fetch response | Non-AP server creates malformed actor record silently; wrong inbox URL stored | Check `Content-Type` for `application/activity+json` or `application/ld+json` before decoding |
| Superuser token passed to client-side JavaScript | Token exposed in browser; reusable until expiry by any script that reads it | Keep superuser auth check and token storage server-side only (Go route or SvelteKit server function) |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| "Connect" button shows success immediately before remote accepts | Admin thinks connection is live; wonders why no content appears | Show "pending" status immediately with explanation "Waiting for remote admin to accept" |
| No feedback when actor fetch fails (timeout, 404, wrong URL) | Admin sees blank or generic 500; retries the same wrong URL | Surface specific error: "Remote URL returned 404", "Request timed out after 10s", "Not an ActivityPub actor (Content-Type: text/html)" |
| Dashboard refreshed only on page load | Accepted connections show as pending; admin tries reconnecting | Auto-refresh pending rows every 20-30s or subscribe to PocketBase realtime `follows` collection |
| Disconnect removes connection silently | Admin does not know if the remote was notified | Show confirmation: "Disconnect sent — remote may briefly still show the connection" |
| Multiple peers listed without direction indicator | Admin cannot tell who followed whom; cannot determine which peers require their own approval | Show direction explicitly: "You follow them", "They follow you", "Mutual" alongside status |
| Pending inbound follow shows no information about the requesting instance | Admin cannot make an informed approve/reject decision | Show remote instance domain, actor name, and NodeInfo software version for each pending inbound follow |

---

## "Looks Done But Isn't" Checklist

- [ ] **Initiate Follow endpoint:** Verify `CreateFollowActivity` is NOT called directly in the route handler — the hook must be the sole delivery path.
- [ ] **Actor discovery endpoint:** Verify `util.SafeHTTPClient()` is used, not `http.DefaultClient` or the `httpClient` module variable from `federation/activity.go`.
- [ ] **Actor discovery endpoint:** Verify `Content-Type` header is checked before decoding the response body.
- [ ] **Actor discovery endpoint:** Verify self-follow (remote IRI equals local instance IRI) is rejected before any follows record is created.
- [ ] **Approve endpoint:** Verify the corresponding `activitypub_activities` Follow record exists and returns 409 (not 500) when missing.
- [ ] **Approve/Reject endpoints:** Verify `CreateAcceptFollowActivity`/`CreateRejectFollowActivity` are NOT called directly in the route handler.
- [ ] **Disconnect endpoint:** Verify inbound vs outbound direction is checked before deleting or updating the follows record.
- [ ] **All admin federation routes:** Verify `e.HasSuperuserAuth()` is the auth guard, not `e.Auth != nil`.
- [ ] **Dashboard UI:** Verify pending connections auto-refresh or subscribe to realtime updates without manual page reload.
- [ ] **UI token handling:** Verify 401 responses from the Go backend redirect to re-authentication instead of showing a blank state or a generic error.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Double Follow delivery | LOW | Remote will deduplicate (standard AP idempotency via `ProcessFollowActivity` existing-record check); no local DB cleanup needed if no duplicate follows row exists |
| Double Accept delivery | LOW | Remote ignores duplicate Accepts; cosmetic only — no local action required |
| Incorrect Undo sent for inbound follow | MEDIUM | Manually re-create the inbound follows record in DB with correct direction; contact remote admin to re-send Follow if they processed the Undo |
| Self-follow loop (before unique index stops it) | MEDIUM | Delete all self-referential follows rows from DB; verify no in-flight goroutines remain (check activity logs) |
| Stale actor record stored after bad discovery | LOW | Delete the actor record from `activitypub_actors`; retry discovery — `GetActorByIRI` will re-fetch on cache miss |
| Missing `activitypub_activities` record blocking Approve | MEDIUM | Manually insert the expected Follow activity row with correct actor/object/type IRIs matching the remote follower and local instance; retry Approve |
| Superuser token expiry causes blank UI | LOW | Admin re-authenticates; no data loss |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Double Follow delivery (Pitfall 1) | "Initiate Follow" API endpoint | Integration test: create follows record via endpoint; verify exactly one Follow row in `activitypub_activities` |
| Double Accept delivery (Pitfall 2) | "Approve/Reject" API endpoint | Integration test: approve via endpoint; verify single Accept row in `activitypub_activities` |
| Incorrect Undo direction (Pitfall 3) | "Disconnect" API endpoint | Unit test: call disconnect on inbound follow; verify `Reject` or status change, not `Undo` delivery |
| Stale actor cache in discovery (Pitfall 4) | "Remote actor discovery" endpoint | Test: cache an actor record, then call discovery endpoint; verify `last_fetched` reset and fresh fetch occurs |
| Self-follow loop (Pitfall 5) | "Remote actor discovery" + "Initiate Follow" validation | Test: submit own instance URL; verify 400 returned and no follows row created |
| Non-AP content-type accepted (Pitfall 6) | "Remote actor discovery" endpoint | Test: point discovery at a JSON REST endpoint; verify 400 with content-type error message |
| SSRF via user-supplied URL (Pitfall 7) | "Remote actor discovery" endpoint | Test: submit `http://127.0.0.1/` and `http://192.168.1.1/`; verify both return 400 blocked |
| Wrong auth guard (Pitfall 8) | All admin federation API endpoints | Test: call each endpoint with regular user token; verify 401 returned |
| Superuser token expiry (Pitfall 9) | UI auth flow design | Manual test: leave UI open 35 minutes; verify 401 is handled with re-authentication prompt |
| Stale dashboard state (Pitfall 10) | Connection dashboard UI | Manual test: approve on remote side; local dashboard should update within 30s without manual refresh |
| Missing `activitypub_activities` for Approve (Pitfall 11) | "Approve" API endpoint | Test: attempt approve when no matching activity record exists; verify 409 (not 500) |

---

## Sources

- Direct code analysis: `db/hooks/follow.go` — `isOutboundInstanceFollow`, `isInstanceFollow`, `InstanceFollowCreateHandler`, `InstanceFollowUpdateHandler`, `InstanceFollowDeleteHandler`
- Direct code analysis: `db/federation/follow.go` — `CreateFollowActivity`, `ProcessFollowActivity`, `CreateAcceptFollowActivity`, `CreateRejectFollowActivity`
- Direct code analysis: `db/federation/actor.go` — `GetActorByIRI`, `assembleActor`, `fetchRemoteActor`, `validateActorResponse`, 2-hour cache logic (line 222-224)
- Direct code analysis: `db/federation/activity.go` — `PostActivity`, module-level `httpClient` variable (line 29)
- Direct code analysis: `db/routes/plugin_system.go` — `e.HasSuperuserAuth()` guard pattern (line 15)
- Direct code analysis: `db/util/network.go` — `SafeHTTPClient`, SSRF protection, `isPrivateOrReservedIP`
- Direct code analysis: `db/main.go` — hook registration order; `OnRecordAfterCreateSuccess("follows")` binding at line 132
- Direct code analysis: `db/federation/instance.go` — `InstanceInboxHandler`, `ProcessFollowActivity` path for inbound follows
- Project context: `.planning/PROJECT.md` — v1.0 known gaps, key decisions, v1.1 active requirements

---
*Pitfalls research for: Federation Connect UI (v1.1)*
*Researched: 2026-06-27*
