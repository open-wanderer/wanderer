# Findings — 260906-fmf harden the privacy policy

Evidence gathered by reading the code directly. Every verdict below carries at least
one `file:line` citation. Where I could not find something, I say "not found" rather
than guessing.

## Q1 — Do comments on private trails reach remote servers? (REV-01)

**Verdict: YES, confirmed.**

- `CreateTrailActivity` (`db/federation/create.go:22`), `CreateSummitLogActivity`
  (`db/federation/create.go:203`) and `CreateListActivity` (`db/federation/create.go:350`)
  each early-return with `if !record.GetBool("public") { return nil }` before anything
  is sent. `CreateCommentActivity` (`db/federation/create.go:100`) has **no** such
  check anywhere in the function body — it runs unconditionally regardless of the
  parent trail's `public` value.
- `db/hooks/comments.go:45` (`CreateCommentHandler`) and `db/hooks/comments.go:65`
  (`UpdateCommentHandler`) call `federation.CreateCommentActivity` unconditionally on
  every comment create/update — no public check happens before the call either.
- (a) **No public gate exists anywhere on the comment path.** This directly confirms
  the reviewer's and the planner's claim.
- (b) **Recipients:** every remote actor `@mentioned` in the comment text
  (`db/federation/create.go:126-139`, via `ActorsFromMentions`), **plus** the trail
  author's inbox unconditionally (`db/federation/create.go:140`:
  `recipients = append(recipients, commentTrailAuthor.GetString("inbox"))`). This
  happens even when the trail author is remote and the trail is private — the trail's
  `public` field is never read in this function.
- (c) **Yes, the payload contains the full comment body.** `ObjectFromComment`
  (`db/util/activitypub.go:667`) sets `commentObject.Content` to
  `comment.GetString("text")` verbatim.
- (d) **What rides along about the parent trail:** only the trail's IRI, via
  `commentObject.InReplyTo` (`db/util/activitypub.go:670`). No trail title,
  description or coordinates are embedded in the comment object itself. (The remote
  recipient could use that IRI to try to fetch the trail separately, but the comment
  activity itself does not carry trail content.)

## Q2 — Does flipping public → private emit any Delete or Undo? (REV-02)

**Verdict: Confirmed for trails; the exact trap is trail-specific — lists and summit
logs behave differently on delete.**

- (a) **Unpublish emits nothing for trails and lists; summit logs inherit the trail's
  state.**
  - `UpdateTrailHandler` (`db/hooks/trails.go:113`) always calls
    `federation.CreateTrailActivity(..., pub.UpdateType)`, which returns nil at
    `db/federation/create.go:22` once `public` is false. Confirmed: unpublishing a
    trail sends no Update and (since it was never sent as a delete either) no
    onward signal of any kind.
  - `UpdateListHandler` (`db/hooks/list.go:90`) calls
    `federation.CreateListActivity(..., pub.UpdateType)`, gated the same way at
    `db/federation/create.go:350`. Same behavior as trails.
  - Summit logs have no independent `public` field; `CreateSummitLogActivity`
    (`db/federation/create.go:203`) gates on the **parent trail's** `public` flag, so
    unpublishing the trail also silently stops summit-log federation for it.
- (b) **Delete-after-unpublish is silent — but only for trails.**
  `CreateTrailDeleteActivity` (`db/federation/delete.go:17`) *also* checks
  `if !r.GetBool("public") { return nil }` before sending anything. A trail that was
  made private and *then* deleted sends no Delete activity at all — remote copies
  from when it was public are never told to remove it. This is the trap the reviewer
  flagged, confirmed.
  By contrast, `CreateListDeleteActivity` (`db/federation/delete.go:209`) and
  `CreateSummitLogDeleteActivity` (`db/federation/delete.go:138`) contain **no**
  equivalent public-status check — they check only whether the author `is_local`
  and otherwise always send a Delete on actual deletion, regardless of the record's
  public state at that moment. So the "delete becomes silent" trap is specific to
  trails, not a blanket rule across all content types.
- (c) **What the delete path emits when still public at deletion time:** a normal
  ActivityPub `Delete` activity referencing the object's IRI, sent to the author's
  followers (`db/federation/delete.go:40-73` for trails; equivalent patterns in the
  list/summit-log/comment delete functions).

## Q3 — What survives account deletion? (REV-03)

