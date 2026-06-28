# Wanderer Instance Federation

## What This Is

Instance-level ActivityPub federation for Wanderer. An instance administrator connects their Wanderer instance to another by establishing a mutual Follow relationship between two instance-level actors. Once connected, all public trails, comments, lists, and summit_logs are synchronized bidirectionally — each piece of content retaining its original actor so ownership and edit permissions remain on the origin instance.

## Core Value

An administrator can connect two Wanderer instances so that public content flows between them automatically, using the same ActivityPub machinery already powering user-level federation.

## Current Milestone: v1.1 Federation Connect UI

**Goal:** Give an admin a UI to manage peer instance connections — paste a remote URL, initiate/approve/reject follows, and disconnect — without touching the PocketBase admin panel.

**Target features:**
- Remote actor discovery: paste a remote instance URL → fetch its ActivityPub actor → auto-create the local actor record
- Outgoing Follow: one-click "Connect" that delivers the Follow activity to the remote
- Incoming Follow management: view pending inbound follows, approve or reject them
- Connection dashboard: all peers with status (pending / accepted / rejected) and direction (following / followed-by / mutual)
- Disconnect: Undo{Follow} and remove peer

**Key open question (research-first):** Where does this UI live? Options: custom Go route served by PocketBase with superuser auth guard, SvelteKit settings page with a new admin flag on user records, or deferred to when PocketBase UI extensions are stable.

## Requirements

### Validated (v1.0)

- ✓ Instance actor (`Application` type) exists at startup with stable RSA keypair and ActivityPub endpoint — v1.0
- ✓ Admin can send/receive Follow requests between instances via PocketBase admin — v1.0
- ✓ Remote admin must accept the Follow before sync begins (mutual approval, not auto-accept) — v1.0
- ✓ Reject{Follow} delivered when admin declines a pending instance follow — v1.0
- ✓ Undo{Follow} sent when admin disconnects from a peer — v1.0
- ✓ Public trails, comments, lists, and summit_logs are federated; private content never leaves its instance — v1.0
- ✓ Create/Update/Delete activities propagate bidirectionally to all accepted peer instances — v1.0
- ✓ Broadcast-loop deduplication: duplicate incoming activities silently dropped — v1.0
- ✓ Delete authorization: only the trail's original author can delete via federation — v1.0
- ✓ NodeInfo 2.1 endpoints expose software identity and live usage counts for peer discovery — v1.0
- ✓ Existing user-level federation unaffected; instance actor is fully additive — v1.0

### Active (v1.1)

- [ ] Admin can paste a remote instance URL and have the system fetch its actor and create the local actor record automatically
- [ ] Admin can initiate a Follow from the UI without touching PocketBase admin panel
- [ ] Admin can view all peer instances with status and direction in a dashboard
- [ ] Admin can approve or reject incoming pending Follow requests from the UI
- [ ] Admin can disconnect from a peer instance from the UI

### Deferred (v1.2+)

- [ ] WebFinger resolution for instance actor (`/.well-known/webfinger`) — enables peers running authorized-fetch mode to discover the instance actor
- [ ] Public→private trail propagation: when `is_public` flips to false, send Delete to all peer instances

### Out of Scope

- Per-user opt-in/opt-out of instance federation — all public content is federated when instances are connected
- Selective content filtering per peer (e.g., share trails but not comments) — all four content types sync together
- Backfill of historical content when a new connection is established — forward-only sync
- Relay-style re-broadcasting (Announce pattern) — direct bilateral Follow avoids broadcast amplification
- Binary media relay (photo files) — remote URLs retained, not mirrored
- Federation with non-Wanderer ActivityPub servers — unknown content types; no shared schema for trails

## Context

**Shipped:** v1.0 Instance Federation (2026-06-26)
**Codebase:** ~2,550 lines added across `db/federation/`, `db/hooks/`, `db/routes/`, `db/migrations/`, `db/util/`, `web/src/routes/api/v1/activitypub/instance/`
**Tech stack:** Go/PocketBase backend only; no SvelteKit or Flutter changes required
**Test coverage:** 16 Go unit tests across `federation/` and `hooks/` packages; 7 UAT bugs found and fixed during manual testing

**Known gaps found during UAT (fixed before release):**
- `actor_type` column missing default for new user actors — fixed in `ActorFromUser`
- Instance actor indexed in Meilisearch — guarded in create/update hooks
- `GetActorByHandle`/`GetActorByIRI` would crash on instance actor lookup — early return added in `assembleActor`
- `Reject` activity not handled in instance inbox — `ProcessRejectActivity` added
- `instanceFollowerInboxes` was unidirectional — now queries both inbound and outbound accepted follows
- SvelteKit proxy forwarded original `Content-Length` after re-serializing body — stripped before forwarding
- Trail photo ingestion panic (`photos[i]` on empty slice) — fixed to `append`

**Next milestone setup:** Requires `/gsd-new-milestone` to define requirements and roadmap for v1.1.

## Constraints

- **Tech Stack**: Go/PocketBase backend only for federation logic; no SvelteKit or Flutter changes required for v1
- **ActivityPub compatibility**: Must use standard ActivityPub types — `Application` actor type for instance actor, same activity types already in use (Create, Update, Delete, Follow, Accept, Undo)
- **Privacy hard constraint**: `is_public = false` records must never be included in outgoing activities — checked at fanout time
- **Online-only**: No offline sync buffer; if a remote instance is unreachable, the activity is dropped (existing behavior for user federation)
- **No breaking changes**: Existing user-level federation must be unaffected; instance actor is additive

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Extend existing ActivityPub actor model rather than a new sync protocol | Reuses all existing delivery, signing, and inbox processing infrastructure | ✓ Good — zero new dependencies, delivery pipeline worked without modification |
| PocketBase admin UI only (no web/mobile admin) | Fastest path to v1; instance federation is an infrequent admin operation | ✓ Good — sufficient for v1; next milestone adds proper UI |
| Mutual approval (not auto-accept) for instance follows | Prevents unwanted data ingestion; admins must explicitly consent on both sides | ✓ Good — critical for trust model |
| No historical backfill on new connection | Avoids complex state reconciliation; forward-only sync is simpler and sufficient | ✓ Good — no issues reported |
| Filter `actor_type` on the followee (object), not the actor | Remote actors fetched via GetActorByIRI do not have `actor_type` populated locally; the local instance actor always has `actor_type=instance` | ✓ Good — correct and testable |
| `instanceFollowerInboxes` queries both directions | A follows B AND B follows A both need fanout — bilateral federation requires delivery to all accepted peers regardless of who initiated | ✓ Good — caught by UAT, fixed before release |
| `isOutboundInstanceFollow` separate from `isInstanceFollow` | Create handler needs directional check (follower only); Update/Delete handlers legitimately need EITHER-direction (to Accept/Reject inbound follows) | ✓ Good — prevents double-delivery and mis-firing |
| Strip `content-length` in SvelteKit instance inbox proxy | Body re-serialized by `JSON.stringify`; original Content-Length causes "unexpected EOF" in Go handler | ✓ Good — same fix should be applied to user inbox proxy if it has the same pattern |

---
*Last updated: 2026-06-27 after v1.1 milestone start (Federation Connect UI)*
