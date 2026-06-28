# Phase 3: Fanout and Safety - Pattern Map

**Mapped:** 2026-06-25
**Files analyzed:** 4 (all modifications to existing files; no new files)
**Analogs found:** 4 / 4

---

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---------------|------|-----------|----------------|---------------|
| `db/federation/activity.go` | utility/helper | request-response | `db/federation/activity.go:31` (`followerInboxes`) | exact — `instanceFollowerInboxes` mirrors `followerInboxes` |
| `db/federation/create.go` | service | event-driven | `db/federation/create.go` itself (existing `CreateTrailActivity`, `processCreateOrUpdate*`) | self-referential extension |
| `db/federation/delete.go` | service | event-driven | `db/federation/delete.go` itself; `processDeleteCommentActivity` is the model for `processDeleteTrailActivity` SAFE-02 fix | self-referential extension |
| `db/federation/instance.go` | middleware/handler | request-response | `db/routes/activitypub.go` user inbox handler; `db/federation/instance.go` itself (existing switch block) | exact — same switch dispatch pattern |

---

## Pattern Assignments

### `db/federation/activity.go` — ADD `instanceFollowerInboxes`

**Analog:** `db/federation/activity.go` lines 31–53 (`followerInboxes`)

**Existing helper to mirror** (lines 31–53):
```go
func followerInboxes(app core.App, actorId string) ([]string, error) {
    rows, err := app.DB().
        Select("aa.inbox").
        From("follows f").
        InnerJoin("activitypub_actors aa", dbx.NewExp("f.follower = aa.id")).
        Where(dbx.NewExp("f.followee = {:followee} AND f.status = 'accepted' AND aa.inbox != ''",
            dbx.Params{"followee": actorId})).
        Rows()
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var inboxes []string
    for rows.Next() {
        var inbox string
        if err := rows.Scan(&inbox); err != nil {
            return nil, err
        }
        inboxes = append(inboxes, inbox)
    }
    return inboxes, rows.Err()
}
```

**Idempotency / not-found guard pattern** (from `db/federation/instance.go` lines 56–62):
```go
existing, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
if err == nil && existing != nil {
    return nil
}
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    return fmt.Errorf("checking instance actor existence: %w", err)
}
```

**New function to add** (D-01/D-02 — insert after `followerInboxes`):
```go
// instanceFollowerInboxes returns inbox URLs for all accepted followers of the
// local instance actor. Returns (nil, nil) if the instance actor has not yet
// been seeded (D-02: startup-safe behavior).
func instanceFollowerInboxes(app core.App) ([]string, error) {
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        return nil, fmt.Errorf("ORIGIN not set")
    }
    iri := origin + "/api/v1/activitypub/instance"
    instanceActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return followerInboxes(app, instanceActor.Id)
}
```

**Required imports already present in `activity.go`:** `os`, `fmt`. Must add `"database/sql"` and `"errors"` — these are already imported in `instance.go` and `follow.go`, so the pattern is established.

---

### `db/federation/create.go` — Fanout injection (D-03) + Dedup guards (D-04) + Comment privacy gate (D-08)

#### Pattern A: Fanout injection in `Create*Activity` functions

**Analog:** `db/federation/create.go` lines 91–97 (`CreateTrailActivity`) — existing fanout block:
```go
inboxes, err := followerInboxes(app, trailAuthor.Id)
if err != nil {
    return err
}
recipients := append(mentions, inboxes...)

return PostActivity(app, trailAuthor, activity, recipients)
```

**Pattern to apply** (add between `recipients` construction and `PostActivity` call, in all 4 `Create*Activity` functions):
```go
instanceInboxes, err := instanceFollowerInboxes(app)
if err != nil {
    return err
}
recipients = append(recipients, instanceInboxes...)

return PostActivity(app, trailAuthor, activity, recipients)
```

Note: `PostActivity` deduplicates via `slices.Sort` + `slices.Compact` (activity.go lines 101–102) — no manual dedup needed.

#### Pattern B: Dedup guard in `processCreateOrUpdate*` functions (D-04/SAFE-01)

**Analog:** `db/federation/follow.go` lines 85–95 (idempotency guard in `ProcessFollowActivity`):
```go
existing, existErr := app.FindFirstRecordByFilter(
    "follows",
    "follower={:follower} && followee={:followee}",
    dbx.Params{"follower": actor.Id, "followee": object.Id},
)
if existErr == nil && existing != nil {
    return nil // already recorded; treat as duplicate delivery
}
if existErr != nil && !errors.Is(existErr, sql.ErrNoRows) {
    return existErr
}
```

**Pattern to apply** (insert at the top of each `processCreateOrUpdate*` function, before any record fetch or DB write):
```go
objectIRI := activity.Object.GetID().String()
existing, err := app.FindFirstRecordByData("trails", "iri", objectIRI)
if err == nil && existing != nil {
    return nil // already have this content — silent dedup (D-04)
}
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    return err
}
```

Replace `"trails"` with `"comments"`, `"summit_logs"`, or `"lists"` for the respective sub-functions.

