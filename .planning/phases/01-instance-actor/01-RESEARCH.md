# Phase 1: Instance Actor - Research

**Researched:** 2026-06-24
**Domain:** PocketBase/Go federation — ActivityPub Application actor creation, schema migration, startup hook, HTTP endpoint
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** The instance actor's `name` field is set dynamically at startup from the `ORIGIN` env var, formatted as `"Wanderer at {domain}"` (e.g., `"Wanderer at trails.example.com"`). The domain is extracted from the ORIGIN URL hostname.
- **D-02:** `preferred_username` remains `"instance"` as specified in INST-01 — only `name` is dynamic.
- **D-03:** Phase 1 advertises the outbox URL (`{ORIGIN}/api/v1/activitypub/instance/outbox`) in the actor JSON but does NOT implement it. Fetching the outbox URL returns 404 until Phase 2. No stub OrderedCollection is needed.
- **D-04:** `initInstanceActor()` and the GET handler (`/api/v1/activitypub/instance`) live in a new dedicated file: `db/federation/instance.go`. This keeps instance-actor concerns separate from the general user-actor code in `db/federation/actor.go`.
- **D-05:** Route registration for the GET endpoint follows the existing pattern in `db/main.go` (calling into `db/federation/instance.go` handler), consistent with how other federation routes are wired.

### Claude's Discretion

- RSA key size: follow whatever the existing user actor generation uses (no new policy needed).
- Startup hook type: use whichever PocketBase hook runs after migrations and before serving (researcher to confirm — likely `OnServe` or equivalent).
- Migration timestamp: researcher picks the next available timestamp; follows the existing `db/migrations/` pattern.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INST-01 | Instance actor record exists in `activitypub_actors` with `actor_type = "Application"`, RSA keypair, and stable IRI at `{ORIGIN}/api/v1/activitypub/instance` | Confirmed: `activitypub_actors` collection exists at `pbc_1295301207`; need to add `actor_type` text field via migration; `generateKeyPair()` in `util/activitypub.go` is reusable |
| INST-02 | GET `{ORIGIN}/api/v1/activitypub/instance` returns valid ActivityPub JSON (id, type, inbox, outbox, publicKey, manuallyApprovesFollowers: true) | Confirmed: route pattern follows `registerRoutes()` in `db/main.go`; handler lives in `db/federation/instance.go`; `manuallyApprovesFollowers` must be added manually to JSON — not present in go-ap Actor struct |
| INST-04 | `initInstanceActor()` runs at startup, creates actor if not exists, never regenerates keypair if actor already exists | Confirmed: `OnServe` hook via `onBeforeServeHandler` in `db/main.go` is the correct startup hook; `initData()` is the established pattern to call initialization functions |
</phase_requirements>

---

## Summary

Phase 1 is a self-contained Go/PocketBase change with three parts: a schema migration to add an `actor_type` column to `activitypub_actors`, a startup function `initInstanceActor()` that creates the Application actor idempotently, and a GET endpoint at `/api/v1/activitypub/instance` that serves the actor as ActivityPub JSON.

All required patterns are already established in the codebase. The RSA keypair generation function `generateKeyPair()` exists in `db/util/activitypub.go` (2048-bit RSA). `security.Encrypt`/`security.Decrypt` from `github.com/pocketbase/pocketbase/tools/security` is used for private key storage. The `ORIGIN` env var is consumed identically across `actor.go`, `activity.go`, and the new `initInstanceActor()`. The `OnServe` hook (called via `onBeforeServeHandler`) is the correct startup point — it runs after migrations are applied and before HTTP serving begins.

One critical pitfall exists: the existing `assembleActor()` function in `federation/actor.go` assumes every `is_local` actor has a non-empty `user` field and will call `app.FindRecordById("users", "")` for the instance actor, which returns an error. The GET endpoint for the instance actor MUST NOT call `assembleActor()`. Instead, it reads the record directly from the database and serializes it to ActivityPub JSON independently.

