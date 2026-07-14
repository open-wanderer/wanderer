---
phase: quick-260714-qma
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/provider/auth_provider.dart
  - app/lib/entities/local_settings_entity.dart
  - app/lib/objectbox.g.dart
  - app/lib/objectbox-model.json
autonomous: true
requirements: [QUICK-260714-qma]
---

<objective>
Revert the speculative persisted consecutive-500-counter self-heal logic added in commits `d9531951` and `dd56c62d`, while keeping the timeout-gated `build()` await/offline-fallback logic (the legitimate, independent race-condition fix). This logic was built on an unconfirmed thundering-herd/SQLite-contention theory now superseded by evidence of an unrelated real root cause (a Meilisearch token expiry bug, fixed separately). Full rationale in /Users/christianbeutel/.claude/plans/sometimes-when-i-open-tidy-reef.md ("Part 1" revision section).
</objective>

<context>
@app/lib/provider/auth_provider.dart
@app/lib/entities/local_settings_entity.dart
</context>

<tasks>

<task type="auto">
  <name>Task 1: Strip the 500-counter self-heal logic from auth_provider.dart</name>
  <files>app/lib/provider/auth_provider.dart</files>
  <action>
In build()'s catch block, remove the `_isServerError(err)` branch entirely so a 5xx (and any other non-auth error) falls through to the final "any other error -> return savedUserEntity" fallback branch. Keep the `_isAuthError` branch (401/403/404 -> deferred logout via Future.microtask) and the `on TimeoutException` branch unchanged.

Remove these members entirely:
- `static const _maxConsecutiveValidationFailures = 3;`
- the `_localSettingsBox` getter
- `bool _isServerError(Object err)`
- `int _bumpConsecutiveValidationFailures()`
- `void _resetConsecutiveValidationFailures()`

Remove the `_resetConsecutiveValidationFailures();` call inside `_updateUserEntity()` (immediately after `_box.put(userEntity);`).

Remove `import 'package:wanderer/entities/local_settings_entity.dart';` from the top of the file (it becomes unused after the above removals).

Keep everything else in build() unchanged: `final validation = _updateUserEntity(savedUserEntity.id);`, the `unawaited(validation.catchError((_) => null));` error sink, `await validation.timeout(const Duration(seconds: 3))`, the success branch, and the TimeoutException branch.
  </action>
  <verify>
    <automated>cd app && flutter analyze lib/provider/auth_provider.dart</automated>
  </verify>
  <done>auth_provider.dart no longer references consecutiveAuthValidationFailures, _isServerError, or LocalSettingsEntity; flutter analyze reports no errors; build()'s timeout-gated await and 401/403/404 logout behavior are unchanged.</done>
</task>

<task type="auto">
  <name>Task 2: Revert the persisted counter field and regenerate ObjectBox bindings</name>
  <files>app/lib/entities/local_settings_entity.dart, app/lib/objectbox.g.dart, app/lib/objectbox-model.json</files>
  <action>
Remove the `int consecutiveAuthValidationFailures` field (and its constructor wiring) from `LocalSettingsEntity`, added in commit `d9531951`. Do not touch the existing `themeMode` field or the entity's id strategy.

Then regenerate ObjectBox bindings from the `app/` directory:
`dart run build_runner build --delete-conflicting-outputs`

This updates `objectbox.g.dart` and `objectbox-model.json` to drop the removed property, keeping generated code in sync with the entity. Do not hand-edit either generated file.
  </action>
  <verify>
    <automated>cd app && dart run build_runner build --delete-conflicting-outputs && ! grep -q consecutiveAuthValidationFailures lib/objectbox.g.dart && ! grep -q consecutiveAuthValidationFailures lib/objectbox-model.json && flutter analyze lib/entities/local_settings_entity.dart lib/provider/local_settings_provider.dart</automated>
  </verify>
  <done>LocalSettingsEntity no longer has consecutiveAuthValidationFailures; regenerated objectbox.g.dart and objectbox-model.json have no references to it; themeMode field and get-or-create usage in local_settings_provider.dart are untouched; flutter analyze reports no errors.</done>
</task>

</tasks>

<verification>
- `cd app && flutter analyze lib/provider/auth_provider.dart lib/entities/local_settings_entity.dart lib/provider/local_settings_provider.dart` reports no errors.
- `grep -c consecutiveAuthValidationFailures app/lib/provider/auth_provider.dart app/lib/entities/local_settings_entity.dart app/lib/objectbox.g.dart app/lib/objectbox-model.json` all return 0.
- `build()` still awaits `_updateUserEntity(...).timeout(const Duration(seconds: 3))` with the offline-fallback TimeoutException branch and the 401/403/404 deferred-logout branch intact.
- `register`/`login`/`refresh`/`logout` public signatures unchanged.
</verification>

<success_criteria>
- All traces of the persisted 500-counter self-heal logic (code + schema field) are removed.
- Timeout-gated build() validation and 401/403/404 logout behavior are fully preserved.
- ObjectBox generated bindings are back in sync with the entity definitions.
</success_criteria>

<output>
Create `.planning/quick/260714-qma-revert-speculative-persisted-500-counter/260714-qma-SUMMARY.md` when done.
</output>