**Critical:** Apply dedup regardless of `activity.Type` (both Create and Update per D-04/CONTEXT.md). The research OPEN QUESTION recommends Create-only dedup, but D-04 is explicit — implement as specified and flag in plan if clarification needed.

#### Pattern C: Comment privacy gate (D-08/SAFE-03)

**Analog:** `db/federation/create.go` lines 199–206 (`CreateSummitLogActivity` — parent trail public check):
```go
summitLogTrail, err := app.FindRecordById("trails", summitLog.GetString("trail"))
if err != nil {
    return err
}
if !summitLogTrail.GetBool("public") {
    // only broadcast the log if the trail it belongs to is public
    return nil
}
```

**Gap confirmed:** `CreateCommentActivity` (line 100) fetches `commentTrail` at line 112 but has no `if !commentTrail.GetBool("public")` guard. The existing trail fetch is:
```go
commentTrail, err := app.FindRecordById("trails", comment.GetString("trail"))
if err != nil {
    return err
}
```

**Pattern to add** (immediately after the `commentTrail` fetch, before `commentTrailAuthor` fetch):
```go
if !commentTrail.GetBool("public") {
    return nil
}
```

---

### `db/federation/delete.go` — Fanout injection (D-03) + SAFE-02 signature fix + `CreateCommentDeleteActivity` restructure

#### Pattern A: Fanout injection in `Create*DeleteActivity` functions

**Analog:** `db/federation/delete.go` lines 68–73 (`CreateTrailDeleteActivity` — existing fanout):
```go
recipients, err := followerInboxes(app, author.Id)
if err != nil {
    return err
}

return PostActivity(app, author, activity, recipients)
```

**Pattern to apply** (for `CreateTrailDeleteActivity`, `CreateSummitLogDeleteActivity`, `CreateListDeleteActivity`):
```go
recipients, err := followerInboxes(app, author.Id)
if err != nil {
    return err
}
instanceInboxes, err := instanceFollowerInboxes(app)
if err != nil {
    return err
}
recipients = append(recipients, instanceInboxes...)

return PostActivity(app, author, activity, recipients)
```

#### Pattern B: `CreateCommentDeleteActivity` restructure (Pitfall 3)

**Current code** (delete.go lines 122–125) — literal slice, not appendable:
```go
err = PostActivity(app, author, activity, []string{to + "/inbox"})
```

Also, the current early-return at lines 102–104 blocks all fanout when `commentTrailAuthor.GetBool("is_local")`:
```go
if commentTrailAuthor.GetBool("is_local") {
    return nil
}
```

**Restructured pattern** (send to instance followers even when trail author is local, but only send to trail author when they are remote):
```go
recipients := []string{}
if !commentTrailAuthor.GetBool("is_local") {
    recipients = append(recipients, to+"/inbox")
}
instanceInboxes, err := instanceFollowerInboxes(app)
if err != nil {
    return err
}
recipients = append(recipients, instanceInboxes...)

err = PostActivity(app, author, activity, recipients)
```

Remove the `if commentTrailAuthor.GetBool("is_local") { return nil }` early exit (lines 102–104).

#### Pattern C: SAFE-02 — `processDeleteTrailActivity` signature fix + ownership check

**Analog:** `db/federation/delete.go` lines 309–325 (`processDeleteCommentActivity` — the established ownership-check pattern):
```go
func processDeleteCommentActivity(app core.App, actor *core.Record, activity pub.Activity) error {
    object := activity.Object.GetID().String()

    comment, err := app.FindFirstRecordByData("comments", "iri", object)
    if err != nil {
        return err
    }

    if comment.GetString("author") != actor.Id {
        return fmt.Errorf("actor is not comment author")
    }

    err = app.Delete(comment)
    if err != nil {
        return err
    }
    return nil
}
```

**Also:** `db/federation/delete.go` lines 328–341 (`processDeleteSummitLogActivity`) confirms the same pattern:
```go
if summitLog.GetString("author") != actor.Id {
    return fmt.Errorf("actor is not summit log author")
}
```

**Current `processDeleteTrailActivity`** (delete.go lines 293–307 — missing actor param and ownership check):
```go
func processDeleteTrailActivity(app core.App, activity pub.Activity) error {

    object := activity.Object.GetID().String()
    trail, err := app.FindFirstRecordByData("trails", "iri", object)
    if err != nil {
        return err
    }

    err = util.DeleteFromFeed(app, trail.Id)
    if err != nil {
        return err
    }

    return app.Delete(trail)
}
```

**New signature and body** (D-06/D-07):
```go
func processDeleteTrailActivity(app core.App, actor *core.Record, activity pub.Activity) error {
    object := activity.Object.GetID().String()
    trail, err := app.FindFirstRecordByData("trails", "iri", object)
    if err != nil {
        return err
    }

    if trail.GetString("author") != actor.Id {
        return fmt.Errorf("actor is not trail author")
    }

    err = util.DeleteFromFeed(app, trail.Id)
    if err != nil {
        return err
    }

    return app.Delete(trail)
}
```

