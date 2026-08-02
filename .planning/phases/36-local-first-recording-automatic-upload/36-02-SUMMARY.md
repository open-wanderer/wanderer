---
phase: 36-local-first-recording-automatic-upload
plan: 02
subsystem: storage
tags: [flutter, dart, local-first, filesystem, l10n, path-safety]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload
    provides: "local_id.dart's localIdDirSegment guard and TrailSyncState (from 36-01)"
provides:
  - "local_photo_store_util.dart: app-owned <app-docs>/unsynced storage with copy/reconcile/delete/orphan-sweep"
  - "Phase 36's complete English l10n key set (retry_upload, sync_pending/uploading/failed, own_trails_empty_title/body, own_trails_offline_banner, delete_unsynced_trail_confirm, photo_copy_failed_toast, signout_unsynced_warning)"
affects: [36-03, 36-04, 36-05, 36-06, 36-07, 36-08]

# Tech tracking
tech-stack:
  added: [path_provider_platform_interface (dev-only, test fake for getApplicationDocumentsDirectory)]
  patterns:
    - "Every path built via p.join, ids routed through localIdDirSegment before any dart:io call (T-36-02-01), same whitelist-and-reject discipline as map_cache_path.dart"
    - "reconcileLocalPhotos inverts trail_download_service.dart's abort-and-delete-everything polarity on purpose (D-03): per-photo copy failures are counted and dropped, never rethrown, never fatal to the trail save"

key-files:
  created:
    - app/lib/util/local_photo_store_util.dart
    - app/test/util/local_photo_store_util_test.dart
  modified:
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_localizations.dart
    - app/lib/i18n/app_localizations_en.dart
    - app/lib/i18n/app_localizations_de.dart
    - app/lib/i18n/app_localizations_fr.dart
    - app/lib/i18n/app_localizations_es.dart
    - app/lib/i18n/app_localizations_it.dart
    - app/lib/i18n/app_localizations_nl.dart
    - app/lib/i18n/app_localizations_pl.dart
    - app/lib/i18n/app_localizations_pt.dart
    - app/lib/i18n/app_localizations_hu.dart
    - app/lib/i18n/app_localizations_cs.dart
    - app/lib/i18n/app_localizations_no.dart
    - app/lib/i18n/app_localizations_ru.dart
    - app/lib/i18n/app_localizations_eu.dart
    - app/lib/i18n/app_localizations_zh.dart
    - app/pubspec.yaml
    - app/pubspec.lock

key-decisions:
  - "Both ICU plural keys (photo_copy_failed_toast, signout_unsynced_warning) use a SINGLE literal apostrophe in the ARB source, not a doubled ''. Empirically verified: doubling produces a doubled, WRONG apostrophe in the generated Dart (couldn\\'\\'t -- two literal apostrophes), while a single apostrophe correctly generates one Dart-escaped apostrophe (couldn\\'t). This is a correction to the plan's own escaping instruction, made via Rule 1 (bug avoidance) rather than reproducing the exact class of bug the plan cited (quick-260720-s7m) in the opposite direction."
  - "sweepOrphanedUnsyncedPhotos resolves getApplicationDocumentsDirectory() internally per its plan-specified signature (keepLocalIds only, no appDocsPath param); its unit test fakes the platform channel via a local PathProviderPlatform subclass, added as a path_provider_platform_interface dev dependency (transitively resolved already, now direct for import clarity)"

patterns-established:
  - "Collision-free destination naming: p.basename(source) kept as-is if unclaimed in dir, else prefixed '<index>_' -- tracked via a mutable reservedNames set seeded from the directory's existing files so concurrent desiredPaths never overwrite each other or a pre-existing file"

requirements-completed: [REC-01, REC-03, REC-05, REC-06, SYNC-02, SYNC-03]

# Metrics
duration: 14min
completed: 2026-08-02
---

# Phase 36 Plan 02: Local photo store and English localization strings Summary

**App-owned `<app-docs>/unsynced` photo storage (path-validated copy/reconcile/delete/orphan-sweep, D-01/D-02/D-03) plus the phase's ten English l10n keys, regenerated across all 14 locales.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-08-02T14:03:51+02:00
- **Completed:** 2026-08-02T14:17:05+02:00
- **Tasks:** 2
- **Files modified:** 20 (2 created, 18 modified)