**CORRECTION (post-checkpoint):** the paragraph originally here claimed
`activitypub_actors.user` has no `cascadeDelete` and that account deletion therefore
removes nothing but the login record. That was a misread: `"cascadeDelete": true` is
present at `db/migrations/1747061257_created_activitypub_actors.go:230`, inside the
same field object as `"name": "user"` at line 236 — an earlier `grep -B 5` context
window ended at line 231 and excluded line 230. Corrected and independently
re-verified below, including against the vendored PocketBase v0.38.0 core source and
its own test suite.

**Verdict: account deletion normally cascades correctly through the actor to trails,
lists and summit logs — but fails outright (deleting nothing at all) if the user has
any comment that still exists, because of one specific relation-field
misconfiguration. This is a different and more precise defect than either the
original miswrite or the reviewer's framing — see Deferred section.**

- (a) **The cascade chain, verified relation by relation.** Deleting a `users` record
  cascades via `activitypub_actors.user` (`cascadeDelete: true`,
  `db/migrations/1747061257_created_activitypub_actors.go:230`) to delete the actor.
  Deleting the actor in turn cascades (all `cascadeDelete: true`) to: `trails.author`
  (`db/migrations/1747061260_trails_add_new_author.go:17`), `lists.author`
  (`db/migrations/1747674856_updated_lists.go:17`), `summit_logs.author`
  (`db/migrations/1747061264_summit_logs_add_new_author.go:17`), `waypoints.author`
  (`db/migrations/1775997520_updated_waypoints.go:17`), `follows.follower`/`.followee`
  (`db/migrations/1747061270_follows_add_new_f_f.go:69,86`),
  `notifications.recipient`/`.author`
  (`db/migrations/1747995473_updated_notifications.go:23,40`), and
  `trail_share.actor`/`list_share.actor`/`trail_like.actor`/`feed.actor`/`.author`
  (`db/migrations/1749553104_updated_trail_share.go:17`,
  `1749566277_updated_list_share.go:27`, `1749717023_created_trail_like.go:44`,
  `1752231390_created_feed.go:31,44`). Every one of these is a clean cascade delete.
  The single exception is `comments.author`
  (`db/migrations/1747061262_comments_add_new_author.go:17`), which is
  `"cascadeDelete": false` **and** `"required": true`. A full sweep of every
  migration referencing `pbc_1295301207` (the `activitypub_actors` collection id)
  turned up no other `cascadeDelete: false` + `required: true` combination.
- (b) **Why that one field blocks the whole deletion.** PocketBase's cascade
  implementation (`deleteRefRecords`, vendored at
  `~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/core/record_model.go:1600-1610`
  relative to `$HOME`) checks, per referencing record: if `CascadeDelete` is set,
  delete the reference and continue; otherwise, if the field is `Required` and would
  become empty, **return an error instead of saving** — it never silently empties a
  required relation. PocketBase's own test suite exercises exactly this path
  (`core/record_model_test.go:2244-2249`, "delete existing record while being part of
  a non-cascade required relation", asserting `app.Delete` returns an error), and the
  HTTP layer surfaces it as a 400 with "Failed to delete record. Make sure that the
  record is not part of a required relation reference." (`apis/record_crud.go:594`).
  All of the cascade steps in (a) run inside one database transaction
  (`core/record_model.go:1487-1497`; nested `RunInTransaction` calls reuse the same
  `*dbx.Tx` per `core/db_tx.go:26-29`), so the moment PocketBase reaches a surviving
  comment authored by the actor being deleted, that error rolls back the entire
  transaction: the `users` record, the actor, and everything else are **not**
  deleted either. The account-deletion entry point,
  `web/src/routes/api/v1/user/[id]/+server.ts:131`
  (`const r = await remove(event, Collection.users)` →
  `web/src/lib/util/api_util.ts:152-159`, a bare `pb.collection('users').delete(id)`),
  has no special handling for this. The frontend caller,
  `deleteAccount()` in `web/src/routes/settings/account/+page.svelte:50-54`, has no
  `try`/`catch` around the delete call, so the thrown `APIError`
  (`web/src/lib/stores/user_store.ts:141-151`) prevents the subsequent `logout()`/
  `goto("/")` from running, with no error surfaced to the user.
- (c) **What the surviving snapshots embed** (this part of the original finding was
  correct and is unaffected by the correction): `ObjectFromTrail`
  (`db/util/activitypub.go:503`) builds `activityContent` from
  `trail.GetString("description")` directly, and `db/util/activitypub.go:509-514`
  sets `trailObject.Location` to a `pub.Place` carrying `trail.GetString("location")`,
  `trail.GetFloat("lat")` and `trail.GetFloat("lon")` verbatim.
- (d) **Yes, readable without authentication** (also unaffected by the correction).
  `activitypub_activities` has `"listRule": ""` and `"viewRule": ""`
  (`db/migrations/1747061258_created_activitypub_activities.go:13-14`) — in
  PocketBase, an empty-string rule (as opposed to `null`) means the collection is
  readable by anyone, no auth required. The read path is
  `web/src/routes/api/v1/activitypub/activity/[id]/+server.ts` (`GET`), which calls
  `show<Activity>(event, Collection.activitypub_activities)`
  (`web/src/lib/util/api_util.ts:74-85`) — `show()` performs a plain
  `event.locals.pb.collection(...).getOne(...)` with no auth check of its own, relying
  entirely on the collection's `viewRule`. `actor` on this collection is a plain
  `url` text field, not a relation
  (`db/migrations/1747061258_created_activitypub_activities.go:90-100`), so no
  cascade rule — correctly configured or not — could ever reach these rows anyway.
- (e) **When deletion succeeds (no surviving comments), does it federate a Delete for
  the deleted content?** Yes, for currently-public content. Cascade-deleting a trail
  via `app.Delete()` runs the record through the same hook chain as any other delete
  (`core/record_model.go:1490-1494`: `e.Next()` — which fires the collection's
  registered `OnRecordAfterDeleteSuccess` hook — runs *before* the cascade recurses
  further), so `DeleteTrailHandler` fires normally and calls
  `federation.CreateTrailDeleteActivity`, which is still gated on
  `r.GetBool("public")` (`db/federation/delete.go:17`, see Q2). No separate Delete
  activity is federated for the *actor* itself — no code path constructs one.

**Net verdict:** the reviewer's original point 3 (residual ActivityPub snapshots
survive with description/location data, readable without auth) stands and is
confirmed at (c)/(d). But the account-deletion mechanism itself is not "does
nothing" — it is "works correctly unless the user has ever left a comment that still
exists, in which case it fails completely and silently." Both are real, both are
documented in `260906-fmf-deferred-items.md`.

