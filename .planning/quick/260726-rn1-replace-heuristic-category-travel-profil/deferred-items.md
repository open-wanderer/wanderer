# Deferred Items — 260726-rn1

Out-of-scope discoveries found while executing this task. **Not fixed** per the
executor's scope boundary (only issues directly caused by this task's changes
are auto-fixed).

## 1. `test/components/route_planner/settings_tab_test.dart` — 4 pre-existing failures

**Status:** Pre-existing at `HEAD` (`4b192622`), NOT caused by this task.

**Symptom:**
```
_TypeError: Null check operator used on a null value
  at SettingsTab.build (lib/components/route_planner/settings_tab.dart:32:46)
```

**Cause:** `settings_tab.dart` calls `AppLocalizations.of(context)!`, but the test
harness pumps a bare `MaterialApp` with no `localizationsDelegates` /
`supportedLocales`, so `AppLocalizations.of(context)` returns `null` and the `!`
throws before the widget renders anything.

**Verified pre-existing:**
- `git show HEAD:app/lib/components/route_planner/settings_tab.dart` already had
  `AppLocalizations.of(context)!` (at line 31 — this task's import shifted it to 32).
- `git show HEAD:app/test/components/route_planner/settings_tab_test.dart` contains
  zero occurrences of `localizationsDelegates`.
- The throw occurs at line 32, before any line this task touched in that file
  (the `bucketIcon` call is at ~line 58).

**Failing tests:**
- `SettingsTab renders exactly 5 bucket options and an auto-routing switch`
- `tapping the 'Biking / Road' card calls switchProfile('bicycle', roadOpts)`
- `the option matching the current state is visually marked selected`
- `toggling the auto-routing switch flips autoRoutingEnabled`

**Suggested fix (separate task):** add `AppLocalizations.localizationsDelegates`
and `AppLocalizations.supportedLocales` to the test's `MaterialApp`, in all 4 tests.
Note that once that is done, the harness will also need a
`subcategoryProvider.overrideWith(...)` (mirroring the fix already applied to
`travel_profile_sheet_test.dart` in this task), because `SettingsTab` now watches it
for bucket-icon resolution — it currently never reaches that line because of the
l10n throw.

## 2. `lib/util/icon_util.dart` — 37 `deprecated_member_use` infos

**Status:** Pre-existing, unrelated to this task.

`flutter analyze` reports 37 info-level `deprecated_member_use` diagnostics, all in
`lib/util/icon_util.dart`, for renamed Font Awesome icon constants
(`lastfmSquare` → `squareLastfm`, `twitterSquare` → `squareTwitter`, etc.).
Info-level only; `flutter analyze` still exits 0. Untouched by this task.
