---
phase: 07-privacy
reviewed: 2026-06-20T00:00:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - app/lib/routes/settings_privacy_screen.dart
  - app/test/routes/settings_privacy_screen_test.dart
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 07: Code Review Report

**Reviewed:** 2026-06-20
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Two files reviewed: the `SettingsPrivacyScreen` widget and its widget test. The screen
implements three `RadioGroup` sections (account, trails, lists) backed by `settingsProvider`
and calls `saveToServer` on each change.

No security vulnerabilities or data-loss bugs were found. Three warnings surfaced: an
unawaited future that deviates from project convention and is inconsistent with how other
callers handle the same pattern; a nullable `settings.id` that can silently produce a
malformed API URL in `saveToServer`; and a test gap where the save path is entirely
untested with no provider override for `toastProvider`. Two info items cover code
duplication (the tripled fallback `SettingsPrivacy` constructor) and a dead import.

---

## Warnings

### WR-01: Unawaited `Future` returned by `_save` — inconsistent with project convention

**File:** `app/lib/routes/settings_privacy_screen.dart:95` (also lines 139, 183)

**Issue:** `_save` returns `Future<void>` but every call site in `onChanged` discards
the future without wrapping it in `unawaited()`. Elsewhere in the project the same
pattern is handled explicitly:

```
// navigation_launch_util.dart:214
unawaited(_recacheNav(store, trail.id, response));
// navigation_screen.dart:87
unawaited(_positionSource.start());
```

Discarding the future without `unawaited()` silences the `unawaited_futures` lint (when
enabled) and makes the intent ambiguous to readers. More importantly, if `_save` ever
gains a code path that throws *before* the `try/catch` (e.g., a synchronous assert or
a refactor that moves the guard outside the try block), the unhandled rejection will be
swallowed with no diagnostic.

**Fix:** Wrap each call site with `unawaited()`, importing `dart:async`:

```dart
// at the top of the file
import 'dart:async';

// each call site (lines 95, 139, 183)
unawaited(_save(ref, l10n, settings.copyWith(privacy: updated)));
```

---

### WR-02: `settings.id` is nullable — `saveToServer` constructs `/settings/null` when `id` is absent

**File:** `app/lib/provider/settings_provider.dart:36` (called from settings_privacy_screen.dart via `_save`)

**Issue:** `Settings.id` is declared `String?` (settings.dart:96). When `saveToServer`
is called it interpolates the id directly into the URL path:

```dart
.post('/settings/${settings.id}', data: settings.toJson());
```

If `id` is `null` this produces the literal URL `/settings/null`, which the server will
return 404 for. The privacy screen passes `settings.copyWith(privacy: updated)` where
`settings` comes from `ref.watch(settingsProvider)`. In practice `id` should always be
populated after login, but there is no guard, so a race condition (user changes a radio
before `auth_provider` finishes calling `updateFromServer`) or a freshly created account
can hit this silently — the user sees an error toast with no indication of the root cause.

**Fix:** Guard in `saveToServer` before the request:

```dart
Future<void> saveToServer(Settings settings) async {
  final id = settings.id;
  if (id == null || id.isEmpty) {
    throw StateError('Cannot save settings: id is null');
  }
  final response = await ref
      .read(apiProvider)
      .post('/settings/$id', data: settings.toJson());
  final updated = Settings.fromJson(response.data as Map<String, dynamic>);
  await updateFromServer(updated);
}
```

The privacy screen's `_save` already catches all exceptions and surfaces a toast, so the
`StateError` will be caught and shown to the user rather than silently misfiring.

---

### WR-03: Test does not override `toastProvider` — save path untested and would crash

**File:** `app/test/routes/settings_privacy_screen_test.dart:21`

**Issue:** Both test cases override `settingsProvider` with `overrideWithValue(...)`,
which replaces the notifier with a plain sync value. If any `onChanged` callback were
triggered, the code would call `ref.read(settingsProvider.notifier).saveToServer(...)`,
which throws because the overridden provider has no notifier instance. Additionally,
the null-settings guard on the `onChanged` paths calls `ref.read(toastProvider.notifier).add(...)`,
which is also not overridden — if triggered in a test this would attempt to resolve
`objectBoxProvider` transitively and crash.

The test comment on line 64–66 acknowledges the save path is untested ("TODO: tap-to-save
assertion requires an apiProvider/HTTP override fixture"). However there is no test that
exercises the null-settings guard branch (showing the error toast when `settings == null`)
even though that code path is live.

**Fix (minimum):** Add a test that overrides `settingsProvider.overrideWithValue(null)`
and taps a radio tile, asserting a `ToastMessage` with `ToastType.error` appears. This
requires overriding `toastProvider` and reading it back:

```dart
settingsProvider.overrideWithValue(null),
toastProvider.overrideWith(() => Toast()),
```

Then assert that after tapping any `RadioListTile` the toast state contains one error
message with `l10n.error_saving_settings`.

---

## Info

### IN-01: Tripled fallback `SettingsPrivacy` constructor — code duplication

**File:** `app/lib/routes/settings_privacy_screen.dart:89` (also 133, 177)

**Issue:** Each of the three `onChanged` callbacks contains an identical
`const SettingsPrivacy(account: 'public', trails: 'private', lists: 'private')` fallback.
These match the `groupValue` defaults on lines 72, 116, and 160, so the logic is consistent,
but any future change to a default value requires updating four locations instead of one.

**Fix:** Extract a single top-level or class-level constant:

```dart
static const _defaultPrivacy = SettingsPrivacy(
  account: 'public',
  trails: 'private',
  lists: 'private',
);
```

Replace all three inline `const SettingsPrivacy(...)` literals and the three `?? "public"`
/ `?? "private"` null-coalescings with references to `_defaultPrivacy.account`,
`_defaultPrivacy.trails`, `_defaultPrivacy.lists`.

---

### IN-02: Duplicated null-settings toast block in three `onChanged` callbacks

**File:** `app/lib/routes/settings_privacy_screen.dart:75` (also 119, 163)

**Issue:** The guard that fires a toast when `settings == null` is copy-pasted verbatim
into all three `onChanged` closures. The same behaviour could be achieved by extracting
a helper or checking once at the top of `build` (the screen could be disabled or show a
loading state when `settings` is null rather than handling the error individually in each
callback).

**Fix:** Move the null guard to a single shared helper or handle at the `build` level:

```dart
// Option A — early return in build
if (settings == null) {
  return const Scaffold(body: Center(child: CircularProgressIndicator()));
}
// settings is non-null below; onChanged can be simplified to a direct .copyWith call
```

This removes three duplicate toast blocks and makes the null state explicit in the UI.

---

_Reviewed: 2026-06-20_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
