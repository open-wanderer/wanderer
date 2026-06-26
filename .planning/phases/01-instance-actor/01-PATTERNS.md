# Phase 1: Instance Actor - Pattern Map

**Mapped:** 2026-06-24
**Files analyzed:** 3 new/modified files
**Analogs found:** 3 / 3

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `db/federation/instance.go` | service + route handler | request-response | `db/util/activitypub.go` (init) + `db/federation/actor.go` (handler shape) | role-match |
| `db/migrations/1782289500_add_actor_type_to_activitypub_actors.go` | migration | batch (schema DDL) | `db/migrations/1780734977_updated_activitypub_actors.go` | exact |
| `db/main.go` (two additions) | config / entry point | — | `db/main.go` itself (existing `initData` + `registerRoutes`) | exact |

---

## Pattern Assignments

### `db/federation/instance.go` (service + route handler, request-response)

**Analogs:** `db/util/activitypub.go` (keypair + record creation) and `db/federation/actor.go` (package/import conventions)

#### Imports pattern (`db/federation/actor.go` lines 1–24)

```go
package federation

import (
    "context"
    "crypto/x509"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "pocketbase/util"
    "strings"
    "time"

    pub "github.com/go-ap/activitypub"
    "github.com/go-fed/httpsig"

    "github.com/pocketbase/dbx"
    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/tools/security"
)
```

For `instance.go`, the subset needed is:
```go
package federation

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "database/sql"
    "encoding/pem"
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "os"
    "strings"
    "time"

    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/tools/security"
)
```

#### Keypair generation + encryption pattern (`db/util/activitypub.go` lines 27–101)

This is the canonical pattern for creating a local actor record with RSA keypair. Copy this entire flow for `InitInstanceActor()`:

```go
// db/util/activitypub.go lines 27-101 (ActorFromUser)
func ActorFromUser(app core.App, u *core.Record) (*core.Record, error) {
    encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
    if len(encryptionKey) == 0 {
        return nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
    }

    collection, err := app.FindCollectionByNameOrId("activitypub_actors")
    if err != nil {
        return nil, err
    }
    priv, pub, err := generateKeyPair()
    if err != nil {
        return nil, err
    }
    privBytes := x509.MarshalPKCS1PrivateKey(priv)

    privEncrypted, err := security.Encrypt(privBytes, encryptionKey)
    if err != nil {
        return nil, err
    }

    pubBytes, err := x509.MarshalPKIXPublicKey(pub)
    if err != nil {
        return nil, err
    }
    pubPem := pem.EncodeToMemory(&pem.Block{
        Type:  "PUBLIC KEY",
        Bytes: pubBytes,
    })

    record := core.NewRecord(collection)

    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return nil, fmt.Errorf("ORIGIN environment variable not set")
    }
    // ...
    url, err := url.Parse(origin)
    if err != nil {
        return nil, err
    }
    domain := strings.TrimPrefix(url.Hostname(), "www.")

    record.Set("is_local", true)
    record.Set("public_key", string(pubPem))
    record.Set("private_key", privEncrypted)
    record.Set("last_fetched", time.Now())

    err = app.Save(record)
    if err != nil {
        return nil, err
    }
    return record, nil
}
```

**Key difference for `InitInstanceActor()`:** Before the keypair block, add an idempotency check:
```go
iri := origin + "/api/v1/activitypub/instance"
existing, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
if err == nil && existing != nil {
    return nil  // actor exists; never regenerate keypair
}
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    return fmt.Errorf("checking instance actor: %w", err)
}
```

#### Private generateKeyPair helper (`db/util/activitypub.go` lines 103–111)

`generateKeyPair` is unexported in `util` package. Copy it verbatim as a private helper in `instance.go`:

```go
func generateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
    priv, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return nil, nil, err
    }
    pub := &priv.PublicKey
    return priv, pub, nil
}
```

#### Record field-setting pattern (`db/util/activitypub.go` lines 76–93)

```go
record.Set("username", ...)
record.Set("preferred_username", ...)
record.Set("domain", domain)
record.Set("iri", id)
record.Set("inbox", id+"/inbox")
record.Set("outbox", id+"/outbox")
record.Set("is_local", true)
record.Set("public_key", string(pubPem))
record.Set("private_key", privEncrypted)
record.Set("last_fetched", time.Now())
```

For instance actor, add `actor_type` and omit `user`, `followers`, `following`, `icon`, `summary`:
```go
record.Set("actor_type", "Application")
record.Set("preferred_username", "instance")
record.Set("username", fmt.Sprintf("Wanderer at %s", domain))
record.Set("domain", domain)
record.Set("iri", iri)
record.Set("inbox", iri+"/inbox")
record.Set("outbox", iri+"/outbox")
record.Set("is_local", true)
record.Set("public_key", string(pubPem))
record.Set("private_key", privEncrypted)
record.Set("last_fetched", time.Now())
```

#### GET handler pattern (`db/federation/actor.go` lines 98–116 shape, but WITHOUT assembleActor)

