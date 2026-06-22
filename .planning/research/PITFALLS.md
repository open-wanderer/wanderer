# Domain Pitfalls: ActivityPub Instance-Level Federation

**Domain:** ActivityPub instance actor (Application type) federation, Wanderer-to-Wanderer instance sync
**Researched:** 2026-06-22
**Confidence:** HIGH — grounded in existing Wanderer federation code, ActivityPub spec (W3C), and Mastodon implementation patterns

---

## Critical Pitfalls

Mistakes that trigger data storms, privacy leaks, or require architectural rewrites.

---

### Pitfall 1: Broadcast Loop — Instance A Re-Announces Content from Instance B Back to Instance B

**What goes wrong:**

When Instance B sends a `Create` activity to Instance A's instance actor inbox, Instance A stores the trail locally (via `processCreateOrUpdateTrailActivity`). If the *local save hook* then fires `CreateTrailActivity` for the newly-stored record, it calls `followerInboxes(app, trailAuthor.Id)` — which now includes Instance B (because Instance B is a follower of that remote actor's local copy). The activity returns to Instance B, which re-processes it, potentially creating another local copy and firing another `Create` downstream.

**Why it happens in Wanderer specifically:**

The existing hook in `db/hooks/trails.go` guards against re-broadcast with `if !userActor.GetBool("is_local") { return nil }` — this correctly skips federation for incoming remote content. However, this guard exists on trail hooks. When the instance actor is added, the fanout logic for the instance actor itself also needs this same "is this activity originating locally or was it received from a peer?" check. Without it, the instance actor will re-deliver activities it *received* to all its own followers (other instances), who will do the same.

**The W3C spec mechanism:**

The ActivityPub spec mandates (§7.1.2): *"This is the first time the server has seen this Activity"* as a prerequisite for inbox forwarding. It also requires servers to MUST de-duplicate the final recipient list. The practical implementation is an `activitypub_activities` IRI check before any re-delivery: if the activity IRI is already recorded in `activitypub_activities`, drop it immediately.

**Consequences:**

- Exponential activity storm across all connected instances
- Each instance's `PostActivity` goroutine pool (capped at 5 concurrent requests in `activity.go`) gets saturated
- CPU/memory spike that mimics a DDoS (the ActivityPub spec warns about this explicitly)
- Data duplication: multiple local copies of the same trail record

**Prevention:**

At the top of `ActivitypubActivityProcess` in `activitypub.go`, before dispatching to any processor, check whether the incoming activity IRI already exists in `activitypub_activities`:

```go
existing, err := app.FindFirstRecordByData("activitypub_activities", "iri", activity.GetID().String())
if err == nil && existing != nil {
    return e.JSON(http.StatusOK, nil) // already seen — drop silently
}
```

Additionally: the instance actor's fanout — when it delivers activities to peer instances — must only deliver activities *created locally*, never relay activities that arrived from a remote instance. The `is_local` check on the originating actor's record is the correct signal. This guard already exists for user actors; it must be explicitly preserved and tested when the instance actor is introduced into `followerInboxes`.

**Detection (warning signs):**

- Duplicate `iri` values appearing in `activitypub_activities`
- Rapidly growing `trails` or `summit_logs` tables with duplicate IRI fields
- The 5-slot semaphore in `PostActivity` consistently saturated (visible via goroutine dumps or slow inbox response times)

**Phase:** Address in the phase that implements the instance actor's inbox processing and fanout inclusion.

---

### Pitfall 2: Private Content Leak Through Missed `is_public` Check at Instance Fanout

**What goes wrong:**

When an instance actor is added to `followerInboxes`, activities for *all* followers of a given content actor are delivered — including the instance actor representing the peer. The public check `if !trail.GetBool("public") { return nil }` exists in `CreateTrailActivity` today (line 22 of `create.go`). But there are two leak vectors:

1. **Visibility change after initial publish:** A trail is created public (activity sent), then later set `public = false`. The `UpdateTrailActivity` (called via `UpdateTrailHandler`) still calls `CreateTrailActivity` with `pub.UpdateType`. If the `if !trail.GetBool("public")` check is read from the incoming record *after* the update (which it is — it reads `trail.GetBool("public")`), a `public = false` update correctly suppresses the Update activity. But the *original Create* is already federated. The trail copy sits on the peer and the peer has no Delete signal. (This is a Delete propagation gap — see Pitfall 4 — but starts as a privacy issue.)

