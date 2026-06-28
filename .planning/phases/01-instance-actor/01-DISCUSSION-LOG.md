# Phase 1: Instance Actor - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-24
**Phase:** 1-Instance Actor
**Areas discussed:** Actor display name, Outbox in Phase 1, Code organization

---

## Actor Display Name

| Option | Description | Selected |
|--------|-------------|----------|
| "Wanderer at {domain}" | Derived from ORIGIN env var at startup (e.g., "Wanderer at trails.example.com"). Makes peer lists readable — admins can identify which instance they're connected to at a glance. | ✓ |
| "Wanderer" | Static string. Simple, but two instances would have identical names, making the peer list ambiguous for admins. | |
| Just "instance" | Matches preferred_username exactly. Technically valid but completely opaque to peer admins. | |

**User's choice:** "Wanderer at {domain}" (Recommended)
**Notes:** Domain extracted from ORIGIN hostname at startup. preferred_username stays locked to "instance" per INST-01.

---

## Outbox in Phase 1

| Option | Description | Selected |
|--------|-------------|----------|
| 404 until Phase 2 | Advertise the URL in actor JSON, implement nothing in Phase 1. Clean and honest — instance-to-instance federation doesn't need the outbox to function. | ✓ |
| Empty OrderedCollection stub | Return minimal valid AP JSON from the outbox URL in Phase 1. More spec-correct but adds work to this phase. | |

**User's choice:** 404 until Phase 2 (Recommended)
**Notes:** Outbox URL is included in the GET response JSON but the endpoint itself returns 404. Phase 2 will implement the inbox; outbox can be deferred further.

---

## Code Organization

| Option | Description | Selected |
|--------|-------------|----------|
| New file: db/federation/instance.go | Keeps instance-specific logic separate from actor.go (which handles user actors). Easier to find and extend in future phases. actor.go is already substantial. | ✓ |
| In db/federation/actor.go | Consistent with "all actor logic in one place". Simpler now, but actor.go grows further and mixes user-actor and instance-actor concerns. | |

**User's choice:** New file: db/federation/instance.go (Recommended)
**Notes:** Route registration still wires through db/main.go per existing pattern.

---

## Claude's Discretion

- RSA key size: follow existing user-actor key generation (no new policy)
- Startup hook type: researcher to confirm correct PocketBase lifecycle hook (likely OnServe)
- Migration timestamp: researcher picks next available timestamp

## Deferred Ideas

None — discussion stayed within Phase 1 scope.
