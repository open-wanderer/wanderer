# Feature Research

**Domain:** ActivityPub admin federation management UI — v1.1 Federation Connect UI
**Researched:** 2026-06-27
**Confidence:** HIGH (codebase fully read; ecosystem research from Mastodon, Pleroma/Akkoma, GoToSocial, Misskey relay/federation admin UIs)

---

## Context: What v1.1 Is Building

v1.0 shipped the protocol layer (instance actor, inbox, Follow lifecycle, fanout). Admins currently manage everything through the PocketBase admin panel by directly creating and updating records in the `follows` and `activitypub_actors` collections.

v1.1 goal: give an admin a real UI to manage peer instance connections without touching the database.

The existing v1.0 hooks (`InstanceFollowCreateHandler`, `InstanceFollowUpdateHandler`, `InstanceFollowDeleteHandler`) already handle the ActivityPub delivery side — any change to a `follows` record triggers the right activity. The UI layer needs only to drive those records correctly. The hooks are the stable API; the UI is the new surface.

---

## Table Stakes (Admins Expect These)

Features that any competent federation admin UI must have. Missing any of these forces admins back to the PocketBase panel, which defeats the purpose.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Paste-a-URL connect flow | Every fediverse relay/federation admin tool (Mastodon relay, Pleroma relay, GoToSocial domain allowlist) uses a single URL input as the entry point. Mastodon admins paste a relay's `/inbox` URL. Wanderer should accept a simpler base URL. | LOW | Accept `https://remote.example.com`. The UI resolves NodeInfo and actor IRI internally — admin should not need to know internal URL paths. |
| Remote instance preview before sending Follow | Admins need to confirm the remote is a compatible Wanderer instance before committing. Mastodon and Pleroma skip this step (they send the Follow immediately), which leads to silent failures. This is table stakes for Wanderer because non-Wanderer instances will not understand the content types. | MEDIUM | Fetch NodeInfo to confirm `software.name == "wanderer"`, then fetch `/api/v1/activitypub/instance` to show actor name, domain, user count, trail count. Show a "Connect" confirm button after preview. |
| Peer dashboard with status per connection | Mastodon relay list, Pleroma relay list, and Misskey federation tab all show status inline. An admin must immediately see the state of each peer without clicking into a detail view. | LOW | Status comes from the `follows` table (`status` field: pending / accepted / rejected). Direction (Outbound / Inbound / Mutual) derived from whether local instance actor is follower, followee, or both. |
| Approve / Reject incoming pending follows | Inbound follows from remote instances land in `follows` with `status=pending`. No user-facing action exists today. This is the most critical missing UI piece — without it admins cannot complete inbound connections. | LOW | The `InstanceFollowUpdateHandler` hook already delivers Accept/Reject when `status` is changed. The UI sends a PATCH to the follow record. |
| Disconnect a peer | Mastodon relay list and Pleroma relay admin both expose a remove/disconnect action per relay. Admins must terminate connections without touching the DB. | LOW | `InstanceFollowDeleteHandler` already delivers `Undo{Follow}` on DELETE. The UI deletes the follow record(s). Mutual connections require deleting both directions atomically — needs a dedicated Go endpoint. |
| Clear, inline error feedback | Mastodon's relay UI shows "Waiting for relay's approval" and silently gets stuck — a documented pain point (GitHub issue #14961). GoToSocial's domain permission UI gives no feedback on unreachable domains. Admins need concrete, actionable error messages. | MEDIUM | Surface errors from the discovery and Follow-delivery steps synchronously in the connect flow, not in server logs. |
| Admin-only access guard | The federation management page must not be accessible to regular users. | MEDIUM | Requires an admin identity marker on users OR PocketBase superuser-only access. See Gaps section. |

---

## Differentiators (Competitive Advantage)