## Q4 — What are the real transport and storage safeguards? (REV-04)

- **Transport:** wanderer ships no TLS terminator. The shipped `docker-compose.yml`
  sets `ORIGIN: http://localhost:3000` in plain HTTP for both the `db` and `web`
  services (`docker-compose.yml`, `db.environment.ORIGIN` and `web.environment.ORIGIN`
  keys). HTTPS is entirely the operator's reverse-proxy responsibility; wanderer's own
  code does not enforce or add it. No code path found that forces the client to
  upgrade scheme — the app connects to whatever scheme the configured instance URL
  uses.
- **Federation authenticity:** outbound activities are signed with HTTP Signatures —
  `httpsig.NewSigner(...)` appears in `db/federation/actor.go:316`,
  `db/federation/actor.go:405` and `db/federation/activity.go:110`. Inbound activities
  are verified: `db/routes/activitypub.go:106` calls
  `util.VerifySignature(e.App, e.Request, actor.GetString("public_key"))`, implemented
  at `db/util/activitypub.go:709-749` using `httpsig.NewVerifier` and RSA-SHA256.
  **Signing authenticates the sender; it does not encrypt the activity body.** The
  content of a signed activity is still sent as plaintext JSON over whatever
  transport is in use (see Transport above).
- **Passwords:** no custom password-hashing code was found anywhere under `db/`.
  Auth flows (`db/routes/auth_token.go`, and the `web/src/routes/api/v1/auth/*`
  routes) delegate entirely to PocketBase's built-in `users` auth collection, which
  hashes credentials internally (PocketBase never exposes or requires plaintext
  password handling in application code here). No evidence of a custom/weaker scheme.
- **Files:** confirmed — no file field anywhere in `db/migrations/` sets
  `"protected": true` (repo-wide grep for the exact string returns zero matches), and
  no `OnFileDownloadRequest` hook or equivalent access-control wrapper exists in
  `db/main.go` or `db/hooks/`. Photos, GPX files and avatars are served by direct URL
  (e.g. `db/util/activitypub.go:481`, `:486` construct
  `origin/api/v1/files/trails/{id}/{filename}`). An unguessable filename is the only
  practical barrier; anyone holding the link can fetch the file.
