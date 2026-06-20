---
phase: 08-account-profile
plan: "03"
subsystem: flutter-app
tags: [flutter, riverpod, image_picker, dio, form, material, widget-test]

dependency_graph:
  requires:
    - phase: 08-01
      provides: "image_picker dependency, iOS NSPhotoLibraryUsageDescription, AppLocalizations.account getter, Auth.refresh() notifier method"
    - phase: 08-02
      provides: "EmailChangeSheet and PasswordChangeSheet ConsumerStatefulWidgets with userId param"
  provides:
    - SettingsAccountScreen (app/lib/routes/settings_account_screen.dart): filled Account & Profile screen
    - Widget test (app/test/routes/settings_account_screen_test.dart): structural section assertions
  affects:
    - Phase 09 (Notifications screen) — nothing depends on this screen; it is the terminal plan of v1.2

tech_stack:
  added: []
  patterns:
    - "_BioSection as ConsumerStatefulWidget holding TextEditingController inside a ConsumerWidget parent (avoids converting the outer screen to StatefulWidget)"
    - "GestureDetector on CircleAvatar + adjacent IconButton for avatar tap target (both call same handler)"
    - "Avatar upload via FormData.fromMap + MultipartFile.fromFile, then authProvider.refresh() re-fetches UserEntity"
    - "Change-aware Save: controller.text != (settings?.bio ?? '') as the enable predicate (normalises null and '' as equal)"
    - "showDialog<bool> with AlertDialog returning true/false for destructive confirm gate"
    - "context.mounted guard after each await in ConsumerWidget methods (not mounted — context.mounted is appropriate in ConsumerWidget build-method helpers)"

key_files:
  created:
    - app/test/routes/settings_account_screen_test.dart
  modified:
    - app/lib/routes/settings_account_screen.dart

key_decisions:
  - "_BioSection extracted as inner ConsumerStatefulWidget so a TextEditingController can live in a ConsumerWidget-hosted screen without converting the whole screen to StatefulWidget"
  - "hintText: l10n.add_bio used in TextField (not a separate placeholder widget) — causes 'Add Bio' to appear twice in widget tree; test updated to use findsWidgets instead of findsOneWidget"
  - "context.mounted used (not mounted) for async guards in ConsumerWidget helper methods — State.mounted only applies in ConsumerState subclasses; ConsumerWidget methods have BuildContext not State"
  - "Colors.red.shade400 for destructive delete row and AlertDialog delete action — colorScheme.error maps to #FEF2F2 (light background token) which is illegible as foreground text"

requirements-completed: [ACCT-01, ACCT-02, ACCT-03, ACCT-04, ACCT-05]

duration: 18min
completed: "2026-06-20"
---

# Phase 08 Plan 03: SettingsAccountScreen — Account & Profile UI Summary

**Filled SettingsAccountScreen stub with all five ACCT sections: CircleAvatar gallery upload with multipart POST, change-aware bio TextField, email/password modal sheets, and AlertDialog-gated account deletion with logout.**

## Performance

- **Duration:** ~18 min
- **Started:** 2026-06-20T13:05:00Z
- **Completed:** 2026-06-20T13:23:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `SettingsAccountScreen` filled from stub (13 lines) to 320 lines implementing all five ACCT requirements
- **ACCT-01 Avatar:** `CircleAvatar(radius: 40)` with `NetworkImage` + dicebear fallback, tappable via `GestureDetector` + edit `IconButton`; on pick posts `FormData{'avatar': MultipartFile}` to `/user/$userId/file` then calls `authProvider.notifier.refresh()`
- **ACCT-02 Bio:** `_BioSection` (inner `ConsumerStatefulWidget`) with `TextEditingController`, 4-line `TextField`, `ElevatedButton` enabled iff `controller.text != (settings?.bio ?? '')`, saves via `settingsProvider.notifier.saveToServer(settings.copyWith(bio: ...))`
- **ACCT-03/04 Sheets:** Two `ListTile` rows open `EmailChangeSheet` / `PasswordChangeSheet` via `showModalBottomSheet(isScrollControlled: true)` passing `user.id`
- **ACCT-05 Delete:** `ListTile` with `Colors.red.shade400` foreground, `AlertDialog(confirm_deletion / account_delete_confirm)` returning `bool`; on confirm: `DELETE /user/$userId` then `authProvider.notifier.logout()`
- Widget test created mirroring `settings_privacy_screen_test.dart` harness; asserts all five sections present by localized copy

## Task Commits

1. **Task 1: Fill SettingsAccountScreen** — `700d9a0c` (feat)
2. **Task 2: Widget test** — `3e2ff993` (feat)

## Files Created/Modified

- `app/lib/routes/settings_account_screen.dart` — 320 lines; fills stub with all five ACCT sections; imports image_picker, dio, EmailChangeSheet, PasswordChangeSheet, authProvider, settingsProvider, toastProvider
- `app/test/routes/settings_account_screen_test.dart` — 87 lines; `_StubAuth extends Auth` returning fixture `UserEntity`; asserts 'Account' AppBar, 'Add Bio', `TextField`, 'Change email', 'Change password', 'Delete Account'

## Decisions Made