2. **Hook fires on remote-content saves:** `CreateTrailHandler` fires on *all* trail record saves, including when `processCreateOrUpdateTrailActivity` saves a remote trail locally. The `is_local` guard in `CreateTrailHandler` prevents outbound federation, but `public` field of a remote trail is whatever the remote actor set it to. If a remote instance sends a private trail in an activity (a bug on the sending side), Wanderer must not trust the `public` flag from the remote — it must verify the activity was addressed to the public audience constant (`https://www.w3.org/ns/activitystreams#Public` in the `to` field).

**Prevention:**

- Keep and test the `if !trail.GetBool("public") { return nil }` guard in *every* Create/Update/Delete fanout function — do not inline or skip it for the instance actor path.
- In `ProcessCreateOrUpdateActivity`, verify the incoming activity's `to` field contains `as:Public` before accepting it as public content. Reject (log and return nil) activities from remote actors claiming public content that do not address the public collection.
- When a trail's `public` field flips to `false`, trigger a `CreateTrailDeleteActivity` to all peers so cached copies are removed. The current `UpdateTrailHandler` does not do this — it would need a before/after comparison of the `public` field.

**Detection:**

- Query peer instances for trail IRIs that were made private locally; if they still serve the content, the Delete was missed.
- Log every outbound activity and verify none have `public = false` records.

**Phase:** Treat the `public = false → Delete` propagation as a first-class requirement before instance federation goes live.

---

## Critical Pitfall 3: HTTP Signature Mismatch for the Application Actor

**What goes wrong:**

The instance actor has a different key ID than user actors. In `PostActivity`, the `pubID` is built as `actor.GetString("iri") + "#main-key"`. For user actors this is something like `https://wanderer.example.com/api/v1/activitypub/actor/abc123#main-key`. For the instance actor it would be `https://wanderer.example.com/api/v1/activitypub/actor/instance#main-key` (or whatever IRI is chosen). The receiving instance must be able to fetch *that specific actor URL* and find the `publicKey.publicKeyPem` field.

There are three common failure modes unique to the Application actor:

1. **WebFinger doesn't resolve the instance actor.** WebFinger is normally keyed by `acct:username@domain`. If the instance actor's `preferred_username` is `instance`, remote servers doing `GET /.well-known/webfinger?resource=acct:instance@wanderer.example.com` must get back the instance actor's IRI. If WebFinger is not extended to cover the Application actor, remote instances cannot resolve the key and every signature verification fails with 401.

2. **Key fetch requires authentication that the instance actor can't provide.** If the remote instance runs in "secure mode" (Mastodon's authorized fetch), every GET to a resource must itself be signed. The instance actor must be able to sign these key-fetch GET requests. In the current `fetchRemoteActor` flow, the signing actor is retrieved from `ctx.Value("actor")` — this context key is set per-request by the web handler. For background deliveries made by the instance actor, no request context exists. The signing must explicitly use the instance actor's key.

3. **`Content-Type` negotiation for Application actors.** Some ActivityPub implementations treat Application-type actors differently and may return `406 Not Acceptable` or an HTML page instead of JSON-LD when fetching the actor profile if the `Accept` header isn't exactly `application/activity+json`. The existing `fetchRemoteActor` sends `Accept: application/ld+json` — this works for most servers but can fail for strict implementations. Use `application/activity+json, application/ld+json` as the Accept value.

**Prevention:**

- Extend the WebFinger endpoint (`/.well-known/webfinger`) to return the instance actor for `acct:instance@{domain}` (or whatever `preferred_username` is chosen).
- Ensure the instance actor record has `private_key` populated and encrypted at startup, just like user actors.
- In the background delivery goroutines in `PostActivity`, always load the signing actor from the database by ID, not from a request context — the instance actor's ID should be resolvable without a live HTTP request context.
- Test signature verification by having a peer instance attempt to verify an inbound activity from the instance actor before considering the feature complete.

**Detection:**

- Peer instances returning `401 Unauthorized` on inbound posts from the instance actor
- Peer instances returning `404` when trying to fetch `{origin}/api/v1/activitypub/actor/instance`
- WebFinger returning no results for `acct:instance@{domain}`

**Phase:** Must be verified in the phase that creates the instance actor at startup.

---

## Critical Pitfall 4: Delete Activity Propagation Failures Leave Stale Copies on Peers

