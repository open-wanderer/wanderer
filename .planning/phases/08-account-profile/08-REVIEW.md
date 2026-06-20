---
phase: 08-account-profile
reviewed: 2026-06-20T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - app/lib/provider/auth_provider.dart
  - app/lib/components/settings/email_change_sheet.dart
  - app/lib/components/settings/password_change_sheet.dart
  - app/lib/routes/settings_account_screen.dart
  - app/test/provider/auth_provider_refresh_test.dart
  - app/test/routes/settings_account_screen_test.dart
  - app/pubspec.yaml
  - app/ios/Runner/Info.plist
  - app/lib/i18n/app_en.arb
findings:
  critical: 3
  warning: 4
  info: 3
  total: 10
status: issues_found
---

# Phase 08: Code Review Report

**Reviewed:** 2026-06-20T00:00:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Phase 08 adds an account management screen (avatar, bio, email, password, delete account) backed by three new files plus an updated auth provider. The architecture is sound and follows existing Riverpod + go_router patterns, but several correctness gaps were found:

- A `catchError` return-type mismatch in `auth_provider.dart` can swallow 404 errors silently rather than triggering logout.
- The password-change sheet sends a `POST /user/{id}` body that the SvelteKit API explicitly rejects for missing `oldPassword`, making the password-change flow non-functional.
- Missing `NSCameraUsageDescription` in `Info.plist` will crash iOS when the system photo picker is invoked under certain iOS/picker configurations.
- No "confirm new password" field in `PasswordChangeSheet`, allowing silent typos.
- Avatar success toast uses the generic key `l10n.avatar` ("Avatar") instead of a meaningful success string.

---

## Critical Issues

### CR-01: `catchError` return type mismatch silences 404 logout

**File:** `app/lib/provider/auth_provider.dart:39-44`

**Issue:** `_updateUserEntity()` returns `Future<UserEntity?>`. The `.catchError()` callback returns `null` (inferred as `Null`, not `UserEntity?`). In Dart's type system this is a static warning that is sometimes silently coerced, but depending on SDK version and sound null-safety mode the callback may fail to satisfy the `FutureOr<UserEntity?> Function(Object)` type, causing the error handler to be ignored entirely. When the network call fails with a 404 (deleted/banned user), `logout()` is never called and the app continues presenting the user as authenticated with stale local data.

```dart
// Current (broken handler may be silently ignored):
_updateUserEntity(savedUserEntity.id).catchError((err) {
  if (err is DioException && err.response?.statusCode == 404) {
    logout();
  }
  return null;          // type Null — not FutureOr<UserEntity?>
});

// Fix — explicit cast satisfies the type constraint:
_updateUserEntity(savedUserEntity.id).catchError((err) {
  if (err is DioException && err.response?.statusCode == 404) {
    logout();
  }
  return null as UserEntity?;
});
```

Additionally the `.catchError()` call is not awaited (fire-and-forget), so even if the handler fires correctly, `logout()` races against the `return savedUserEntity` on line 46. Prefer a try/catch inline:

```dart
try {
  await _updateUserEntity(savedUserEntity.id);
} on DioException catch (err) {
  if (err.response?.statusCode == 404) {
    logout();
    return null;
  }
}
return savedUserEntity;
```

---

### CR-02: Password change request body is rejected by the API

**File:** `app/lib/components/settings/password_change_sheet.dart:35-42`

**Issue:** The sheet calls `POST /user/{userId}` (the user update endpoint). That endpoint's `UserUpdateSchema` requires `oldPassword` when changing the password. The sheet sends `oldPassword: v['currentPassword']` — but the form field is named `'currentPassword'`, so `v['currentPassword']` is correct — **however**, the field name sent to the API is also `'oldPassword'` (line 38), which matches the schema. **The real bug is a different one:** the SvelteKit `POST /api/v1/user/[id]` endpoint at line 96 calls `pb.collection('users').update()` which does not accept `oldPassword` in a standard PocketBase update call; PocketBase's REST API requires `oldPassword` alongside `password` and `passwordConfirm` only in the `PATCH /api/collections/users/records/{id}` endpoint, not in a custom update. The SvelteKit handler passes `oldPassword` through in `updateData` but PocketBase silently ignores unknown fields on `.update()`, so the password is changed **without verifying the old password**, and then re-authentication with `safeData.password` uses the new password — meaning any authenticated session can change the password with no knowledge of the current one.

