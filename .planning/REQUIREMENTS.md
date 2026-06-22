# Requirements: Wanderer Instance Federation

**Defined:** 2026-06-22
**Core Value:** An administrator can connect two Wanderer instances so that public content flows between them automatically, using the same ActivityPub machinery already powering user-level federation.

## v1 Requirements

### Instance Actor

- [ ] **INST-01**: Instance actor record exists in `activitypub_actors` with `actor_type = "Application"`, RSA keypair, and stable IRI at `{ORIGIN}/api/v1/activitypub/instance`
- [ ] **INST-02**: GET `{ORIGIN}/api/v1/activitypub/instance` returns valid ActivityPub JSON (id, type, inbox, outbox, publicKey, manuallyApprovesFollowers: true)
- [ ] **INST-03**: POST `{ORIGIN}/api/v1/activitypub/instance/inbox` accepts HTTP-signed activities from authenticated remote actors
- [ ] **INST-04**: `initInstanceActor()` runs at startup, creates actor if not exists, never regenerates keypair if actor already exists

### Follow Lifecycle

- [ ] **FLCL-01**: Admin can initiate an outgoing Follow to a remote instance actor by supplying the remote IRI in PocketBase admin; local instance actor sends a Follow activity to the remote inbox
- [ ] **FLCL-02**: Incoming Follow from an `Application`-type actor creates a `pending` follow record in the `follows` collection (not auto-accepted — requires admin approval)
- [ ] **FLCL-03**: When admin sets a pending instance follow status to `accepted` in PocketBase, an `Accept{Follow}` activity is automatically delivered to the remote instance's inbox
- [ ] **FLCL-04**: When admin sets a pending instance follow status to `rejected` in PocketBase, a `Reject{Follow}` activity is delivered to the remote instance's inbox (remote must receive this to unblock its UI)
- [ ] **FLCL-05**: Admin can unfollow a peer instance; `Undo{Follow}` activity is sent to the remote; local instance stops including that peer in fanout

### Content Synchronization

- [ ] **SYNC-01**: When a public trail, summit_log, list, or comment is created, a `Create` activity is delivered to the inboxes of all accepted instance actor followers (in addition to existing user-level fanout)
- [ ] **SYNC-02**: When a public trail, summit_log, list, or comment is updated, an `Update` activity is delivered to all accepted instance actor followers
- [ ] **SYNC-03**: When a trail, summit_log, list, or comment is deleted, a `Delete` activity is delivered to all accepted instance actor followers; receiving instances remove the local cached copy

### Safety and Correctness

- [ ] **SAFE-01**: Incoming activity IRI is checked against `activitypub_activities` before dispatch; duplicate activities are silently dropped (prevents broadcast loop / re-delivery storms)
- [ ] **SAFE-02**: `processDeleteTrailActivity` verifies the deleting actor is the trail's original author before removing the local copy (fixing existing missing ownership check)
- [ ] **SAFE-03**: Outgoing fanout checks `is_public = true` before including any record; `is_public = false` records are never included regardless of connection state
- [ ] **SAFE-04**: NodeInfo endpoint at `/.well-known/nodeinfo` and `/.well-known/nodeinfo/2.1` returns instance software metadata (name: wanderer, version, user count, post count) so peer instances can identify Wanderer software

## v2 Requirements

### Content Lifecycle

- **VIS-01**: When a trail's `public` field changes from `true` to `false`, a `Delete` activity is sent to all instance followers so they remove their cached copy (public→private propagation)
- **VIS-02**: When a trail is made public after previously being private, a `Create` activity is sent to all instance followers

### Discovery

- **DISC-01**: WebFinger (`/.well-known/webfinger`) extended to resolve the instance actor IRI, enabling peers running "authorized fetch" mode to discover the instance actor via WebFinger lookup

### Admin Visibility

- **ADMIN-01**: Wanderer web settings panel shows connected peer instances with status (pending/accepted/rejected), direction (following/followed-by/mutual), and date connected

## Out of Scope

| Feature | Reason |
|---------|--------|
| Historical backfill on new connection | Complex state reconciliation; forward-only sync is sufficient; explicit PROJECT.md decision |
| Federation with non-Wanderer ActivityPub servers | Unknown content types; no shared schema for trails/summit_logs; deferred to future |
| Per-user opt-out of instance federation | All public content is federated when instances are connected; per-user control is v2+ |
| Web or mobile admin UI for federation management | PocketBase admin is sufficient for v1; admin is infrequent operation |
| Relay-style re-broadcasting (Announce pattern) | Direct bilateral Follow is simpler and avoids broadcast amplification |
| Binary media relay (photo files) | Storage/bandwidth cost; remote URLs retained, not mirrored |
| Private content federation | Hard privacy constraint — must never leave origin instance |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INST-01 | Phase 1 | Pending |
| INST-02 | Phase 1 | Pending |
| INST-03 | Phase 2 | Pending |
| INST-04 | Phase 1 | Pending |
| FLCL-01 | Phase 2 | Pending |
| FLCL-02 | Phase 2 | Pending |
| FLCL-03 | Phase 2 | Pending |
| FLCL-04 | Phase 2 | Pending |
| FLCL-05 | Phase 2 | Pending |
| SYNC-01 | Phase 3 | Pending |
| SYNC-02 | Phase 3 | Pending |
| SYNC-03 | Phase 3 | Pending |
| SAFE-01 | Phase 3 | Pending |
| SAFE-02 | Phase 3 | Pending |
| SAFE-03 | Phase 3 | Pending |
| SAFE-04 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-22*
*Last updated: 2026-06-22 after initial definition*
