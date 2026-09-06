# Quick Task 260906-fmf: Harden the privacy policy Summary

Hardened `docs/src/pages/privacy.astro` against reviewer feedback on PR #1203, after
verifying every claim in `db/` and the `feature/app` tree. A first-pass Q3 finding
was itself factually wrong (caught at the checkpoint) and has been corrected and
re-verified against PocketBase's own cascade-delete implementation and test suite —
see "Correction" below. No claim in the final rewritten text goes beyond what the
code actually implements.

## Verified findings (Q1-Q4)

Full evidence with 43 `file:line` citations is in
`.planning/quick/260906-fmf-harden-the-privacy-policy/260906-fmf-findings.md`.

- **Q1 (REV-01):** Confirmed. `CreateCommentActivity` (`db/federation/create.go:100`)
  has no public-status gate, unlike the trail/list/summit-log create paths. Comments
  are sent to `@mentioned` remote actors plus the trail author's inbox
  unconditionally, carrying the full comment text (`db/util/activitypub.go:667`) —
  regardless of whether the parent trail is public. Only the trail's IRI rides along,
  not its content.
- **Q2 (REV-02):** Confirmed, but trail-specific. Unpublishing suppresses onward
  federation for trails, lists and (via the parent trail) summit logs alike
  (`db/federation/create.go:22,203,350`). The "delete becomes silent after unpublish"
  trap is unique to trails: `CreateTrailDeleteActivity` (`db/federation/delete.go:17`)
  gates on `public`, but `CreateListDeleteActivity` and `CreateSummitLogDeleteActivity`
  (`db/federation/delete.go:209`, `:138`) do not — they always send a Delete on
  actual deletion.
- **Q3 (REV-03), corrected:** Account deletion cascades correctly through the actor
  to trails, lists, summit logs, waypoints, follows, notifications, shares, likes and
  feed entries (all `cascadeDelete: true` — verified relation by relation). It fails
  outright — deleting nothing at all, including the login record — only if the user
  has a comment that still exists, because `comments.author` is `required: true`
  with `cascadeDelete: false`
  (`db/migrations/1747061262_comments_add_new_author.go:17`), and PocketBase's own
  cascade logic refuses to empty a required relation, rolling back the whole
  transaction instead (verified against
  `~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/core/record_model.go:1600-1610`
  and its test at `core/record_model_test.go:2244-2249`). Independently of that,
  `activitypub_activities` rows are never cleaned up (not a relation field —
  `db/migrations/1747061258_created_activitypub_activities.go:90-100`), embed
  description text and `lat`/`lon` coordinates
  (`db/util/activitypub.go:503,509-514`), and are publicly readable with no auth
  (`db/migrations/1747061258_created_activitypub_activities.go:13-14`, exposed via
  `web/src/routes/api/v1/activitypub/activity/[id]/+server.ts`).
- **Q4 (REV-04):** No TLS termination shipped (`docker-compose.yml` sets
  `ORIGIN: http://localhost:3000`); HTTP Signatures authenticate federation but do
  not encrypt (`db/routes/activitypub.go:106`, `db/federation/activity.go:110`);
  passwords are hashed by PocketBase's built-in auth; no file field is `"protected"`
  anywhere in `db/migrations/`, so photos/GPX/avatars are guarded only by an
  unguessable URL; no encryption at rest was found for content (the compose file's
  `POCKETBASE_ENCRYPTION_KEY` only protects PocketBase's internal settings secrets,
  not user content); on Android, cleartext traffic is blocked except for the
  loopback tile proxy (`feature/app:app/android/.../network_security_config.xml:6-9`);
  the mobile session is a plain-file cookie jar, not OS-secure storage
  (`feature/app:app/lib/main.dart:56-57`).

## Correction made at the checkpoint

The first draft of Q3 claimed `activitypub_actors.user` has no `cascadeDelete` and
that account deletion therefore removes nothing but the login record. That was a
misread: `"cascadeDelete": true` is present at
`db/migrations/1747061257_created_activitypub_actors.go:230`, in the same field
object as `"name": "user"` at line 236 — an earlier `grep -B 5` context window
stopped one line short and excluded it. The project coordinator caught this and
supplied citations; I independently re-verified every one of them plus the
surrounding cascade behavior against the vendored PocketBase v0.38.0 source and its
own test suite before accepting the correction, and found the true picture is more
precise than the coordinator's own proposed framing too: it is not "comments survive
with a dangling reference" but "the entire account-deletion request fails and
rolls back" whenever the user has any surviving comment, because `comments.author`
is the *only* `cascadeDelete: false` + `required: true` relation into the actor
(confirmed by sweeping every migration referencing the actor collection). Section 12
of `privacy.astro`, `260906-fmf-findings.md` (Q3, plus the discrepancies/deferred
sections), and `260906-fmf-deferred-items.md` (Defect 1) were all rewritten to match.
This correction is `git log`-visible as a second commit,
`7b34dde4 fix(docs): correct account-deletion claim in privacy policy §12`, layered
on top of the original `214ae28e`, touching only `docs/src/pages/privacy.astro`.