**What goes wrong:**

`CreateTrailDeleteActivity` (`delete.go` line 16) correctly guards `if !author.GetBool("is_local") { return nil }`. This means only locally-authored trails trigger outbound Delete activities — correct. However, the Delete is only triggered from the trail's `OnDelete` hook. Three scenarios leave stale copies:

1. **Trail visibility change (public → private):** When a trail's `public` field is set to false, no Delete activity is sent. The trail copy on peer instances remains permanently. Users on peer instances can continue viewing content the original author has effectively retracted.

2. **Delete of a remote-origin trail:** When Instance B deletes a trail it originally authored and sends a `Delete` to Instance A, Instance A's `processDeleteTrailActivity` (line 293) deletes the local record. But if Instance A also received a re-broadcast of that trail from Instance C (via instance-level fanout), Instance C may not receive Instance A's re-propagation of the Delete — because the re-propagation of Delete activities faces the same loop-prevention problem described in Pitfall 1.

3. **Delete verification gap:** `ProcessDeleteActivity` (line 266) checks `if actor.GetBool("is_local") { return nil }` — it skips processing for local actors (correct). But it does *not* verify that the deleting actor is actually the author of the content being deleted. `processDeleteCommentActivity` (line 309) does check `comment.GetString("author") != actor.Id`. `processDeleteTrailActivity` (line 293) does *not* make this check. A malicious or buggy remote actor could send a Delete for another actor's trail IRI and Wanderer would delete the locally-cached copy.

**Prevention:**

- Add a `public` field change detector in `UpdateTrailHandler`: compare `original.GetBool("public")` with `record.GetBool("public")`. If it transitions `true → false`, call `CreateTrailDeleteActivity` before saving.
- In `processDeleteTrailActivity`, verify the deleting `actor` matches the trail's `author` field: `if trail.GetString("author") != actor.Id { return fmt.Errorf("actor is not trail author") }`.
- For Delete activities arriving from peer instance actors, verify the object IRI's origin domain matches the sending actor's domain as a minimum check.

**Detection:**

- Trails that are private on the origin instance but still publicly accessible on peer instances
- Any `Delete` activity in logs where `actor.Id != trail.author`

**Phase:** The ownership verification fix should be included as a hardening step in the first phase that implements instance actor inbox processing.

---

## Moderate Pitfalls

---

### Pitfall 5: Key Rotation Breaks All Pending and In-Flight Signature Verifications

**What goes wrong:**

If the instance actor's `POCKETBASE_ENCRYPTION_KEY` or the RSA private key stored in the `activitypub_actors` record is regenerated (e.g., during a server rebuild or accidental re-initialization), all previously-issued signatures become invalid. Peer instances cache the public key from the actor JSON. Until their cache expires (Wanderer's cache TTL is 2 hours, per `twoHoursAgo` in `assembleActor`), every inbound activity from the regenerated instance will fail HTTP signature verification with 401.

The ActivityPub ecosystem has no standardized key rotation notification mechanism. Sending an `Update` activity for the instance actor with the new public key is the closest convention, but it requires the old key to sign it (a paradox if the old key is lost). Most implementations handle this through a "retry after re-fetch" pattern: on signature failure, re-fetch the actor document, update the cached public key, and retry.

Wanderer's current `ActivitypubActivityProcess` does not implement this retry-on-401 pattern. It verifies once and returns 401 on failure.

**Prevention:**

- Treat the instance actor's key pair as immutable infrastructure. Store it separately from PocketBase data (e.g., as an environment-provided secret) so it survives database rebuilds.
- Alternatively, implement the retry pattern: on signature verification failure, invalidate the cached actor and re-fetch before failing permanently.
- Document clearly that re-generating `POCKETBASE_ENCRYPTION_KEY` requires notifying all peer instances to flush their actor cache or re-establish federation.
- Do not auto-regenerate the instance actor key pair on startup if the actor record already exists.

**Detection:**

- Sudden `401 Unauthorized` responses from all peer instances simultaneously
- Log entries showing `Failed to parse private key` or `Invalid http signature` spikes

**Phase:** Startup initialization phase for the instance actor must include a "key already exists → do not overwrite" guard.

---

### Pitfall 6: Accept/Reject Race Condition in Mutual Instance Follow Approval

**What goes wrong:**