- **At rest:** no encryption-at-rest of user content was found — application-level,
  PocketBase-level, or in the shipped compose file. The compose file does set
  `POCKETBASE_ENCRYPTION_KEY` (`docker-compose.yml`, `db.environment` block), but this
  is PocketBase's own secret-settings encryption key (it encrypts specific internal
  settings values such as SMTP/OAuth secrets stored in PocketBase's settings table) —
  it does **not** encrypt trails, comments, photos, GPX files or the SQLite database
  contents. **None found for user content. Do not describe this as encryption at
  rest in the policy.**
- **Mobile (`feature/app`, read via `git show`):**
  - `network_security_config.xml` (`git show feature/app:app/android/app/src/main/res/xml/network_security_config.xml`,
    lines 6-9) permits cleartext **only** for `127.0.0.1` (the in-app loopback tile
    proxy) — no app-wide cleartext exception exists. The manifest references this
    config explicitly
    (`git show feature/app:app/android/app/src/main/AndroidManifest.xml`, line 40:
    `android:networkSecurityConfig="@xml/network_security_config"`), and no
    `usesCleartextTraffic="true"` attribute was found on the manifest's
    `<application>` tag. So: cleartext is blocked by default except for the local
    loopback proxy — a real, verifiable safeguard, confirmed.
  - Session storage: no `flutter_secure_storage` or equivalent OS-keychain dependency
    appears in `git show feature/app:app/pubspec.yaml` (grep for
    secure_storage/crypto/keychain-related packages returns nothing beyond
    `cookie_jar: ^4.0.9`). The session itself is a cookie persisted via
    `PersistCookieJar` with plain `FileStorage`
    (`git show feature/app:app/lib/main.dart`, lines 56-57:
    `final jar = PersistCookieJar(storage: FileStorage(cookiePath), ...)`) inside the
    app's document directory. This is inside the app's OS sandbox (as section 3
    already states) but is **not** OS-secure storage (Keychain/Keystore) — it is a
    plain file, readable by anything with access to the app's own sandboxed storage
    (i.e., not readable by other apps under normal OS sandboxing, but not
    additionally hardware-encrypted either).

## Discrepancies with the reviewer

1. **Account deletion is more narrowly broken than either the reviewer or an earlier
   draft of this note suggested.** The reviewer's comment characterized the gap as
   "existing ActivityPub snapshots ... are not cleaned up." An earlier draft of this
   finding overcorrected to "account deletion cascades to nothing but the login
   record" — that was a misread of a `cascadeDelete` flag (see the correction note
   at the top of Q3) and has been fixed. The verified reality is narrower than that
   overcorrection but still a real defect: account deletion correctly cascades
   through the actor to trails, lists, summit logs and everything else *except*
   comments, and if the user has any comment that still exists, the required-relation
   constraint on `comments.author` causes the **entire deletion to fail and roll
   back** — not "comments survive," but "nothing is deleted, silently." The
   reviewer's original point about ActivityPub snapshots surviving (Q3(c)/(d)) stands
   independently of this and is confirmed. The policy text and the deferred item
   reflect this corrected, code-verified picture.
2. **The "unpublish is not deletion" trap is trail-specific, not universal.** The
   reviewer's comment discusses trails and lists together ("Changing a public trail
   or list to private..."). Verification found unpublishing suppresses federation
   for both (Q2(a)), but the *silent-delete-after-unpublish* trap (Q2(b)) is unique to
   trails — `CreateListDeleteActivity` and `CreateSummitLogDeleteActivity` have no
   equivalent public-status gate and will still send a Delete when the record is
   actually deleted, regardless of its public flag at that moment. The policy text
   below is scoped to trails for the specific "delete while it's still public if you
   want the deletion to travel" advice, to avoid asserting a universal behavior the
   code doesn't implement for lists/summit logs.

## Deferred

Two real defects were found and are **not fixed** per scope (this task documents
reality; it does not change it). Both are recorded in `260906-fmf-deferred-items.md`:

1. Account deletion fails outright — deleting nothing at all, including the login
   record — if the user has any comment that still exists, because
   `comments.author` is `required: true` with `cascadeDelete: false` while every
   other relation into `activitypub_actors` cascades cleanly (Q3(a)/(b)). When no
   comment blocks it, deletion correctly removes the login, the actor, and the
   user's trails/lists/summit logs/etc.
2. The `activitypub_activities` collection is readable by anyone without
   authentication via `GET /api/v1/activitypub/activity/{id}`, and these rows are
   never cleaned up by account or content deletion regardless of whether that
   deletion succeeds, since `actor` is a plain text field, not a relation (Q3(c)/(d)).

Both are reflected honestly in the rewritten section 12 of `privacy.astro`, worded to
match exactly what the code does — no more, no less.