Features that make Wanderer's federation admin UI meaningfully better than existing fediverse implementations.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Software identity check before connecting | No existing fediverse admin UI verifies the remote is the same software before initiating a follow. Wanderer-to-Wanderer is the only supported federation path; connecting to a Mastodon instance would silently fail (unknown content types). Proactively blocking this at the UI level prevents confusion. | LOW | NodeInfo `software.name` check during the preview step. Show a blocking error if the value is not `"wanderer"`. |
| Direction-aware peer list | Mastodon relay list shows only URL and status. A direction column (Outbound / Inbound / Mutual) tells the admin at a glance whether this instance is a full peer or one-sided. Existing fediverse relay UIs do not surface this because relays are inherently one-directional (pub/sub, not bilateral Follow). | LOW | Derived from two `follows` records: one where local actor is follower, one where it is followee, both with `status=accepted`. Mutual = both exist. |
| Pending inbound request context | Show when the inbound Follow was received and which remote domain/actor sent it. No existing fediverse relay UI shows this because relay follows auto-accept. Wanderer is the only implementation that requires bilateral admin approval. | LOW | `follows.created` + actor domain/preferred_username are already stored in v1.0. Show them inline in the pending section. |
| Instance metadata in peer list | Show remote instance domain, Wanderer version, user count, and public trail count fetched at connect time. Helps admins recognise peers and notice version skew. | MEDIUM | Store fetched NodeInfo data at connect time; re-fetch on page load if stale (> 24h). Options: add columns to `activitypub_actors` or re-fetch live on admin page load (acceptable for low-traffic admin page in v1.1). |

---

## Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Auto-accept all incoming instance follows | Reduces friction — no need to approve each request. | Breaks the mutual consent model established in v1.0. An admin could unknowingly sync with an untrusted instance. The `pending` → `accepted` step is the trust gate. | Keep manual approve/reject. Make the UI fast enough that approval takes one click. |
| Connect via handle (`@instance@domain`) | WebFinger handle resolution feels familiar to fediverse users. | The Wanderer instance actor does not have a standard WebFinger handle — it is an `Application` type at a fixed IRI. FEP-d556 (server-level actor discovery via WebFinger) is still exploratory and not yet standardised. v1.0 has no WebFinger endpoint for the instance actor. | Accept a plain base URL (`https://domain.example`). Resolve the actor IRI by fetching NodeInfo then the known Wanderer path convention `/api/v1/activitypub/instance`. Reliable for Wanderer-to-Wanderer. |
| Per-peer content filtering | Admins may want to share trails but not comments with specific peers. | Out of scope per PROJECT.md. Adds significant fanout complexity. Contradicts the "same ActivityPub machinery" constraint. | Document clearly in the UI that all four public content types sync when connected. |
| Historical backfill on new connection | Admins want existing content to appear on a new peer immediately. | Out of scope per PROJECT.md. Forward-only sync avoids complex state reconciliation. | Explain in the UI that only content created after connection is established will sync. |
| Relay-style one-to-many subscription | Mastodon uses a relay pattern that might be expected. | Wanderer uses bilateral Follow, not relay Announce. A relay would require a third-party relay server. Bilateral is already implemented and avoids broadcast amplification. | The peer dashboard IS the equivalent of the relay list — bilateral instead of hub-and-spoke. |
| Domain blocklist | Admins may want to pre-emptively block hostile domains. | Out of scope for v1.1. Adds a separate data model and inbox filter. | Defer to v1.2+. For now, admins simply do not connect to untrusted instances, and incoming follows from unknown instances require manual approval. |

---

## UX Flows (Concrete, Step-by-Step)

These are the four flows the UI must implement. Each step maps to existing v1.0 hooks or a new API endpoint.

### Flow 1: Connect to a Remote Instance (Outbound Follow)

1. Admin navigates to `/settings/federation` (new SvelteKit page).
2. Admin pastes a base URL: `https://remote.wanderer.example`.
3. UI sends a discovery request to a new Go endpoint: `POST /api/v1/federation/discover` with `{"url": "https://remote.wanderer.example"}`.
4. Go endpoint performs the discovery sequence server-side (required because fetches must be HTTP-signed):
   a. Fetch `/.well-known/nodeinfo` → resolve NodeInfo 2.1 href.
   b. Fetch NodeInfo 2.1 → extract `software.name`, `software.version`, `usage.users.total`, `usage.localPosts`.
   c. If `software.name != "wanderer"`: return 422 `{"error": "not_wanderer", "actual_software": "..."}`.
   d. Fetch `/api/v1/activitypub/instance` (the known Wanderer instance actor IRI) → validate actor fields using existing `validateActorResponse`.
   e. Upsert the remote actor into `activitypub_actors` using existing `GetActorByIRI` / `assembleActor` path.
   f. Return preview payload: `{actor: {id, domain, preferred_username}, nodeinfo: {version, users, posts}}`.