More critically for the Flutter side: the sheet sends `data: {'oldPassword': ..., 'password': ..., 'passwordConfirm': ...}` but the SvelteKit schema (`UserUpdateSchema`) may not pass `oldPassword` through `updateData` if the field is stripped. Verify the schema passes `oldPassword` to PocketBase and that PocketBase validates it; if not, this is an **authorization bypass** — any authenticated user who obtains another user's ID can change their password.

**Fix (Flutter side — ensure field names match schema):**

```dart
await ref.read(apiProvider).post(
  '/user/${widget.userId}',
  data: {
    'oldPassword': v['currentPassword'],
    'password': v['newPassword'],
    'passwordConfirm': v['newPassword'],
  },
);
```

*Field names are actually correct in the current code. The real fix needed is on the API side to ensure PocketBase receives and validates `oldPassword`. Additionally, add a "confirm new password" field in the sheet (see WR-01).*

**However**, the combined effect is: the `POST /user/{id}` route does not forward `oldPassword` to PocketBase's native auth mechanism and therefore does not enforce old-password verification. This is an authorization gap — any logged-in user session can change their own password with no proof of knowing the current one.

---

### CR-03: Missing `NSCameraUsageDescription` in iOS `Info.plist`

**File:** `app/ios/Runner/Info.plist`

**Issue:** `image_picker` 1.x on iOS calls `PHPickerViewController` (photo library) which does not require camera permission. However, `image_picker` documentation explicitly states that `NSPhotoLibraryUsageDescription` alone is sufficient only for read access; on some iOS versions the system picker may also surface camera options or the plugin may fall back to `UIImagePickerController` which **always** requires `NSCameraUsageDescription`. If `NSCameraUsageDescription` is absent and the system camera picker is triggered — including by future plugin upgrades — the app will terminate immediately at the OS level without a crash report.

```xml
<!-- Add before or after NSPhotoLibraryUsageDescription -->
<key>NSCameraUsageDescription</key>
<string>Wanderer needs camera access so you can take a new profile photo.</string>
```

---

## Warnings

### WR-01: No "confirm new password" field in `PasswordChangeSheet`

**File:** `app/lib/components/settings/password_change_sheet.dart:91-110`

**Issue:** The sheet has only `currentPassword` and `newPassword` fields. There is no confirmation field. A typo in `newPassword` is accepted silently; the backend stores the wrong password and the user is locked out until they use password reset. This is a standard UX requirement for password change flows.

**Fix:** Add a `confirmNewPassword` field and a cross-field validator:

```dart
WandererTextField(
  name: 'confirmNewPassword',
  label: l10n.password_confirm,
  isPassword: true,
  validator: FormBuilderValidators.compose([
    FormBuilderValidators.required(),
    (val) {
      final pw = _formKey.currentState?.fields['newPassword']?.value as String?;
      return (val != pw) ? l10n.passwords_must_match : null;
    },
  ]),
),
```

---

### WR-02: Silent generic error handler discards server detail in `PasswordChangeSheet`

**File:** `app/lib/components/settings/password_change_sheet.dart:55-62`

**Issue:** The `catch (_)` block shows a generic `l10n.error_updating_password` toast and swallows the actual server response (e.g., wrong old password, validation failure, 403). `EmailChangeSheet` correctly parses `ApiError.fromJson` and surfaces the server message; `PasswordChangeSheet` should do the same.

**Fix:**

```dart
} on DioException catch (error) {
  String message;
  try {
    final apiError = ApiError.fromJson(error.response?.data);
    message = apiError.message;
  } catch (_) {
    message = error.message ?? l10n.error_updating_password;
  }
  if (!mounted) return;
  ref.read(toastProvider.notifier).add(
    ToastMessage(
      type: ToastType.error,
      icon: FontAwesomeIcons.circleExclamation,
      text: message,
    ),
  );
}
```