The mutual approval flow for instance follows is: Admin A sends Follow → Admin B accepts → sync begins. The current `ProcessFollowActivity` (follow.go line 64) auto-accepts any incoming Follow from a remote actor immediately. This behavior works for user-level follows where any remote user can follow a local user. For instance-level follows it is the wrong policy: the `PROJECT.md` requirement is *"Remote admin must accept the Follow before sync begins (mutual approval)."*

The race condition is: if Admin A sends a Follow to Instance B while Instance B simultaneously sends a Follow to Instance A, both sides auto-accept (under current behavior) and both sides start receiving all content from the other — before either admin has reviewed the connection. Content from an untrusted or misconfigured peer starts populating the local database.

**Prevention:**

- When the actor sending a Follow is `type = Application` (an instance actor), do *not* auto-accept. Instead, set the follow record status to `pending` and require an explicit admin action to accept.
- This is a type-check on the incoming actor: `if actor.GetString("type") == "Application" { followRecord.Set("status", "pending") }` — the auto-accept path in `ProcessFollowActivity` must be gated on actor type.
- The Accept activity must then be sent manually when the admin approves via the PocketBase admin UI.

**Detection:**

- Follow records appearing with `status = accepted` for Application-type actors without admin action
- Content from unknown instances appearing in local feeds without admin approval

**Phase:** The follow handling change is foundational — it must be implemented before the instance actor is activated.

---

### Pitfall 7: Object Ownership Confusion on Update — Remote Content Modified by Wrong Actor

**What goes wrong:**

The ActivityPub spec (§7.3) states: *"The receiving server MUST take care to be sure that the Update is authorized to modify its object. At minimum, this may be done by ensuring that the Update and its object are of same origin."*

`ProcessCreateOrUpdateActivity` (create.go line 412) dispatches to `processCreateOrUpdateTrailActivity` which calls `util.TrailFromActivity` — this function upserts based on the trail's IRI. If two different remote actors both send `Update` activities referencing the same trail IRI (possible if one is a buggy relay), the second update overwrites the first without any actor-ownership check.

**Prevention:**

- Before applying an Update to a locally-cached remote trail, verify `trail.GetString("author") == actor.Id`. If not, log the discrepancy and discard the Update.
- The same check should apply to summit_logs and lists.
- This check is already present for comments (`processCreateOrUpdateCommentActivity` does not overwrite — it uses upsert by IRI) but trails, summit_logs, and lists need the actor ownership guard.

**Detection:**

- Trail records whose `author` field does not match the `actor` IRI in the last received Update activity
- Unexpected field value changes on locally-cached remote trails

**Phase:** Address in the phase that implements Update activity processing for the instance actor path.

---

### Pitfall 8: Fan-Out Storm When a High-Volume Instance First Connects

**What goes wrong:**

`PROJECT.md` explicitly states: *"No historical backfill on new connection — only new activities after the Follow is accepted."* This is the correct decision. However, the risk is in *what happens at the moment of Accept*: if the Follow Accept triggers any outbox sync (the commented-out `util.SyncOutbox` call in `ProcessAcceptActivity`, line 155 of `follow.go`), and that code is ever uncommented or reimplemented, it could result in delivering thousands of stored activities at once.

The secondary fan-out risk: when a large instance (Instance B with 10,000 trails) accepts a Follow from Instance A, any user on Instance B who creates a new trail will immediately fan-out to Instance A's inbox. If many users are active simultaneously, Instance A's inbox handler will receive a burst of concurrent POST requests. Wanderer's `PostActivity` goroutine model uses a semaphore of 5 (`semaphore.NewWeighted(5)` in `activity.go` line 99) — this is per-*outbound* call. The *inbound* inbox endpoint has no rate limiting.

**Prevention:**

- Keep `SyncOutbox` permanently deleted or commented out. Do not reintroduce it.
- Add per-source-instance rate limiting to `ActivitypubActivityProcess`: if a single remote IP or actor domain exceeds N requests per second, return `429 Too Many Requests`. PocketBase middleware can enforce this.
- Log and alert when inbound activity queue depth exceeds a threshold.

**Detection:**

- Inbox endpoint response times spiking immediately after a Follow Accept
- Go goroutine count spiking in backend metrics
- Database write latency increasing after a new instance connection is established

**Phase:** Rate limiting should be planned in the same phase as instance actor inbox processing.

---

## Minor Pitfalls

---

### Pitfall 9: Activity IRI Collision Between User and Instance Actor Namespaces

**What goes wrong:**