5. UI renders a preview card: remote domain, Wanderer version, user count, public trail count.
6. Admin clicks "Connect". UI sends `POST /api/v1/federation/follow` with the remote actor's local record ID.
7. Go endpoint creates a `follows` record (follower = local instance actor ID, followee = remote actor ID, status = pending).
8. `InstanceFollowCreateHandler` fires → delivers `Follow` activity to remote instance inbox.
9. UI shows the peer in the list with badge "Pending (awaiting remote approval)".

**Error states surfaced at step 4 (returned as JSON, shown inline in the UI):**

| Error code | UI message |
|------------|-----------|
| `unreachable` | "Cannot reach that instance. Check the URL and try again." |
| `not_https` | Rejected client-side before POST. "URL must start with https://." |
| `not_wanderer` | "Remote is not a Wanderer instance (found: mastodon). Only Wanderer-to-Wanderer federation is supported." |
| `already_connected` | "Already connected to this instance." |
| `already_pending` | "A follow request to this instance is already pending." |
| `discovery_failed` | "Could not read instance information. The remote may not support federation discovery." |
| `actor_invalid` | "Remote instance actor is missing required fields. The remote may be misconfigured." |

### Flow 2: Approve an Incoming Pending Follow

1. Admin loads `/settings/federation` → a "Pending Requests" section lists inbound follows with `status=pending`.
2. Each row shows: remote domain, actor name, received timestamp.
3. Admin clicks "Approve" → UI PATCHes the `follows` record: `{"status": "accepted"}`.
4. `InstanceFollowUpdateHandler` fires → delivers `Accept{Follow}` to the remote instance inbox.
5. Row moves to the "Connected Peers" section with direction badge "Inbound" (or "Mutual" if there is also an outbound accepted follow to the same peer).

**Status after approve:** The remote now delivers Create/Update/Delete activities to our instance inbox. We begin delivering our public content to their instance actor's inbox.

### Flow 3: Reject an Incoming Pending Follow

1. Admin clicks "Reject" on a pending inbound row.
2. UI PATCHes the `follows` record: `{"status": "rejected"}`.
3. `InstanceFollowUpdateHandler` fires → delivers `Reject{Follow}` to the remote inbox.
4. Row moves to a "Rejected" section or is removed after a brief confirmation display.

**Why Reject must be sent (not just ignored):** Mastodon relay GitHub issue #14961 documents the exact problem — when an admin only closes the request without sending Reject, the remote stays permanently in "waiting for approval" state. Sending `Reject{Follow}` is the ActivityPub-correct way to close the request definitively.

### Flow 4: Disconnect from a Peer

1. Admin clicks "Disconnect" on an accepted peer row.
2. Confirmation dialog: "This will stop content syncing with remote.example. Continue?"
3. UI sends `DELETE /api/v1/federation/disconnect?actor_id=X` (new Go endpoint).
4. Go endpoint deletes both follow records for this peer (outbound and inbound if mutual) in a single DB transaction.
5. `InstanceFollowDeleteHandler` fires for each deleted record → delivers `Undo{Follow}` to the remote inbox for each direction.
6. Peer is removed from the Connected Peers list.

**Why a dedicated endpoint (not direct PocketBase DELETE):** Mutual connections have two `follows` records. Deleting them one at a time from the frontend is not atomic and could leave a half-open state (we stop receiving their content but continue delivering ours, or vice versa). A single Go endpoint wraps both deletions.

---

## Feature Dependencies

