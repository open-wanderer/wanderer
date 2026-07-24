---
phase: 21-route-planner-handoff-entry-point
plan: 02
subsystem: mobile-settings-model
tags: [flutter, freezed, objectbox, dart, settings]

# Dependency graph
requires:
  - phase: 21-route-planner-handoff-entry-point
    plan: 01
    provides: (no direct code dependency — parallel wave-1 plan; both feed into 21-03/21-04)
provides:
  - "Behavior freezed class + Settings.behavior field — bool? allowAutoGeolocate, int? mapClusteringMaxZoom, bool? showTrailStartMarker"
  - "SettingsEntity.behaviorJson — ObjectBox JSON-blob persistence for Behavior, mirroring privacyJson"
affects: [21-03, 21-04, route-planner-entry-point-gps-gate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Nested freezed settings type (Behavior) added following the exact SettingsPrivacy/SettingsLocation template — factory constructor, fromJson, part directives, unchanged part 'settings.freezed.dart'/'settings.g.dart' declarations"
    - "ObjectBox JSON-blob-per-field persistence (behaviorJson) mirroring privacyJson's exact null-check/encode/decode shape — not a flattened boolean column"

key-files:
  created: []
  modified:
    - app/lib/models/settings.dart
    - app/lib/entities/settings_entity.dart
    - app/lib/models/settings.freezed.dart (generated)
    - app/lib/models/settings.g.dart (generated)
    - app/lib/objectbox-model.json (generated)
    - app/lib/objectbox.g.dart (generated)
    - app/lib/provider/router_provider.g.dart (generated — hash recompute side effect, no functional change)

key-decisions:
  - "Behavior's three fields (allowAutoGeolocate, mapClusteringMaxZoom, showTrailStartMarker) are all nullable, matching web's optional/nullable Behavior type and the project's null-reads-as-not-opted-in convention (A2) — no non-null defaults baked into the freezed factory"
  - "behaviorJson uses the identical encode/decode shape as privacyJson (jsonEncode(settings.behavior!.toJson()) on fromModel; Behavior.fromJson(jsonDecode(behaviorJson!)) on toModel) rather than a flattened bool column, per Pitfall 4"

patterns-established:
  - "Any future nested Settings type follows: freezed class matching SettingsPrivacy's shape + a matching <field>Json String? on SettingsEntity, encoded/decoded identically"

requirements-completed: [HANDOFF-03]

# Metrics
duration: 6min
completed: 2026-07-17
---

# Phase 21 Plan 02: Settings.behavior (allowAutoGeolocate) Data-Layer Port Summary

**Ported the `Behavior` nested type (`allowAutoGeolocate`, `mapClusteringMaxZoom`, `showTrailStartMarker`) from web's `Settings.behavior` onto the Flutter `Settings` freezed model, with `SettingsEntity.behaviorJson` persisting it via the same JSON-blob strategy as `privacyJson` — closing the D-03 gap so Plan 03's GPS gate at planner entry has a real field to read.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-17T14:50:51Z
- **Completed:** 2026-07-17T14:56:34Z
- **Tasks:** 1
- **Files modified:** 2 hand-edited (`settings.dart`, `settings_entity.dart`) + 5 regenerated (freezed/json_serializable/objectbox codegen)

## Accomplishments

- Added `Behavior` freezed class to `settings.dart` — `bool? allowAutoGeolocate`, `int? mapClusteringMaxZoom`, `bool? showTrailStartMarker`, all nullable, mirroring `web/src/lib/models/settings.ts:55-59`'s `Behavior` type field-for-field
- Added `Behavior? behavior` field to the `Settings` freezed class, alongside `privacy`/`notifications`
- Added `String? behaviorJson` to `SettingsEntity` (field + constructor param), encoded in `fromModel` and decoded in the `toModel` extension using the exact null-check/encode/decode shape `privacyJson` already uses
- Regenerated all codegen (`dart run build_runner build --delete-conflicting-outputs`) — `settings.freezed.dart`, `settings.g.dart`, and `objectbox.g.dart`/`objectbox-model.json` (new `behaviorJson` ObjectBox property, id `14:4465420488338205867`) all picked up the new field
- `flutter analyze lib/models/settings.dart lib/entities/settings_entity.dart` reports no issues; whole-app `flutter analyze` shows zero new issues (46 pre-existing info/warning-level issues, none touching the changed files)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Behavior freezed class + Settings.behavior field + SettingsEntity.behaviorJson, then regenerate codegen (D-03)** - `4aba3904` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/lib/models/settings.dart` - Added `Behavior` freezed class + `Settings.behavior` field
- `app/lib/entities/settings_entity.dart` - Added `behaviorJson` field, `fromModel` encode, `toModel` decode
- `app/lib/models/settings.freezed.dart`, `app/lib/models/settings.g.dart` - Regenerated (freezed + json_serializable for `Behavior`/`Settings`)
- `app/lib/objectbox-model.json`, `app/lib/objectbox.g.dart` - Regenerated (new `behaviorJson` ObjectBox column)
- `app/lib/provider/router_provider.g.dart` - Regenerated hash literal only (riverpod_generator side effect of the full-project `build_runner` run; `router_provider.dart` source itself is unchanged — confirmed via `git diff`, no functional impact)

## Decisions Made

- All three `Behavior` fields nullable — no freezed default values — so an absent/unset `allowAutoGeolocate` reads as `null`, never silently resolving to `true` (privacy-safe default, matches web's `?? false` convention, A2 in RESEARCH.md)
- `behaviorJson` strictly mirrors `privacyJson`'s JSON-blob-per-field storage strategy (Pitfall 4) rather than a flattened boolean ObjectBox column

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written; both files edited per the plan's exact `<action>` spec.

### Notes on acceptance-criteria greps

One of the plan's acceptance-criteria greps (`grep -n "Behavior.fromJson(jsonDecode(behaviorJson" app/lib/entities/settings_entity.dart`) assumes a single-line decode call. `dart format` (confirmed via `dart format --output=none --set-exit-if-changed`, 0 changes) wraps the call across two lines — identical to how the existing `privacyJson`/`SettingsPrivacy.fromJson` decode is already formatted in this file. The decode logic is structurally identical to the mirrored pattern; only the literal single-line grep doesn't match post-formatting. Verified correctness instead via `flutter analyze` (clean) and the `behaviorJson` occurrence-count criterion (5 occurrences — exceeds the required minimum of 3).

## Issues Encountered

None blocking. `flutter test` (whole-suite, informational per RESEARCH's own scope note — not a hard acceptance gate for this plan) surfaced 4 pre-existing failures rather than the 3 logged in STATE.md's Pending Todos: `feed_item_test.dart` (x2, previously known), `settings_screen_test.dart` (x1, previously known), and `settings_account_screen_test.dart` (x1, newly observed — `find.text('Add Bio')` finds 0 widgets). Confirmed via source read that `settings_account_screen_test.dart`'s only `Settings(...)` reference (`Settings(id: '1', bio: 'hello')`) does not touch `behavior` — this plan's diff has no causal link to the bio-rendering assertion under test. Logged to `.planning/phases/21-route-planner-handoff-entry-point/deferred-items.md`, not fixed (out of `files_modified` scope for this plan).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `Settings.behavior?.allowAutoGeolocate` is ready for Plan 03's GPS gate at Route Planner entry (`ref.read(settingsProvider)...behavior?.allowAutoGeolocate == true` gates `foregroundPositionStreamProvider` subscription, else falls back to `mapCameraProvider`/fixed default)
- No settings-screen toggle UI was added, per D-03's explicit scope limit — the field is model-only until a future settings phase surfaces it
- No blockers for Plans 03/04

---
*Phase: 21-route-planner-handoff-entry-point*
*Completed: 2026-07-17*

## Self-Check: PASSED

- FOUND: app/lib/models/settings.dart
- FOUND: app/lib/entities/settings_entity.dart
- FOUND: .planning/phases/21-route-planner-handoff-entry-point/21-02-SUMMARY.md
- FOUND commit: 4aba3904
