# Feature Landscape: Instance-Level ActivityPub Federation

**Domain:** ActivityPub instance-to-instance federation (trail/hiking app)
**Researched:** 2026-06-22
**Confidence:** HIGH for protocol conventions; MEDIUM for admin UX patterns (drawn from Mastodon, Pleroma/Akkoma, Bonfire)

---

## Context: What Wanderer Is Building

Instance actors (`type = Application`) that let two Wanderer admins establish a bilateral Follow relationship. Once both sides accept, public trails, comments, lists, and summit_logs sync bidirectionally. The existing `follows` collection and `ProcessFollowActivity` / `ProcessAcceptActivity` machinery already handles the protocol layer; the gap is creating the instance actor and wiring it into the fanout.

The existing `ProcessFollowActivity` **auto-accepts** all incoming follows from remote actors (`status: "accepted"` immediately). For instance-level federation PROJECT.md requires **mutual approval**, meaning this auto-accept behavior must be gated differently for Application-type actors.

---

## Table Stakes

Features an admin will expect before trusting this system. Missing any of these means the feature is unusable or unsafe.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Instance actor exists at a stable, well-known URL | Every AP implementation expects `GET /api/v1/activitypub/instance` to return a valid actor JSON document | Low | `type: Application`, `preferredUsername: "instance"`, owns an inbox, outbox, public key |
| HTTP-signed inbox delivery for instance actor | Remote server must be able to verify the origin of activities sent by your instance actor | Low | Same signing machinery used for user actors; just needs the instance actor's keypair |
| Admin can send a Follow to a remote instance actor by URL or handle | Core connection flow. Without it admins cannot initiate federation | Low | PocketBase admin UI — custom action or collection hook on a "peer_instances" collection |
| Remote admin must explicitly Accept before sync starts | Prevents surprise data ingestion. All mature AP implementations require bilateral consent for trusted connections | Low | Current code auto-accepts; must gate Application-type follows differently — require manual Accept |
| Incoming Follow from a remote instance actor enters `pending` state | Admin must review and approve before content flows inbound | Low | Change `ProcessFollowActivity` to skip auto-accept when actor `type == Application` |
| Admin can Accept a pending instance Follow | Completes the inbound side of the connection | Low | Send `Accept{Follow}` activity; set status to `accepted` in `follows` |
| Admin can Reject a pending instance Follow | Closes the connection attempt without data flowing | Low | Send `Reject{Follow}` activity; set status to `rejected` or delete row |
| Admin can Undo a Follow to disconnect from a peer (unfollow) | Standard protocol mechanism for severing outbound sync | Low | Send `Undo{Follow}`; remove or mark the local follow row |
| Admin can revoke an accepted inbound Follow (remove a peer that follows us) | Stops delivering our content to a peer that no longer should receive it | Low | Send `Reject` against the existing accepted follow (or `Undo Accept`); remove row |
| Connection status is visible per peer: `pending_out` / `pending_in` / `accepted` / `rejected` | Admins need to know the state of each relationship | Low | A query over `follows` where follower or followee is the instance actor |
| Public content only: `is_public = false` records never included in instance fanout | Privacy hard requirement — already documented as a constraint in PROJECT.md | Low | Check at fanout time, same as for user federation; already partially coded in hooks |
| Remote content retains original actor IRI | Receiving instance must not allow editing of content that originated elsewhere | Low | Already enforced by existing user federation; instance federation inherits same rule |
| Instance actor's inbox rejects activities from blocked domains | If a peer is later found harmful, block must stop content delivery immediately | Medium | Domain blocklist check at inbox entry point before processing |
| Admin can view a list of known peer instances with their connection status | Operational visibility — without this admins can't manage the feature | Low | Query `follows` joined to `activitypub_actors` where actor type = Application |

---

## Differentiators