**CRITICAL:** Do NOT follow `GetActorByIRI` or `GetActorByHandle` — both call `assembleActor()` which calls `app.FindRecordById("users", "")` and fails for userless actors. Instead, look up by IRI directly and serialize manually:

```go
// InstanceActorGet handler shape — derived from routes/activitypub.go GET pattern
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

**Why `map[string]any` instead of `pub.Actor`:** The `go-ap` library's `Actor` struct has no `ManuallyApprovesFollowers` field (verified by grep of mod cache). The `map[string]any` approach is the only way to include it.

---

### `db/migrations/1782289500_add_actor_type_to_activitypub_actors.go` (migration, batch/DDL)

**Analog:** `db/migrations/1780734977_updated_activitypub_actors.go` — exact same pattern.

#### Full migration pattern (lines 1–52)

```go
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

        // Backfill existing user actors so future queries on actor_type='Person' work
        if _, err := app.DB().NewQuery(
            "UPDATE activitypub_actors SET actor_type = 'Person' WHERE actor_type = '' OR actor_type IS NULL",
        ).Execute(); err != nil {
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

**Collection ID `pbc_1295301207`** is the stable identifier for `activitypub_actors` used in every existing migration for this table. The field ID `text_actor_type_001` is a new unique ID (not present in any existing migration).

---

### `db/main.go` (two additions to existing file)

**Pattern source:** `db/main.go` lines 163–197 (`registerRoutes`) and lines 214–223 (`initData`).

#### Route registration addition (after line 188, within `registerRoutes`)

```go
// db/main.go lines 184-188 — existing AP routes for context:
se.Router.POST("/activitypub/activity/process", routes.ActivitypubActivityProcess)
se.Router.GET("/activitypub/actor", routes.ActivitypubActor)
se.Router.GET("/activitypub/actor/{id}/{follow}", routes.ActivitypubActorFollow)
se.Router.GET("/activitypub/trail/{id}", routes.ActivitypubTrail)
se.Router.GET("/activitypub/comment/{id}", routes.ActivitypubComment)

// ADD after existing AP routes:
se.Router.GET("/activitypub/instance", federation.InstanceActorGet)
```

**Note:** The route is `/activitypub/instance`, not `/api/v1/activitypub/instance`. The `/api/v1` prefix is added by the SvelteKit proxy layer upstream (verified: all existing AP routes follow this bare-path pattern).

**Required import addition:** `"pocketbase/federation"` must be added to `db/main.go` imports (or inline the call via a route handler in `routes/` — but D-05 says to follow `registerRoutes` convention, which calls package-level handler funcs directly).

#### initData addition (lines 214–223)

```go
// Current initData (db/main.go lines 214-223):
func initData(app core.App, client meilisearch.ServiceManager) error {
    initCategories(app)
    initPlugins(app)
    initMeilisearchConfig(client)
    go func() {
        backfillPolylines(app)
        initMeilisearchDocuments(app, client)
    }()
    return nil
}

// MODIFIED — add federation.InitInstanceActor call BEFORE the goroutine:
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

**Why synchronous (not goroutine):** Unlike `backfillPolylines`/`initMeilisearchDocuments`, the instance actor must exist before the HTTP server accepts any AP requests. `initCategories` and `initPlugins` use the same synchronous pattern for the same reason.

---

## Shared Patterns

### env var access
**Source:** `db/util/activitypub.go` lines 28–30, 64–66; `db/federation/actor.go` lines 164–167
**Apply to:** `db/federation/instance.go` — both `InitInstanceActor` and `InstanceActorGet`
```go
origin := os.Getenv("ORIGIN")
if origin == "" {
    return nil, fmt.Errorf("ORIGIN environment variable not set")
}
encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
if len(encryptionKey) == 0 {
    return nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
}
```

### Direct DB record lookup (bypasses collection rules)
**Source:** `db/federation/actor.go` lines 99–100; `db/util/activitypub.go` line 57
**Apply to:** `InstanceActorGet` and the idempotency check in `InitInstanceActor`
```go
record, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
```
Never use `e.App.Collection("activitypub_actors").List(...)` — that enforces collection rules and blocks userless actors.

### Error handling for missing records
**Source:** `db/federation/actor.go` lines 77–78
```go
if err != nil && err == sql.ErrNoRows {
    // not found — create new
} else if err != nil {
    return nil, err
}
```
Replicate this pattern in `InitInstanceActor` for the idempotency check (use `errors.Is(err, sql.ErrNoRows)`).

### `app.Save(record)` + error propagation
**Source:** `db/util/activitypub.go` lines 95–98
```go
err = app.Save(record)
if err != nil {
    return nil, err
}
```

---

## No Analog Found

None. All three files/changes have strong analogs in the codebase.

---

## Metadata

**Analog search scope:** `db/federation/`, `db/util/`, `db/migrations/`, `db/main.go`
**Files scanned:** 5 source files read in full
**Pattern extraction date:** 2026-06-24
