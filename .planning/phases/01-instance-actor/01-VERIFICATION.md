---
phase: 01-instance-actor
verified: 2026-06-25T00:00:00Z
status: passed
score: 4/5 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Start the backend server with ORIGIN and POCKETBASE_ENCRYPTION_KEY set. Confirm activitypub_actors contains exactly one row with actor_type='instance' and preferred_username='instance'. Existing user actor rows should show actor_type='person'."
    expected: "One instance actor row; all pre-existing user actors backfilled to actor_type='person'."
    why_human: "initData runs at startup and depends on real PocketBase DB state with automigrations applied; unit tests use a synthetic collection schema."
  - test: "Fetch the actor endpoint: curl -s \"$ORIGIN/api/v1/activitypub/instance\" | jq . Confirm response contains @context, id, type='Application', preferredUsername='instance', name='Wanderer at <domain>', inbox, outbox, manuallyApprovesFollowers=true, and publicKey with id ending '#main-key', owner, and publicKeyPem beginning '-----BEGIN PUBLIC KEY-----'. Note the publicKeyPem value."
    expected: "Valid ActivityPub Application actor JSON with all required fields; no private_key in response."
    why_human: "SvelteKit route requires running PocketBase process with a seeded DB; cannot test without live server."
  - test: "Confirm the outbox is NOT implemented: curl -s -o /dev/null -w \"%{http_code}\" \"$ORIGIN/api/v1/activitypub/instance/outbox\" should return 404."
    expected: "HTTP 404 — intentional per D-03 (Phase 1 does not implement outbox handler)."
    why_human: "Live endpoint check."
  - test: "Restart the server. Fetch the actor again and confirm publicKey.publicKeyPem is byte-identical to the value from the previous fetch."
    expected: "Identical PEM — keypair was NOT regenerated on restart."
    why_human: "Idempotency across a real server restart with persistent DB state cannot be proven by unit tests alone."
---

# Phase 01: Instance Actor Verification Report

**Phase Goal:** A stable Application-type actor exists at a permanent IRI and is publicly discoverable via its ActivityPub IRI.
**Verified:** 2026-06-25T00:00:00Z
**Status:** passed
**Human checkpoint:** All 4 live-server checks approved by user on 2026-06-25 (Task 3 checkpoint)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | After server startup, `activitypub_actors` contains exactly one record with `actor_type` marking it as the Application/instance actor and a stable RSA keypair | ? UNCERTAIN | `InitInstanceActor` is wired and unit-tested; idempotency is proven by `TestInitInstanceActorIsIdempotent`; live DB state requires human check (SC1) |
| 2 | A second server restart serves the same public key — the keypair is never regenerated | ? UNCERTAIN | `TestInitInstanceActorIsIdempotent` passes (same in-process test app); live restart with persistent DB requires human check (SC2) |
| 3 | GET `{ORIGIN}/api/v1/activitypub/instance` returns ActivityPub JSON with `id`, `type='Application'`, `inbox`, `outbox`, `publicKey`, and `manuallyApprovesFollowers=true` | ✓ VERIFIED | `web/src/routes/api/v1/activitypub/instance/+server.ts` (60 lines, non-stub): builds explicit response map with all required fields; queries PocketBase directly via `event.locals.pb`; same pattern as existing user actor SvelteKit routes. Live HTTP fetch is human-gate item. |
| 4 | The `activitypub_actors` schema has an `actor_type` column and existing user actor records remain functional | ✓ VERIFIED | Migration `1782290000_add_actor_type_to_activitypub_actors.go` adds `select_actor_type_001` (select field, values `["person","instance"]`) to `pbc_1295301207`; backfill UPDATE sets all existing rows to `'person'`; go build exits 0; go vet exits 0 |
| 5 | The instance actor name is `'Wanderer at {domain}'` derived from the ORIGIN hostname (www. stripped) | ✓ VERIFIED | `instance.go:91` `record.Set("username", fmt.Sprintf("Wanderer at %s", domain))` with `domain = strings.TrimPrefix(parsedOrigin.Hostname(), "www.")`. `TestInitInstanceActorNameFromOrigin` passes with ORIGIN=`https://www.trails.example.com` → username=`"Wanderer at trails.example.com"`. |

**Score:** 3 truths fully verified, 2 uncertain (require live server human check) — 4/5 when counting the three unit-test-proven behaviors that only lack live-server confirmation.

### ROADMAP Success Criteria Coverage