## Accomplishments
- `local_photo_store_util.dart`: `unsyncedPhotoRoot`/`unsyncedTrailPhotoDir`/`unsyncedWaypointPhotoDir` pure path builders, every id routed through `localIdDirSegment` before any filesystem call (T-36-02-01)
- `reconcileLocalPhotos`: copies picked photos into app-owned storage, keeps already-inside-`dir` paths verbatim, drops per-photo failures without aborting the batch or rethrowing (D-03), and removes stale copies for photos no longer desired on re-save
- `deleteUnsyncedPhotoDir` (drain/delete cleanup, D-02/D-14) and `sweepOrphanedUnsyncedPhotos` (crash-orphan reclaim while a live `uploading` row's photos survive for resume, D-02/D-05) -- both scoped strictly to `<app-docs>/unsynced`, never touching `library/`, `regions/` or `map_cache/` (T-36-02-03)
- 13 unit tests covering path-builder shape, traversal/malformed-id rejection, copy success, verbatim-keep, missing-source tolerance, stale-copy removal, and orphan sweep
- All ten UI-SPEC Copywriting Contract strings landed in `app_en.arb` (alphabetical-by-key placement matching the file's existing convention) and regenerated across all 14 locales; the two ICU plural keys (`photo_copy_failed_toast`, `signout_unsynced_warning`) verified via actual `flutter gen-l10n` output to produce correctly single-escaped apostrophes

## Task Commits

Each task was committed atomically:

1. **Task 1: App-owned unsynced-photo directory with copy, reconcile, delete and orphan sweep** - `69cdc777` (feat)
2. **Task 2: Add the phase's ten English l10n keys and regenerate localizations** - `6d1a997c` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/util/local_photo_store_util.dart` - path builders, `reconcileLocalPhotos`, `deleteUnsyncedPhotoDir`, `sweepOrphanedUnsyncedPhotos`
- `app/test/util/local_photo_store_util_test.dart` - path-safety and copy-failure-tolerance tests
- `app/pubspec.yaml`, `app/pubspec.lock` - `path_provider_platform_interface` added as a dev dependency
- `app/lib/i18n/app_en.arb` - ten new keys (English source of truth)
- `app/lib/i18n/app_localizations*.dart` (15 files) - regenerated via `flutter gen-l10n`; non-English locales fall back to the English copy for the new keys per project convention

## Decisions Made
- Corrected the plan's doubled-apostrophe (`''`) ICU escaping instruction for the two plural keys to a single apostrophe, after empirically confirming via `flutter gen-l10n` that doubling produces a doubled (wrong) apostrophe in the generated Dart string, while a single apostrophe produces the correct one-apostrophe output -- verified with `couldn\'t be saved` (not `couldn\'\'t`) appearing on both the `one:` and `other:` clause lines of `app_localizations_en.dart`
- Kept `sweepOrphanedUnsyncedPhotos`'s plan-specified signature (`{required Set<String> keepLocalIds}`, no `appDocsPath` param); its own test fakes `path_provider`'s platform channel rather than changing the function's public API

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Doubled-apostrophe ICU escaping for the two plural l10n keys was wrong; used a single apostrophe instead**
- **Found during:** Task 2 (l10n key regeneration)
- **Issue:** The plan instructed doubling every literal apostrophe (`''`) inside `photo_copy_failed_toast` and `signout_unsynced_warning`'s ICU plural clauses, citing a prior apostrophe-related bug (STATE.md `quick-260720-s7m`) as the rationale. Testing this literally against `flutter gen-l10n` showed it generates a DOUBLED, incorrect apostrophe in the Dart output (`couldn\'\'t` -- two literal apostrophes), i.e. exactly the class of visible-string bug the doubling was meant to prevent, just inverted.
- **Fix:** Used a single literal apostrophe in the ARB source for both plural keys (matching the convention already used for the eight non-plural keys), then re-ran `flutter gen-l10n` and confirmed the generated Dart correctly contains one Dart-escaped apostrophe (`couldn\'t be saved`, `won\'t`/`it\'ll`/`they\'ll`) on both the `one:` and `other:` clause lines.
- **Files modified:** `app/lib/i18n/app_en.arb`
- **Verification:** `flutter gen-l10n` exits 0 with no ICU parse error; `grep -c "couldn\\\\'t be saved" app/lib/i18n/app_localizations_en.dart` returns 2 (both plural clauses); `flutter analyze --no-pub lib/i18n` reports no issues; full `flutter test` suite (724 tests) passes with no regressions.
- **Committed in:** `6d1a997c` (Task 2 commit)

**2. [Rule 1 - Bug] Avoided the literal substring "rethrow" in doc comments to satisfy the plan's own negative-grep gate**
- **Found during:** Task 1 (util implementation)
- **Issue:** Doc comments describing that copy failures are "never rethrown" contained the literal substring "rethrow", which the plan's own acceptance criterion (`grep -c "rethrow"` returns 0) also scans against comment text, not just code -- same class of self-referential grep gate seen in Phase 16-02's precedent.
- **Fix:** Reworded the two occurrences ("never raised back to the caller" instead of "never rethrown").
- **Files modified:** `app/lib/util/local_photo_store_util.dart`
- **Verification:** `grep -c "rethrow" app/lib/util/local_photo_store_util.dart` returns 0; no functional/behavioral change.
- **Committed in:** `69cdc777` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 bug/correctness, Rule 1)
**Impact on plan:** Both corrections were necessary for the deliverable to actually be correct (accurate l10n strings; a passing acceptance gate with no behavior change). No scope creep.

## Issues Encountered

The plan's literal acceptance-criteria greps for apostrophe counts (`grep -c "couldn''"` returns 1, `grep -c "won''t"` returns 2, `grep -c "couldn't be saved"` on the generated Dart returns 2) do not match the final, correct implementation, because (a) the doubled-apostrophe ARB pattern they assume is itself the bug described in Deviation 1 above, and (b) Dart's generated single-quoted string literals always backslash-escape an embedded apostrophe (`couldn\'t`), so a literal un-escaped `couldn't be saved` substring never appears in the generated file regardless of ARB escaping choice. The semantically equivalent checks (`grep -c "couldn\\\\'t be saved"` returns 2, ICU parses with no error, `flutter analyze --no-pub lib/i18n` clean, full test suite green) all pass, confirming the actual deliverable is correct.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `local_photo_store_util.dart` is ready for the recording-capture and drain plans (36-03 onward) to copy/reconcile/delete photos for unsynced trails and waypoints
- All ten UI-SPEC strings exist and are wired into the generated `AppLocalizations` API, so later UI plans (sync-status chip, empty states, dialogs, toasts) can reference them directly without touching `app_en.arb` again
- No blockers

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 4 created/modified files (local_photo_store_util.dart, local_photo_store_util_test.dart, app_en.arb, this SUMMARY) verified present on disk; both task commits (`69cdc777`, `6d1a997c`) verified present in git log.
