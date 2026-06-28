# Wanderer Instance Federation — Human Test Plan

Covers all four phases of the federation milestone. Tests run against the two-instance local setup
(`wanderer-a.mac.lan` and `wanderer-b.mac.lan`). Run them in order — each section depends on the one before it.

## Findings

| # | Test | Result | Issue |
|---|------|--------|-------|
| 1.1 | Instance actor created on startup | PASS | — |
| 1.2 | Instance actor endpoint | PASS | — |
| 1.3 | Keypair survives restart | PASS | — |
| 1.4 | Existing user actors unaffected | PARTIAL | Newly created users get blank `actor_type` instead of `person`; migration backfill works but the column default is missing |
| 1.5 | NodeInfo discovery | PASS | — |
| 1.6 | NodeInfo 2.1 | PASS | — |
| 1.7 | NodeInfo excludes private content | PASS | — |
| — | Instance actor search isolation | FAIL | Instance actors are indexed in Meilisearch and appear in user searches; they should be excluded from indexing |
| — | `GetActorByHandle` / `GetActorByIRI` safety | FAIL | Both functions can resolve the instance actor and then crash at `actor.go:171` because no user record exists for it; neither function guards against `actor_type = instance` |
| 2.1 | Incoming Follow stored as pending | PASS | — |
| 2.2 | Accept delivers Accept{Follow} | PASS | — |
| 2.3 | Reject delivers Reject{Follow} | FAIL | `InstanceInboxHandler` hits the `default:` case and returns "Unsupported activity type" — `activity.Type` is `"Reject"` but there is no `case pub.RejectType` in the inbox dispatch switch |
| 2.4 | Undo{Follow} on delete | PASS | — |
| 2.5 | Outgoing Follow from admin | PASS | — |
| 4.1 | User actor endpoint still works | PASS | — |
| 4.2 | User follow still auto-accepts | PASS | — |
| 4.3 | `actor_type` defaults to `person` on new user | PASS | — |
| 3.* | Content fanout — all tests | FAIL | `instanceFollowerInboxes` (activity.go:73) only queries records where the local instance is the *followee*, so outgoing follows (A follows B) are never included. Fanout must be bidirectional: deliver to accepted followers AND to accepted instances the local instance follows |
| 3.1b | Incoming Create via instance inbox | FAIL | When B publishes after A→B follow, `InstanceInboxHandler` crashes with "unexpected EOF" at instance.go:114 — body is likely already read/drained before the second read attempt |
| 3.1c | Trail photo ingestion | FAIL | `TrailFromActivity` panics at activitypub.go:298 — `photos` slice allocated empty then written by index; fixed to `append` |
| 3.1 | Create public trail fans out | PASS | — |
| 3.2 | Update fans out | PASS | — |
| 3.3 | Delete fans out | PASS | — |
| 3.4 | Private trail never fanned out | PASS | — |
| 3.5 | Duplicate Create silently dropped | NOT TESTED | — |
| 3.6 | Delete by non-author rejected | NOT TESTED | — |
| 3.7 | Incoming Create processed via instance inbox | PASS | — |

## Next Milestone Notes

- **Connect UI needed:** Testing the follow lifecycle currently requires manually creating a foreign instance actor record in PocketBase. The next milestone needs an admin UI that lets an operator paste a remote instance URL and have the actor fetched and the follow record created automatically.

---

## Prerequisites

- [ ] Both PocketBase instances running (`PocketBase (wanderer-a)` on :8090, `PocketBase (wanderer-b)` on :8091)
- [ ] Both Meilisearch instances running (`MeiliSearch (wanderer-a)` on :7700, `MeiliSearch (wanderer-b)` on :7701)
- [ ] Caddy running with `wanderer-a.mac.lan` and `wanderer-b.mac.lan` in `/etc/hosts`
- [ ] At least one user account created in each PocketBase admin panel
- [ ] `WANDERER_VERSION` env var is optional — shows `"dev"` if unset