```
[Peer list]
    └──reads──> follows table + activitypub_actors table (v1.0, exists)

[Direction column in peer list]
    └──reads──> both follow directions queryable (v1.0, exists)

[Connect flow]
    ├──requires──> POST /api/v1/federation/discover (NEW Go endpoint)
    ├──uses──>     GetActorByIRI / assembleActor / validateActorResponse (v1.0, exists)
    └──requires──> POST /api/v1/federation/follow → creates follows record (NEW Go endpoint)
                       └──triggers──> InstanceFollowCreateHandler (v1.0, exists)

[Approve inbound]
    └──requires──> PATCH follows.status = "accepted" (NEW: SvelteKit route or direct PocketBase PATCH)
                       └──triggers──> InstanceFollowUpdateHandler (v1.0, exists)

[Reject inbound]
    └──requires──> PATCH follows.status = "rejected" (NEW: same as above)
                       └──triggers──> InstanceFollowUpdateHandler (v1.0, exists)

[Disconnect]
    └──requires──> DELETE /api/v1/federation/disconnect?actor_id=X (NEW Go endpoint, atomic)
                       └──triggers──> InstanceFollowDeleteHandler per record (v1.0, exists)

[Admin auth guard]
    └──requires──> is_admin flag on users collection (NEW: migration) OR superuser-only Go route
```

### Dependency Notes

- **All four flows depend on v1.0 hooks — the hooks are the stable API.** The UI drives the database; the hooks deliver the ActivityPub activities. No changes to federation logic are needed.
- **Connect flow requires server-side discovery endpoint.** The SvelteKit page cannot call the remote instance directly — fetches must be HTTP-signed using the instance actor's private key. A new Go endpoint handles the signed discovery sequence.
- **Mutual disconnect requires a new Go endpoint.** Two `follows` records cannot be deleted atomically via two separate PocketBase REST calls from the frontend. A single endpoint wraps both in a DB transaction.
- **Admin auth guard is the key unresolved dependency.** See Gaps section.

---

## MVP Definition

### v1.1 Launch With (All Five Requirements from PROJECT.md)

- [ ] **Connect flow** — paste a base URL, see a preview card (domain, software version, user count, trail count), click "Connect" to initiate an outbound Follow. Surfaces all error states inline.
- [ ] **Pending inbound approvals** — a dedicated section listing inbound `follows` with `status=pending`, each with Approve and Reject buttons. Shows remote domain, actor name, and received timestamp.
- [ ] **Peer dashboard** — a list of all instance-actor follows with status badge (Pending / Accepted / Rejected) and direction (Outbound / Inbound / Mutual).
- [ ] **Disconnect** — a Disconnect button per accepted peer that atomically removes the follow(s) and delivers Undo{Follow} via the existing hooks.
- [ ] **Admin auth guard** — `/settings/federation` is accessible only to admins; regular users receive a 403 or redirect. Requires a decision on the admin identity model (see Gaps).

### Add After Validation (v1.2+)

- [ ] **Refresh peer metadata** — a "Refresh" button per peer that re-fetches NodeInfo and updates stored version/count data. Trigger: admins report stale version or count information.
- [ ] **Connection activity log** — a timeline of Follow / Accept / Reject / Undo events per peer. Trigger: admins ask "why did this connection drop?"
- [ ] **WebFinger instance actor** — enables discovery via `@instance@domain` handle if FEP-d556 stabilises. Deferred per PROJECT.md.
- [ ] **Domain blocklist** — pre-emptively refuse Follow requests from domains on a blocklist. Currently admins can achieve the same by rejecting inbound follows manually.
- [ ] **Email notification on inbound pending follow** — notify the admin when a new follow request arrives so they do not miss it. Requires email configuration.

### Future Consideration (v2+)

- [ ] **Public peer list endpoint** — expose the accepted peers list publicly so other instances can discover federation relationships. Raises privacy considerations.
- [ ] **Bulk disconnect** — select multiple peers for disconnection. Useful during instance decommission.
- [ ] **Import/export peer list** — JSON or CSV export of peer connections. Useful when migrating or managing multiple instances.

---

## Feature Prioritization Matrix

