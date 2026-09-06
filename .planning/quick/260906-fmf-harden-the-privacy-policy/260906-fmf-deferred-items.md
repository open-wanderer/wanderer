# Deferred items — 260906-fmf harden the privacy policy

Real privacy/reliability defects surfaced during verification for the privacy-policy
hardening task. Per task scope, these are **documented, not fixed** — this task
changes only `docs/src/pages/privacy.astro` and its own planning notes. Application
code (`db/`, `web/`) is read-only for this task.

**Correction note:** this Defect 1 write-up supersedes an earlier draft that
incorrectly claimed `activitypub_actors.user` has no `cascadeDelete` and that account
deletion therefore removes nothing but the login record. That was wrong — the field
does carry `"cascadeDelete": true`
(`db/migrations/1747061257_created_activitypub_actors.go:230`, in the same field
object as `"name": "user"` at line 236; an earlier read used a `-B 5` grep context
that excluded line 230). The corrected finding below was independently re-verified
against the vendored PocketBase v0.38.0 core source and its own test suite.

## Defect 1: account deletion fails outright (deletes nothing) if the user has any surviving comments

**Severity: High.** This is a reliability defect with a privacy consequence: it is
not that content silently survives — the whole deletion request fails and nothing is
removed, most likely with no clear explanation shown to the user.

**Verified cascade chain when a `users` record is deleted**
(`DELETE /api/v1/user/{id}` → `web/src/routes/api/v1/user/[id]/+server.ts:131` →
`remove()` at `web/src/lib/util/api_util.ts:152-159` → bare
`pb.collection('users').delete(id)`):

| Relation | `cascadeDelete` | `required` | Result |
|---|---|---|---|
| `activitypub_actors.user` → `users` | `true` (`db/migrations/1747061257_created_activitypub_actors.go:230`) | `false` | actor **is** deleted |
| `trails.author` → actor | `true` (`db/migrations/1747061260_trails_add_new_author.go:17`) | `true` | trails **are** deleted |
| `lists.author` → actor | `true` (`db/migrations/1747674856_updated_lists.go:17`) | `true` | lists **are** deleted |
| `summit_logs.author` → actor | `true` (`db/migrations/1747061264_summit_logs_add_new_author.go:17`) | `true` | summit logs **are** deleted |
| `waypoints.author` → actor | `true` (`db/migrations/1775997520_updated_waypoints.go:17`) | `true` | waypoints **are** deleted |
| `follows.follower` / `.followee` → actor | `true` (`db/migrations/1747061270_follows_add_new_f_f.go:69,86`) | `true` | follow rows **are** deleted |
| `notifications.recipient` / `.author` → actor | `true` (`db/migrations/1747995473_updated_notifications.go:23,40`) | `true` | notifications **are** deleted |
| `trail_share.actor`, `list_share.actor`, `trail_like.actor`, `feed.actor`/`.author` → actor | `true` (`db/migrations/1749553104_updated_trail_share.go:17`, `1749566277_updated_list_share.go:27`, `1749717023_created_trail_like.go:44`, `1752231390_created_feed.go:31,44`) | `true` | rows **are** deleted |
| `comments.author` → actor | **`false`** (`db/migrations/1747061262_comments_add_new_author.go:17`) | **`true`** | **blocks the entire deletion** |

`comments.author` is the *only* relation into `activitypub_actors` that combines
`cascadeDelete: false` with `required: true`. Every other referencing collection
cascades cleanly.