**Primary recommendation:** Implement `db/federation/instance.go` containing both `InitInstanceActor()` and the GET handler; keep the migration minimal (one `actor_type` text field with default `"Person"`); call `InitInstanceActor()` from `initData()` in `db/main.go`; register the GET route in `registerRoutes()`.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Instance actor schema (actor_type column) | Database / Storage | — | PocketBase migration adds a column to `activitypub_actors`; no API or UI layer involved |
| Instance actor initialization (keypair, record creation) | Backend (Go) | — | `initInstanceActor()` is a startup side-effect in the Go binary; no HTTP request involved |
| GET /api/v1/activitypub/instance | API / Backend | — | Custom route registered on `se.Router`; returns ActivityPub JSON; no SvelteKit involvement |
| RSA keypair generation and storage | Backend (Go) | Database / Storage | `generateKeyPair()` + `security.Encrypt()` write into PocketBase; encryption key from env |

---

## Standard Stack

This phase installs no new packages. All dependencies already exist in `db/go.mod`.

### Core (already present in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/pocketbase/pocketbase` | v0.38.0 | App framework, DB, hooks, route registration | Existing backend runtime [VERIFIED: go.mod] |
| `github.com/go-ap/activitypub` | v0.0.0-20250905102448 | `ApplicationType`, `ApplicationNew()`, `PublicKey` struct, JSON marshaling | Existing AP library already used for user actors [VERIFIED: go.mod] |
| `github.com/pocketbase/pocketbase/tools/security` | (part of PB v0.38.0) | `security.Encrypt()` / `security.Decrypt()` for RSA private key storage | Already used in `util/activitypub.go` and `federation/actor.go` [VERIFIED: codebase grep] |
| `crypto/rsa`, `crypto/x509`, `encoding/pem` | Go stdlib | RSA keygen, DER marshaling, PEM encoding | Already used in `util/activitypub.go::generateKeyPair()` [VERIFIED: codebase grep] |
| `os` | Go stdlib | `os.Getenv("ORIGIN")`, `os.Getenv("POCKETBASE_ENCRYPTION_KEY")` | Established pattern across `actor.go`, `activity.go` [VERIFIED: codebase grep] |
| `net/url` | Go stdlib | URL parsing to extract hostname for `name` field | Already used in `util/activitypub.go::ActorFromUser()` [VERIFIED: codebase grep] |

### No New Packages
No npm, pip, or additional Go dependencies are required. [VERIFIED: full dependency scan of phase scope]

---

## Package Legitimacy Audit

> No new packages are installed in this phase. All code uses existing dependencies already present in `db/go.mod`.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
Server startup
    │
    ▼
app.OnBootstrap() ──► hooks.OnBootstrapHandler()  [sets AppName/AppURL from env]
    │
    ▼
