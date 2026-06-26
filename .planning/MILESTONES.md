# Milestones

## v1.0 Instance Federation (Shipped: 2026-06-26)

**Phases completed:** 4 phases, 9 plans, 18 tasks

**Key accomplishments:**

- Stable Application-type instance actor with idempotent RSA keypair init and ActivityPub GET endpoint, backed by actor_type schema migration.
- PocketBase migration 1782290001 appends "rejected" to follows.status select field (collection 8obn1ukumze565i), enabling the Reject{Follow} lifecycle path without breaking existing pending/accepted values
- Task 1 (TDD): Actor-type branch in ProcessFollowActivity
- Five correctness defects in Phase 02 follow-lifecycle code fixed: double-delivery guards on two hooks, directional guard preventing inbound-follow mis-firing, panic-safe type assertion on Accept objects, and honest 400 error propagation in the instance inbox.
- instanceFollowerInboxes helper added to activity.go: resolves instance actor by ORIGIN-derived IRI, delegates to followerInboxes JOIN, returns (nil, nil) when actor not yet seeded
- 1. [Rule 1 - Bug] pub.Place struct literal had non-existent `Parent` field
- NodeInfo 2.1 and JRD discovery endpoints implemented with TDD, serving live user/post counts and software identity to peer ActivityPub instances (SAFE-04)

---
