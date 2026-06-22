# Technology Stack: Instance Actor for ActivityPub Federation

**Project:** Wanderer Instance Federation
**Researched:** 2026-06-22
**Overall confidence:** HIGH for actor type and library support; MEDIUM for endpoint conventions (no finalized spec)

---

## Core Question

What is the standard stack for implementing an ActivityPub instance actor in the existing Go/PocketBase/go-ap codebase?

---

## Actor Type: Use `Application`

**Recommendation:** `pub.ApplicationType` ("Application")

**Rationale — verified from multiple sources:**

1. **go-ap/activitypub library (confirmed from source at `v0.0.0-20250905102448-e9df599e4528`):** `ApplicationType` is defined as `ActivityVocabularyType = "Application"` and `Application = Actor` (type alias). The library ships `ApplicationNew(id ID) *Application` as a dedicated constructor. `Application` is a first-class type in `ActorTypes`, identical in structure to `Person` — the same `Actor` struct, just with `Type = "Application"`. There is no structural difference at the Go level; only the serialized `"type"` field changes.

2. **Mastodon live instance (confirmed by fetching `https://mastodon.social/actor`):** Mastodon's instance actor uses `"type": "Application"` with `preferredUsername` set to the domain name (e.g., `"mastodon.social"`). This is the most widely deployed reference implementation.

3. **Pixelfed documentation:** Uses `Application` for the instance-wide actor that signs GET requests to remote instances.

4. **FEP-2677 (draft, SocialHub):** Identifies Application as the type for the application actor. The FEP is not yet finalized, but the pattern it describes (Application type, discoverable via NodeInfo or WebFinger) matches Mastodon's existing behavior.

5. **`Service` vs `Application`:** `Service` was semantically intended to represent "a server" per the AS2 Primer, while `Application` was intended for "a software application / frontend." In practice, both are accepted by Mastodon and major implementations — the ActivityPub spec explicitly states actors "don't have to be" one of the standard types. However, **`Application` is what Mastodon itself uses** for its instance actor, making it the safer choice for broadest interoperability.

**What NOT to use:**
- `Service` — semantically ambiguous, fewer reference implementations use it for instance actors
- `Group` — semantically wrong; implies a collection of people
- `Person` — wrong; remote servers use this to infer a human user account

**Confidence: HIGH** — verified from live Mastodon instance JSON and go-ap source.

---

## IRI and URL Pattern

**Recommendation:** `{ORIGIN}/actor` (e.g., `https://wanderer.example.com/actor`)

**Rationale:**
- Mastodon uses `https://mastodon.social/actor` — the de-facto standard
- The trailing `/actor` path is the most widely recognized pattern in the Fediverse
- Pleroma uses `/internal/fetch` and Lemmy uses the root path (`/`), but `/actor` is the pattern most remote servers expect and is what relay software checks first

**Sub-resources (inbox, outbox, followers, following):**

| Endpoint | URL pattern | Required? |
|----------|------------|----------|
| `inbox` | `{ORIGIN}/actor/inbox` | Yes — receives Follow, Accept, Undo activities |
| `outbox` | `{ORIGIN}/actor/outbox` | Yes — required by ActivityPub spec for a valid actor; can be empty OrderedCollection |
| `followers` | `{ORIGIN}/actor/followers` | Yes — remote instances check this when accepting Follows; can be empty/stub |
| `following` | `{ORIGIN}/actor/following` | Recommended — symmetric to followers; stub acceptable |

Mastodon's instance actor **omits** dedicated followers/following collection URLs and sets `manuallyApprovesFollowers: true`. For Wanderer's use case (mutual consent before sync begins), `manuallyApprovesFollowers: true` is appropriate and matches the PROJECT.md requirement for explicit admin approval on both sides.

**keyId convention:**
The public key ID must follow the `{actor IRI}#main-key` pattern, which is already established in the codebase:

```go
pubID := actor.GetString("iri") + "#main-key"
```

This pattern is already used by `PostActivity` and `FetchCollection` in `db/federation/activity.go` and `db/federation/actor.go`. The instance actor must use the same pattern.

**Confidence: HIGH** for `/actor` path. MEDIUM for exact sub-resource paths — these are convention, not spec-mandated.