---

## 1 · Instance Identity (Phase 1 + Phase 4)

### 1.1 Instance actor created on startup

In the PocketBase admin panel for wanderer-a (`https://wanderer-a.mac.lan/_/`):

- Open **Collections → activitypub_actors**
- **Expected:** One record with `actor_type = instance`, `preferred_username = instance`, and a non-empty `public_key` field

### 1.2 Instance actor endpoint

```bash
curl -s https://wanderer-a.mac.lan/api/v1/activitypub/instance | jq .
```

**Expected:** Valid ActivityPub JSON containing:
- `"type": "Application"`
- `"id"` starting with `https://wanderer-a.mac.lan`
- `"inbox"` URL
- `"outbox"` URL
- `"publicKey"` object with `id`, `owner`, and `publicKeyPem`
- `"manuallyApprovesFollowers": true`

### 1.3 Keypair survives restart

- Note the `publicKeyPem` value from 1.2
- Restart the `PocketBase (wanderer-a)` process
- Re-run the curl from 1.2
- **Expected:** `publicKeyPem` is byte-identical to the value noted before restart

### 1.4 Existing user actors unaffected

In **Collections → activitypub_actors**:

**Expected:** All pre-existing user actor rows show `actor_type = person` (not blank, not `instance`)

**Result: PARTIAL** — Pre-existing actors show `actor_type = person` (migration backfill works). Newly registered users after migration do NOT get `actor_type = person`; the field is blank on new actor records. Default value not set on the column.

### 1.5 NodeInfo discovery

```bash
curl -s https://wanderer-a.mac.lan/.well-known/nodeinfo | jq .
```

**Expected:** JSON with a `links` array containing one object where:
- `"rel"` is `"http://nodeinfo.diaspora.software/ns/schema/2.1"`
- `"href"` points to `https://wanderer-a.mac.lan/.well-known/nodeinfo/2.1`

### 1.6 NodeInfo 2.1

```bash
curl -sv https://wanderer-a.mac.lan/.well-known/nodeinfo/2.1 | jq .
```

**Expected:**
- Response header `Content-Type` includes `profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`
- Body contains `"software": { "name": "wanderer", "version": "..." }`
- `"usage.users.total"` matches the number of user accounts created in wanderer-a
- `"usage.localPosts"` counts only **public** trails (see 1.7)

### 1.7 NodeInfo excludes private content

- Create a trail in wanderer-a with `is_public = false`
- Re-run the NodeInfo 2.1 curl
- **Expected:** `localPosts` count does NOT increase

---

## 2 · Follow Lifecycle (Phase 2)

These tests require both instances running. You will use the PocketBase admin panels to drive the lifecycle.

### 2.1 Incoming Follow stored as pending (not auto-accepted)

In the **wanderer-b** PocketBase admin (`https://wanderer-b.mac.lan/_/`):

- Open **Collections → follows**
- Create a new record:
  - `follower`: `https://wanderer-b.mac.lan/api/v1/activitypub/instance`
  - `followee`: `https://wanderer-a.mac.lan/api/v1/activitypub/instance`
  - `status`: `pending`

In the **wanderer-a** PocketBase admin:

- Open **Collections → follows**
- **Expected:** A new record appears with `status = pending` for the wanderer-b actor — it is NOT automatically set to `accepted`

### 2.2 Accept delivers Accept{Follow} to remote

In the **wanderer-a** admin panel:

- Find the pending follow from 2.1
- Change `status` from `pending` to `accepted` and save

**Expected:**
- The record on wanderer-a now shows `status = accepted`
- In **wanderer-b's** PocketBase admin → **Collections → activitypub_activities**, an `Accept` activity record appears with the wanderer-a instance as actor

### 2.3 Reject delivers Reject{Follow}

- Repeat 2.1 to create a second pending follow from wanderer-b → wanderer-a
- In wanderer-a admin, change `status` to `rejected` and save