| Feature | Admin Value | Implementation Cost | Priority |
|---------|-------------|---------------------|----------|
| Connect flow (URL input + server-side discovery + preview + Follow) | HIGH | MEDIUM (new Go discover + follow endpoints) | P1 |
| Software identity check at preview (block non-Wanderer) | HIGH | LOW (NodeInfo check inside discover endpoint) | P1 |
| Pending inbound approvals (Approve/Reject buttons) | HIGH | LOW (PATCH to follows record, hooks handle delivery) | P1 |
| Peer dashboard (list + status + direction) | HIGH | LOW (query follows + activitypub_actors) | P1 |
| Disconnect (atomic two-direction delete) | HIGH | LOW (new Go disconnect endpoint, hooks handle delivery) | P1 |
| Admin auth guard | HIGH | MEDIUM (depends on admin identity decision) | P1 |
| Inline error messages for all connect failure modes | MEDIUM | LOW (returned from Go endpoint) | P1 |
| Pending request context (received timestamp, remote actor name) | MEDIUM | LOW (already in DB) | P2 |
| Direction column (Outbound / Inbound / Mutual) | MEDIUM | LOW (derived from existing data) | P2 |
| Remote instance metadata in peer list (version, counts) | MEDIUM | MEDIUM (store NodeInfo at connect time) | P2 |
| PocketBase realtime subscription for status updates | LOW | LOW (PocketBase realtime already exists) | P2 |
| Refresh peer metadata button | LOW | LOW | P3 |
| Connection activity log | LOW | MEDIUM | P3 |

**Priority key:** P1 = required for v1.1 launch; P2 = should have, add in v1.1 if low effort otherwise v1.2; P3 = defer

---

## Competitor Feature Analysis

| Feature | Mastodon (relay UI) | Pleroma/Akkoma (relay UI) | GoToSocial (domain perms) | Wanderer v1.1 Plan |
|---------|---------------------|--------------------------|--------------------------|-------------------|
| Entry point | Paste relay inbox URL (e.g. `https://relay.example.com/inbox`) | Paste relay actor URL | Domain name only | Paste instance base URL (`https://instance.example`) |
| Remote preview before connecting | None — Follow sent immediately | None — Follow sent immediately | None | Yes: NodeInfo + actor card with domain, version, user/trail counts |
| Software compatibility check | None | None | None | Yes: block if `software.name != "wanderer"` |
| Status shown | idle / pending / accepted / rejected | actor URL + `followed_back` bool | domain block / allow (no follow states) | pending / accepted / rejected with direction |
| Direction shown | Not applicable (relay is one-way pub/sub) | Not applicable (one-way) | Not applicable (no Follow model) | Outbound / Inbound / Mutual |
| Approve inbound follows | Not applicable (relays auto-accept or use their own admin UI) | Not applicable | Not applicable | Yes — dedicated Approve / Reject buttons per pending inbound row |
| Disconnect | Remove relay from list | DELETE relay with optional `force` flag (Pleroma Admin API) | Remove domain block/allow | Delete follow record(s) via atomic Go endpoint |
| Error states in UI | "Waiting for relay's approval" (gets silently stuck — documented issue) | Error only in server logs | Silent on unreachable domain | Inline error per step: unreachable, not_wanderer, already_connected, actor_invalid |
| Inbound pending request detail | Not applicable | Not applicable | Not applicable | Shows remote domain, actor name, received timestamp |

**Observation:** No existing fediverse admin UI (a) verifies remote software type before connecting, (b) shows connection direction, or (c) provides an explicit approve/reject UI for inbound instance follows. All three are Wanderer-specific because no other implementation uses mutual-approval bilateral instance Follow as its connection model.

---

## Where Does the UI Live? (Architectural Decision)

Three options, evaluated against the codebase:

**Option A: SvelteKit `/settings/federation` page (recommended)**
- Fits the established pattern: `settings/plugins`, `settings/maintenance`, `settings/privacy`, `settings/account` all exist.
- Requires adding an `is_admin: bool` field to the `users` PocketBase collection via migration.
- `+page.server.ts` load function checks `locals.user.is_admin` and redirects if false.
- API actions go to new SvelteKit server routes at `/api/v1/federation/*` that proxy to Go endpoints.
- The Go endpoints themselves are protected by `e.HasSuperuserAuth()` OR accept a user token and check `is_admin`.
- Zero new frontend infrastructure — same page structure used by maintenance and plugins pages.

**Option B: Go-served HTML page at `/federation-admin` (not recommended)**
- No SvelteKit changes.
- PocketBase superuser token required — only the DB-level admin, not a regular Wanderer user with admin rights.
- Too restrictive: the intended user is a Wanderer admin who logs in via the web UI, not a PocketBase superuser.
- Requires Go template rendering or a new embedded SPA — significant new infrastructure.

