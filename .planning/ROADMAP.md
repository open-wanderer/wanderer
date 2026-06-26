# Roadmap: Wanderer Instance Federation

## Overview

Four phases take the Wanderer Go/PocketBase backend from zero instance-level federation to a fully operational bilateral ActivityPub sync. Phase 1 lays the schema and creates the instance actor. Phase 2 wires up the inbox endpoint and the full Follow lifecycle so admins can connect two instances. Phase 3 activates content fanout with the broadcast-loop guard, privacy gate, and ownership-check fix bundled together — they cannot ship separately. Phase 4 adds the NodeInfo well-known endpoint so peer instances can identify Wanderer software.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Instance Actor** - Schema migration and idempotent instance actor initialization with GET endpoint (completed 2026-06-25)
- [x] **Phase 2: Follow Lifecycle** - Inbox endpoint + admin-gated Follow/Accept/Reject/Undo flow (completed 2026-06-25)
- [x] **Phase 3: Fanout and Safety** - Content sync to instance followers, broadcast-loop deduplication, privacy gate, ownership fix (completed 2026-06-26)
- [ ] **Phase 4: NodeInfo** - Well-known NodeInfo endpoint so peers can identify this Wanderer instance

## Phase Details

### Phase 1: Instance Actor

**Goal**: A stable Application-type actor exists for this Wanderer instance and is publicly discoverable via its ActivityPub IRI
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: INST-01, INST-02, INST-04
**Success Criteria** (what must be TRUE):

  1. After server startup, `activitypub_actors` contains exactly one record with `actor_type = "Application"` and a stable RSA keypair
  2. A second server restart does not regenerate the keypair — the same public key is served each time
  3. GET `{ORIGIN}/api/v1/activitypub/instance` returns valid ActivityPub JSON containing `id`, `type: "Application"`, `inbox`, `outbox`, and `publicKey`
  4. The `activitypub_actors` schema has an `actor_type` column (default `"Person"`) without breaking existing user actor records

**Plans**: 1 plan
Plans:

- [x] 01-01-PLAN.md — Schema migration, idempotent instance actor init, and GET endpoint

### Phase 2: Follow Lifecycle

**Goal**: An admin can connect two Wanderer instances by initiating or receiving a Follow, approving or rejecting it, and disconnecting later — with correct AP activities delivered at each step
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: INST-03, FLCL-01, FLCL-02, FLCL-03, FLCL-04, FLCL-05
**Success Criteria** (what must be TRUE):

  1. POST `{ORIGIN}/api/v1/activitypub/instance/inbox` accepts HTTP-signed activities and routes them to the existing activity processor
  2. An incoming Follow from an `Application`-type actor is stored as `pending` in the `follows` collection and is NOT auto-accepted
  3. When an admin sets a pending instance follow to `accepted` in PocketBase, an `Accept{Follow}` activity is delivered to the remote instance's inbox
  4. When an admin sets a pending instance follow to `rejected` in PocketBase, a `Reject{Follow}` activity is delivered to the remote instance's inbox
  5. An admin-initiated Undo removes the peer from the follows collection and sends an `Undo{Follow}` activity to the remote

**Plans**: 3 plansPlans:
**Wave 1**

- [x] 02-01-PLAN.md — Migration: add "rejected" to follows.status (prereq for FLCL-04)
- [x] 02-02-PLAN.md — Instance inbox endpoint + incoming Follow → pending (INST-03, FLCL-02)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-03-PLAN.md — Admin lifecycle hooks: outgoing Follow, Accept, Reject, Undo (FLCL-01/03/04/05)

### Phase 3: Fanout and Safety

**Goal**: Public content created, updated, or deleted on this instance is automatically delivered to all accepted instance-actor followers, with broadcast-loop prevention, privacy enforcement, and correct delete authorization in place
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: SYNC-01, SYNC-02, SYNC-03, SAFE-01, SAFE-02, SAFE-03
**Success Criteria** (what must be TRUE):

  1. When a public trail, summit_log, list, or comment is created, accepted instance actor followers receive a `Create` activity in addition to existing user-level fanout
  2. When a public trail, summit_log, list, or comment is updated or deleted, accepted instance actor followers receive the corresponding `Update` or `Delete` activity
  3. A `Create` activity received from a peer that was already processed (IRI exists in `activitypub_activities`) is silently dropped — no duplicate delivery occurs
  4. A trail with `is_public = false` is never included in outgoing fanout activities regardless of instance follow state
  5. A `Delete{Trail}` activity from a remote actor is only applied if the actor is the trail's original author — unauthorized delete attempts are rejected

**Plans**: 3 plans
Plans:
**Wave 1**

- [x] 03-01-PLAN.md — instanceFollowerInboxes helper + unit tests (SYNC-01/02/03 foundation)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 03-02-PLAN.md — create.go fanout injection, SAFE-01 dedup guards, SAFE-03 comment privacy gate

**Wave 3** *(blocked on Waves 1-2)*

- [x] 03-03-PLAN.md — SAFE-02 delete authorization, SYNC-03 delete fanout, D-10 instance inbox content dispatch

### Phase 4: NodeInfo

**Goal**: Peer instances and federation tools can identify this server as Wanderer software via the NodeInfo well-known endpoints
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: SAFE-04
**Success Criteria** (what must be TRUE):

  1. GET `/.well-known/nodeinfo` returns a JSON discovery document linking to the NodeInfo 2.1 endpoint
  2. GET `/.well-known/nodeinfo/2.1` returns valid NodeInfo JSON with `software.name: "wanderer"`, a version string, and current user and post counts

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Instance Actor | 1/1 | Complete    | 2026-06-25 |
| 2. Follow Lifecycle | 4/4 | Complete   | 2026-06-25 |
| 3. Fanout and Safety | 3/3 | Complete   | 2026-06-26 |
| 4. NodeInfo | 0/? | Not started | - |
