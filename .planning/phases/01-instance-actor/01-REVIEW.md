---
phase: 01-instance-actor
reviewed: 2026-06-25T00:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - db/federation/instance.go
  - db/federation/instance_test.go
  - db/main.go
  - db/migrations/1782290000_add_actor_type_to_activitypub_actors.go
  - web/src/routes/api/v1/activitypub/instance/+server.ts
findings:
  critical: 2
  warning: 4
  info: 2
  total: 8
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-06-25T00:00:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

This phase introduces the instance-level ActivityPub actor: a Go `InitInstanceActor` function that idempotently seeds an `Application`-type actor record, a migration that adds the `actor_type` select field, and a SvelteKit endpoint that serves the actor document. The core happy-path logic is sound. However, two critical defects exist: (1) the `assembleActor` function in the existing `actor.go` will crash with a database error whenever the instance actor is resolved through any of the generic `GetActorByIRI` / `GetActorByHandle` code paths, because it unconditionally looks up a `users` record that the instance actor does not have; and (2) the `actor_type` field introduced by the migration is never used as a filter or guard in any code path other than the narrow SvelteKit GET endpoint, meaning the instance actor is completely transparent to all existing federation logic and can be processed as if it were a user actor. There are also four warnings covering: the `initData` return value being silently discarded, code duplication of the key-generation function, the migration's down-migration not backfilling the instance actor's `actor_type` column before removing the column, and the inbox/outbox URIs advertised in the actor document having no corresponding route handlers.

---

## Critical Issues

### CR-01: `assembleActor` crashes when instance actor is resolved via generic lookup paths

**File:** `db/federation/actor.go:170-177`

**Issue:** `assembleActor` unconditionally branches on `is_local = true` and then calls `app.FindRecordById("users", dbActor.GetString("user"))`. The instance actor record has `is_local = true` but no `user` field value — `GetString("user")` returns an empty string. PocketBase's `FindRecordById` with an empty ID returns an error, causing `assembleActor` to return `nil, err`. Any caller that resolves the instance actor's IRI (e.g., an inbound ActivityPub activity whose `actor` field is the instance actor IRI, processed by `routes/activitypub.go:96` via `GetActorByIRI`) will receive this error and the activity will be rejected.

The six callers of `GetActorByIRI` and `GetActorByHandle` in the codebase all go through `assembleActor`. As federation with peer instances is added (the explicit goal of this phase), the instance actor's IRI will appear as the `actor` on incoming `Follow`, `Accept`, and `Undo` activities — all of which will fail.

**Fix:** Guard the `users` lookup with an `actor_type` check before entering the user-specific branch:

```go
func assembleActor(app core.App, ctx context.Context, dbActor *core.Record, includeFollows bool) (*core.Record, error) {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return nil, fmt.Errorf("ORIGIN environment variable not set")
    }

    private := false
    if dbActor.GetBool("is_local") {
        // Instance actor has no backing user record; skip user-specific enrichment.
        if dbActor.GetString("actor_type") == "instance" {
            if err := app.Save(dbActor); err != nil {
                return nil, err
            }
            return dbActor, nil
        }

        user, err := app.FindRecordById("users", dbActor.GetString("user"))
        // ... rest of existing logic unchanged
```

---

### CR-02: Instance actor is invisible to all federation guards — no `actor_type` filter anywhere

**File:** `db/federation/instance.go:89`, cross-referenced with `db/federation/actor.go`, `db/federation/follow.go`, `db/federation/create.go`, `db/routes/activitypub.go`

**Issue:** The `actor_type` field is set to `"instance"` on the record but is never read by any federation processing code. This means:

1. The instance actor can be matched by `GetActorByHandle` with handle `instance@<domain>` because the filter only checks `preferred_username` and `is_local`. Any remote actor that sends a Follow to `instance@<domain>` via the user-follow path will attempt the broken `assembleActor` path described in CR-01.
2. When the instance actor is eventually used as the signing actor for outbound requests (future work, but the architecture assumes it), `fetchRemoteActor` and `FetchCollection` will sign requests using whichever actor is in the context — there is no check preventing a user's private key from being substituted for the instance actor's key, or vice versa.
3. `IndexActors` / `documentFromActorRecord` in `db/util/meilisearch.go` index the instance actor into the Meilisearch `actors` index with no `actor_type` field in the document, making it searchable and returnable as a user actor to clients.

The root cause is that `actor_type` was introduced as a database-level marker only, with no runtime enforcement. Every code path that operates on `activitypub_actors` records should filter or branch on `actor_type` where the distinction matters.

**Fix:** At minimum, the three places below need `actor_type` guards before this phase ships:

```go
// db/util/meilisearch.go — documentFromActorRecord
// Skip indexing instance actors (they are not user-searchable profiles)
func documentFromActorRecord(r *core.Record) (map[string]any, error) {
    if r.GetString("actor_type") == "instance" {
        return nil, fmt.Errorf("skipping instance actor from search index")
    }
    // ... existing code
}

// db/federation/actor.go — GetActorByHandle
filter := "preferred_username={:username}&&actor_type!='instance'&&"
// ... rest of filter

// db/routes/activitypub.go — ActivitypubActor (user-profile resolution endpoint)
// Add actor_type check before returning the actor to prevent the instance
// actor from being served as a user profile
```

---

## Warnings

### WR-01: `initData` return value silently discarded

**File:** `db/main.go:157`

**Issue:** `onBeforeServeHandler` calls `initData(se.App, client)` but discards the returned `error`. `initData` has the signature `func initData(app core.App, client meilisearch.ServiceManager) error` and includes `federation.InitInstanceActor(app)` which only logs its error internally. The broader function can also return errors from `initCategories` (line 280-311) and `initPlugins`. These are all swallowed silently at the call site.