---

## go-ap/activitypub Library Support

**Version in use:** `v0.0.0-20250905102448-e9df599e4528`

### ApplicationNew constructor

```go
actor := pub.ApplicationNew(pub.IRI(id))
// Sets actor.Type = pub.ApplicationType ("Application")
// Returns *pub.Actor (Application = Actor type alias)
```

The returned `*pub.Actor` has all the same fields as any other actor type. Setting inbox, outbox, followers, following, publicKey, and preferredUsername is identical to the existing user actor code in `util/activitypub.go`.

### Serialization behavior

`Actor.MarshalJSON()` always writes the `"type"` field from `actor.Type`. Setting `actor.Type = pub.ApplicationType` produces `"type": "Application"` in the JSON output — no additional configuration required.

### Receiving/deserializing Application actors

`Actor.UnmarshalJSON()` uses `JSONLoadActor()` which reads the `"type"` field generically. The existing `fetchRemoteActor()` in `db/federation/actor.go` decodes into `pub.Actor` directly — it already handles Application-type remote actors without modification. The `validateActorResponse()` function checks for ID, inbox, outbox, username/name, and publicKey — all of which the instance actor will have.

### No library changes needed

The existing codebase already imports `pub "github.com/go-ap/activitypub"` and uses its full API. There are no new library dependencies required. The `ApplicationNew()` constructor is a drop-in for the existing `ActorFromUser()` pattern in `util/activitypub.go`.

**Confidence: HIGH** — verified from library source files in the Go module cache.

---

## WebFinger for Instance Actor

**Pattern from Mastodon:** The instance actor at `https://mastodon.social/actor` is discoverable via WebFinger using the resource `acct:mastodon.social@mastodon.social` (domain as both user and host).

**For Wanderer v1:** WebFinger support for the instance actor is useful but not strictly required for the core use case. The admin-to-admin Follow workflow can work without WebFinger if the admin supplies the full instance actor IRI directly. However, to be compatible with remote servers that discover actors via WebFinger, the `/.well-known/webfinger` handler should return a `self` link pointing to `{ORIGIN}/actor` when queried for the instance's own domain handle.

**Existing WebFinger infrastructure:** The existing `iriFromHandle()` function in `actor.go` already knows how to query remote WebFinger endpoints. The local WebFinger handler (in the SvelteKit web layer) would need a new route for the instance actor. This is out of scope for v1 per PROJECT.md (Go/PocketBase only, no SvelteKit changes).

**v1 recommendation:** Skip WebFinger for the instance actor in v1. Admins connect instances by supplying the remote instance actor IRI directly (e.g., `https://remote.wanderer.example.com/actor`). This matches the PROJECT.md constraint of "PocketBase admin UI only."

**Confidence: MEDIUM** — WebFinger discovery pattern is established but not spec-required for the Follow workflow.

---

## HTTP Signature Signing for the Instance Actor

The existing `PostActivity()` in `db/federation/activity.go` already handles all HTTP signing via `go-fed/httpsig`. The instance actor needs:

1. An RSA 2048-bit keypair (same as user actors — use existing `generateKeyPair()` in `util/activitypub.go`)
2. The private key stored encrypted with `POCKETBASE_ENCRYPTION_KEY` (same as user actors)
3. The public key stored as PEM-encoded PKIX public key (same format as user actors)
4. A `#main-key` fragment on the actor IRI for the `keyId` in HTTP signatures

All of this is already present in the codebase. The instance actor can use `PostActivity()` directly with no modifications.

**Confidence: HIGH** — directly derived from existing codebase.

---

## activitypub_actors Schema: What Needs to Change

The existing `activitypub_actors` collection has no `type` field (actor type is not stored). For user actors this is fine since they are always `Person`. For the instance actor, identifying it as `Application` requires either:

**Option A (recommended):** Add a `type` text field to `activitypub_actors` with a default of `"Person"` and set `"Application"` for the instance actor. This makes the type queryable and allows the HTTP actor endpoint to serialize the correct type.

**Option B:** Identify the instance actor by `preferred_username = "instance"` or `user IS NULL AND is_local = true`, and hardcode the type in the serialization layer.

Option A is cleaner and more extensible. It requires one new migration.

