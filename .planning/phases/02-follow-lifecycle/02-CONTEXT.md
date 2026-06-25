# Phase 2: Follow Lifecycle - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire up the instance actor inbox endpoint and the full Follow lifecycle so admins can connect two Wanderer instances bidirectionally:

1. POST `/api/v1/activitypub/instance/inbox` accepts HTTP-signed ActivityPub activities from remote instances
2. Incoming Follow from an `Application`-type actor is stored as `pending` in the `follows` collection (not auto-accepted)
3. Admin accepts or rejects a pending Follow via PocketBase admin panel; `Accept{Follow}` or `Reject{Follow}` is delivered to the remote inbox
4. Admin initiates an outgoing Follow by creating a record in the `follows` collection via PocketBase admin; the local instance actor sends a `Follow` activity to the remote
5. Admin unfollows a peer instance by deleting the follow record; `Undo{Follow}` is sent to the remote

**In scope:** INST-03, FLCL-01, FLCL-02, FLCL-03, FLCL-04, FLCL-05
**Out of scope:** Content fanout (Phase 3), broadcast-loop deduplication (Phase 3), NodeInfo (Phase 4), custom admin web UI (v2 ADMIN-01)

</domain>

<decisions>
## Implementation Decisions

### Outgoing Follow Trigger (FLCL-01)
- **D-01:** Admin initiates an outgoing Follow by creating a record directly in the `follows` collection via PocketBase's native admin CRUD UI. A Go `OnRecordAfterCreate` hook fires, fetches the remote actor's ActivityPub JSON (to discover their inbox URL), and delivers the `Follow` activity to the remote inbox.
- **D-02:** No custom admin UI is built in this phase. A custom admin panel for federation management is deferred to v2 (ADMIN-01). For v1, admin uses PocketBase's native collection record form to supply the remote actor IRI.

### Instance Inbox Handler (INST-03)
- **D-03:** The instance inbox endpoint (`POST /api/v1/activitypub/instance/inbox`) gets a **dedicated handler in `db/federation/instance.go`**. It does NOT route through the existing user-inbox `ProcessActivity()` dispatcher. This keeps instance-actor concerns isolated from user-level federation, consistent with Phase 1's `instance.go` isolation pattern.
- **D-04:** HTTP signature verification on the instance inbox reuses whatever signature-check function is already implemented for user inboxes. No new verification logic is introduced.

### Auto-Accept Bypass for Application-Type Follows (FLCL-02)
- **D-05:** `ProcessFollowActivity()` is extended with a branch: if the Follow's *object* (the followee) resolves to the local instance actor (`actor_type == "instance"` and `is_local == true`), the follow record is stored with `status = "pending"` and the function returns without sending an `Accept`. If the object is a user actor (existing user-level follow), the existing auto-accept logic runs unchanged. **Note:** Original discussion assumed filtering on the sending actor's `actor_type == "Application"`, but research confirmed remote actors fetched via `GetActorByIRI` do not have `actor_type` populated locally, making sender-side filtering unreliable. Object-side filtering (checking the followee is the local instance actor) is the correct approach and is what must be implemented. This is the minimal change that satisfies FLCL-02 without affecting user federation.

### Accept/Reject/Undo Lifecycle Hooks (FLCL-03, FLCL-04, FLCL-05)
- **D-06:** `OnRecordAfterUpdate` hook on the `follows` collection: when `status` changes to `accepted`, deliver `Accept{Follow}` to the remote instance's inbox; when `status` changes to `rejected`, deliver `Reject{Follow}`.
- **D-07:** `OnRecordAfterDelete` hook on the `follows` collection: when an instance follow record is deleted (admin removes it in PocketBase admin), deliver `Undo{Follow}` to the remote instance's inbox.
- **D-08:** `Reject{Follow}` delivery is **mandatory** (not optional). The remote instance's UI cannot recover if it never receives the rejection. This was flagged in STATE.md and must not be skipped.

