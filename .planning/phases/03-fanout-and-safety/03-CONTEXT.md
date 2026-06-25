# Phase 3: Fanout and Safety - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Activate content synchronization so that public trails, comments, lists, and summit_logs created/updated/deleted on this instance are automatically delivered to all accepted instance-actor followers. Bundle three safety constraints that cannot ship separately:

1. Outgoing fanout: each `Create*Activity` and `Delete*Activity` function includes accepted instance-actor followers as additional recipients
2. Broadcast-loop prevention (SAFE-01): incoming `processCreate*` functions check for duplicate content IRIs before inserting
3. Privacy hard gate (SAFE-03): outgoing fanout checks `is_public = true` before including any record
4. Delete authorization fix (SAFE-02): `processDeleteTrailActivity` verifies the deleting actor is the trail's original author

**In scope:** SYNC-01, SYNC-02, SYNC-03, SAFE-01, SAFE-02, SAFE-03
**Out of scope:** NodeInfo (Phase 4), public→private propagation (v2 VIS-01), per-user opt-out (v2), historical backfill (explicitly out of scope in REQUIREMENTS.md)

</domain>

<decisions>
## Implementation Decisions

### Fanout Injection

- **D-01:** Add `instanceFollowerInboxes(app core.App) ([]string, error)` helper function in `db/federation/activity.go`. This function computes the instance actor IRI as `os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"`, fetches the actor record via `FindFirstRecordByData("activitypub_actors", "iri", iri)`, and calls `followerInboxes(app, instanceActor.Id)`.
- **D-02:** If the instance actor record is not found (`sql.ErrNoRows`), `instanceFollowerInboxes` returns `(nil, nil)` — empty slice, no error. Fanout proceeds with only user-level followers. This makes startup ordering safe (instance actor is always created before first content hook can fire, but defensive regardless).
- **D-03:** Each of the 4 outgoing `Create*Activity` functions (trail, comment, summitLog, list) and each of the 4 outgoing `Delete*Activity` functions calls `instanceFollowerInboxes(app)` and appends the result to the existing `recipients` slice before calling `PostActivity`. No structural changes to existing fanout logic — additive only.

### Broadcast-Loop Deduplication (SAFE-01)

- **D-04:** Dedup is enforced inside each `processCreateOrUpdate*` function (trail, comment, summitLog, list), not at the inbox handler level. Each function checks whether a content record with the incoming object IRI already exists in the relevant collection (`trails`, `comments`, `summit_logs`, `lists`). If found: return `nil` silently (idempotent delivery — remote gets 200, no retry triggered).
- **D-05:** No changes to `activitypub_activities` for incoming dedup tracking. Content records themselves serve as the dedup store (their `iri` field is the unique key). This is consistent with how `ProcessFollowActivity` already uses `FindFirstRecordByFilter("follows", ...)` for its own idempotency guard.

### Delete Authorization Fix (SAFE-02)

- **D-06:** Change `processDeleteTrailActivity` signature from `(app core.App, activity pub.Activity)` to `(app core.App, actor *core.Record, activity pub.Activity)`. Update the single call site in `ProcessDeleteActivity` to pass `actor`. This is consistent with `processDeleteCommentActivity` and `processDeleteSummitLogActivity` which already accept the actor.
- **D-07:** Add ownership check at the top of `processDeleteTrailActivity`: compare `actor.GetString("iri")` against `trail.GetString("author")` (the author field stores the actor IRI). If they do not match: return `fmt.Errorf(...)` describing the unauthorized delete attempt. This propagates as a 400 response to the remote via the instance inbox handler's error propagation.

### Comment Privacy Gate (SAFE-03)

- **D-08:** Add a parent trail `public` check at the top of `CreateCommentActivity` (whole-function gate, not instance-fanout-only). Fetch the comment's parent trail record and check `trail.GetBool("public")`. If false: return `nil` immediately — no user-level or instance-level fanout. Consistent with how `CreateSummitLogActivity` gates on `trail.GetBool("public")`.
- **D-09:** Existing gates in `CreateTrailActivity` (`trail.GetBool("public")`) and `CreateListActivity` (`list.GetBool("public")`) are already in place. `CreateSummitLogActivity` already gates on the parent trail's `public` field. No changes needed for those three.

### Instance Inbox — Incoming Content Activities

- **D-10:** Extend `InstanceInboxHandler` in `db/federation/instance.go` to dispatch incoming `Create` and `Update` activities to `ProcessCreateOrUpdateActivity`, and `Delete` activities to `ProcessDeleteActivity`. These are the same functions already used by user-level inboxes — no new processing logic needed. Phase 3 adds these switch cases to the existing `switch activity.Type` block.

### Claude's Discretion

