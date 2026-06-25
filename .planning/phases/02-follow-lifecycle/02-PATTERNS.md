# Phase 2: Follow Lifecycle - Pattern Map

**Mapped:** 2026-06-25
**Files analyzed:** 4 (3 modified, 1 new)
**Analogs found:** 4 / 4

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `db/federation/instance.go` (add `InstanceInboxHandler`) | handler | request-response | `db/routes/activitypub.go` `ActivitypubActivityProcess` | exact |
| `db/federation/follow.go` (extend `ProcessFollowActivity`) | service | event-driven | `db/federation/follow.go` existing body | self-analog (modification) |
| `db/hooks/follow.go` (add 3 instance lifecycle handlers) | hook | event-driven | `db/hooks/follow.go` existing `CreateFollowHandler`/`DeleteFollowHandler` | exact |
| `db/main.go` (register new route + 3 hooks) | config/wiring | — | `db/main.go` existing `setupEventHandlers` + `registerRoutes` | exact |
| `db/migrations/XXXXXXXXXX_add_rejected_to_follows_status.go` (new) | migration | — | `db/migrations/1782290000_add_actor_type_to_activitypub_actors.go` | exact |

---

## Pattern Assignments

### `db/federation/instance.go` — add `InstanceInboxHandler`

**Analog:** `db/routes/activitypub.go` lines 66–131 (`ActivitypubActivityProcess`)

**Imports pattern** (instance.go lines 1–18, activitypub.go lines 1–17):
```go
// instance.go already imports these; add the ones below that are missing:
import (
    "fmt"
    "io"
    "net/http"
    "os"

    pub "github.com/go-ap/activitypub"
    "github.com/pocketbase/pocketbase/core"
    "pocketbase/util"
    // federation package is the same package; call ProcessFollowActivity etc. directly
)
```

**Core inbox handler pattern** (activitypub.go lines 66–131):
```go
func ActivitypubActivityProcess(e *core.RequestEvent) error {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return fmt.Errorf("ORIGIN not set")
    }

    body, err := io.ReadAll(e.Request.Body)
    if err != nil {
        return err
    }
    var activity pub.Activity
    err = activity.UnmarshalJSON(body)
    if err != nil {
        return err
    }

    // Identify recipient by matching inbox IRI from X-Forwarded-Path header
    inbox := fmt.Sprintf("%s%s", origin, e.Request.Header.Get("X-Forwarded-Path"))
    recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "inbox", inbox)
    if err != nil {
        return err
    }

    // Find or fetch the sending actor
    actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", activity.Actor.GetID().String())
    if err != nil {
        if err == sql.ErrNoRows {
            ctx, err := util.GetSafeActorContext(e.Request, recipient)
            if err != nil {
                return err
            }
            actor, err = federation.GetActorByIRI(e.App, ctx, activity.Actor.GetID().String(), false)
            if err != nil {
                return err
            }
        } else {
            return err
        }
    }

    // Verify HTTP signature
    verified, err := util.VerifySignature(e.App, e.Request, actor.GetString("public_key"))
    if err != nil || !verified {
        e.App.Logger().Error(err.Error())
        return e.UnauthorizedError("Invalid http signature", err)
    }

    // Dispatch by activity type
    switch activity.Type {
    case pub.FollowType:
        err = federation.ProcessFollowActivity(e.App, actor, activity)
    case pub.AcceptType:
        err = federation.ProcessAcceptActivity(e.App, actor, activity)
    case pub.UndoType:
        err = federation.ProcessUndoActivity(e.App, actor, activity)
    }
    return e.JSON(http.StatusOK, err)
}
```

**Adaptation for `InstanceInboxHandler`:**
- The `recipient` lookup uses `inbox` derived from `X-Forwarded-Path`. For the instance inbox this will resolve to the instance actor record (its `inbox` field is `{ORIGIN}/api/v1/activitypub/instance/inbox`). Use this as `recipient` for `GetSafeActorContext`.
- The `switch` only needs `FollowType`, `AcceptType`, and `UndoType` — no Create/Update/Delete/Announce/Like.
- Since `InstanceInboxHandler` lives in the `federation` package, call `ProcessFollowActivity`, `ProcessAcceptActivity`, `ProcessUndoActivity` without the package qualifier.

---

### `db/federation/follow.go` — extend `ProcessFollowActivity`

**Analog:** `db/federation/follow.go` lines 63–133 (self-analog; existing function body)