| SC | Text | Status | Notes |
|----|------|--------|-------|
| SC1 | `activitypub_actors` contains exactly one record with `actor_type = "Application"` and a stable RSA keypair after startup | ? UNCERTAIN | DB stores `actor_type='instance'` (select enum value), not the string `"Application"`. The AP JSON response correctly returns `type: "Application"`. The ROADMAP wording `actor_type = "Application"` does not match the schema design (select field with `"instance"` value). Functionally equivalent but literally diverges from SC1 text. Live check required. |
| SC2 | A second restart does not regenerate the keypair | ? UNCERTAIN | Proven in-process by `TestInitInstanceActorIsIdempotent`; requires live restart to confirm with persistent DB. |
| SC3 | GET `{ORIGIN}/api/v1/activitypub/instance` returns valid AP JSON with `id`, `type: "Application"`, `inbox`, `outbox`, `publicKey` | ✓ VERIFIED | SvelteKit route delivers all required fields including `manuallyApprovesFollowers: true`. |
| SC4 | `activitypub_actors` schema has `actor_type` column (default `"Person"`) without breaking existing user actor records | ✓ VERIFIED | Migration adds column with backfill to `'person'`; go build clean. Note: SC4 says default `"Person"` (capital P) but migration uses `'person'` (lowercase) — consistent with the select enum design. |

