# Phase 2: Follow Lifecycle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-25
**Phase:** 2-Follow Lifecycle
**Areas discussed:** Outgoing Follow trigger, Inbox routing strategy, Auto-accept bypass design

---

## Outgoing Follow Trigger

| Option | Description | Selected |
|--------|-------------|----------|
| follows record + hook | Admin creates a follows record in PocketBase admin native CRUD; Go OnRecordAfterCreate hook fires, fetches remote actor JSON, delivers Follow activity | ✓ |
| Custom admin endpoint | Lightweight POST endpoint (e.g. /api/v1/activitypub/admin/follow) called via curl/Postman | |
| Custom admin UI now | Build a lightweight form page in PocketBase admin UI this phase (~3–5 hours extra) | |

**User's choice:** follows record + hook (Recommended)
**Notes:** User initially asked about extending the PocketBase dashboard UI. After reviewing the effort (~3–5 hours, requires custom HTML/JS served from Go), user confirmed the native CRUD approach is sufficient for v1. Custom admin UI is deferred to v2 (ADMIN-01).

---

## Inbox Routing Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated handler in instance.go | Instance inbox handler in db/federation/instance.go, routes only instance-level activities | ✓ |
| Extend existing processor | Route to the same ProcessActivity() dispatcher used for user inboxes | |

**User's choice:** Dedicated handler in instance.go (Recommended)
**Notes:** Keeps instance-actor concerns isolated from user-level federation. Consistent with Phase 1's pattern of putting instance code in instance.go.

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse existing sig-check | Call the same signature verification function already used for user inboxes | ✓ |
| Claude decides | Researcher identifies the cleanest approach | |

**User's choice:** Reuse existing sig-check (Recommended)
**Notes:** No new verification logic needed — same function for instance and user inboxes.

---

## Auto-Accept Bypass Design

| Option | Description | Selected |
|--------|-------------|----------|
| Actor-type branch inside existing function | Add if-branch in ProcessFollowActivity(): Application-type → pending, Person → existing auto-accept | ✓ |
| Separate ProcessInstanceFollowActivity() | Route Application-type follows to a new function before reaching ProcessFollowActivity() | |
| Claude decides | Researcher determines cleanest approach | |

**User's choice:** Actor-type branch inside existing function (Recommended)
**Notes:** Minimal change to existing code; user-level federation behavior untouched.

| Option | Description | Selected |
|--------|-------------|----------|
| Record delete triggers Undo hook | OnRecordAfterDelete fires Undo{Follow} delivery; admin deletes follow record in PocketBase admin | ✓ |
| Status field value | Admin sets status to 'unfollowed'; preserves audit trail but adds non-standard status value | |

**User's choice:** Record delete triggers Undo hook (Recommended)
**Notes:** Consistent hook-based pattern across all lifecycle events. Admin deletes the record to unfollow.

---

## Claude's Discretion

- Researcher determines exact PocketBase hook types (`OnRecordAfterCreate`, etc.) and call patterns
- Researcher identifies existing HTTP signature verification function and call site
- Researcher checks for existing `FetchActor()` utility in `db/federation/` for fetching remote actor JSON
- Researcher determines how to filter Accept/Reject/Undo hooks so they only fire for instance-level follows (not user follows)

## Deferred Ideas

- **Custom admin UI for federation management** — Admin wants a proper UI to initiate follows, view connection status (pending/accepted/rejected), direction, and date. Deferred to v2 as ADMIN-01 in REQUIREMENTS.md.
