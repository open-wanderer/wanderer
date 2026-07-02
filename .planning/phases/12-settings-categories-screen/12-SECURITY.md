---
phase: 12
slug: settings-categories-screen
status: closed
threats_open: 0
asvs_level: 1
created: 2026-07-02
---

# Security Audit — Phase 12: Settings Categories Screen

**Audited:** 2026-07-02
**ASVS Level:** 1
**Disposition:** SECURED — 8/8 threats closed
**Block-on:** high (no high-severity open threats)

This audit verifies that every declared threat mitigation in the Phase 12 threat
register is actually present in the code on disk (not merely in planning intent).
Implementation files were treated as read-only; no code was modified.

---

## Threat Verification

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-12-01 | Elevation of Privilege | mitigate | CLOSED | Client reorder/upsert payloads never send a `user` field; server injects owner from session. |
| T-12-02 | Tampering | mitigate | CLOSED | Meilisearch filter interpolates a server-fetched 15-char PB id, never free-text. |
| T-12-03 | Information Disclosure | mitigate | CLOSED | Own-trail count queries the author-scoped `/profile/@{ownHandle}/trails` with the own `@`-prefixed handle. |
| T-12-04 | Tampering | accept | CLOSED | Accepted-risk entry logged; server re-validates every posted category id + scopes writes to session user. |
| T-12-05 | Tampering | accept | CLOSED | Accepted-risk entry logged; server re-validates every posted subcategory id against the parent category + scopes writes to session user. |
| T-12-06 (info) | Information Disclosure | accept | CLOSED | Accepted-risk entry logged; view-trails push uses the own `@`-prefixed session handle to an author-scoped endpoint. |
| T-12-06 (DoS) | Denial of Service | accept→fixed | CLOSED | CR-01 type-guard fix present in `router_provider.dart`: `state.extra` guarded with `if (extra is! Category)` and falls back to `SettingsCategoriesScreen`. |
| T-12-SC | Tampering (supply chain) | accept | CLOSED | Accepted-risk entry logged; zero packages installed this phase (`tech-stack.added: []` in all four SUMMARYs). |

---

## Evidence Detail

### T-12-01 — EoP: reorder/upsert never carry a client-trusted owner (mitigate) — CLOSED
Verified at both trust-boundary sides:

- **Client (Flutter):**
  - `app/lib/provider/category_preference_provider.dart:54-60` — `reorder` posts
    `{'categories': orderedCategoryIds}` only; `upsert` (43-49) posts
    `{'category', 'visible'}` only. No `user` key.
  - `app/lib/provider/subcategory_preference_provider.dart:55-64` — `reorder`
    posts `{'category', 'subcategories'}`; `upsert` (43-49) posts
    `{'subcategory', 'visible'}`. No `user` key.
- **Server (SvelteKit → PocketBase):**
  - `web/src/routes/api/v1/user-category-preference/+server.ts:84-87` and
    `.../user-subcategory-preference/+server.ts:82-85` — the PUT handlers build
    the payload as `{ ...safeData, user: event.locals.user.id }`, injecting the
    owner from the session and overriding any client-supplied value.
  - `.../reorder/+server.ts` (both) require `event.locals.user` (401 otherwise).
- **Server (Go/PocketBase custom route):**
  - `db/routes/category_preferences.go:30,47` — reorder handlers pass
    `e.Auth.Id` (session-derived) as `userID`; the request struct carries only the
    id list, never a user id.

### T-12-02 — Tampering: Meilisearch filter id is not free-text (mitigate) — CLOSED
- `app/lib/util/own_trail_count.dart:21-25` — `field` is a hard-coded literal
  (`subcategory_id` / `category_id`); `id` is the caller-supplied PB id
  interpolated as `"$field IN ['$id']"`. Call sites pass `category.id` /
  `sub.id` from server-fetched provider data
  (`settings_categories_screen.dart:446`, `settings_subcategories_screen.dart:416`),
  never user-typed text.
- The Zod schemas constrain every id to `z.string().length(15)`
  (`category_preference_schema.ts:4,9`, `subcategory_preference_schema.ts:4,9`),
  matching the vetted PB-id shape.

### T-12-03 — Info Disclosure: own-trail count is author-scoped (mitigate) — CLOSED
- `app/lib/util/own_trail_count.dart:18-21,27` — resolves the handle as
  `'@${user.preferredUsername}'` from `authProvider` (own session) and POSTs to
  `/profile/$handle/trails`. The count only ever queries the requesting user's own
  author-scoped endpoint; returns 0 for an anonymous user (line 19).