Features that improve the admin experience and build trust but are not blockers for v1.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Per-peer delivery health indicator (last successful delivery timestamp, recent failure count) | Helps admins detect broken connections without reading logs | Medium | Store `last_delivered_at` and `delivery_failure_count` on the peer follow row; Pleroma exposes `/api/v1/pleroma/federation_status` for exactly this |
| Admin note field per peer instance | Lets admins record why a connection was made or any concerns | Low | Private-only; never federated; Mastodon domain blocks use this pattern (`private_comment`) |
| Domain blocklist at the instance level | Pre-emptively refuse Follow requests from domains known to be harmful | Medium | Check incoming Follow actor domain against a blocklist table before creating a pending row |
| Allowlist mode: only federate with explicitly approved instances | For privacy-sensitive deployments that don't want open federation | High | Mastodon's "limited federation mode"; gate the inbox to reject activities from domains not on the allowlist |
| Peer instance metadata display (software name, version, user count) | NodeInfo endpoint (`/.well-known/nodeinfo`) exposes this; useful for the peer list UI | Medium | Fetch NodeInfo on first connection; cache and refresh periodically |
| Disconnect notification activity | Inform the remote instance actor when you Undo a Follow, so they can clean up their follow row promptly | Low | Standard ActivityPub `Undo{Follow}` — the remote side SHOULD handle this |
| Import/export peer list as JSON or CSV | Admins running multiple instances or migrating can transfer their peer connections | Medium | Mastodon does this for domain blocklists; useful for Wanderer networks |
| Delivery retry with exponential back-off for unreachable peers | Avoids permanently losing activities when a peer is temporarily down | High | Existing code drops on failure; a job queue with retries is a significant addition |

---

## Anti-Features

Things to deliberately NOT build in v1. Each has a specific reason.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Web or mobile admin UI for federation management | PROJECT.md explicitly scopes this out; PocketBase admin UI is sufficient for infrequent admin operations | PocketBase collection UI + custom admin actions |
| Per-user opt-in/opt-out of instance federation | Adds significant complexity (per-record consent tracking, UI, notification); out of scope per PROJECT.md | Document that all public content federates when instances are connected |
| Selective content-type filtering per peer (e.g., trails but not comments) | Selective filtering requires per-activity type routing, adds fanout complexity, and the benefit is marginal for a hiking app where trail + summit_log + comment form a coherent unit | Federate all four public content types together |
| Historical backfill when a new connection is established | State reconciliation for potentially thousands of trails is complex and risky; forward-only is a well-established pattern in the fediverse | Document clearly: only new activities after Follow acceptance are synced |
| Relay-style re-broadcasting (Announce forwarding) | Mastodon's relay protocol uses `Announce` to re-broadcast to all subscribers; for Wanderer this would cause duplicate delivery and content attribution issues | Direct bilateral Follow relationships; no relay pattern |
| Auto-accepting instance follows | Mastodon auto-accepts user follows (unless the user has locked their account); but for instance-level federation, surprise data ingestion from unknown instances is a trust and safety risk | Always require manual Accept for Application-type follows |
| Federated DMs or private messages | ActivityPub DMs addressed to specific actors propagate to remote inboxes; this creates a privacy surface that does not apply to trail data | Never include non-public activities in instance fanout |
| Cross-instance search indexing | Federated instances could receive each other's trails into Meilisearch; this creates a large and potentially stale index | Index only locally-stored records (remote content received via federation is stored locally anyway) |
| Federation with generic Mastodon/Pleroma instances | Wanderer's content types (Trail, SummitLog, List) are not understood by Mastodon; a Mastodon instance following the Wanderer instance actor would receive unrenderable activities | Scope instance federation to Wanderer-to-Wanderer only in v1; interop with generic AP servers is a future milestone |

---

## Feature Dependencies

```
Instance actor created at startup
  → HTTP-signed inbox for instance actor
    → Admin can send outgoing Follow (requires keypair)
    → Admin can receive incoming Follow (requires inbox route)

Admin sends Follow to peer
  → Remote peer receives pending_in Follow
    → Remote admin Accepts
      → Originating instance receives Accept, status → accepted
        → Content fanout includes instance actor's followers
          → Create/Update/Delete activities delivered to peer inbox

Admin Accepts incoming Follow
  → Send Accept{Follow} to remote instance actor inbox
    → Remote instance status → accepted
      → Remote instance begins receiving our public content

Admin blocks peer domain
  → Inbox rejects activities from that domain
  → Outgoing fanout skips that peer's inbox
```

