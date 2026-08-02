---
phase: 35-offline-trail-creation
status: complete
executed: 2026-08-01
requirements: [OFFUI-01, OFFUI-02, OFFUI-03, OFFUI-04]
plans_executed: 0
commits: [21c4b1ee]
---

# Phase 35: Offline Trail Creation — Summary

**Executed without plans.** This phase was implemented directly rather than through
`/gsd-plan-phase` + `/gsd-execute-phase`, so there are no `35-NN-PLAN.md` / `35-NN-SUMMARY.md`
pairs and no plan-authored `<threat_model>` blocks. This file is the phase record. Written after
the fact from the diff and a review pass, not from a plan.

## Requirement coverage

| Req | Delivered | Where |
|-----|-----------|-------|
| OFFUI-01 | Map renders from downloaded regions offline instead of going blank | `trail_create_screen.dart:628`, `trail_panel.dart`, `trail_detail_map_screen.dart` |
| OFFUI-02 | Tag autocomplete degrades to no suggestions; free-form tags still reach the save payload | `tag_provider.dart` |
| OFFUI-03 | `.gpx` import works with no connection | `trail_source_select_screen.dart` (gate relaxed); conversion path from Phase 34 |
| OFFUI-04 | Non-GPX import offline explains the constraint instead of failing generically | `trail_import_util.dart:69-83` |

## Root cause of OFFUI-01, and the rename it forced

`TrailMap`'s `offline:` argument was fed `trail.isOffline` — the *downloaded-trail* flag, not a
connectivity signal. A trail merely being edited while the device happened to be offline has it
`false`, so the online basemap style was chosen and never loaded. All three `TrailMap` mount
sites now derive `offline:` from `onlineStatusProvider`. Online, network tiles are preferred
even for a downloaded trail.

`Trail.isOffline` was renamed `Trail.isLocal` and documented so the two ideas cannot be
conflated again. **It could not be deleted** — seven live consumers ask about provenance, not
connectivity: local thumbnail file vs network (`trail_card`, `trail_list_item`), the
"downloaded" and mutually-exclusive "available offline" badges (`trail_panel`), hiding the
comments/summit-log TabBar for a local-only trail (`trail_panel`), and — load-bearing — routing
delete to `trailLibraryProvider.deleteTrail` (un-download) rather than a server delete
(`trail_dropdown`). That last one would have turned "remove download" into "delete from server".

Note `isLocal` now spans two axes in this codebase: `Actor.isLocal` means federation-local
(this instance), `Trail.isLocal` means device-local. The receiver type disambiguates at every
call site.

## OFFUI-04's non-obvious dependency

`OnlineStatus` is optimistic (`build() => true`) and settles only from ordinary request traffic.
In airplane mode with no prior failed request it still reads online, so the non-GPX guard would
have missed entirely and produced the generic error. `importTrailFile` therefore refreshes the
status before the guard. Doing it there rather than in the picker also covers the OS
share-intent entry point (`main.dart:185`), which calls `importTrailFile` directly.

## Delivered alongside (not OFFUI-scoped)

- Library empty states split: "nothing downloaded" vs "nothing matches the search"
- Hardcoded source-select descriptions and three library strings moved into l10n (en + de)
- Share profile URL gained its missing `@` handle prefix — it had been pointing at a 404 — plus
  a topography-background redesign; `/profile/share` moved out of the shell to render full-screen
- Saving no longer leaves the form reporting dirty, which asked the user to discard changes
  they had just saved
- Theme tokens replaced hardcoded greys; the empty-state SVG takes the localized title as its
  semantics label

## Verification

- `flutter analyze lib/` — no new issues (only pre-existing `icon_util` deprecations)
- `flutter test` — 684 pass, 1 skip, 4 fail
- **The 4 failures are pre-existing and unrelated**: `settings_tab_test.dart`,
  `AppLocalizations.of(context)!` returning null. Confirmed failing at clean HEAD before this
  work, via a separate worktree.
- 7 tests added: OFFUI-04's guard (offline non-GPX refused specifically and before any network
  call; online non-GPX falls through to the generic error; offline GPX unaffected) and
  tag-provider error discrimination. The discriminating tag test was verified to fail with the
  fix disabled.

### On-device verification — confirmed by the implementer 2026-08-01

All four OFFUI criteria are user-observable and were exercised on device in airplane mode by the
implementer, who also wrote the feature work:

1. Downloaded trail open in create/edit, airplane mode → basemap renders from the offline
   region rather than blank.
2. Airplane mode → typing in the tag field shows no suggestions, no error toast, no thrown
   exception; a free-form tag survives to the save payload.
3. Airplane mode → `.gpx` import completes and lands on a populated create screen.
4. Airplane mode → `.kml`/`.fit` import shows the offline-specific message, not the generic one.

Recorded as implementer-confirmed rather than as an independent UAT session: unlike Phase 34,
where a separate `34-UAT.md` round found four defects, the same person built and checked this.

### Security gate

Scoped audit run — see `35-SECURITY.md`.
