---
phase: 01-instance-actor
plan: 01
subsystem: federation/instance-actor
tags: [activitypub, instance-actor, migration, go, pocketbase]
dependency_graph:
  requires: []
  provides: [instance-actor-record, actor-type-column, GET-activitypub-instance]
  affects: [db/federation, db/migrations, db/main.go]
tech_stack:
  added: []
  patterns:
    - idempotent-startup-init (InitInstanceActor with IRI existence check before keypair creation)
    - map-based-ap-json (manuallyApprovesFollowers via map[string]any instead of go-ap struct)
    - programmatic-test-collection (Bootstrap + ImportCollectionsByMarshaledJSON without full migration chain)
key_files:
  created:
    - db/federation/instance.go
    - db/migrations/1782290000_add_actor_type_to_activitypub_actors.go
    - db/federation/instance_test.go
  modified:
    - db/main.go
decisions:
  - D-01: instance actor name is "Wanderer at {domain}" derived from ORIGIN hostname (www. stripped)
  - D-02: preferred_username stays "instance" (not the full name)
  - D-03: outbox URL advertised but returns 404 until Phase 2 (no stub OrderedCollection)
  - D-04: instance actor lives in db/federation/instance.go (separate from user actor code)
  - D-05: route registered in main.go registerRoutes() following existing AP route pattern
  - IMPL: buildInstanceActorJSON extracted as private helper to enable direct testing without HTTP request event
  - IMPL: test uses programmatic collection creation (not RunAllMigrations) to avoid Meilisearch external dependency
metrics:
  duration: "~25 minutes"
  completed: "2026-06-24"
  tasks_completed: 3
  tasks_total: 3
  files_created: 3
  files_modified: 1
---

# Phase 01 Plan 01: Instance Actor Summary

**One-liner:** Stable Application-type instance actor with idempotent RSA keypair init and ActivityPub GET endpoint, backed by actor_type schema migration.

## What Was Built

A complete vertical slice delivering the instance actor for Wanderer:

1. **Schema migration** (`db/migrations/1782290000_add_actor_type_to_activitypub_actors.go`): Adds `actor_type` select column (field id `select_actor_type_001`, values `["person","instance"]`, `maxSelect: 1`) to `activitypub_actors` collection (`pbc_1295301207`). Backfills all existing user actor rows to `actor_type='person'`.

2. **Instance actor init and handler** (`db/federation/instance.go`):
   - `InitInstanceActor(app core.App) error` — idempotent startup function. Checks for existing actor by IRI before creating; returns nil immediately if found (never regenerates keypair). Creates the Application actor record with `actor_type="instance"`, `preferred_username="instance"`, `username="Wanderer at {domain}"`, stable RSA-2048 keypair (encrypted private key via `security.Encrypt`).
   - `buildInstanceActorJSON(record, iri)` — private helper building the AP JSON map including `manuallyApprovesFollowers: true` (absent from go-ap Actor struct, requires explicit map construction).
   - `InstanceActorGet(e *core.RequestEvent) error` — HTTP handler serving the actor document. Looks up by IRI directly (NOT via actor assembly helpers — those fail for userless actors).

3. **Main wiring** (`db/main.go`):
   - Added `pocketbase/federation` import.
   - Route `GET /activitypub/instance` to `federation.InstanceActorGet` in `registerRoutes()`.
   - Synchronous `federation.InitInstanceActor(app)` call in `initData()` BEFORE the goroutine block.

4. **Tests** (`db/federation/instance_test.go`): 4 tests covering actor creation, idempotency, name derivation, and AP JSON shape. Uses `core.NewBaseApp` + `ImportCollectionsByMarshaledJSON` to create a test environment without the full migration chain (avoids Meilisearch external service dependency).

## TDD Gate Compliance