```go
// db/main.go:157
initData(se.App, client)   // error return ignored
```

**Fix:**
```go
func onBeforeServeHandler(client meilisearch.ServiceManager) func(se *core.ServeEvent) error {
    return func(se *core.ServeEvent) error {
        registerRoutes(se, client)
        registerCronJobs(se.App, client)
        if err := initData(se.App, client); err != nil {
            se.App.Logger().Warn(fmt.Sprintf("initData warning: %v", err))
            // do not return err — non-fatal startup failures should not abort serving
        }
        return se.Next()
    }
}
```

---

### WR-02: Key-generation function duplicated between packages

**File:** `db/federation/instance.go:22-28`

**Issue:** `generateInstanceKeyPair` is a line-for-line copy of `generateKeyPair` in `db/util/activitypub.go:103-111`. The comment in `instance.go` acknowledges this ("Copied from db/util/activitypub.go (unexported there; duplicated here)"). If the key size or algorithm ever changes in one copy, the other diverges silently. The two functions generate identical key material; there is no technical reason for the duplication.

**Fix:** Export `generateKeyPair` from `db/util/activitypub.go` as `GenerateKeyPair` (or move it to a shared internal package) and call it from `instance.go`:

```go
// db/util/activitypub.go
func GenerateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) { ... }

// db/federation/instance.go
priv, pub, err := util.GenerateKeyPair()
```

---

### WR-03: Migration `down` function removes the column without clearing the instance actor row first

**File:** `db/migrations/1782290000_add_actor_type_to_activitypub_actors.go:42-51`

**Issue:** The `down` migration simply removes the `actor_type` field from the collection schema. If the instance actor was created (by `InitInstanceActor`) before the down-migration runs, the instance actor row remains in the `activitypub_actors` table with no distinguishing field. Any code that previously filtered on `actor_type = 'instance'` to exclude it would stop working correctly after rollback, and the orphaned record could be treated as a user actor. Additionally, the down migration does not delete the instance actor record itself, which means a partially-rolled-back state leaves an actor with `is_local=true` and no `user` relation, which is the crash condition described in CR-01.

**Fix:** The down migration should delete the instance actor row (identified by `preferred_username = 'instance' AND is_local = true`) before removing the column:

```go
func(app core.App) error {
    // Delete the instance actor created by this phase before removing its type column.
    if _, err := app.DB().NewQuery(
        "DELETE FROM activitypub_actors WHERE preferred_username = 'instance' AND is_local = 1",
    ).Execute(); err != nil {
        return err
    }

    collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
    if err != nil {
        return err
    }
    collection.Fields.RemoveById("select_actor_type_001")
    return app.Save(collection)
},
```

---

### WR-04: Advertised `inbox` and `outbox` URLs have no route handlers

**File:** `web/src/routes/api/v1/activitypub/instance/+server.ts:43-44`

**Issue:** The actor document exposes `inbox` and `outbox` fields (populated from `actor.inbox` and `actor.outbox`, which `InitInstanceActor` sets to `<iri>/inbox` and `<iri>/outbox`). Peer servers that attempt to deliver activities to the instance actor's inbox or fetch its outbox will receive 404 responses. The ActivityPub spec requires that both endpoints be operational for a conforming actor. The SvelteKit route tree has no `instance/inbox/+server.ts` or `instance/outbox/+server.ts` (confirmed: only `+server.ts` exists under `web/src/routes/api/v1/activitypub/instance/`).

**Fix:** Either implement `inbox` and `outbox` sub-routes before advertising these URLs, or omit `inbox` and `outbox` from the actor document until the handlers exist. Omitting is the safer short-term option:

```typescript
const r = {
    "@context": [...],
    id: iri,
    type: "Application",
    preferredUsername: actor.preferred_username,
    name: actor.username,
    // inbox and outbox omitted until handlers are implemented
    manuallyApprovesFollowers: true,
    publicKey: { ... },
};
```

---

## Info

### IN-01: `iri` variable computed from `env.ORIGIN` rather than from `actor.iri`

**File:** `web/src/routes/api/v1/activitypub/instance/+server.ts:28,39`

**Issue:** The `id` field in the actor document is set to `iri` which is computed as `` `${env.ORIGIN}/api/v1/activitypub/instance` ``. The `actor` record fetched from the database also has an `iri` field (`actor.iri`) set to the same value by `InitInstanceActor`. Using the environment variable directly instead of `actor.iri` means a discrepancy between `env.ORIGIN` and the stored `actor.iri` (e.g., if `ORIGIN` changes after initial creation) would produce an actor document whose `id` does not match the record's canonical IRI. Using `actor.iri` as the source of truth is more resilient.

**Fix:**
```typescript
const iri = actor.iri;  // use the stored canonical IRI
```

---

### IN-02: Test hardcodes a numeric encryption key that is valid ASCII but not a random key

**File:** `db/federation/instance_test.go:80,121,166`

**Issue:** All three test cases use `"0123456789abcdef0123456789abcdef"` as the encryption key. This is exactly 32 bytes (correct for AES-256) and the tests pass, but using sequential numeric characters is not representative of a real key and could mask byte-ordering or encoding issues in the encrypt/decrypt path. This is a test quality concern, not a production bug.

**Fix:** Use a randomly-looking but fixed test key:
```go
t.Setenv("POCKETBASE_ENCRYPTION_KEY", "t3st-k3y-f0r-unit-t3sts-32bytes!")
```

---

_Reviewed: 2026-06-25T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