### Claude's Discretion
- Researcher determines which existing function handles HTTP signature verification and how to call it from the instance inbox handler.
- Researcher determines which PocketBase lifecycle hook type fires reliably after a `follows` record is created/updated/deleted (likely `OnRecordAfterCreate`, `OnRecordAfterUpdate`, `OnRecordAfterDelete`).
- Researcher identifies whether a `FetchActor()` utility already exists in `db/federation/` for fetching remote actor JSON, or whether one needs to be written.
- Hook filtering: researcher must ensure the Accept/Reject/Undo hooks only fire for instance-level follows (where the followee or follower is the instance actor), not for user-level follows.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and Roadmap
- `.planning/REQUIREMENTS.md` §"Follow Lifecycle" — FLCL-01 through FLCL-05 and INST-03 are the locked requirements for this phase; acceptance criteria defined there
- `.planning/ROADMAP.md` §"Phase 2: Follow Lifecycle" — success criteria (inbox accepts signed activities, Follow → pending, Accept/Reject/Undo delivery)

### Existing Federation Code
- `db/federation/instance.go` — Phase 1 output; instance actor GET handler and `initInstanceActor()` live here; Phase 2 adds inbox handler to this file
- `db/federation/follow.go` — existing `ProcessFollowActivity()` that currently auto-accepts; **must be extended** with actor-type branch (D-05); researcher must read this before planning
- `db/federation/activity.go` — `PostActivity()` and `followerInboxes()`; `PostActivity()` is how all outgoing activities are delivered; reuse for Accept/Reject/Undo delivery
- `db/federation/actor.go` — existing actor fetch/create patterns; researcher checks for a `FetchActor()` or equivalent utility for fetching remote actor JSON
- `db/main.go` — where route registrations and PocketBase hook registrations happen; new instance inbox route and follow lifecycle hooks registered here

### Schema
- `db/migrations/1747061257_created_activitypub_actors.go` — existing `activitypub_actors` schema; `actor_type` column was added in Phase 1
- The `follows` collection — researcher must check current fields (`followee`, `follower`, `status`) to understand what the admin must fill in when creating an outgoing Follow record

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `PostActivity()` in `db/federation/activity.go` — the delivery function for all outgoing AP activities; reuse for Accept, Reject, Undo delivery from lifecycle hooks
- HTTP signature verification — already in place for user inboxes; instance inbox handler must call the same function (researcher identifies exact call site)
- `activitypub_actors` collection with `actor_type` field — Phase 1 added `actor_type = "Application"` for the instance actor; the instance inbox handler and hook filters can use this to distinguish instance vs. user actors

### Established Patterns
- Phase 1 set the pattern: instance-actor code in `db/federation/instance.go`, routes/hooks registered in `db/main.go`
- Hooks fire on PocketBase collection events (`OnRecordAfterCreate`, `OnRecordAfterUpdate`, `OnRecordAfterDelete`) — the same pattern used throughout `db/main.go` for other business logic
- `ORIGIN` env var used via `os.Getenv("ORIGIN")` for IRI construction

### Integration Points
- `db/main.go` — registers the new `POST /api/v1/activitypub/instance/inbox` route and all three follow lifecycle hooks
- `db/federation/follow.go` `ProcessFollowActivity()` — modified with actor-type branch; existing user-level follow behavior must remain unchanged
- `follows` collection — the central state store for all follow relationships; hooks on this collection drive the lifecycle

</code_context>

<specifics>
## Specific Ideas

No specific UI or behavioral references from discussion — implementation follows from REQUIREMENTS.md acceptance criteria and the decisions above.

</specifics>

<deferred>
## Deferred Ideas

- **Custom admin UI for federation management** — Admin wants a proper UI to initiate follows and view connection status. This is v2 scope (ADMIN-01 in REQUIREMENTS.md). For v1, PocketBase native admin CRUD is sufficient.

</deferred>

---

*Phase: 2-Follow Lifecycle*
*Context gathered: 2026-06-25*
