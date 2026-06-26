# Wanderer Instance Federation

## What This Is

Instance-level ActivityPub federation for Wanderer. An instance administrator connects their Wanderer instance to another by establishing a mutual Follow relationship between two instance-level actors. Once connected, all public trails, comments, lists, and summit_logs are synchronized bidirectionally — each piece of content retaining its original actor so ownership and edit permissions remain on the origin instance.

## Core Value

An administrator can connect two Wanderer instances so that public content flows between them automatically, using the same ActivityPub machinery already powering user-level federation.

## Requirements

### Validated

- [x] Public trails, comments, lists, and summit_logs are federated; private content never leaves its instance — Validated in Phase 03 (fanout-and-safety)
- [x] Update and Delete activities propagate to all following instances — Validated in Phase 03 (fanout-and-safety)
- [x] Instance fanout uses the same delivery infrastructure as user-level federation — Validated in Phase 03 (fanout-and-safety)

### Active

- [ ] Instance actor exists for each Wanderer deployment (a special actor in `activitypub_actors` with `type = Application`)
- [ ] Admin can send a Follow request to a remote instance actor via PocketBase admin UI
- [ ] Remote admin must accept the Follow before sync begins (mutual approval)
- [ ] Accepted Follow causes the origin to receive Create/Update/Delete activities for all public content from the followed instance
- [ ] Public trails, comments, lists, and summit_logs are federated; private content never leaves its instance
- [ ] Update and Delete activities propagate to all following instances
- [ ] Remote content retains its original actor IRI — users on the receiving instance cannot edit it
- [ ] Admin can view connected instances and connection status (pending/accepted/rejected) in PocketBase admin UI
- [ ] Admin can unfollow (disconnect from) a peer instance

### Out of Scope

- Web or mobile UI for federation management — PocketBase admin only for v1
- Per-user opt-in/opt-out of instance federation — all public content is federated when instances are connected
- Selective content filtering per peer (e.g., share trails but not comments) — all four content types sync together
- Backfill of historical content when a new connection is established — only new activities after the Follow is accepted

## Context

The existing codebase has a fully functional ActivityPub federation layer:

- `db/federation/` — `actor.go`, `follow.go`, `create.go`, `update.go`, `delete.go`, `activity.go`, etc.
- `activitypub_actors` PocketBase collection stores both local and remote actors with IRI, inbox, outbox, public/private key pairs
- `follows` collection tracks follower/followee relationships with `status` (pending/accepted)
- `PostActivity()` handles HTTP-signed delivery to remote inboxes
- `ProcessFollowActivity()` handles incoming Follow requests and auto-accepts them

The key gap: all existing actors represent individual users. An instance actor (`type = Application`, `preferred_username = instance`) needs to be created at startup and included in the fanout when any public content is created/updated/deleted.

Currently, `followerInboxes()` already performs an efficient JOIN to find accepted followers — the instance actor just needs to be a valid entry in `activitypub_actors` so it can send and receive activities.

## Constraints

- **Tech Stack**: Go/PocketBase backend only for federation logic; no SvelteKit or Flutter changes required for v1
- **ActivityPub compatibility**: Must use standard ActivityPub types — `Application` actor type for instance actor, same activity types already in use (Create, Update, Delete, Follow, Accept, Undo)
- **Privacy hard constraint**: `is_public = false` records must never be included in outgoing activities — checked at fanout time
- **Online-only**: No offline sync buffer; if a remote instance is unreachable, the activity is dropped (existing behavior for user federation)
- **No breaking changes**: Existing user-level federation must be unaffected; instance actor is additive

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Extend existing ActivityPub actor model rather than a new sync protocol | Reuses all existing delivery, signing, and inbox processing infrastructure | — Pending |
| PocketBase admin UI only (no web/mobile admin) | Fastest path to v1; instance federation is an infrequent admin operation | — Pending |
| Mutual approval (not auto-accept) for instance follows | Prevents unwanted data ingestion; admins must explicitly consent on both sides | — Pending |
| No historical backfill on new connection | Avoids complex state reconciliation; forward-only sync is simpler and sufficient | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-06-26 after Phase 03 (fanout-and-safety) completion*