---

## Privacy and Safety Controls — Explicit Callout

These are the controls that determine admin trust in the feature. They must all be present before launch.

### Hard Requirements (non-negotiable)

1. **Public-only gate at fanout**: Before including any record in an outgoing activity, check `is_public == true`. This check must happen in the fanout function itself, not upstream, so it cannot be bypassed by future code changes.

2. **Mutual consent via ActivityPub Accept**: No content flows in either direction until both admins have explicitly accepted the Follow relationship. The existing `ProcessFollowActivity` must NOT auto-accept Application-type follows (it currently auto-accepts all remote follows without checking actor type).

3. **Reject activity on refusal**: When an admin rejects an incoming Follow, send a `Reject{Follow}` activity to the remote instance actor's inbox. Per the AP spec, this tells the remote server definitively not to proceed. Failing to send Reject leaves the remote in a permanently pending state (known Mastodon relay UX bug).

4. **Instance actor does not represent a user**: The instance actor (`preferredUsername = "instance"`, `type = Application`) must never be linked to a user record. It has no bio, avatar, or social graph visible to end users.

5. **Domain blocklist checked at inbox entry**: If a peer is added to a blocklist after a connection is established, incoming activities from that domain must be refused at the inbox handler before any processing occurs. Domain blocks are additive: they override existing accepted connections.

### Soft Requirements (important but not launch-blocking)

6. **Private note per peer**: Admins should be able to record why a connection was made or flagged, visible only to other admins on that instance. This follows the Mastodon `private_comment` pattern on domain blocks.

7. **Connection state is auditable**: The `follows` collection rows for instance actors should be queryable by admins to understand the full history of connection attempts, including rejected and unfollowed peers. Do not delete rows on rejection — set `status = "rejected"` so the record remains.

8. **No leaking of remote content to third instances**: Wanderer does not relay (re-Announce) content it received from a peer to other peers. Each instance sends its own Create activities directly to its followers. This prevents the relay amplification problem common in fediverse deployments.

---

## Connection Status State Machine

Standard across Mastodon, Pleroma, and ActivityPub relay implementations:

```
[none]
  → admin initiates Follow → pending_out (we sent Follow, awaiting Accept)
    → remote accepts → accepted (bidirectional when both sides have accepted)
    → remote rejects → rejected (terminal; admin must re-initiate to retry)
    → admin withdraws → [none] (Undo{Follow} sent)

[none]
  → remote sends Follow → pending_in (remote wants to follow us)
    → admin accepts → accepted
    → admin rejects → rejected (terminal; Reject{Follow} sent to remote)
    → admin ignores → pending_in (stays until admin acts or remote withdraws)

[accepted]
  → admin unfollows → [none] (our outgoing Follow undone; we stop receiving their content)
  → admin revokes inbound → [none] (we refuse their follow; they stop receiving our content)
  → admin blocks domain → [blocked] (overrides accepted; all delivery halted)

[blocked]
  → admin unblocks → prior status restored (or [none] if connection must be re-established)
```

Statuses to store in `follows.status` field for instance-actor rows: `pending`, `accepted`, `rejected`. Direction (inbound vs outbound) is determined by whether the local instance actor is in `follower` or `followee` position.

---

## Peer Instance List: What the Admin UI Should Show

Based on Mastodon's admin instance details page (PR #32948) and Pleroma's federation status endpoint:

| Column | Source | Notes |
|--------|--------|-------|
| Instance domain | `activitypub_actors.domain` | Extracted from actor IRI |
| Connection direction | `follows.follower` vs `follows.followee` == local instance actor | "Following them" / "They follow us" / "Mutual" |
| Status | `follows.status` | pending / accepted / rejected |
| Last activity received | `follows.updated` or delivery timestamp | Indicates health of connection |
| Software / version | NodeInfo fetch (differentiator) | Wanderer version on peer; skip if NodeInfo unavailable |
| Date connected | `follows.created` | When the Follow was first sent or received |
| Admin note | Custom field on follow row | Visible only to local admins |
| Actions | — | Accept / Reject (for pending_in); Unfollow (for accepted outbound); Revoke (for accepted inbound); Block domain |