**Current code to branch** (lines 79–92):
```go
// a remote actor has requested the follow — accept immediately
if !actor.GetBool("is_local") {
    followCollection, err := app.FindCollectionByNameOrId("follows")
    if err != nil {
        return err
    }
    followRecord := core.NewRecord(followCollection)
    followRecord.Set("follower", actor.Id)
    followRecord.Set("followee", object.Id)
    followRecord.Set("status", "accepted")   // <-- this line changes
    err = app.Save(followRecord)
    if err != nil {
        return err
    }
}
// ... Accept delivery continues below ...
```

**Target pattern after change (actor-type branch):**
```go
if !actor.GetBool("is_local") {
    followCollection, err := app.FindCollectionByNameOrId("follows")
    if err != nil {
        return err
    }
    followRecord := core.NewRecord(followCollection)
    followRecord.Set("follower", actor.Id)
    followRecord.Set("followee", object.Id)

    // NEW: check if the Follow targets the local instance actor (object, not actor)
    // object.actor_type == "instance" && object.is_local == true means this Follow
    // is directed at our Application actor; require admin approval.
    if object.GetString("actor_type") == "instance" && object.GetBool("is_local") {
        followRecord.Set("status", "pending")
        // Store the incoming Follow in activitypub_activities so the update hook
        // can look it up when constructing Accept/Reject.
        // (Store activity record here — see RESEARCH.md open question 3)
        return app.Save(followRecord)   // Early return: no Accept sent
    }

    followRecord.Set("status", "accepted")
    err = app.Save(followRecord)
    if err != nil {
        return err
    }
}
// existing Accept + notification logic (person path) runs unchanged below
```

**Key detail:** Filter on the *followee* (`object`) actor_type, not the sender (`actor`) actor_type. Remote actors fetched by `GetActorByIRI` do not have `actor_type` populated locally; the local instance actor always has `actor_type = "instance"` and `is_local = true` set by `InitInstanceActor` (instance.go lines 88–89).

**Accept activity construction pattern** (follow.go lines 94–103) — reused for Accept delivery from hook:
```go
recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)
id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

acceptActivity := pub.AcceptNew(pub.IRI(id), originalFollowActivity)
acceptActivity.Actor = activity.Object   // the local actor's IRI
PostActivity(app, object, acceptActivity, []string{actor.GetString("inbox")})
```

---

### `db/hooks/follow.go` — add 3 instance lifecycle handlers

**Analog:** `db/hooks/follow.go` lines 1–24 (existing `CreateFollowHandler`/`DeleteFollowHandler`)

**Existing handler shape to copy:**
```go
package hooks

import (
    "pocketbase/federation"
    "github.com/pocketbase/pocketbase/core"
)

func CreateFollowHandler() func(e *core.RecordRequestEvent) error {
    return func(e *core.RecordRequestEvent) error {
        e.Next()
        federation.CreateFollowActivity(e.App, e.Record)
        return nil
    }
}

func DeleteFollowHandler() func(e *core.RecordRequestEvent) error {
    return func(e *core.RecordRequestEvent) error {
        federation.CreateUnfollowActivity(e.App, e.Record)
        return e.Next()
    }
}
```

**New handlers use `*core.RecordEvent` (not `*core.RecordRequestEvent`)** because they bind to `AfterSuccess` hooks (model lifecycle, not HTTP request lifecycle):

```go
// InstanceFollowCreateHandler — fires after admin creates a follows record in PocketBase admin.
// Sends outgoing Follow activity to the remote instance.
func InstanceFollowCreateHandler() func(e *core.RecordEvent) error {
    return func(e *core.RecordEvent) error {
        if !isInstanceFollow(e.App, e.Record) {
            return e.Next()
        }
        if err := federation.CreateFollowActivity(e.App, e.Record); err != nil {
            e.App.Logger().Error(fmt.Sprintf("instance follow activity failed: %v", err))
        }
        return e.Next()
    }
}

// InstanceFollowUpdateHandler — fires after admin changes status on a follows record.
// Delivers Accept{Follow} or Reject{Follow} to the remote instance.
func InstanceFollowUpdateHandler() func(e *core.RecordEvent) error {
    return func(e *core.RecordEvent) error {
        if !isInstanceFollow(e.App, e.Record) {
            return e.Next()
        }
        newStatus := e.Record.GetString("status")
        oldStatus := e.Record.Original().GetString("status")
        if oldStatus == newStatus {
            return e.Next()
        }
        // Deliver Accept or Reject based on new status
        // (call federation.CreateAcceptFollowActivity / federation.CreateRejectFollowActivity)
        return e.Next()
    }
}

// InstanceFollowDeleteHandler — fires after admin deletes a follows record.
// Sends Undo{Follow} to the remote instance.
func InstanceFollowDeleteHandler() func(e *core.RecordEvent) error {
    return func(e *core.RecordEvent) error {
        if !isInstanceFollow(e.App, e.Record) {
            return e.Next()
        }
        if err := federation.CreateUnfollowActivity(e.App, e.Record); err != nil {
            e.App.Logger().Error(fmt.Sprintf("instance unfollow activity failed: %v", err))
        }
        return e.Next()
    }
}
```