## Discrepancies

- The reviewer framed the account-deletion gap as "ActivityPub snapshots ... not
  cleaned up." That framing is correct on its own terms (Q3(c)/(d) confirm it) but
  incomplete: it doesn't capture the comment-blocks-everything failure mode, which
  is the more severe and more surprising defect. The rewritten policy states both.
- The reviewer discussed "a public trail or list" together for the unpublish/delete
  trap. Verification found the silent-delete-after-unpublish behavior is trail-only
  (see Q2 above) — the policy's actionable advice ("delete while still public") is
  scoped to trails specifically to avoid asserting a behavior lists/summit logs don't
  have.

## Deferred

Two real defects were found and are **documented, not fixed** (per task scope — this
task changes only `privacy.astro` and its own planning notes) in
`.planning/quick/260906-fmf-harden-the-privacy-policy/260906-fmf-deferred-items.md`:

1. Account deletion fails outright — deleting nothing, including the login record —
   if the user has any comment that still exists, due to the
   `comments.author` required/non-cascading relation. When no comment blocks it,
   deletion correctly cascades through the actor to all authored content.
2. `activitypub_activities` is readable by anyone without authentication via
   `GET /api/v1/activitypub/activity/{id}`, and these rows are never cleaned up by
   account or content deletion — successful or not — since `actor` is a plain text
   field, not a relation.

Both are stated plainly in the new section 12 of the policy.

## Section renumbering (important for PR reviewers)

A new section 9 ("Security and safeguards") was inserted after the federation
section, so every section from the old 9 onward shifted up by one:

| Old # | New # | Title |
|-------|-------|-------|
| 9 | 10 | Demo instance |
| 10 | 11 | Legal bases |
| 11 | 12 | Retention and deletion |
| 12 | 13 | Your rights |
| 13 | 14 | Children |
| 14 | 15 | Changes |
| 15 | 16 | Contact |

Sections 1-8 are unchanged in number. **If reviewer slothful-vassal's original
comment referenced "section 11" (account deletion / retention), that content is now
in section 12.** The `#federation` (section 8) and `#location` (section 5) anchors,
and both cross-reference links to `#federation`, are unchanged and still resolve
correctly.

## What changed in `privacy.astro`

- **Section 8 (Federation):** replaced "Only public content federates" with a
  narrower claim plus a new "Comments are the exception" bullet (REV-01); added an
  "Unpublishing is not deletion" bullet with the trail-specific silent-delete
  consequence (REV-02); narrowed "Deletion is a request, not a guarantee" to cover
  deletion only, removing the old conflation with unpublishing.
- **New section 9 (Security and safeguards):** in-transit, federation-signing,
  passwords, files, at-rest, on-device and in-app-disclosure sub-points, each bounded
  strictly to what Q4 verified — no encryption-at-rest claim anywhere (REV-04).
- **Section 12 (Retention and deletion, was 11):** states precisely what account
  deletion does and does not remove — correct in the common case, fails outright
  when comments exist — plus the surviving, publicly-readable activity-record
  disclosure (REV-03). Rewritten a second time after the checkpoint correction.
- `LAST_UPDATED` bumped to "6 September 2026".
- No changes to the `<style>` block, no new CSS classes, no changes outside
  `docs/src/pages/privacy.astro`.

## Design-hook note

The design-review hook flagged two pre-existing `font-size` values in the `<style>`
block (`h1` at line ~571, `.lede` at line ~579) as off the documented type ramp on
every edit in this file. These lines were not touched by this task — the plan
explicitly forbids modifying the `<style>` block — so they are left as-is and are
out of scope here.

## Self-Check: PASSED

- `.planning/quick/260906-fmf-harden-the-privacy-policy/260906-fmf-findings.md` — FOUND, Q3 corrected
- `.planning/quick/260906-fmf-harden-the-privacy-policy/260906-fmf-deferred-items.md` — FOUND, Defect 1 corrected
- `docs/src/pages/privacy.astro` — FOUND, modified as described
- Commit `214ae28e` (`fix(docs): harden privacy policy federation, safeguards and retention claims`) — FOUND in `git log`
- Commit `7b34dde4` (`fix(docs): correct account-deletion claim in privacy policy §12`) — FOUND in `git log`
- Task 1 gate (Q1-Q4 headings, discrepancies/deferred sections, ≥8 citations) — PASSED (43 citations)
- Task 2 gate (16 balanced sections, sequential `<h2>` numbers, anchors intact, REV-01/REV-02 phrasing gates, no encryption-at-rest claim, `LAST_UPDATED` bumped) — PASSED
- `git diff --stat` against the pre-task commit (`bffad96a`) touches only `docs/src/pages/privacy.astro` — CONFIRMED