**Option C: PocketBase UI extension (deferred)**
- PocketBase UI is embedded Svelte 4; plugin system not yet stable.
- Deferred in v1.0 for exactly this reason; still not ready in v1.1.

**Decision: Option A.** The settings page pattern is established, the routing infrastructure exists, and the User model already passes through SvelteKit auth. The only new work is the `is_admin` migration, the new `/settings/federation` page, and the Go backend endpoints.

---

## Gaps to Address in Phase Planning

1. **Admin identity model** — Wanderer has no `is_admin` flag on users today. Two sub-options:
   - (a) Add `is_admin: bool` to `users` collection via a PocketBase migration. Set manually in PocketBase admin panel or via a first-run promotion flow. The SvelteKit page checks this flag. Go endpoints also validate it via the user's auth token.
   - (b) Require PocketBase superuser credentials. Simpler but too restrictive — instance admins log in as regular Wanderer users, not PocketBase superusers.
   Option (a) is recommended. The migration is simple; the flag is a standard RBAC pattern used in other SvelteKit+PocketBase projects.

2. **Atomic mutual-disconnect endpoint** — A dedicated `DELETE /api/v1/federation/disconnect?actor_id=X` Go endpoint must delete both follow directions in one DB transaction. The existing PocketBase REST DELETE route works per-record only.

3. **NodeInfo data persistence** — The connect flow fetches NodeInfo at preview time. For v1.1 the simplest approach is to re-fetch on every admin page load (acceptable for a low-traffic admin page). In v1.2, persist version/count to the `activitypub_actors` record (add columns) and refresh periodically.

4. **Follow status polling / realtime** — After sending an outbound Follow, the peer shows "pending" until the remote sends Accept back. The `InstanceInboxHandler` updates `follows.status` to `accepted` when Accept arrives. Options: (a) poll the peer list every N seconds while any row is in pending state; (b) use PocketBase realtime subscription to `follows` collection (already available in PocketBase SDK). Option (b) is preferred — event-driven and no unnecessary requests.

5. **Collection API access rules** — PocketBase collection rules must allow the admin user to PATCH `follows` records for instance-actor rows. The current rules likely only allow the authenticated user to modify their own follow records. A rule change or a Go middleware endpoint is needed for the approve/reject flow.

---

## Sources

- [Mastodon Relay model states (idle/pending/accepted/rejected) — source code](https://github.com/mastodon/mastodon/blob/main/app/models/relay.rb)
- [Mastodon "Add federation relay support" PR — original relay admin UI design](https://github.com/mastodon/mastodon/pull/7998)
- [Mastodon relay stuck "Waiting for relay's approval" — why Reject must be sent](https://github.com/tootsuite/mastodon/issues/14961)
- [Pleroma admin relay API (GET/POST/DELETE)](https://docs.pleroma.social/backend/development/API/admin_api/)
- [Akkoma admin API relay endpoints](https://docs.akkoma.dev/stable/development/API/admin_api/)
- [GoToSocial admin settings panel — domain permissions](https://docs.gotosocial.org/en/latest/admin/settings/)
- [GoToSocial federation modes (blocklist/allowlist)](https://docs.gotosocial.org/en/latest/admin/federation_modes/)
- [FEP-d556: Server-Level Actor Discovery via WebFinger — still exploratory](https://socialhub.activitypub.rocks/t/fep-d556-server-level-actor-discovery-using-webfinger/3861)
- [Mastodon follow_requests API — approve/reject pattern](https://docs.joinmastodon.org/methods/follow_requests/)
- [PocketBase Go routing and RequireSuperuserAuth / HasSuperuserAuth](https://pocketbase.io/docs/go-routing/)
- [Adding relays to Mastodon — URL format and status states](https://dustinrue.com/2023/01/adding-relays-to-your-mastodon-instance/)
- Wanderer codebase: `db/federation/follow.go`, `db/federation/instance.go`, `db/federation/actor.go`, `db/hooks/follow.go`, `db/routes/nodeinfo.go`, `web/src/routes/settings/`

---

*Feature research for: Wanderer Instance Federation Connect UI (v1.1)*
*Researched: 2026-06-27*