---

## Content Types: What Should and Should Not Federate

| Content Type | Federate at Instance Level? | Rationale |
|---|---|---|
| Trails (`is_public = true`) | Yes | Core content; the primary reason instances connect |
| Summit logs (`is_public = true`) | Yes | Directly tied to trails; completes the trail data picture |
| Lists (`is_public = true`) | Yes | Curated trail collections; useful for discovery across instances |
| Comments (`is_public = true`) | Yes | Social layer; context for trails |
| Any record with `is_public = false` | Never | Hard privacy constraint |
| User private settings, DMs, follows between users | Never | Not ActivityPub public-addressed; not in Wanderer's federation scope |
| Photos/media attached to trails | Conditionally | Media URLs reference the origin instance; receiving instances render via URL. No binary relay needed. If origin goes offline, media breaks — acceptable for v1 |
| Activities from non-Wanderer instances | Reject at inbox | Instance actor should only process Create/Update/Delete for Wanderer content types; unknown activity types should be logged and dropped |

---

## MVP Recommendation

Build in this order — each step is independently testable:

1. **Instance actor creation** — Idempotent startup routine creates the Application actor in `activitypub_actors` if absent. No UI needed.
2. **Instance actor inbox route** — New HTTP route receiving activities addressed to the instance actor; initially just logs and returns 200.
3. **Outgoing Follow** — Admin triggers Follow from the instance actor to a remote instance actor URL; creates a `pending` follow row.
4. **Incoming Follow approval** — `ProcessFollowActivity` gates Application-type actors to `pending` status instead of auto-accept; admin Accept sends `Accept{Follow}`; admin Reject sends `Reject{Follow}`.
5. **ProcessAcceptActivity for instance actor** — When remote sends Accept, set our outgoing follow to `accepted`.
6. **Fanout to instance followers** — Include the instance actor in `followerInboxes()` query; existing Create/Update/Delete hooks deliver to peer inboxes automatically once the follow is accepted.
7. **Privacy gate** — Verify `is_public` check is present in fanout; add it if not.
8. **Admin visibility** — PocketBase admin collection view on `follows` filtered by instance-actor follower/followee; no custom UI needed.

Defer: delivery health metrics, NodeInfo fetch, domain blocklist, allowlist mode, import/export.

---

## Sources

- [Mastodon moderation actions — domain blocks, severity levels](https://docs.joinmastodon.org/admin/moderation/)
- [Mastodon Admin::DomainBlock API entity fields](https://docs.joinmastodon.org/entities/Admin_DomainBlock/)
- [Mastodon admin domain_blocks API methods](https://docs-p.joinmastodon.org/methods/admin/domain_blocks/)
- [Mastodon admin instance details page redesign (PR #32948)](https://github.com/mastodon/mastodon/pull/32948) — source for peer instance list UI
- [W3C ActivityPub specification — Follow/Accept/Reject semantics](https://www.w3.org/TR/activitypub/)
- [Mastodon ActivityPub spec — instance actor, Application type, relay](https://docs.joinmastodon.org/spec/activitypub/)
- [Mastodon instance peers endpoint](https://docs.joinmastodon.org/methods/instance/)
- [Activity-Relay (yukimochi) — relay subscription protocol](https://github.com/yukimochi/Activity-Relay)
- [pub-relay (noellabo) — Follow as:Public subscribe mechanism](https://github.com/noellabo/pub-relay)
- [FediMod FIRES — federation management policy framework](https://fires.fedimod.org/concepts/federation-management.html)
- [Bonfire federation interoperability guide — instance blocks, privacy controls](https://docs.bonfirenetworks.org/federation-interoperability.html)
- [Mastodon relay stuck "Waiting for approval" — why Reject must be sent](https://github.com/tootsuite/mastodon/issues/14961)
- [Pleroma/Akkoma federation reachability timeout, admin delete instance content](https://akkoma.dev/AkkomaGang/akkoma)