**Call site update** (delete.go line 277 in `ProcessDeleteActivity`):
```go
// Before:
case strings.Contains(object, "trail"):
    err = processDeleteTrailActivity(app, activity)
// After:
case strings.Contains(object, "trail"):
    err = processDeleteTrailActivity(app, actor, activity)
```

**CRITICAL:** `trail.GetString("author")` returns a PocketBase record ID (15-char alphanumeric), NOT an IRI. Compare against `actor.Id`, not `actor.GetString("iri")`. This is confirmed by `db/util/trail_access.go` and the `processDeleteCommentActivity` pattern above.

---

### `db/federation/instance.go` — Extend `InstanceInboxHandler` switch (D-10)

**Analog:** `db/federation/instance.go` lines 172–187 (existing switch block):
```go
switch activity.Type {
case pub.FollowType:
    if err := ProcessFollowActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Follow activity", err)
    }
case pub.AcceptType:
    if err := ProcessAcceptActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Accept activity", err)
    }
case pub.UndoType:
    if err := ProcessUndoActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Undo activity", err)
    }
default:
    return e.BadRequestError("Unsupported activity type", nil)
}
```

**`recipient` variable** is already available at line 138:
```go
recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "inbox", inbox)
```

**New cases to insert before `default:`** (D-10):
```go
case pub.CreateType:
    fallthrough
case pub.UpdateType:
    if err := ProcessCreateOrUpdateActivity(e.App, actor, recipient, activity); err != nil {
        return e.BadRequestError("Failed to process Create/Update activity", err)
    }
case pub.DeleteType:
    if err := ProcessDeleteActivity(e.App, actor, activity); err != nil {
        return e.BadRequestError("Failed to process Delete activity", err)
    }
```

**Pattern source:** Per-case `e.BadRequestError(...)` established in Phase 2 (`.planning/phases/02-follow-lifecycle/02-03-PLAN.md`) and confirmed by the existing switch cases above.

**`recipient` usage:** `ProcessCreateOrUpdateActivity` (create.go line 412) uses `recipient *core.Record` for `util.InsertIntoFeed` calls. Passing the instance actor record as `recipient` is correct — the instance actor is a valid `*core.Record` with an `.Id` field.

---

## Shared Patterns

### Not-found / idempotency guard
**Source:** `db/federation/follow.go` lines 85–95, `db/federation/instance.go` lines 56–62
**Apply to:** `instanceFollowerInboxes` (D-02), all 4 `processCreateOrUpdate*` dedup guards (D-04)
```go
if err == nil && existing != nil {
    return nil
}
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    return err
}
```

### Public gate (early-return nil)
**Source:** `db/federation/create.go` line 22 (`CreateTrailActivity`), lines 203–206 (`CreateSummitLogActivity`)
**Apply to:** `CreateCommentActivity` (D-08) — add after `commentTrail` is fetched
```go
if !record.GetBool("public") {
    return nil
}
```

### Instance fanout append
**Source:** `db/federation/activity.go` lines 101–102 (PostActivity deduplication)
**Apply to:** All 8 outgoing activity functions (D-03)
```go
instanceInboxes, err := instanceFollowerInboxes(app)
if err != nil {
    return err
}
recipients = append(recipients, instanceInboxes...)
```

### Ownership check before delete
**Source:** `db/federation/delete.go` lines 317–319 (`processDeleteCommentActivity`), lines 336–338 (`processDeleteSummitLogActivity`)
**Apply to:** `processDeleteTrailActivity` (D-06/D-07/SAFE-02)
```go
if record.GetString("author") != actor.Id {
    return fmt.Errorf("actor is not <type> author")
}
```

### Error propagation in inbox handler
**Source:** `db/federation/instance.go` lines 173–187 (existing switch cases)
**Apply to:** New Create/Update/Delete cases in `InstanceInboxHandler` (D-10)
```go
if err := ProcessXxxActivity(...); err != nil {
    return e.BadRequestError("Failed to process Xxx activity", err)
}
```

---

## No Analog Found

None. All patterns have direct equivalents in the existing codebase.

---

## Anti-Patterns Called Out by Research

| Anti-Pattern | Why Wrong | Correct Pattern |
|---|---|---|
| `actor.GetString("iri") != trail.GetString("author")` | `author` stores a record ID, not an IRI | `trail.GetString("author") != actor.Id` |
| Dedup at inbox handler level | D-04 places it inside each `processCreateOrUpdate*` | Add check at top of each sub-function |
| `errors.Is(err, sql.ErrNoRows)` returning error in `instanceFollowerInboxes` | Breaks outgoing fanout on startup | Return `(nil, nil)` on `sql.ErrNoRows` |
| Adding `ctx context.Context` to `CreateListActivity` | Function has no mention resolution | Do not add ctx param |
| `PostActivity(app, author, activity, []string{to + "/inbox"})` literal in `CreateCommentDeleteActivity` | Cannot append instance inboxes to a literal | Extract to `recipients` variable first |

---

## Metadata

**Analog search scope:** `db/federation/` (activity.go, create.go, delete.go, follow.go, instance.go)
**Files scanned:** 5
**Pattern extraction date:** 2026-06-25
