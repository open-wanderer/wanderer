# Phase 5: Federation Admin API - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-27
**Phase:** 05-federation-admin-api
**Areas discussed:** Discovery → Follow coupling, Peer list endpoint scope, Instance name in preview card

---

## Discovery → Follow Coupling

### Q1: When an admin hits /federation/follow, what does it expect as input?

| Option | Description | Selected |
|--------|-------------|----------|
| Actor ID from discovery | Discovery stores the remote actor and returns its ID. /follow receives that ID and just creates the follows record. No HTTP in the follow handler — clean SAFE-07 separation. | ✓ |
| Remote URL, re-resolved | /follow accepts a URL, fetches and upserts the actor itself. Discovery is purely informational. Adds HTTP to the follow handler. | |
| Follow record ID (pre-created) | Discovery creates both the actor AND a pending follows record. /follow just triggers delivery (conflicts with SAFE-07). | |

**User's choice:** Actor ID from discovery
**Notes:** Clean SAFE-07 separation — no HTTP in follow handler.

---

### Q2: What does /federation/discover return when the remote is valid?

| Option | Description | Selected |
|--------|-------------|----------|
| Actor ID + preview card data | Returns local activitypub_actors record ID plus NodeInfo preview fields. Phase 6 passes actor ID to /follow. | ✓ |
| Full actor record | Returns the full activitypub_actors record. More data, slightly larger response. | |
| Preview card only (no actor ID) | Display data only. Phase 6 must call discover again before /follow — extra round-trip. | |

**User's choice:** Actor ID + preview card data
**Notes:** Clean handoff — actor_id is the cursor between discover and follow.

---

### Q3: If /federation/follow receives an actor ID that doesn't exist locally?

| Option | Description | Selected |
|--------|-------------|----------|
| 400 Bad Request — discovery required first | Enforces the two-step flow. Simple handler — just finds actor by ID or returns 400. | ✓ |
| Silently fetch and store the actor | Fallback to self-contained resolution. Adds HTTP to follow handler, loosens SAFE-07 intent. | |
| You decide | Let Claude choose. | |

**User's choice:** 400 Bad Request
**Notes:** Enforces the two-step discover-first workflow.

---

## Peer List Endpoint Scope

### Q1: Should Phase 5 include GET /federation/peers?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — lightweight JSON endpoint | Returns follow_id, direction, status, domain. Phase 6 wraps this. Follows Phase 5 = API, Phase 6 = HTML separation. | ✓ |
| No — Phase 6 queries PocketBase directly | Phase 6 queries follows collection directly. Fewer Phase 5 endpoints but Phase 6 does more data joining. | |
| Yes — full actor data included | Returns everything including actor IRI, inbox, NodeInfo preview data. Heavier response. | |

**User's choice:** Yes — lightweight JSON (follow_id, direction, status, domain)

---

### Q2: What fields per peer?

| Option | Description | Selected |
|--------|-------------|----------|
| follow_id, direction, status, domain | Minimal: everything Phase 6 needs for the dashboard. | ✓ |
| Add actor IRI and inbox | Include actor IRI and inbox URL for profile links. | |
| Add cached NodeInfo preview data | Also include version and counts. Requires storing in actor record. | |

**User's choice:** follow_id, direction, status, domain

---

### Q3: How is "mutual" direction computed?

| Option | Description | Selected |
|--------|-------------|----------|
| Single row labeled "mutual" | Handler detects both directions, collapses to one entry. Cleaner for Phase 6. | ✓ |
| Two separate rows | Return both follow records. Phase 6 deduplicates. | |
| You decide | Let Claude pick. | |

**User's choice:** Single row labeled "mutual"
**Notes (user clarification):** Instance follows are automatically mutual — if both parties have accepted, A follows B and B follows A. The handler groups by remote domain and collapses two accepted-direction records into one "mutual" entry.

---

## Instance Name in Preview Card

### Q1: Where should the preview card's "instance name" come from?

| Option | Description | Selected |
|--------|-------------|----------|
| Domain extracted from URL | Hostname from admin-supplied URL (e.g., "trails.example.com"). Always available, no extra fetch. | ✓ |
| ActivityPub actor "name" field | Fetch remote instance actor and use its name field. Requires one extra HTTP fetch. | |
| You decide | Let Claude pick — domain is fine for admin-facing tool. | |

**User's choice:** Domain extracted from URL

---

### Q2: What data sources does /federation/discover fetch?

| Option | Description | Selected |
|--------|-------------|----------|
| NodeInfo only (+ GetActorByIRI) | GET /.well-known/nodeinfo → follow link → NodeInfo 2.1. Yields version + counts. Actor fetched via GetActorByIRI. | ✓ |
| NodeInfo + explicit actor fetch | Same plus extra GET to actor IRI. Redundant since GetActorByIRI already fetches. | |
| Actor only (skip NodeInfo) | Just fetch actor. No version or counts — doesn't meet DISC-01. | |

**User's choice:** NodeInfo only (reuse GetActorByIRI for actor)

---

### Q3: How is "is Wanderer" confirmed?

| Option | Description | Selected |
|--------|-------------|----------|
| NodeInfo software.name == "wanderer" | Strict check: exact string match. Simple and deterministic. | ✓ |
| NodeInfo present + actor has actor_type field | Two-factor check. More permissive, harder to test. | |
| You decide | Let Claude pick. | |

**User's choice:** NodeInfo software.name == "wanderer"

---

## Claude's Discretion

- Error response format: `e.JSON(http.StatusXxx, map[string]any{"error": "..."})` — consistent with existing handlers
- Route file: `db/routes/federation_admin.go` following nodeinfo.go pattern
- Superuser auth: whether to use shared middleware or inline check per handler
- DISC-02 error message strings for each failure case
- Disconnect direction logic for mutual peers (both records need updating — researcher to handle per CONN-04)

## Deferred Ideas

- Storing NodeInfo preview data (version, counts) in actor record for dashboard display without re-fetch — v2 UX-02
- Realtime status updates via PocketBase subscription — v2 UX-01
- WebFinger instance actor resolution — v2 DISC-03
- Mutual disconnect behavior (handling both inbound and outbound records atomically) — deferred to researcher/planner to implement per CONN-04