---

### WR-03: Avatar upload success toast uses non-descriptive key `l10n.avatar`

**File:** `app/lib/routes/settings_account_screen.dart:53-58`

**Issue:** On successful avatar upload the toast text is `l10n.avatar` which resolves to the string `"Avatar"`. This is the entity label, not a success message. The user sees the word "Avatar" pop up as a notification, which is not actionable or confirmatory. `EmailChangeSheet` uses `l10n.email_updated`; a parallel `avatar_updated` key (or the existing `l10n.settings_saved`) should be used.

**Fix:**

```dart
// Option A — add key "avatar_updated": "Avatar updated" to app_en.arb and use:
text: l10n.avatar_updated,

// Option B — reuse existing key (settings_saved = "Settings saved"):
text: l10n.settings_saved,
```

---

### WR-04: `didUpdateWidget` condition in `_BioSectionState` can overwrite user edits

**File:** `app/lib/routes/settings_account_screen.dart:255-266`

**Issue:** The guard on line 262-264 reads:

```dart
if (_controller.text == _persisted ||
    _controller.text == (oldWidget.settings?.bio ?? '')) {
  _controller.text = newPersisted;
}
```

The second branch — `_controller.text == (oldWidget.settings?.bio ?? '')` — matches when the user has deleted everything in the field (empty string) AND the old persisted value was also empty. In that case `_controller.text` gets reset to `newPersisted`, discarding the user's intentional clear. Specifically: if `_persisted` was `"hello"`, the user clears the field (text = `""`), and then `oldWidget.settings?.bio` also happens to be `""` (e.g., initial load), the condition fires and resets the field. The first branch (`_controller.text == _persisted`) is the correct guard; the second branch is redundant and incorrect.

**Fix:** Remove the second branch:

```dart
if (_controller.text == _persisted) {
  _controller.text = newPersisted;
}
```

---

## Info

### IN-01: Auth refresh test exercises no meaningful behavior

**File:** `app/test/provider/auth_provider_refresh_test.dart:27-33`

**Issue:** The only test asserts `expect(_assertRefreshExists, isNotNull)` — a function reference is always non-null. The test proves nothing about runtime behavior and will never fail even if `refresh()` is completely removed from `Auth` (the file simply won't compile). The comment acknowledges this but leaves the test body as permanently green. Future readers may rely on this test file as coverage evidence when there is none.

**Fix:** Either remove the file and note that integration tests cover this, or add a minimal notifier test using `ProviderContainer` with a stub dependency (at minimum verify that calling `refresh()` when `state.value` is null does not throw).

---

### IN-02: Avatar success toast shown before `refresh()` completes

**File:** `app/lib/routes/settings_account_screen.dart:49-58`

**Issue:** `refresh()` is awaited on line 49, but the toast is shown after the `context.mounted` check on lines 51-58 regardless of whether `refresh()` succeeded or threw. If `refresh()` throws (network error), the success toast still appears because the exception is swallowed by the outer `catch (_)` block at line 60. The user sees "Avatar" (see WR-03) even if the server data could not be reloaded.

**Fix:** Move the toast after the `refresh()` await and wrap both in a single try-block, or check that `refresh()` did not put `authProvider` into `AsyncError` state before showing success.

---

### IN-03: Localization key `visibilty` is a misspelling

**File:** `app/lib/i18n/app_en.arb:483`

**Issue:** `"visibilty": "Visibility"` and `"visibilty_status": "Visibility status"` both misspell "visibility" as "visibilty". This key is used elsewhere in the codebase via `l10n.visibilty`. This is not new to phase 08 but is present in the reviewed file.

**Fix:**

```json
"visibility": "Visibility",
"visibility_status": "Visibility status",
```

Update all call sites that reference `l10n.visibilty` and `l10n.visibilty_status` accordingly.

---

_Reviewed: 2026-06-20T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