**Why this blocks everything, not just the comment:** PocketBase's cascade-delete
logic (`deleteRefRecords`, vendored at
`~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/core/record_model.go:1600-1610`)
checks, per referencing record: if `CascadeDelete` is set, delete the reference; else
if the field is `Required` and would become empty, **return an error instead of
saving** — it never leaves a record with an emptied required relation. This exact
behavior is asserted by PocketBase's own test suite
(`core/record_model_test.go:2244-2249`, "delete existing record while being part of a
non-cascade required relation" — expects `app.Delete` to return an error), and the
HTTP API layer wraps it into a 400 with the message "Failed to delete record. Make
sure that the record is not part of a required relation reference."
(`apis/record_crud.go:594`).

All of this cascade logic — from the `users` row down through the actor to every
referencing collection — runs inside a **single database transaction**
(`core/record_model.go:1487-1497`; nested `RunInTransaction` calls reuse the same
`*dbx.Tx`, confirmed at `core/db_tx.go:26-29`). So the moment PocketBase reaches a
comment still authored by the actor being deleted, the whole transaction returns an
error and rolls back — the `users` record is **not** deleted, the actor is **not**
deleted, nothing is deleted. The frontend does not appear to surface this clearly:
`deleteAccount()` in
`web/src/routes/settings/account/+page.svelte:50-54` has no `try`/`catch` around
`await users_delete($currentUser!)`, so a thrown `APIError` (from
`web/src/lib/stores/user_store.ts:141-151`) prevents the subsequent `logout()` and
`goto("/")` calls from running, with no error toast shown — the user most likely sees
nothing happen.

**Net effect:**
- **User has never commented, or has deleted all their comments first:** account
  deletion succeeds and correctly removes the login record, the actor, and every
  trail/list/summit_log/waypoint/follow/notification/share/like/feed-entry
  associated with that actor. Cascade-deleting trails/lists/summit logs goes through
  the same `app.Delete()` path as a normal delete, so it fires the usual
  `OnRecordAfterDeleteSuccess` hooks and does send a federated Delete activity for
  content that was still public at the time (subject to the same public-only gate
  documented in policy section 8).
- **User still has any comment authored on any trail:** the deletion request fails
  outright. Nothing is removed — not the comment, not the trails, not the login.

## Defect 2: `activitypub_activities` is readable by anyone, unauthenticated, and never purged

**Severity: Medium on its own; compounds with Defect 1's comment-blocking case (where
literally nothing is deleted) and applies unconditionally otherwise.**
`activitypub_activities` has `"listRule": ""` and `"viewRule": ""`
(`db/migrations/1747061258_created_activitypub_activities.go:13-14`), which in
PocketBase means public read access with no authentication required.
`GET /api/v1/activitypub/activity/{id}`
(`web/src/routes/api/v1/activitypub/activity/[id]/+server.ts`) exposes this directly
via `show()` (`web/src/lib/util/api_util.ts:74-85`), which performs no additional auth
check of its own. `actor` on this collection is stored as a plain `url` text field,
not a `relation` (`db/migrations/1747061258_created_activitypub_activities.go:90-100`),
so no cascade rule — even a correctly configured one — could ever reach these rows.

Each activity's `object` field is a full ActivityPub object snapshot, including (for
trails) description text and precise `lat`/`lon` coordinates
(`db/util/activitypub.go:503`, `:509-514`). These snapshots are never cleaned up by
account deletion (successful or not) or by deleting the underlying content, so a
user's trail descriptions and coordinates remain fetchable indefinitely by anyone who
has or guesses an activity id — with no login required and no relationship to the
content's current visibility or existence.

## Recommendation for a future fix (not implemented here)

- Either add `"cascadeDelete": true` to `comments.author`, so a user's comments are
  removed along with the rest of their content, or relax `comments.author` from
  `required` to optional so the reference can be cleared instead of blocking
  deletion. Either fixes the transaction-aborting failure mode.
- Surface delete errors to the user in `web/src/routes/settings/account/+page.svelte`
  (`deleteAccount()`) instead of letting a thrown `APIError` disappear silently.
- Set an authenticated `viewRule`/`listRule` on `activitypub_activities` (e.g.
  restricted to superusers or the record's own actor), since its current use case
  (serving activity objects to remote federated servers) does not require public
  browsability through the collection API — federated delivery already happens via
  `PostActivity`, not via clients reading this collection.

This file exists purely as a record for maintainers; no code changes were made as
part of the 260906-fmf task.