- Error message text for SAFE-02 unauthorized delete (content of the `fmt.Errorf` string)
- Whether `instanceFollowerInboxes` de-duplicates against user-level inboxes (edge case where the instance actor and a user happen to share a follower; dedup is nice-to-have)

</decisions>

<specifics>
## Specific Ideas

- No strong preferences on error message wording for SAFE-02 — functional clarity is sufficient
- Dedup behavior on broadcast loop: "already have it → return nil silently" mirrors the existing `ProcessFollowActivity` idempotency pattern exactly

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and Roadmap
- `.planning/REQUIREMENTS.md` §"Content Synchronization" — SYNC-01, SYNC-02, SYNC-03: which content types and activity types must be delivered to instance followers
- `.planning/REQUIREMENTS.md` §"Safety and Correctness" — SAFE-01, SAFE-02, SAFE-03: dedup, delete authorization, and privacy gate requirements
- `.planning/ROADMAP.md` §"Phase 3: Fanout and Safety" — success criteria (5 criteria, each maps to a requirement)

### Outgoing Fanout (SYNC-01/02/03)
- `db/federation/activity.go` — `followerInboxes()` and `PostActivity()` — the delivery infrastructure; `instanceFollowerInboxes()` will be added here
- `db/federation/create.go` — `CreateTrailActivity`, `CreateCommentActivity`, `CreateSummitLogActivity`, `CreateListActivity` — each handles Create and Update (via `typ` param); these get the `instanceFollowerInboxes` append
- `db/federation/delete.go` — `CreateTrailDeleteActivity`, `CreateCommentDeleteActivity`, `CreateSummitLogDeleteActivity`, `CreateListDeleteActivity` — these get the same append; `processDeleteTrailActivity` gets the actor fix here

### Incoming Content (SYNC-01/02/03 receiving side + SAFE-01)
- `db/federation/create.go` — `ProcessCreateOrUpdateActivity` and its 4 sub-functions — dedup guards go inside each `processCreateOrUpdate*` function
- `db/federation/delete.go` — `ProcessDeleteActivity` and `processDeleteTrailActivity` — ownership check fix goes here
- `db/federation/instance.go` — `InstanceInboxHandler` — new `pub.CreateType`, `pub.UpdateType`, `pub.DeleteType` cases added to switch block

### Phase 2 Patterns (apply same conventions)
- `.planning/phases/02-follow-lifecycle/02-03-PLAN.md` — established error propagation pattern (per-case `e.BadRequestError`); apply same in inbox handler extension
- `db/hooks/follow.go` — `isOutboundInstanceFollow` and directional guard patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `followerInboxes(app, actorId string) ([]string, error)` in `db/federation/activity.go:29` — existing JOIN query for accepted followers; `instanceFollowerInboxes` uses this directly
- `PostActivity(app, actor, activity, recipients)` in `db/federation/activity.go` — handles goroutine, HTTP signing, delivery; unchanged
- `ProcessCreateOrUpdateActivity(app, actor, recipient, activity)` in `db/federation/create.go:412` — existing dispatcher for incoming Create/Update; instance inbox handler will call this
- `ProcessDeleteActivity(app, actor, activity)` in `db/federation/delete.go:266` — existing dispatcher for incoming Delete; already receives actor; calls `processDeleteTrailActivity` which needs the fix

### Established Patterns
- Privacy gate pattern: `if !record.GetBool("public") { return nil }` — used in `CreateTrailActivity:22`, `CreateSummitLogActivity:203`, `CreateListActivity:350`
- Idempotency guard pattern: `FindFirstRecordByFilter("follows", ...)` returning `nil` on duplicate — in `ProcessFollowActivity`; SAFE-01 mirrors this per content type
- Error propagation pattern from Phase 2: per-case `e.BadRequestError(...)` in `InstanceInboxHandler` switch — extend same switch with Create/Update/Delete cases

### Integration Points
- `InstanceInboxHandler` switch block (`db/federation/instance.go:180+`) — add `pub.CreateType`, `pub.UpdateType`, `pub.DeleteType` cases that call `ProcessCreateOrUpdateActivity` and `ProcessDeleteActivity`
- `CreateTrailDeleteActivity` recipients slice (currently `followerInboxes(app, author.Id)`) — append `instanceFollowerInboxes(app)` result
- `processDeleteTrailActivity` call site in `ProcessDeleteActivity:277` — update to pass `actor`

</code_context>

<deferred>
## Deferred Ideas

- Public→private trail propagation (send Delete to instance followers when `is_public` changes false): v2 VIS-01, explicitly out of scope
- Private→public trail propagation (send Create when trail becomes public): v2 VIS-02, explicitly out of scope
- De-duplicating instance follower inboxes against user-level follower inboxes (edge case): Claude's discretion, not a hard requirement

</deferred>

---

*Phase: 03-fanout-and-safety*
*Context gathered: 2026-06-25*