- **RED commit** `6c4c007d`: `test(01-01): add failing tests for instance actor` — tests failed to compile because `InitInstanceActor` and `buildInstanceActorJSON` were undefined.
- **GREEN commit** `aaf885e8`: `feat(01-01): create instance actor, migration, and GET endpoint` — all 4 tests pass.

## Task Commits

| Task | Type | Commit | Description |
|------|------|--------|-------------|
| 1 (RED) | test | `6c4c007d` | Failing tests for instance actor |
| 2 (GREEN) | feat | `aaf885e8` | Migration + instance.go + main.go wiring |
| 2.5 | refactor | `5e62ec34` | actor_type → select field with person/instance values |
| 3 | checkpoint | `approved` | Live restart + endpoint verification — all 4 checks passed |

## Phase-Level Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `go vet ./federation/ ./migrations/` | PASS |
| Unit tests (4 tests) | PASS |
| Anti-pattern guard (0 anti-pattern refs in instance.go) | PASS |
| Route wiring in main.go | PASS |
| Init call synchronous in initData (before goroutine) | PASS |
| Migration references pbc_1295301207, select_actor_type_001, backfill | PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Test harness skips Meilisearch migration**
- **Found during:** Task 2 GREEN run
- **Issue:** `RunAllMigrations()` in `newTestApp` hit migration `1742167033_init_meilisearch.go` which calls external Meilisearch service. Fails with "unsupported protocol scheme" when MEILI_URL is unset.
- **Fix:** Replaced `RunAllMigrations()` with `Bootstrap()` (system migrations only, creates `_collections`) + `ImportCollectionsByMarshaledJSON()` to create just the `activitypub_actors` collection programmatically. Added `_ "github.com/pocketbase/pocketbase/migrations"` import for PocketBase internal system migrations (creates `_collections` table).
- **Files modified:** `db/federation/instance_test.go`
- **Commit:** `aaf885e8`

**2. [Rule 1 - Bug] Anti-pattern guard violated by comments**
- **Found during:** Task 2 acceptance criteria check
- **Issue:** The `InstanceActorGet` docstring originally named `assembleActor`, `GetActorByIRI`, and `GetActorByHandle`, causing `grep -c` to return 2 instead of 0.
- **Fix:** Rewrote the comment to describe the behavior without naming the anti-pattern functions.
- **Files modified:** `db/federation/instance.go`
- **Commit:** `aaf885e8`

## Outstanding: Task 3 — Human Verification (Checkpoint)

Task 3 is a `type="checkpoint:human-verify"` gate requiring live server verification. Four checks:
1. After startup, one Application actor row in `activitypub_actors`, existing user actors backfilled to Person
2. `curl "$ORIGIN/api/v1/activitypub/instance"` returns valid AP JSON with all required fields
3. Outbox URL returns 404 (expected — Phase 1 doesn't implement it)
4. After server restart, `publicKeyPem` is byte-identical

## Known Stubs

None. The outbox URL is advertised (`{ORIGIN}/api/v1/activitypub/instance/outbox`) but deliberately returns 404 per D-03 — this is intentional design, documented in CONTEXT.md. Phase 2 will implement inbox/outbox.

## Threat Flags

None — all new surface is accounted for in the plan threat model:
- `GET /activitypub/instance` exposes only public actor fields via explicit `map[string]any`
- `private_key` column is never read into the response
- Idempotency check prevents keypair regeneration on restart
- Unique index on `iri` enforces single-record constraint

## Self-Check: PASSED

| Item | Status |
|------|--------|
| `db/federation/instance.go` exists | FOUND |
| `db/federation/instance_test.go` exists | FOUND |
| `db/migrations/1782290000_add_actor_type_to_activitypub_actors.go` exists | FOUND |
| `.planning/phases/01-instance-actor/01-01-SUMMARY.md` exists | FOUND |
| Commit `6c4c007d` (test RED) exists | FOUND |
| Commit `aaf885e8` (feat GREEN) exists | FOUND |