Currently, all activity IRIs are constructed as `{ORIGIN}/api/v1/activitypub/activity/{recordId}`. Instance actor activities will also use this format. If the instance actor's ID string overlaps with any existing user actor path, remote servers may dereference the wrong actor when resolving the `actor` field of an activity.

**Prevention:**

- The instance actor's IRI should use a fixed, clearly distinct path: `{ORIGIN}/api/v1/activitypub/actor/instance` (not a random ID). This makes it unambiguous and resolvable.
- Reserve `instance` as the `preferred_username` for the Application actor — ensure no user can register with that username.

**Phase:** Startup initialization.

---

### Pitfall 10: `processDeleteCommentActivity` Skips Delete if Trail Author is Local

**What goes wrong:**

In `delete.go` line 101-103: `if commentTrailAuthor.GetBool("is_local") { return nil }`. This means: if a local user authors a trail, and a remote user posts a comment on that trail, and then deletes the comment, Wanderer does *not* send a Delete activity for the comment. The comment deletion is handled locally. This is correct for comments on local trails — but when the instance actor enables sync, the comment may have been replicated to peer instances via instance-level fanout. Those peer instances will not receive the Delete and will retain the stale comment indefinitely.

**Prevention:**

- When instance-level federation is active, `CreateCommentDeleteActivity` must also target peer instances that received the comment via instance actor fanout, not just the trail author.
- The simplest fix: call `followerInboxes(app, instanceActor.Id)` in addition to the trail author inbox when sending comment Delete activities.

**Phase:** Identified during integration testing — should be flagged for the Delete propagation phase.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Instance actor creation at startup | Key overwrite on restart destroys federation (Pitfall 5) | Check if actor record exists before generating keypair |
| Instance actor WebFinger registration | Signature verification fails on all peer GET requests (Pitfall 3) | Extend WebFinger handler before testing any peer connection |
| Follow handler for Application actors | Auto-accept connects untrusted peers without admin review (Pitfall 6) | Type-check actor before setting status = accepted |
| Instance actor added to `followerInboxes` | Re-broadcast storm if incoming activities are re-delivered (Pitfall 1) | Activity IRI deduplication check at inbox entry point |
| Privacy field enforcement in fanout | Private trails leaked to peer instances (Pitfall 2) | `public = false → Delete` propagation must be implemented |
| Update activity processing | Remote content overwritten by wrong actor (Pitfall 7) | Actor ownership check before any upsert |
| Delete activity processing | Unauthorized deletion of locally cached trails (Pitfall 4) | Author-match check in `processDeleteTrailActivity` |
| First peer instance connection | Fan-out burst overwhelms inbox (Pitfall 8) | Rate limiting on inbound inbox POST |
| Comment delete propagation | Stale comments on peer instances (Pitfall 10) | Include instance actor followers in comment Delete fanout |

---

## Sources

- [ActivityPub W3C Recommendation — §7 Server-to-Server Interactions, §7.1.2 Inbox Forwarding](https://www.w3.org/TR/activitypub/)
- [Mastodon ActivityPub Documentation — Instance Actor, HTTP Signatures](https://docs.joinmastodon.org/spec/activitypub/)
- [ActivityPub HTTP Signatures — SWICG Draft](https://swicg.github.io/activitypub-http-signature/)
- [Key Rotation Notification — SocialHub ActivityPub Discussion](https://socialhub.activitypub.rocks/t/key-rotation-notification/562)
- [Pixelfed Private Content Leak Vulnerability — Hachyderm Community Analysis](https://community.hachyderm.io/blog/2025/04/03/pixelfed-vulnerability-and-impacts-to-federation/)
- [Mastodon Source — activitypub/activity.rb deduplication patterns](https://github.com/mastodon/mastodon/blob/main/app/lib/activitypub/activity.rb)
- [ActivityPub Relay — pub-relay implementation (loop by Announce)](https://github.com/noellabo/pub-relay)
- [ActivityPub Must Verify Ownership on Delete — W3C Issue #294](https://github.com/w3c/activitypub/issues/294)
- [WordPress ActivityPub — Accept/Reject race condition PR #524](https://github.com/Automattic/wordpress-activitypub/pull/524)
- Wanderer codebase: `db/federation/activity.go`, `create.go`, `delete.go`, `follow.go`, `actor.go`, `db/routes/activitypub.go`, `db/hooks/trails.go`