### T-12-04 — Tampering: category reorder id list (accept) — CLOSED (accepted risk logged)
- `db/util/category_preference.go:89-159` (`ReorderUserCategoryPreferences`):
  requires a non-empty session `userID`; validates the posted list contains all
  known categories, rejects unknown ids (line 110-112) and duplicates (113-116);
  loads existing rows filtered to `user = {:user}` and writes priorities scoped to
  that session user inside a transaction. Client ids come from `categoryProvider`
  (server data). See accepted-risk entry AR-01.

### T-12-05 — Tampering: subcategory reorder id list scoped to parent (accept) — CLOSED (accepted risk logged)
- `db/util/category_preference.go:161-241` (`ReorderUserSubcategoryPreferences`):
  requires session `userID` and a non-empty `categoryID`; validates every posted
  subcategory id belongs to the posted parent category
  (`FindRecordsByFilter("subcategories", "category = {:category}"...)`, lines
  169-183, 191-193), rejects unknown/duplicate ids, and scopes preference writes to
  `user = {:user}`. Client passes `widget.category.id` +
  `subcategoryProvider` ids. See accepted-risk entry AR-02.

### T-12-06 (info) — Info Disclosure: view-trails push (accept) — CLOSED (accepted risk logged)
- `settings_categories_screen.dart:516-535,553` and
  `settings_subcategories_screen.dart:486-505,525` — resolve the handle as
  `'@$username'` from `authProvider` (own session), abort with an error toast on a
  null username (never build `'@null'`), then push `/profile/$handle/trails` (an
  author-scoped endpoint). See accepted-risk entry AR-03.

### T-12-06 (DoS) — null-extra cast crash (accept → REAL bug, fixed CR-01) — CLOSED
Per the required-reading note, this was found to be a real bug and had to be
verified as fixed on disk, not accepted:
- `app/lib/provider/router_provider.dart:200-211` — the `subcategories` route
  builder reads `state.extra` into a local, type-guards with
  `if (extra is! Category)`, and falls back to `const SettingsCategoriesScreen()`
  instead of blind-casting `state.extra as Category`. The unguarded cast the
  original plan described (Plan 04 Task 2) is NOT present; the CR-01 fix IS
  present. This mirrors the sibling `/trail/:id/navigate` guard (lines 247-252).

### T-12-SC — Supply chain (accept) — CLOSED (accepted risk logged)
- All four SUMMARYs declare `tech-stack.added: []`. No new pub packages this phase;
  all widgets are first-party Flutter Material + existing `go_router` / Font
  Awesome deps. See accepted-risk entry AR-04.

---

## Accepted Risks Log

| ID | Threat | Rationale | Compensating Control |
|----|--------|-----------|----------------------|
| AR-01 | T-12-04 (category reorder tampering) | Posted category ids are UI-controllable but originate from `categoryProvider` server data. | Server re-validates every id against the canonical `categories` table and scopes all writes to the session user (`db/util/category_preference.go:89-159`). |
| AR-02 | T-12-05 (subcategory reorder tampering) | Posted subcategory ids + parent id are UI-controllable. | Server re-validates every subcategory id belongs to the posted parent category and scopes writes to the session user (`db/util/category_preference.go:161-241`). |
| AR-03 | T-12-06 info (view-trails navigation) | Handle + filter are seeded client-side before an author-scoped push. | Handle is the own `@`-prefixed session handle from `authProvider`; destination endpoint is author-scoped server-side, so only the user's own trails are ever shown. Null-username aborts (no `'@null'`). |
| AR-04 | T-12-SC (supply chain) | No new dependency surface introduced. | Zero packages installed (`tech-stack.added: []` across all four plan SUMMARYs). |

---

## Unregistered Flags

None. No SUMMARY in this phase contains a `## Threat Flags` section, and no new
attack surface was found beyond the eight registered threats.

---

## Notes

- This phase went through post-execution live iteration (reorder-guard timing,
  AsyncValue combine, parent-visibility cascade, subcategory chips, row restyle).
  All verification above was performed against the CURRENT state of the files on
  disk, including uncommitted working-tree changes, not solely the SUMMARY text.
- The reorder SvelteKit endpoints (`.../reorder/+server.ts`) do not themselves
  perform ownership re-validation — they Zod-parse and forward to the Go/PocketBase
  custom routes (`event.locals.pb.send(...)`), where the ownership scoping and id
  re-validation actually live (`db/util/category_preference.go`). The mitigation is
  present, just one layer deeper than the SvelteKit proxy.