Automigrate runs db/migrations/*.go
    │  └─► 1782289500_add_actor_type_to_activitypub_actors.go
    │           adds actor_type TEXT column DEFAULT 'Person'
    │
    ▼
app.OnServe() ──► onBeforeServeHandler()
    │  ├─► registerRoutes()
    │  │       └─► se.Router.GET("/activitypub/instance", InstanceActorGet)
    │  └─► initData()
    │           └─► federation.InitInstanceActor(app)
    │                   ├─► check IRI exists in activitypub_actors (by IRI)
    │                   ├─► [if exists] return (no keypair regen)
    │                   └─► [if not exists] create record:
    │                           actor_type="Application"
    │                           preferred_username="instance"
    │                           name="Wanderer at {domain}"
    │                           iri="{ORIGIN}/api/v1/activitypub/instance"
    │                           inbox="{ORIGIN}/api/v1/activitypub/instance/inbox"
    │                           outbox="{ORIGIN}/api/v1/activitypub/instance/outbox"
    │                           is_local=true
    │                           public_key=PEM
    │                           private_key=Encrypt(DER)
    │
    ▼
HTTP GET /activitypub/instance
    │
    ▼
InstanceActorGet(e *core.RequestEvent)
    │  ├─► find record by IRI from activitypub_actors
    │  └─► serialize as ActivityPub JSON (NOT via assembleActor)
    │           {
    │             "@context": [...],
    │             "id": "{ORIGIN}/api/v1/activitypub/instance",
    │             "type": "Application",
    │             "preferredUsername": "instance",
    │             "name": "Wanderer at {domain}",
    │             "inbox": "...",
    │             "outbox": "...",
    │             "manuallyApprovesFollowers": true,
    │             "publicKey": { "id": "...#main-key", "owner": "...", "publicKeyPem": "..." }
    │           }
    └─► e.JSON(200, actorJSON)
```

### Recommended Project Structure

New file only:

```
db/
├── federation/
│   ├── instance.go      ← NEW: InitInstanceActor() + InstanceActorGet handler
│   ├── actor.go         ← unchanged (user actors)
│   └── activity.go      ← unchanged
├── migrations/
│   └── 1782289500_add_actor_type_to_activitypub_actors.go  ← NEW
└── main.go              ← two additions: route + initData call
```

### Pattern 1: Migration — Adding a Text Column with Default

Follow exactly the field-addition pattern from `1776675228_updated_activitypub_actors.go` and the field object format from `1780734977_updated_activitypub_actors.go`.

```go
// Source: db/migrations/1776675228_updated_activitypub_actors.go (VERIFIED: codebase read)
package migrations

import (
    "github.com/pocketbase/pocketbase/core"
    m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
    m.Register(func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
        if err != nil {
            return err
        }

        if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
            "autogeneratePattern": "",
            "hidden": false,
            "id": "text_actor_type_001",
            "max": 0,
            "min": 0,
            "name": "actor_type",
            "pattern": "",
            "presentable": false,
            "primaryKey": false,
            "required": false,
            "system": false,
            "type": "text"
        }`)); err != nil {
            return err
        }

        return app.Save(collection)
    }, func(app core.App) error {
        // down: remove actor_type field
        collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
        if err != nil {
            return err
        }
        collection.Fields.RemoveById("text_actor_type_001")
        return app.Save(collection)
    })
}
```

**Default value note:** PocketBase text fields do not support `DEFAULT` SQL clauses in the field JSON descriptor. Existing rows get an empty string (`""`). The instance actor will explicitly set `actor_type = "Application"`. User actors created by `ActorFromUser()` do not set this field — they will have `""` (empty string), which is acceptable since `assembleActor()` does not need `actor_type`. If strictly needed, an `UPDATE` in the up migration can backfill `actor_type = "Person"` for existing rows, but the schema constraint (INST-04 clause about existing records not being broken) only requires the column exists and existing actors remain functional, not that they have a populated value.

[ASSUMED: PocketBase text field migration cannot specify a SQL DEFAULT — based on reading multiple migration files in this codebase; no PocketBase migration documentation was consulted for this specific claim.]

### Pattern 2: Startup Initialization — OnServe Hook

The correct hook is `OnServe()`, called via `onBeforeServeHandler()` in `db/main.go`. This runs after all migrations are applied and before the HTTP server begins accepting requests. [VERIFIED: codebase read of `db/main.go`]

```go
// Source: db/main.go (VERIFIED: codebase read)
// In initData():
func initData(app core.App, client meilisearch.ServiceManager) error {
    initCategories(app)
    initPlugins(app)
    initMeilisearchConfig(client)
    federation.InitInstanceActor(app)  // ← add this call
    go func() {
        backfillPolylines(app)
        initMeilisearchDocuments(app, client)
    }()
    return nil
}
```

`InitInstanceActor()` must run synchronously (not in a goroutine) because the actor record must exist before the HTTP server accepts any ActivityPub requests.

### Pattern 3: RSA Key Generation and Storage

Reuse exactly what `ActorFromUser()` does in `db/util/activitypub.go`. [VERIFIED: codebase read]

```go
// Source: db/util/activitypub.go::ActorFromUser() (VERIFIED: codebase read)
priv, pub, err := generateKeyPair()   // 2048-bit RSA
privBytes := x509.MarshalPKCS1PrivateKey(priv)
privEncrypted, err := security.Encrypt(privBytes, encryptionKey)
pubBytes, err := x509.MarshalPKIXPublicKey(pub)
pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

record.Set("public_key", string(pubPem))
record.Set("private_key", privEncrypted)
```

`generateKeyPair()` is unexported (lowercase). Either call it via a public wrapper or duplicate it in `instance.go`. The simplest approach: move `generateKeyPair()` from `util/activitypub.go` to an exported function, or copy it into `federation/instance.go`. [VERIFIED: `generateKeyPair` is in package `util`, lowercase = unexported]

### Pattern 4: ApplicationNew() from go-ap

The `go-ap/activitypub` library provides `pub.ApplicationNew(id)` and `pub.ApplicationType`. The library's `Actor` struct does NOT have a `ManuallyApprovesFollowers` field. [VERIFIED: grep of go-ap actor.go in mod cache]

For the GET endpoint response, `manuallyApprovesFollowers: true` must be added by constructing a custom map or struct and serializing it with `e.JSON()` rather than using the library's Actor struct directly.

```go
// Source: derived from go-ap actor.go and existing route pattern (VERIFIED: library + routes/activitypub.go)
// Approach: build a plain map for the response so manuallyApprovesFollowers is included
origin := os.Getenv("ORIGIN")
iri := origin + "/api/v1/activitypub/instance"

actorJSON := map[string]any{
    "@context": []any{
        "https://www.w3.org/ns/activitystreams",
        "https://w3id.org/security/v1",
    },
    "id":                          iri,
    "type":                        "Application",
    "preferredUsername":           record.GetString("preferred_username"),
    "name":                        record.GetString("username"),
    "inbox":                       record.GetString("inbox"),
    "outbox":                      record.GetString("outbox"),
    "manuallyApprovesFollowers":   true,
    "publicKey": map[string]any{
        "id":           iri + "#main-key",
        "owner":        iri,
        "publicKeyPem": record.GetString("public_key"),
    },
}
return e.JSON(http.StatusOK, actorJSON)
```

### Pattern 5: Domain Extraction from ORIGIN

`ActorFromUser()` already demonstrates this pattern: [VERIFIED: db/util/activitypub.go]

```go
url, err := url.Parse(origin)
if err != nil {
    return nil, err
}
domain := strings.TrimPrefix(url.Hostname(), "www.")
name := fmt.Sprintf("Wanderer at %s", domain)
```

### Anti-Patterns to Avoid

- **Calling `assembleActor()` for the instance actor:** `assembleActor()` assumes every `is_local` actor has a non-empty `user` relation field and calls `app.FindRecordById("users", "")` which will error. The instance actor has no user. The GET handler must NOT go through `assembleActor()`. [VERIFIED: federation/actor.go lines 170-174]
- **Regenerating the keypair on startup:** `InitInstanceActor()` must check for an existing record by IRI before creating. The IRI is `{ORIGIN}/api/v1/activitypub/instance` and has a unique index (`idx_rpT7QJwWTm`). If a record exists, return immediately without touching the keypair.
- **Running InitInstanceActor in a goroutine:** Unlike `backfillPolylines` and `initMeilisearchDocuments`, the instance actor init must complete synchronously so it is available immediately when the server starts accepting requests.
- **Using `GetActorByIRI` or `GetActorByHandle` to look up the instance actor in the GET handler:** These will call `assembleActor()`, triggering the user-field assumption bug.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| RSA keypair generation | Custom RSA code | `generateKeyPair()` in `db/util/activitypub.go` | Already 2048-bit RSA, matches existing user actor format |
| Private key encryption | Custom AES/GCM | `security.Encrypt()` from `github.com/pocketbase/pocketbase/tools/security` | Must match `security.Decrypt()` used in `PostActivity()` |
| HTTP signature for future outbound | Custom signer | `github.com/go-fed/httpsig` already in go.mod | Used consistently for all existing outbound AP requests |
| JSON-LD context for ActivityPub | Custom context injection | `jsonld.WithContext(jsonld.IRI(pub.ActivityBaseURI), jsonld.IRI(pub.SecurityContextURI))` from `go-ap/jsonld` | Used in `PostActivity()`; ensure `@context` array matches |

**Key insight:** The entire RSA key lifecycle (generate → encrypt → store → decrypt → use) is already implemented for user actors. The instance actor init is a parameterization of the same pipeline with `user = ""` and `actor_type = "Application"`.

---

## Common Pitfalls

### Pitfall 1: assembleActor() Breaks for Userless Local Actors

**What goes wrong:** `GetActorByHandle()` and `GetActorByIRI()` always call `assembleActor()`. For any `is_local` actor, `assembleActor()` unconditionally runs `app.FindRecordById("users", dbActor.GetString("user"))`. When `user` is empty (the instance actor), this returns `sql.ErrNoRows`, which propagates as an error.
**Why it happens:** `assembleActor()` was written when all local actors were user actors. The instance actor is the first local actor without a user.
**How to avoid:** The GET handler (`InstanceActorGet`) must look up the instance actor record directly — `app.FindFirstRecordByData("activitypub_actors", "iri", iri)` — and serialize the record manually. Do not call `federation.GetActorByIRI()` or `federation.GetActorByHandle()`.
**Warning signs:** A `FindRecordById: no rows` error in the instance actor GET handler at startup is this bug.

### Pitfall 2: Keypair Regeneration on Restart

**What goes wrong:** If `InitInstanceActor()` does not check for an existing record before creating one, each restart generates a new keypair. The existing public key is published to peers; a new key invalidates all cached references and breaks HTTP signature verification.
**Why it happens:** Forgetting to check IRI existence before `app.Save(newRecord)`.
**How to avoid:** Use `app.FindFirstRecordByData("activitypub_actors", "iri", instanceIRI)` at the top of `InitInstanceActor()`. If a record is returned without error, return immediately. Only proceed to key generation if `sql.ErrNoRows`.
**Warning signs:** The success criterion "a second server restart does not regenerate the keypair" would fail; a new public key appears in each restart's log.

### Pitfall 3: Migration Timestamp Collision

**What goes wrong:** Using a timestamp already used by an existing migration causes a PocketBase automigrate error or silent skip.
**Why it happens:** Copy-pasting an existing migration file without updating its timestamp.
**How to avoid:** The latest migration in `db/migrations/` is `1780734977_updated_activitypub_actors.go`. Choose any timestamp greater than `1780734977`. The current Unix time is approximately `1782289477`, which is safe. Use `1782289500_add_actor_type_to_activitypub_actors.go` as the filename.
**Warning signs:** PocketBase startup log shows migration error or duplicate migration warning.

### Pitfall 4: manuallyApprovesFollowers Not in go-ap Actor Struct

**What goes wrong:** Returning the `go-ap` library's `pub.Actor` or `pub.Application` struct directly (marshaled via `e.JSON()`) will not include `manuallyApprovesFollowers` in the response. INST-02 requires it.
**Why it happens:** The go-ap library follows the ActivityPub core spec; `manuallyApprovesFollowers` is from the ActivityPub context extension (used by Mastodon etc.) and is not in the struct.
**How to avoid:** Construct the response as a `map[string]any` that includes `"manuallyApprovesFollowers": true` explicitly. Do not use the library struct for the HTTP response serialization. [VERIFIED: grep of entire go-ap module — no ManuallyApproves field found]
**Warning signs:** A remote Mastodon instance shows the Follow as auto-accepted (Phase 2 symptom); INST-02 acceptance test fails to find `manuallyApprovesFollowers` in the response.

### Pitfall 5: PocketBase Access Rules Block the GET Endpoint

**What goes wrong:** The `activitypub_actors` collection `viewRule` is `"user.settings_via_user.privacy.account != 'private' || user = @request.auth.id"`. A lookup of the instance actor (which has no `user`) may be blocked by this rule when accessed through PocketBase's collection API. However, the GET handler accesses the database directly via `app.FindFirstRecordByData()` and bypasses collection rules — this is safe.
**Why it happens:** Confusing PocketBase's collection API (enforces rules) with direct Go app DB access (bypasses rules).
**How to avoid:** Always use `app.FindFirstRecordByData()` / `app.FindRecordById()` in server-side handler code, never `e.App.Collection(...).List()` with auth enforcement. All existing federation routes follow this pattern. [VERIFIED: db/routes/activitypub.go]
**Warning signs:** 403 Forbidden when fetching the instance actor; only happens if handler accidentally routes through the collection rules layer.

### Pitfall 6: actor_type Column Default Value

**What goes wrong:** PocketBase migration field descriptors for text fields do not support a `"default"` property that translates to a SQL `DEFAULT` clause. Existing rows will have `actor_type = ""` (empty string) after the migration runs.
**Why it happens:** PocketBase schema migrations work at the ORM level, not raw SQL level.
**How to avoid:** Either (a) accept empty string for existing user actors (safe — nothing reads `actor_type` today), or (b) add an `UPDATE activitypub_actors SET actor_type = 'Person' WHERE actor_type = '' OR actor_type IS NULL` statement in the migration up function using `app.DB().NewQuery(...)`. The CONTEXT.md requirement is "default `'Person'`, no existing records broken" — option (b) satisfies the letter; option (a) satisfies the spirit if no code reads `actor_type` in this phase.
**Warning signs:** Future code querying `actor_type = 'Person'` misses existing user actors.

---

## Code Examples

Verified patterns from authoritative codebase sources:

### `generateKeyPair()` — Reuse or Re-export

The function in `db/util/activitypub.go` is unexported. Two options:

**Option A:** Export it (rename to `GenerateKeyPair()` and update the call site in `ActorFromUser()`).

**Option B:** Duplicate it in `db/federation/instance.go` as a private helper.

Option A is cleaner. Option B avoids touching `util/activitypub.go`.

```go
// Source: db/util/activitypub.go (VERIFIED: codebase read)
func generateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
    priv, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return nil, nil, err
    }
    pub := &priv.PublicKey
    return priv, pub, nil
}
```

### Complete `InitInstanceActor()` Function Shape

```go
// Source: pattern derived from ActorFromUser() in db/util/activitypub.go (VERIFIED: codebase read)
func InitInstanceActor(app core.App) error {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return fmt.Errorf("ORIGIN not set")
    }
    encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
    if len(encryptionKey) == 0 {
        return fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
    }

    iri := origin + "/api/v1/activitypub/instance"

    // Idempotency check: if actor already exists, do nothing
    existing, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
    if err == nil && existing != nil {
        return nil  // actor exists; never regenerate keypair
    }
    // Only proceed for sql.ErrNoRows; surface other errors
    if err != nil && err != sql.ErrNoRows {
        return fmt.Errorf("checking instance actor: %w", err)
    }

    parsedOrigin, err := url.Parse(origin)
    if err != nil {
        return fmt.Errorf("parsing ORIGIN: %w", err)
    }
    domain := strings.TrimPrefix(parsedOrigin.Hostname(), "www.")

    // Generate keypair
    priv, pub, err := generateKeyPair()
    // ... (key encoding, encryption — same as ActorFromUser())

    collection, err := app.FindCollectionByNameOrId("activitypub_actors")
    // ...

    record := core.NewRecord(collection)
    record.Set("actor_type", "Application")
    record.Set("preferred_username", "instance")
    record.Set("username", fmt.Sprintf("Wanderer at %s", domain))
    record.Set("iri", iri)
    record.Set("inbox", iri+"/inbox")
    record.Set("outbox", iri+"/outbox")
    record.Set("is_local", true)
    record.Set("domain", domain)
    record.Set("public_key", string(pubPem))
    record.Set("private_key", privEncrypted)
    record.Set("last_fetched", time.Now())

    return app.Save(record)
}
```

### GET Endpoint Handler Shape

```go
// Source: pattern derived from ActivitypubActor() in db/routes/activitypub.go (VERIFIED: codebase read)
// Note: Does NOT call assembleActor() — instance actor has no user field
func InstanceActorGet(e *core.RequestEvent) error {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return fmt.Errorf("ORIGIN not set")
    }
    iri := origin + "/api/v1/activitypub/instance"

    record, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", iri)
    if err != nil {
        return e.NotFoundError("Instance actor not found", err)
    }

    actorJSON := map[string]any{
        "@context": []any{
            "https://www.w3.org/ns/activitystreams",
            "https://w3id.org/security/v1",
        },
        "id":                        iri,
        "type":                      "Application",
        "preferredUsername":         record.GetString("preferred_username"),
        "name":                      record.GetString("username"),
        "inbox":                     record.GetString("inbox"),
        "outbox":                    record.GetString("outbox"),
        "manuallyApprovesFollowers": true,
        "publicKey": map[string]any{
            "id":           iri + "#main-key",
            "owner":        iri,
            "publicKeyPem": record.GetString("public_key"),
        },
    }
    return e.JSON(http.StatusOK, actorJSON)
}
```

### Route Registration in main.go

```go
// Source: db/main.go registerRoutes() (VERIFIED: codebase read)
// Add to registerRoutes():
se.Router.GET("/activitypub/instance", federation.InstanceActorGet)
```

The URL prefix `/api/v1` is added by the SvelteKit proxy layer, not in the Go route registration. All existing federation routes use bare paths like `/activitypub/actor`, `/activitypub/trail/{id}`. [VERIFIED: db/main.go registerRoutes()]

### Call Site in initData()

```go
// Source: db/main.go initData() (VERIFIED: codebase read)
func initData(app core.App, client meilisearch.ServiceManager) error {
    initCategories(app)
    initPlugins(app)
    initMeilisearchConfig(client)
    if err := federation.InitInstanceActor(app); err != nil {
        app.Logger().Error(fmt.Sprintf("Failed to initialize instance actor: %v", err))
    }
    go func() {
        backfillPolylines(app)
        initMeilisearchDocuments(app, client)
    }()
    return nil
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `isLocal` field name | `is_local` field name | Migration `1780734977_updated_activitypub_actors.go` | All Go code uses `is_local`; the migration renamed it [VERIFIED: codebase] |
| `followerCount` / `followingCount` | `follower_count` / `following_count` | Migration `1776675228_updated_activitypub_actors.go` | Field names changed; Go code references snake_case names [VERIFIED: codebase] |
| User actor `user` field required | Instance actor has no `user` field | Phase 1 (this phase) | `assembleActor()` must NOT be used for the instance actor |

**Note:** The URL prefix the Go router uses is **not** `/api/v1/` — that prefix is handled upstream (SvelteKit proxy). The Go binary registers routes without this prefix (e.g., `/activitypub/actor`, not `/api/v1/activitypub/actor`). The success criteria reference `{ORIGIN}/api/v1/activitypub/instance`, which is the public-facing URL; internally the Go route is `/activitypub/instance`. [VERIFIED: db/main.go registerRoutes()]

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | PocketBase text field migration cannot specify a SQL DEFAULT via the field JSON descriptor — existing rows get empty string `""` | Common Pitfalls §6, Code Examples §Migration | If PocketBase does support defaults, the backfill UPDATE in the migration is unnecessary but harmless |
| A2 | The SvelteKit proxy adds `/api/v1/` prefix to Go routes, so the Go router registers `/activitypub/instance` not `/api/v1/activitypub/instance` | Code Examples §Route Registration | If the prefix is in the Go router, the route would be registered at the wrong path and all AP endpoints would be broken (easily caught by existing tests) |

**A2 justification:** Every existing Go route in `registerRoutes()` uses `/activitypub/actor`, `/activitypub/trail/{id}` without the `/api/v1/` prefix, and the REQUIREMENTS.md success criteria say the public URL is `{ORIGIN}/api/v1/activitypub/instance`. The proxy pattern is consistent across all observed routes. [VERIFIED: db/main.go]

---

## Open Questions

1. **`generateKeyPair()` accessibility**
   - What we know: the function is unexported (`generateKeyPair`, lowercase) in `db/util/activitypub.go`.
   - What's unclear: whether the planner prefers exporting it (touching `util/activitypub.go`) or duplicating it in `federation/instance.go` (no existing file touched beyond `main.go`).
   - Recommendation: Export as `util.GenerateKeyPair()` and update the single call site in `ActorFromUser()`. This is 2 lines changed in an existing file and avoids duplicated code. Alternative: inline in `instance.go`.

2. **actor_type backfill for existing user actors**
   - What we know: after migration, existing rows have `actor_type = ""` (empty string).
   - What's unclear: whether Phase 2 or later phases will query `actor_type = 'Person'` to distinguish user actors from the instance actor.
   - Recommendation: Include a backfill `UPDATE activitypub_actors SET actor_type = 'Person' WHERE actor_type = ''` in the migration's up function. Cost is negligible; prevents a future bug.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|---------|
| Go 1.25.0 | Compilation | ✓ (inferred from go.mod) | 1.25.0 per go.mod directive | — |
| PocketBase v0.38.0 | Runtime | ✓ (already in go.mod) | v0.38.0 | — |
| `ORIGIN` env var | InitInstanceActor, GET handler | Must be set at runtime | — | Error logged; actor not created |
| `POCKETBASE_ENCRYPTION_KEY` | RSA private key encryption | Must be set at runtime | — | Checked by `verifySettings()` which fatals if absent |

**Missing dependencies with no fallback:** None for this phase — all required dependencies are present.

---

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1` per `.planning/config.json`.

### Applicable ASVS Categories (Level 1)

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | GET endpoint is public (no auth required for AP actor discovery) |
| V3 Session Management | No | Stateless GET endpoint |
| V4 Access Control | Low | Instance actor record read via `app.FindFirstRecordByData()` bypasses collection rules intentionally; no sensitive data exposed (public key is public by design) |
| V5 Input Validation | No | No user input accepted in the GET handler |
| V6 Cryptography | Yes | RSA-2048 via Go stdlib `crypto/rsa` — standard; private key encrypted with AES-GCM via `security.Encrypt()` — existing proven pattern |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Keypair regeneration on restart | Tampering | Idempotency check in `InitInstanceActor()` — if IRI exists, return without touching keypair |
| Private key exposure in API response | Information Disclosure | `private_key` field is `hidden: true` in PocketBase schema; GET handler builds response from `map[string]any` — only `public_key` is included explicitly |
| Instance actor hijacking via IRI collision | Spoofing | Unique index `idx_rpT7QJwWTm` on `iri` column enforces one-record-per-IRI; migration does not change this index |

---

## Sources

### Primary (HIGH confidence)
- `db/federation/actor.go` — existing actor patterns, `assembleActor()` user-field assumption [VERIFIED: codebase read]
- `db/util/activitypub.go` — `ActorFromUser()`, `generateKeyPair()`, key encoding/encryption pattern [VERIFIED: codebase read]
- `db/main.go` — `onBeforeServeHandler()`, `initData()`, `registerRoutes()` — startup hook and route registration pattern [VERIFIED: codebase read]
- `db/migrations/1780734977_updated_activitypub_actors.go` — latest activitypub_actors migration, field JSON format [VERIFIED: codebase read]
- `github.com/go-ap/activitypub@v0.0.0-20250905102448-e9df599e4528/actor.go` — `ApplicationType`, `ApplicationNew()`, `PublicKey` struct, confirmed `ManuallyApproves` field absent [VERIFIED: mod cache grep]
- `db/migrations/` directory — timestamps from `1747061257` to `1780734977` — current migration set [VERIFIED: codebase ls]

### Secondary (MEDIUM confidence)
- REQUIREMENTS.md — INST-01, INST-02, INST-04 acceptance criteria [VERIFIED: codebase read]
- CONTEXT.md — D-01 through D-05 locked decisions [VERIFIED: codebase read]

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; all libraries verified in go.mod and mod cache
- Architecture: HIGH — startup hook, route pattern, and DB access verified directly in source
- Pitfalls: HIGH — `assembleActor()` user-field assumption verified by direct code inspection; keypair idempotency pattern derived from existing `initCategories()` pattern
- Migration pattern: HIGH — multiple migration examples read directly

**Research date:** 2026-06-24
**Valid until:** 2026-09-24 (stable Go/PocketBase stack; 90-day window appropriate)
