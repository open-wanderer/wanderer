# Research Summary: Wanderer Instance Federation

**Synthesized from:** STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md
**Overall confidence:** HIGH

## Executive Summary

Wanderer is adding instance-level ActivityPub federation so two Wanderer server admins can establish a bilateral Follow relationship, after which all public trails, summit_logs, lists, and comments sync bidirectionally. The approach follows the Mastodon instance-actor pattern: create an `Application`-type actor at `{ORIGIN}/api/v1/activitypub/instance`, reuse all existing go-ap and go-fed/httpsig libraries (no new dependencies), and extend the existing `activitypub_actors`/`follows`/`activitypub_activities` collections with one additive schema migration (`actor_type` column, default `"Person"`).

The protocol layer already exists — this feature is a targeted extension, not a rewrite.

## Stack

- **Actor type:** `pub.ApplicationType` (`"Application"`) — verified from live Mastodon instance at mastodon.social/actor and go-ap library source
- **IRI pattern:** `{ORIGIN}/api/v1/activitypub/instance` (Wanderer convention; Mastodon uses `/actor`)
- **Key ID:** `{actorIRI}#main-key` (matches existing user actor convention)
- **Libraries:** No new dependencies — `go-ap/activitypub`, `go-fed/httpsig` already support Application actors identically to Person actors
- **Schema change:** One migration — add `actor_type` text column to `activitypub_actors`, default `"Person"`; instance actor row gets `"Application"`
- **WebFinger:** Useful for peer discovery but can be deferred — admins can supply IRIs directly in v1

## Features (Table Stakes for v1)

1. Instance actor exists at a stable well-known IRI with RSA keypair
2. Instance actor GET endpoint returns valid ActivityPub JSON
3. Instance actor inbox POST endpoint accepts signed activities
4. Outgoing Follow to remote instance actor (admin-initiated via PocketBase)
5. Incoming Follow creates `pending` record (no auto-accept for Application actors)
6. Admin manually accepts/rejects pending instance follows in PocketBase UI
7. Accepted Follow triggers `Accept{Follow}` activity sent to remote inbox
8. Rejected Follow triggers `Reject{Follow}` activity (mandatory — remote gets stuck without it)
9. `Undo{Follow}` on disconnect removes the peer and stops fanout
10. All public trails/summit_logs/lists/comments fanned out to instance actor followers on Create/Update/Delete
11. Privacy gate: `is_public = false` records never included in fanout
12. Incoming activities to instance inbox are deduplicated (broadcast loop prevention)
13. Remote content retains original actor — users on receiving instance cannot edit

**Anti-features (explicitly out of scope for v1):**
- Historical backfill when a new connection is established
- Relay-style re-broadcasting (Announce pattern) — use direct bilateral Follow instead
- Auto-accept for Application-type instance actors
- Federation with non-Wanderer AP servers

## Architecture

**Six targeted component changes — no new subsystems:**

| Component | Change | Risk |
|-----------|--------|------|
| DB schema | Add `actor_type` column to `activitypub_actors` | LOW |
| `initData()` startup | Add `initInstanceActor()` (idempotent find-or-create) | LOW |
| SvelteKit routes | GET `/api/v1/activitypub/instance`, POST `/api/v1/activitypub/instance/inbox` | LOW |
| `ProcessFollowActivity` | Gate auto-accept on `actor_type != "Application"` | MEDIUM — touches existing code path |
| New hook `OnRecordAfterUpdateSuccess("follows")` | Send `Accept{Follow}` or `Reject{Follow}` when admin changes status | LOW |
| `followerInboxes()` / fanout functions | Merge instance actor's accepted followers into delivery set for each Create/Update/Delete | MEDIUM |

**Key constraint:** HTTP signatures on outbound deliveries must use the **content author's key**, not the instance actor key. The `actor` JSON field is the content author; signing with a mismatched key will fail remote verification.

**Fanout data flow (new trail on Instance A → Instance B):**
```
User creates trail on A
  → OnRecordAfterCreateSuccess("trails") hook
  → CreateTrailActivity() called with trail.author as actor
  → followerInboxes(author.id) UNION followerInboxes(instanceActor.id)
  → PostActivity() deduplicates and delivers
  → Instance B instance actor inbox receives Create{Trail}
  → ActivitypubActivityProcess() checks IRI deduplication
  → ProcessCreateActivity() upserts trail with original actor retained
  → InsertIntoFeed() SKIPPED (recipient is instance actor, not a user)
```

## Watch Out For

### Critical

1. **Broadcast loop** — When Instance A stores a trail received from B, the `OnRecordAfterCreateSuccess` hook fires and will re-deliver to A's instance followers (including B). Prevention: check activity IRI against `activitypub_activities` before dispatch. **Must ship in the same phase as fanout activation.**

2. **Auto-accept `ProcessFollowActivity`** — The existing handler auto-accepts all follows unconditionally. A single test connection under current behavior creates an unreviewed sync relationship. Fix the actor-type gate before any instance actor endpoint goes live.

3. **`Reject{Follow}` is mandatory** — Documented Mastodon relay bug: if a rejecting server sends no Reject activity, the remote instance is permanently stuck in "Waiting for approval." The AP spec says SHOULD; production says MUST.

4. **`processDeleteTrailActivity` missing ownership check** — Any remote actor can send a Delete for any trail IRI and Wanderer silently removes the local copy. The comment and summit_log processors have this check; the trail processor does not. Fix in earliest phase that touches delete processing.

### Moderate

5. **Private content leak via visibility change** — When `trail.public` flips to `false`, no Delete activity is sent to peers. They retain the cached copy. `UpdateTrailHandler` must compare before/after `public` field and emit Delete when it goes from `true` to `false`.

6. **Key rotation destroys federation** — The instance actor keypair must be treated as immutable. `initInstanceActor()` must guard: if actor record already exists, never regenerate the key.

7. **Fan-out burst on new connection** — When a high-volume instance first connects, all existing public content is not backfilled (PROJECT.md decision), but any ongoing activity spike may overload the remote inbox. No rate limiting exists today.

8. **HTTP signature mismatch** — Peers running "authorized fetch" mode will reject unsigned GET requests. WebFinger extension for the instance actor ensures key lookup succeeds. Recommend Phase 2 inclusion.

## Suggested Roadmap (4 Coarse Phases)

| Phase | Scope | Prerequisite for |
|-------|-------|-----------------|
| 1 | Schema migration + instance actor lifecycle (init, GET route, key generation) | All other phases |
| 2 | Inbox route + Follow lifecycle (pending-not-auto-accept, Accept/Reject hooks, Undo) | Phase 3 |
| 3 | Fanout extension + broadcast loop deduplication + InsertIntoFeed guard | Phase 4 |
| 4 | Delete propagation hardening (public→private, ownership check, comment gap) | — |

**Fanout (Phase 3) and deduplication must ship together — activating fanout without deduplication creates broadcast storms.**
