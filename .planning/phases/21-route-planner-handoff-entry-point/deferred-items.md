# Phase 21 — Deferred Items

Out-of-scope discoveries logged during plan execution, per SCOPE BOUNDARY (not fixed, not
part of this plan's `files_modified`).

## 21-02: `flutter test` failure count grew from 3 (Phase 18 baseline) to 4

**Found during:** 21-02 Task 1 verification (`flutter test`, full suite).

**Pre-existing baseline (per STATE.md Pending Todos):** 3 failures —
`feed_item_test.dart` (x2), `settings_screen_test.dart` (x1).

**New observation:** `settings_account_screen_test.dart` ("account screen renders all five
sections in fixed order (ACCT-01..05)") also fails — `find.text('Add Bio')` finds 0 widgets.

**Why out of scope for 21-02:** This plan's `files_modified` is `app/lib/models/settings.dart`
and `app/lib/entities/settings_entity.dart` only, adding a nullable `Behavior?` field. The
failing assertion is about bio-section text rendering in `SettingsAccountScreen`, unrelated to
`Behavior`/`allowAutoGeolocate`. The test fixture (`Settings(id: '1', bio: 'hello')`) does not
reference `behavior` at all. Confirmed via source read: no causal link between this plan's diff
and the bio-rendering logic under test.

**Action:** Not fixed. Flagged here for a future cleanup pass; does not block 21-02 or 21-03/04.