**`isInstanceFollow` helper — add to same file or a shared util:**
```go
// isInstanceFollow returns true if the follows record involves the local instance actor
// (as either follower or followee). Uses IRI comparison for reliability.
func isInstanceFollow(app core.App, follow *core.Record) bool {
    origin := os.Getenv("ORIGIN")
    instanceIRI := origin + "/api/v1/activitypub/instance"
    for _, field := range []string{"follower", "followee"} {
        actor, err := app.FindRecordById("activitypub_actors", follow.GetString(field))
        if err == nil && actor.GetString("iri") == instanceIRI {
            return true
        }
    }
    return false
}
```

**Unfollow activity pattern** (undo.go lines 15–68, reused directly via `CreateUnfollowActivity`):
```go
// CreateUnfollowActivity (undo.go:15) already handles:
// 1. Fetch follower+followee actors by record ID
// 2. Find original Follow activity in activitypub_activities by actor+object+type
// 3. Build pub.UndoNew wrapping the Follow
// 4. Call PostActivity to deliver
// Call federation.CreateUnfollowActivity(e.App, e.Record) directly — no new implementation needed.
```

**Imports to add to hooks/follow.go:**
```go
import (
    "fmt"
    "os"
    "pocketbase/federation"
    "github.com/pocketbase/pocketbase/core"
)
```

---

### `db/main.go` — register new route and 3 hooks

**Analog:** `db/main.go` lines 92–147 (`setupEventHandlers`) and lines 164–198 (`registerRoutes`)

**Route registration pattern** (main.go line 185):
```go
// Existing pattern to copy:
se.Router.POST("/activitypub/activity/process", routes.ActivitypubActivityProcess)

// New line to add at line 186 (or directly after existing AP route):
se.Router.POST("/activitypub/instance/inbox", federation.InstanceInboxHandler)
```

**Hook registration pattern** (main.go lines 127–128):
```go
// Existing follows hooks (DO NOT REMOVE):
app.OnRecordCreateRequest("follows").BindFunc(hooks.CreateFollowHandler())
app.OnRecordDeleteRequest("follows").BindFunc(hooks.DeleteFollowHandler())

// New lines to add immediately after (lines 129–131):
app.OnRecordAfterCreateSuccess("follows").BindFunc(hooks.InstanceFollowCreateHandler())
app.OnRecordAfterUpdateSuccess("follows").BindFunc(hooks.InstanceFollowUpdateHandler())
app.OnRecordAfterDeleteSuccess("follows").BindFunc(hooks.InstanceFollowDeleteHandler())
```

**Critical distinction:** Existing `CreateFollowHandler`/`DeleteFollowHandler` use `OnRecordCreateRequest`/`OnRecordDeleteRequest` — these fire only on HTTP API calls (user-follow path). The new `InstanceFollow*` handlers use `OnRecordAfterCreateSuccess`/`OnRecordAfterUpdateSuccess`/`OnRecordAfterDeleteSuccess` — these fire on all saves including admin panel operations. Both sets must coexist.

---

### `db/migrations/XXXXXXXXXX_add_rejected_to_follows_status.go` (new file)

**Analog:** `db/migrations/1782290000_add_actor_type_to_activitypub_actors.go` lines 1–52

**Full migration pattern** (analog lines 1–52):
```go
package migrations

import (
    "github.com/pocketbase/pocketbase/core"
    m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
    m.Register(func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("pbc_1295301207")  // activitypub_actors
        if err != nil {
            return err
        }

        if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
            "hidden": false,
            "id": "select_actor_type_001",
            "maxSelect": 1,
            "name": "actor_type",
            ...
        }`)); err != nil {
            return err
        }

        return app.Save(collection)
    }, func(app core.App) error {
        // rollback: remove the field
        collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
        if err != nil {
            return err
        }
        collection.Fields.RemoveById("select_actor_type_001")
        return app.Save(collection)
    })
}
```

**Adaptation for `follows.status` — add `"rejected"` value:**
```go
// The follows collection has a select field "status" with values ["pending", "accepted"].
// This migration updates that field to ["pending", "accepted", "rejected"].
// Collection ID: "8obn1ukumze565i" (verified in 1747064968_collections_snapshot.go)
// Status field ID: must be read from the migration snapshot (find by name "status" in follows).