- **_BioSection as ConsumerStatefulWidget:** The outer `SettingsAccountScreen` must remain a `ConsumerWidget` (the `const SettingsAccountScreen()` route invocation already exists and `ConsumerWidget` does not hold mutable state). The bio section needs a `TextEditingController`, so it is extracted into a private `_BioSection ConsumerStatefulWidget`. This is the minimal-scope approach — only the bio block becomes stateful.
- **hintText reuse for section header copy:** The bio `TextField(hintText: l10n.add_bio)` causes "Add Bio" to appear in both the section header `Text` widget and the hint; the widget test uses `findsWidgets` not `findsOneWidget` for that assertion.
- **context.mounted in ConsumerWidget:** In `ConsumerWidget` helper methods (`_pickAndUploadAvatar`, `_deleteAccount`), the async guard is `if (!context.mounted) return` — correct for a `ConsumerWidget`. The Phase 02 decision about `mounted` vs `context.mounted` applies only to `ConsumerState` subclasses.
- **Colors.red.shade400:** `colorScheme.error` maps to `#FEF2F2` (Tailwind light-red background) which is illegible as foreground text. UI-SPEC D-12 explicitly permits `Colors.red.shade400` for the destructive row text/icon and dialog action.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] IDE null-aware operator lint on avatar URL**
- **Found during:** Task 1 — immediately after Write (PostToolUse diagnostic)
- **Issue:** `user != null ? user.getFileUrl(...) : null` pattern flagged as information-level lint; Dart prefers `user?.getFileUrl(...)`
- **Fix:** Changed to `user?.getFileUrl(user.serverUrl, user.avatar)` using null-aware operator
- **Files modified:** app/lib/routes/settings_account_screen.dart
- **Verification:** `flutter analyze lib/routes/settings_account_screen.dart` — No issues found
- **Committed in:** 700d9a0c (Task 1 commit, fixed before commit)

**2. [Rule 1 - Bug] Test assertion changed from findsOneWidget → findsWidgets for 'Add Bio'**
- **Found during:** Task 2 widget test first run
- **Issue:** "Add Bio" appears in both the `_sectionHeader` `Text` widget and the `TextField` `hintText`; `findsOneWidget` fails with "Found 2 widgets"
- **Fix:** Changed `expect(find.text('Add Bio'), findsOneWidget)` to `expect(find.text('Add Bio'), findsWidgets)` with an explanatory comment
- **Files modified:** app/test/routes/settings_account_screen_test.dart
- **Verification:** `flutter test test/routes/settings_account_screen_test.dart` exits 0
- **Committed in:** 3e2ff993 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — Bugs)
**Impact on plan:** Both fixes required for correctness. No scope creep.

## Issues Encountered

Pre-existing test failures (not caused by this plan):
- `test/models/feed_item_test.dart` — 2 failures: `type 'int' is not a subtype of type 'String'` / `List<dynamic>?`; confirmed pre-existing via `git stash` check
- `test/routes/settings_language_screen_test.dart` — 1 failure: `scrollUntilVisible` Bad state; confirmed pre-existing via `git stash` check

All account-screen-related tests pass: account screen test (new), settings screen test, privacy screen test.

## Known Stubs

None — all five ACCT sections are fully wired to real API endpoints, real providers, and real l10n keys. No placeholder data or hardcoded display strings.

## Threat Flags

No new network endpoints, auth paths, or schema changes beyond the plan's threat model. T-08-09 through T-08-13 mitigations are implemented:
- T-08-09: AlertDialog confirm gate before `DELETE /user/$userId`
- T-08-10: `user.id` from `authProvider.value` (authenticated user's own id); server enforces 403 for other ids
- T-08-11: `authProvider.notifier.logout()` clears cookie jar + ObjectBox after deletion; go_router redirects to `/login`
- T-08-12: Errors surface via `l10n.error_saving_settings` toast (no PII or token details)
- T-08-13: `authProvider.notifier.refresh()` re-fetches expanded UserEntity after avatar upload; bio saved via `settingsProvider.saveToServer`

## User Setup Required

None — all infrastructure (image_picker iOS permission, `account` l10n key, `Auth.refresh()`) was established in Plan 01.

## Next Phase Readiness

Phase 08 is complete. All five ACCT requirements (ACCT-01..05) are observable from the Account screen. The screen is reachable at the existing `settings/account` go_router route.

No blockers for Phase 09 (Notifications) or any future phase.

## Self-Check: PASSED

- `app/lib/routes/settings_account_screen.dart`: FOUND (320 lines)
- `app/test/routes/settings_account_screen_test.dart`: FOUND (87 lines)
- `class SettingsAccountScreen extends ConsumerWidget`: FOUND
- `Text(l10n.account)` in settings_account_screen.dart: FOUND
- `ImagePicker().pickImage(source: ImageSource.gallery)` in settings_account_screen.dart: FOUND
- `/user/$userId/file` POST in settings_account_screen.dart: FOUND
- `authProvider.notifier).refresh()` in settings_account_screen.dart: FOUND
- `saveToServer` in settings_account_screen.dart: FOUND
- `EmailChangeSheet` in settings_account_screen.dart: FOUND
- `PasswordChangeSheet` in settings_account_screen.dart: FOUND
- `isScrollControlled: true` in settings_account_screen.dart: FOUND
- `AlertDialog` in settings_account_screen.dart: FOUND
- `Colors.red.shade400` in settings_account_screen.dart: FOUND
- `authProvider.notifier).logout()` in settings_account_screen.dart: FOUND
- Commits 700d9a0c, 3e2ff993: FOUND in git log
- `flutter analyze lib/routes/settings_account_screen.dart` — No issues: CONFIRMED
- `flutter test test/routes/settings_account_screen_test.dart` exits 0: CONFIRMED

---
*Phase: 08-account-profile*
*Completed: 2026-06-20*
