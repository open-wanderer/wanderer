---
phase: 08-account-profile
fixed_at: 2026-06-20T00:00:00Z
review_path: .planning/phases/08-account-profile/08-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 4
skipped: 3
status: partial
---

# Phase 08: Code Review Fix Report

**Fixed at:** 2026-06-20T00:00:00Z
**Source review:** .planning/phases/08-account-profile/08-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (CR-01, CR-02, CR-03, WR-01, WR-02, WR-03, WR-04)
- Fixed: 4 (CR-01, CR-03, WR-01+WR-02, WR-04)
- Skipped: 3 (CR-02 false positive; WR-03 code no longer present; WR-01/WR-02 committed together)

## Fixed Issues

### CR-01: `catchError` return type mismatch silences 404 logout

**Files modified:** `app/lib/provider/auth_provider.dart`
**Commit:** 67f9a435
**Applied fix:** Replaced the fire-and-forget `.catchError()` block with an inline `try { await _updateUserEntity(...); } on DioException catch (err)` block. The new form awaits the call (eliminating the race between logout() and `return savedUserEntity`) and uses a typed catch that removes the `Null` vs `UserEntity?` coercion risk under sound null-safety. Returns `null` after calling `logout()` on a 404.

---

### CR-03: Missing `NSCameraUsageDescription` in iOS `Info.plist`

**Files modified:** `app/ios/Runner/Info.plist`
**Commit:** ac430de8
**Applied fix:** Added `NSCameraUsageDescription` key with value "Wanderer needs camera access so you can take a new profile photo." immediately before `NSPhotoLibraryUsageDescription`, matching the existing plist key ordering.

---

### WR-01 + WR-02: No confirm password field; silent error handler in `PasswordChangeSheet`

**Files modified:** `app/lib/components/settings/password_change_sheet.dart`
**Commit:** 63ec9b81
**Applied fix (WR-01):** Added a `confirmNewPassword` `WandererTextField` with a cross-field validator using `FormBuilderValidators.compose` that asserts the value matches `newPassword`, using the existing ARB keys `l10n.password_confirm` and `l10n.passwords_must_match`. The API `passwordConfirm` field now sends `v['confirmNewPassword']` instead of duplicating `v['newPassword']`.
**Applied fix (WR-02):** Added `import 'package:wanderer/models/api_error.dart'` and replaced the generic `catch (_)` with `on DioException catch (error)` that parses `ApiError.fromJson(error.response?.data)` to surface the server-provided message. A generic `catch (_)` fallback below handles non-network exceptions, matching the `EmailChangeSheet` pattern.

---

### WR-04: `didUpdateWidget` condition in `_BioSectionState` can overwrite user edits

**Files modified:** `app/lib/routes/settings_account_screen.dart`
**Commit:** 2bc94523
**Applied fix:** Removed the second branch `_controller.text == (oldWidget.settings?.bio ?? '')` from the guard condition in `didUpdateWidget`. The remaining single-branch check `_controller.text == _persisted` is the correct guard — it only resets the controller when the user has not deviated from the last persisted server value.

---

## Skipped Issues

### CR-02: Password change request body is rejected by the API

**File:** `app/lib/components/settings/password_change_sheet.dart:35-42`
**Reason:** false positive — PocketBase enforces oldPassword at collection level. The SvelteKit handler calls `pb.collection('users').update(id, updateData)` where `updateData` includes `oldPassword`. PocketBase auth collections enforce `oldPassword` at the ORM/database layer for non-superuser updates — the SDK returns 400 if `oldPassword` is missing or wrong. The sheet already sends the correct payload `{oldPassword, password, passwordConfirm}`. No code change required.

---

### WR-03: Avatar upload success toast uses non-descriptive key `l10n.avatar`

**File:** `app/lib/routes/settings_account_screen.dart:53-58`
**Reason:** skipped — code no longer present. The user committed `013d2b10 update account screen` after the review was written. That commit removed the success toast from `_pickAndUploadAvatar` entirely (replaced with `ref.invalidate(ownProfileProvider)` and a loading spinner). The flagged `l10n.avatar` toast text no longer exists in the file.

---

_Fixed: 2026-06-20T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