**Expected:**
- The follow record on wanderer-a shows `status = rejected`
- In wanderer-b's `activitypub_activities`, a `Reject` activity appears

### 2.4 Undo{Follow} on delete

- In wanderer-a admin, delete the accepted follow record from 2.2

**Expected:**
- The follows record is gone from wanderer-a
- In wanderer-b's `activitypub_activities`, an `Undo` activity appears wrapping the original Follow

### 2.5 Outgoing Follow from admin

In **wanderer-a** admin → **Collections → follows**:

- Create a new record:
  - `follower`: `https://wanderer-a.mac.lan/api/v1/activitypub/instance`
  - `followee`: `https://wanderer-b.mac.lan/api/v1/activitypub/instance`
  - `status`: `pending`

**Expected:**
- In wanderer-b's `activitypub_activities`, a `Follow` activity arrives with wanderer-a as actor
- In wanderer-b's `follows` collection, a new record appears with `status = pending`

---

## 3 · Content Fanout (Phase 3)

Prerequisites for this section: wanderer-a and wanderer-b have an **accepted mutual follow** (complete section 2 first, with at least one accepted follow in each direction).

### 3.1 Create public trail fans out

- On wanderer-a, create a public trail (via UI or API) with `is_public = true`
- **Expected:** In wanderer-b's `activitypub_activities`, a `Create` activity appears containing the trail

### 3.2 Update fans out

- Edit the trail created in 3.1 (change name or description) and save
- **Expected:** In wanderer-b's `activitypub_activities`, an `Update` activity appears for the same trail

### 3.3 Delete fans out

- Delete the trail from 3.1 on wanderer-a
- **Expected:** In wanderer-b's `activitypub_activities`, a `Delete` activity appears for the trail's IRI

### 3.4 Private trail is never fanned out

- On wanderer-a, create a trail with `is_public = false`
- **Expected:** No corresponding `Create` activity appears in wanderer-b's `activitypub_activities`
- Update and then delete the private trail
- **Expected:** Still no `Update` or `Delete` activities for it on wanderer-b

### 3.5 Duplicate Create is silently dropped

- Find the IRI of a trail activity already in wanderer-b's `activitypub_activities`
- POST that same `Create` activity again to wanderer-b's instance inbox:
  ```bash
  # Use the JSON body of the existing Create activity
  curl -s -X POST https://wanderer-b.mac.lan/api/v1/activitypub/instance/inbox \
    -H "Content-Type: application/json" \
    -d '<json of existing Create activity>'
  ```
- **Expected:** wanderer-b does NOT create a duplicate trail record; only one copy exists

### 3.6 Delete by non-author is rejected

- Take the IRI of a trail owned by a wanderer-a user
- POST a `Delete` activity to wanderer-b's inbox signed as a **different actor** (not the trail's author)
- **Expected:** The request returns a `400` error; the trail record on wanderer-b is NOT deleted

### 3.7 Incoming Create/Update/Delete processed via instance inbox

- On wanderer-b, look at a trail that arrived via fanout from wanderer-a (created in 3.1)
- **Expected:** The trail record exists in wanderer-b's `trails` collection, with the original wanderer-a user as `author`

---

## 4 · Regression — User-Level Federation Unaffected

These verify that instance-level federation did not break existing user-to-user ActivityPub.

### 4.1 User actor endpoint still works

```bash
curl -s "https://wanderer-a.mac.lan/api/v1/activitypub/user/<handle>" | jq .
```

**Expected:** Valid ActivityPub `Person` actor JSON (same as before this milestone)

### 4.2 User follow still auto-accepts

- Have a user on wanderer-b follow a user on wanderer-a
- **Expected:** The follow is accepted immediately (not stored as `pending`) — the manual-approval gate applies only to `Application`-type (instance) actors

### 4.3 `actor_type` column defaults to `person`

- Create a new user on wanderer-a
- In the admin panel → **activitypub_actors**, find their actor record
- **Expected:** `actor_type = person` (not blank, not `instance`)