**The `user` relation field:** User actors have a non-null `user` relation. The instance actor has no associated user record — `user` must be nullable (already is: `"required": false` in the migration). The instance actor row will have `user = null`, `is_local = true`, `type = "Application"`.

**Existing uniqueness constraint:** `CREATE UNIQUE INDEX idx_rpT7QJwWTm ON activitypub_actors (iri)` — the instance actor IRI `{ORIGIN}/actor` is unique per deployment; no conflict.

**Confidence: HIGH** — derived from direct schema inspection.

---

## Fanout Integration

The `followerInboxes()` function in `db/federation/activity.go` already performs a JOIN query:

```go
SELECT aa.inbox FROM follows f
INNER JOIN activitypub_actors aa ON f.follower = aa.id
WHERE f.followee = {:followee} AND f.status = 'accepted' AND aa.inbox != ''
```

When a remote instance actor follows the local instance actor, that remote instance actor row in `activitypub_actors` will have an `inbox` pointing to the remote instance's shared/actor inbox. The `followerInboxes()` call for the instance actor ID will then return that remote inbox URL — the fanout machinery works without modification.

The only new requirement: when creating/updating/deleting public content, the fanout must also include followers of the instance actor, not just followers of the content's author. This means calling `followerInboxes()` for the instance actor and merging those inboxes with the author's follower inboxes.

**Confidence: HIGH** — directly derived from existing query and data model.

---

## Alternatives Considered

| Approach | Why not recommended |
|----------|-------------------|
| `Service` actor type | Fewer reference implementations; Mastodon's own instance actor uses `Application` |
| `/relay` path for IRI | ActivityPub relay pattern (FEP-ae0c) is a different protocol; relay announces all public content via Announce activities rather than direct Create/Update/Delete |
| Separate sync protocol (not ActivityPub) | PROJECT.md explicitly requires reusing existing ActivityPub machinery |
| Auto-accept instance follows | PROJECT.md explicitly requires mutual admin approval |
| New HTTP client/signing library | go-fed/httpsig already in use; no need for alternatives |

---

## Summary Recommendation

1. **Actor type:** `pub.ApplicationType` ("Application") — matches Mastodon, supported natively by go-ap
2. **IRI:** `{ORIGIN}/actor` — Mastodon-compatible, widely recognized
3. **Endpoints:** inbox at `{ORIGIN}/actor/inbox`, outbox stub at `{ORIGIN}/actor/outbox`, followers stub at `{ORIGIN}/actor/followers`
4. **`manuallyApprovesFollowers: true`** — enforces mutual consent requirement from PROJECT.md
5. **Schema change:** Add `type` text field to `activitypub_actors`, default `"Person"`, set `"Application"` for instance actor
6. **No new library dependencies** — all required functionality exists in go-ap and go-fed/httpsig
7. **WebFinger for instance actor:** defer to a later milestone; not required for admin-only v1

---

## Sources

- go-ap/activitypub library source: `/Users/christianbeutel/go/pkg/mod/github.com/go-ap/activitypub@v0.0.0-20250905102448-e9df599e4528/actor.go`
- Mastodon instance actor live JSON: [https://mastodon.social/actor](https://mastodon.social/actor)
- Mastodon ActivityPub spec: [https://docs.joinmastodon.org/spec/activitypub/](https://docs.joinmastodon.org/spec/activitypub/)
- FEP-2677 discussion: [https://socialhub.activitypub.rocks/t/fep-2677-identifying-the-application-actor/3646](https://socialhub.activitypub.rocks/t/fep-2677-identifying-the-application-actor/3646)
- Instance actor pattern discussion: [https://socialhub.activitypub.rocks/t/so-what-even-is-an-instance-actor/3820](https://socialhub.activitypub.rocks/t/so-what-even-is-an-instance-actor/3820)
- Follow application actor pattern: [https://socialhub.activitypub.rocks/t/follow-application-actor-has-a-way-to-get-public-activities/8473](https://socialhub.activitypub.rocks/t/follow-application-actor-has-a-way-to-get-public-activities/8473)
- Mastodon actor type issue: [https://github.com/mastodon/mastodon/issues/22322](https://github.com/mastodon/mastodon/issues/22322)