func init() {
    m.Register(func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("8obn1ukumze565i")
        if err != nil {
            return err
        }

        statusField := collection.Fields.GetByName("status")
        if statusField == nil {
            return fmt.Errorf("status field not found in follows collection")
        }

        // Cast to SelectField and append the new value
        selectField, ok := statusField.(*core.SelectField)
        if !ok {
            return fmt.Errorf("status is not a SelectField")
        }
        selectField.Values = append(selectField.Values, "rejected")

        return app.Save(collection)
    }, func(app core.App) error {
        // rollback: remove "rejected" from values
        collection, err := app.FindCollectionByNameOrId("8obn1ukumze565i")
        if err != nil {
            return err
        }
        statusField := collection.Fields.GetByName("status")
        if statusField == nil {
            return nil
        }
        selectField, ok := statusField.(*core.SelectField)
        if !ok {
            return nil
        }
        var filtered []string
        for _, v := range selectField.Values {
            if v != "rejected" {
                filtered = append(filtered, v)
            }
        }
        selectField.Values = filtered
        return app.Save(collection)
    })
}
```

---

## Shared Patterns

### PostActivity (outgoing delivery)
**Source:** `db/federation/activity.go` lines 55–end
**Apply to:** All three lifecycle hooks (Create → Follow, Update → Accept/Reject, Delete → Undo)
```go
// PostActivity fires asynchronously in a goroutine. It handles:
// - RSA signing of the outgoing request
// - Semaphore-limited concurrency
// - Deduplication of recipient inboxes
// Call site pattern (from follow.go:47, undo.go:67):
PostActivity(app, actorRecord, activity, []string{remoteActor.GetString("inbox")})
```

### GetActorByIRI (remote actor fetch-or-cache)
**Source:** `db/federation/actor.go` line 98; called from `activitypub.go` lines 95–98
**Apply to:** `InstanceInboxHandler` (fetch unknown sender); `InstanceFollowCreateHandler` if followee actor not yet cached
```go
ctx, err := util.GetSafeActorContext(e.Request, recipient)  // e.Request can be nil in hooks
actor, err = federation.GetActorByIRI(e.App, ctx, iri, false)
// In hooks (no HTTP request), use: util.GetSafeActorContext(nil, instanceActor)
```

### VerifySignature (HTTP signature check)
**Source:** `db/util/activitypub.go` line 615; call site `activitypub.go` lines 106–110
**Apply to:** `InstanceInboxHandler` exclusively
```go
verified, err := util.VerifySignature(e.App, e.Request, actor.GetString("public_key"))
if err != nil || !verified {
    e.App.Logger().Error(err.Error())
    return e.UnauthorizedError("Invalid http signature", err)
}
```

### Instance actor lookup (in hooks without request context)
**Source:** `db/federation/instance.go` lines 48, 52 (IRI construction + FindFirstRecordByData)
**Apply to:** `InstanceFollowUpdateHandler`, `InstanceFollowDeleteHandler` (need instance actor for signing Accept/Reject/Undo)
```go
origin := os.Getenv("ORIGIN")
instanceIRI := origin + "/api/v1/activitypub/instance"
instanceActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", instanceIRI)
```

### Activity record construction (IRI + security.RandomStringWithAlphabet)
**Source:** `db/federation/follow.go` lines 40–43; `db/federation/undo.go` lines 47–49
**Apply to:** Any new function that creates an outgoing Accept or Reject activity
```go
recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)
id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
```

---

## No Analog Found

All files have close analogs. No new patterns from external sources are needed.

---

## Metadata

**Analog search scope:** `db/federation/`, `db/hooks/`, `db/routes/`, `db/migrations/`, `db/main.go`
**Files read:** `instance.go`, `follow.go`, `undo.go`, `activity.go` (header), `activitypub.go` (routes), `hooks/follow.go`, `main.go`, `1782290000_add_actor_type_to_activitypub_actors.go`
**Pattern extraction date:** 2026-06-25