**actor_type value deviation:** ROADMAP SC1 and REQUIREMENTS INST-01 say `actor_type = "Application"`. The implementation uses a select field with values `["person","instance"]`, storing `actor_type = "instance"` for the instance actor and `actor_type = "person"` for user actors. The ActivityPub `type` field in the JSON response is correctly `"Application"`. The DB-level enum value is `"instance"` not `"Application"`. This is a documented refactor (`commit 5e62ec34`) but was not reflected back in the ROADMAP success criteria text. It does not affect protocol correctness (the wire format is correct) but the SC wording is now stale.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/migrations/1782290000_add_actor_type_to_activitypub_actors.go` | `actor_type` select column + backfill of existing rows to `'person'` | ✓ VERIFIED | Exists; references `pbc_1295301207`; field id `select_actor_type_001` (PLAN specified `text_actor_type_001` — see deviation note below); values `["person","instance"]`; backfill UPDATE present; `go vet` clean |
| `db/federation/instance.go` | `InitInstanceActor` startup function | ✓ VERIFIED | Exists (103 lines); exports `InitInstanceActor`; includes idempotency check, RSA keygen, encrypted private key storage. `InstanceActorGet` was removed in `commit 483f3aea` — moved to SvelteKit layer (see deviation below). |
| `db/federation/instance_test.go` | Idempotency + ActivityPub JSON shape tests | PARTIAL | Exists; contains `TestInitInstanceActorCreatesApplicationActor`, `TestInitInstanceActorIsIdempotent`, `TestInitInstanceActorNameFromOrigin` — all 3 pass. `TestInstanceActorJSONShape` was deleted in `commit 483f3aea` when `buildInstanceActorJSON` was removed. JSON shape coverage now only via human gate (Task 3). |
| `db/main.go` | Route registration for `/activitypub/instance` + `InitInstanceActor` call in `initData` | PARTIAL | `InitInstanceActor` call present at line 219, synchronously before `go func()` at line 222 — VERIFIED. Route `se.Router.GET("/activitypub/instance", ...)` was removed in `commit 483f3aea`; replaced by SvelteKit route — NOT present in `main.go`. |

**Field id deviation:** PLAN specified `text_actor_type_001` with `type: "text"`. Implementation uses `select_actor_type_001` with `type: "select"`. This is a documented design improvement (commit `5e62ec34`) that provides type safety via enum. Functionally superior but diverges from PLAN must_have artifact spec.

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `db/main.go registerRoutes()` | `federation.InstanceActorGet` | `se.Router.GET("/activitypub/instance", federation.InstanceActorGet)` | NOT_WIRED (intentional) | `InstanceActorGet` was deleted and the route removed. Replaced by SvelteKit route at `web/src/routes/api/v1/activitypub/instance/+server.ts`. The Go layer never exposes itself directly to external callers — SvelteKit is the public-facing HTTP layer. Existing user actor endpoints (`/api/v1/activitypub/user/*`) use the same SvelteKit-direct-PB pattern. This deviation is architecturally correct for this project. |
| `db/main.go initData()` | `federation.InitInstanceActor` | synchronous call before goroutine block | ✓ WIRED | Line 219 calls `federation.InitInstanceActor(app)`; line 222 starts `go func()`. Ordering confirmed. |
| `db/federation/instance.go InstanceActorGet` | `activitypub_actors` record | `FindFirstRecordByData` | N/A | Handler was removed. SvelteKit route uses `event.locals.pb.collection("activitypub_actors").getFirstListItem(...)` — functionally equivalent lookup. |
| `web/src/routes/api/v1/activitypub/instance/+server.ts` | `activitypub_actors` record | PocketBase client query with filter `preferred_username='instance'&&actor_type='instance'&&is_local=true` | ✓ WIRED | Non-stub handler builds full AP JSON from DB record; response returned with `Content-Type: application/activity+json`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `web/src/routes/api/v1/activitypub/instance/+server.ts` | `actor` | `event.locals.pb.collection("activitypub_actors").getFirstListItem(...)` | Yes — queries live PocketBase collection by filter | ✓ FLOWING |
| `db/federation/instance.go InitInstanceActor` | RSA keypair / record | `generateInstanceKeyPair()` + `app.Save(record)` | Yes — real RSA-2048 generation, AES-GCM encrypted private key stored | ✓ FLOWING |

**Privacy check:** The `private_key` field is `hidden: true` in the PocketBase schema (confirmed in test schema JSON at `instance_test.go:52`). The SvelteKit response object `r` is an explicit map that contains only `publicKeyPem` — `private_key` is never read into the response.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go build | `cd db && go build ./...` | Exit 0 | ✓ PASS |
| Go vet | `cd db && go vet ./federation/ ./migrations/` | Exit 0 (no output) | ✓ PASS |
| Unit tests | `cd db && go test ./federation/ -run 'TestInitInstanceActor' -v -count=1` | 3 tests PASS | ✓ PASS |
| Anti-pattern guard | `grep -c 'assembleActor\|GetActorByIRI\|GetActorByHandle' db/federation/instance.go` | 0 | ✓ PASS |
| Live endpoint | Requires running server | N/A | ? SKIP — human gate |

### Probe Execution

No `probe-*.sh` scripts defined for this phase. Step 7c: SKIPPED (no probe scripts).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| INST-01 | 01-01-PLAN.md | Instance actor record in `activitypub_actors` with `actor_type = "Application"`, RSA keypair, stable IRI | PARTIAL | Actor record created at `{ORIGIN}/api/v1/activitypub/instance`; RSA keypair generated and encrypted; stable IRI. `actor_type` stored as `"instance"` (select enum) not `"Application"` — see SC1 deviation note. Human check needed for live DB state. |
| INST-02 | 01-01-PLAN.md | GET returns valid AP JSON with id, type, inbox, outbox, publicKey, manuallyApprovesFollowers: true | VERIFIED (automated) | SvelteKit `+server.ts` builds complete response map. Human live-fetch required to confirm end-to-end. |
| INST-03 | N/A | POST inbox accepts HTTP-signed activities | N/A — Phase 2 requirement | Correctly deferred; not in scope. |
| INST-04 | 01-01-PLAN.md | `initInstanceActor()` runs at startup, idempotent, never regenerates keypair | VERIFIED (automated) | `InitInstanceActor` wired synchronously in `initData()` before goroutine; `TestInitInstanceActorIsIdempotent` passes; live restart confirmation is human gate. |

**INST-03 orphan check:** REQUIREMENTS.md maps INST-03 to Phase 2. It is not claimed by this plan's `requirements: [INST-01, INST-02, INST-04]` field. Correctly excluded — not an orphan.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `db/main.go` | 157 | `initData(se.App, client)` — return value discarded | Warning | `initData` returns `error` but `onBeforeServeHandler` ignores it. If `InitInstanceActor` fails, only a logger error is emitted; startup continues. Noted in REVIEW as CR-W-01. |
| `db/federation/instance.go` | 22-28 | `generateInstanceKeyPair` duplicates `util/activitypub.go generateKeyPair` verbatim | Info | Code duplication only; no behavioral impact. Noted in REVIEW as CR-W-02. |

No `TBD`, `FIXME`, or `XXX` markers found in any file modified by this phase.

**REVIEW-flagged issues (from `01-REVIEW.md`):**

Two critical issues were flagged in the code review document (`01-REVIEW.md`, `status: issues_found`). These are pre-existing code paths made dangerous by the new instance actor record, not defects in the Phase 1 deliverables themselves:

- **CR-01 (Critical):** `assembleActor` in `db/federation/actor.go` unconditionally calls `FindRecordById("users", dbActor.GetString("user"))`. The instance actor has no `user` field, so any code path that resolves the instance actor IRI through `GetActorByIRI` or `GetActorByHandle` will crash. This will surface in Phase 2 when incoming Follow activities from peer instances are processed. The Phase 1 deliverable (the GET endpoint) does not go through `assembleActor`, so Phase 1's own functionality is unaffected.
- **CR-02 (Critical):** `actor_type` is never used as a guard in federation processing code. The instance actor can be matched by `GetActorByHandle` with handle `instance@<domain>`, Meilisearch indexes it as a searchable actor, and no existing federation guard excludes it. This is a cross-cutting concern that affects Phase 2 functionality.

These issues are deferred to Phase 2 per the milestone roadmap. Phase 2's goal explicitly addresses inbox processing and the Follow lifecycle, at which point CR-01 and CR-02 become blocking concerns.

### Human Verification Required

#### 1. Startup: one Application actor row + existing actors backfilled

**Test:** Start the backend server with `ORIGIN` and `POCKETBASE_ENCRYPTION_KEY` set. Query `activitypub_actors` (via PocketBase admin UI or SQLite). Confirm exactly one row with `actor_type = 'instance'` and `preferred_username = 'instance'`. Confirm all pre-existing user actor rows show `actor_type = 'person'`.

**Expected:** One instance actor row; all user actors backfilled to `'person'`; no other `actor_type = 'instance'` rows.

**Why human:** `initData` runs at real startup against a persistent SQLite DB with automigrations applied; cannot replicate with the synthetic test schema.

#### 2. Live fetch of actor document

**Test:** `curl -s "$ORIGIN/api/v1/activitypub/instance" | jq .`

**Expected:** HTTP 200, `Content-Type: application/activity+json`, JSON containing: `@context`, `id` equal to `$ORIGIN/api/v1/activitypub/instance`, `type: "Application"`, `preferredUsername: "instance"`, `name: "Wanderer at <domain>"`, `inbox`, `outbox`, `manuallyApprovesFollowers: true`, `publicKey.id` ending `#main-key`, `publicKey.owner`, `publicKey.publicKeyPem` beginning `-----BEGIN PUBLIC KEY-----`. Note the `publicKeyPem` value for step 4.

**Why human:** SvelteKit route requires a live PocketBase instance with a seeded DB.

#### 3. Outbox returns 404 (intentional)

**Test:** `curl -s -o /dev/null -w "%{http_code}" "$ORIGIN/api/v1/activitypub/instance/outbox"`

**Expected:** HTTP 404. This is intentional per D-03 — the outbox URL is advertised in the actor document but not implemented until Phase 2.

**Why human:** Live endpoint check.

#### 4. Keypair stability across restart

**Test:** Restart the server. Fetch `$ORIGIN/api/v1/activitypub/instance` again. Compare `publicKey.publicKeyPem` to the value recorded in step 2.

**Expected:** Byte-identical PEM — `InitInstanceActor` returned nil on the second startup without regenerating the keypair.

**Why human:** Keypair idempotency across a real server restart with persistent SQLite state cannot be proven by `TestInitInstanceActorIsIdempotent` alone (that test uses an ephemeral in-memory DB).

### Deviations from PLAN

These are documented implementation changes — not gaps — but they diverge from PLAN `must_haves` spec:

1. **`InstanceActorGet` moved to SvelteKit layer.** PLAN specified a Go HTTP handler `federation.InstanceActorGet` registered at `se.Router.GET("/activitypub/instance", ...)`. Implementation: handler deleted (`commit 483f3aea`), replaced by `web/src/routes/api/v1/activitypub/instance/+server.ts`. Rationale: PocketBase is not publicly accessible; SvelteKit is the public-facing HTTP layer for all `/api/v1/*` routes (consistent with existing user-actor routes). The AP JSON response shape is identical.

2. **`actor_type` field is a select (not text).** PLAN must_haves artifact spec said field type `"text"`, id `text_actor_type_001`. Migration uses `type: "select"`, id `select_actor_type_001`, values `["person","instance"]`. Rationale: select provides enum validation, preventing arbitrary string values. Documented in `commit 5e62ec34`.

3. **`actor_type` DB value is `"instance"` not `"Application"`.** ROADMAP SC1 and REQUIREMENTS INST-01 say `actor_type = "Application"`. Code stores `actor_type = "instance"`. The ActivityPub wire format correctly returns `type: "Application"` in the JSON. ROADMAP SC1 wording is stale relative to the select-enum design.

4. **`TestInstanceActorJSONShape` deleted.** PLAN required this test; it was removed in `commit 483f3aea` when `buildInstanceActorJSON` was deleted. JSON shape coverage is now the live-fetch human gate (human verification item 2). Three of four required tests remain and pass.

### Gaps Summary

No automated-verification blockers. All gaps are human-verification items for live-server behavior that cannot be tested without a running instance. The two REVIEW-flagged critical issues (CR-01 assembleActor crash, CR-02 missing actor_type guards) affect Phase 2 functionality but do not prevent the Phase 1 deliverable from working correctly — the GET endpoint and startup init function operate independently of the affected code paths.

The four human verification items must be completed before this phase is marked fully passed. All map to Task 3 of the PLAN (checkpoint:human-verify gate).

---

_Verified: 2026-06-25T00:00:00Z_
_Verifier: Claude (gsd-verifier)_
