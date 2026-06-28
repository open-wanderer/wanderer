# Phase 1: Instance Actor - Context

**Gathered:** 2026-06-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Create a stable `Application`-type ActivityPub actor for this Wanderer instance:
1. PocketBase migration adds `actor_type` column to `activitypub_actors` (default `"Person"`, no existing records broken)
2. `initInstanceActor()` runs at startup — creates the Application actor if it doesn't exist, never regenerates the RSA keypair if the actor already exists
3. GET `{ORIGIN}/api/v1/activitypub/instance` returns valid ActivityPub JSON for the instance actor

**In scope:** INST-01, INST-02, INST-04 (schema migration, startup init, GET endpoint)
**Out of scope:** Inbox endpoint (Phase 2), Follow lifecycle (Phase 2), content fanout (Phase 3), NodeInfo (Phase 4)

</domain>

<decisions>
## Implementation Decisions

### Actor Display Name
- **D-01:** The instance actor's `name` field is set dynamically at startup from the `ORIGIN` env var, formatted as `"Wanderer at {domain}"` (e.g., `"Wanderer at trails.example.com"`). The domain is extracted from the ORIGIN URL hostname.
- **D-02:** `preferred_username` remains `"instance"` as specified in INST-01 — only `name` is dynamic.

### Outbox Endpoint
- **D-03:** Phase 1 advertises the outbox URL (`{ORIGIN}/api/v1/activitypub/instance/outbox`) in the actor JSON but does NOT implement it. Fetching the outbox URL returns 404 until Phase 2. No stub OrderedCollection is needed.

### Code Organization
- **D-04:** `initInstanceActor()` and the GET handler (`/api/v1/activitypub/instance`) live in a new dedicated file: `db/federation/instance.go`. This keeps instance-actor concerns separate from the general user-actor code in `db/federation/actor.go`.
- **D-05:** Route registration for the GET endpoint follows the existing pattern in `db/main.go` (calling into `db/federation/instance.go` handler), consistent with how other federation routes are wired.

### Claude's Discretion
- RSA key size: follow whatever the existing user actor generation uses (no new policy needed).
- Startup hook type: use whichever PocketBase hook runs after migrations and before serving (researcher to confirm — likely `OnServe` or equivalent).
- Migration timestamp: researcher picks the next available timestamp; follows the existing `db/migrations/` pattern.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` §"Instance Actor" — INST-01, INST-02, INST-04 are the locked requirements for this phase; acceptance criteria are defined there
- `.planning/ROADMAP.md` §"Phase 1: Instance Actor" — success criteria (stable keypair across restarts, valid AP JSON, no breaking changes to existing actors)

### Existing Federation Code
- `db/federation/actor.go` — existing actor fetch/create/key-handling patterns; `GetActorByHandle`, local actor creation, `security.Encrypt/Decrypt` usage
- `db/federation/activity.go` — `followerInboxes()` and `PostActivity()` to understand how ORIGIN env var is used and how actors are passed to the delivery layer
- `db/migrations/1747061257_created_activitypub_actors.go` — existing `activitypub_actors` schema; confirms which fields already exist before adding `actor_type`
- `db/migrations/1747061259_seed_actors.go` — seed migration pattern; useful model for the `initInstanceActor()` migration or startup hook

### Backend Entry Point
- `db/main.go` — where startup hooks and route registrations happen; researcher should confirm which PocketBase lifecycle hook is used for post-migration initialization

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `security.Encrypt` / `security.Decrypt` (`github.com/pocketbase/pocketbase/tools/security`) — already used for RSA private key storage; `initInstanceActor()` must use the same pattern
- `ORIGIN` env var — already consumed by `PostActivity()` in `activity.go`; use the same `os.Getenv("ORIGIN")` call to derive the instance actor IRI and `name` field
- Existing `activitypub_actors` collection — `iri`, `preferred_username`, `inbox`, `outbox`, `public_key`, `private_key`, `is_local`, `domain` fields all present; migration only needs to add `actor_type`

### Established Patterns
- Migration files are timestamp-prefixed Go files in `db/migrations/` — new `actor_type` column migration follows this exact pattern
- Actors are uniquely identified by IRI (unique index `idx_rpT7QJwWTm` on `iri`); `initInstanceActor()` checks for existing record by IRI before creating
- `is_local = true` distinguishes local actors from remote cached actors — instance actor must set `is_local = true`

### Integration Points
- `db/main.go` — registers routes and startup hooks; `initInstanceActor()` is called from here (or its hook is registered here)
- `followerInboxes()` in `activity.go` — Phase 3 will extend this to also return instance-actor follower inboxes; Phase 1 just needs the actor record to exist so Phase 3 can query it

</code_context>

<specifics>
## Specific Ideas

No specific UI or behavioral references from discussion — implementation follows from REQUIREMENTS.md acceptance criteria directly.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 1-Instance Actor*
*Context gathered: 2026-06-24*
