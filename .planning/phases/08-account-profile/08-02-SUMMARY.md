---
phase: 08-account-profile
plan: "02"
subsystem: flutter-app
tags: [flutter, riverpod, form_builder, dio, bottom-sheet, credentials]

dependency_graph:
  requires:
    - phase: 08-01
      provides: "Auth.refresh() public method on Auth Riverpod notifier; account l10n key"
  provides:
    - EmailChangeSheet ConsumerStatefulWidget (app/lib/components/settings/email_change_sheet.dart)
    - PasswordChangeSheet ConsumerStatefulWidget (app/lib/components/settings/password_change_sheet.dart)
  affects:
    - 08-03 (settings_account_screen.dart — opens both sheets via showModalBottomSheet)

tech_stack:
  added: []
  patterns:
    - "ConsumerStatefulWidget bottom-sheet with GlobalKey<FormBuilderState> and autovalidateMode: onUnfocus"
    - "State.mounted guard (not context.mounted) for BuildContext use across async gaps in ConsumerState"
    - "viewInsets.bottom padding pattern for keyboard-safe bottom sheets"
    - "try/catch on apiProvider.post with DioException branch for structured API error extraction"

key_files:
  created:
    - app/lib/components/settings/email_change_sheet.dart
    - app/lib/components/settings/password_change_sheet.dart
  modified: []

key_decisions:
  - "EmailChangeSheet catches DioException and extracts ApiError.fromJson(error.response?.data).message to surface actual server error; falls back to error.message — mirrors RegisterScreen pattern"
  - "PasswordChangeSheet uses a generic l10n.error_updating_password toast on failure (D-11 / T-08-07: never leak server internals for password errors)"
  - "PasswordChangeSheet payload must include oldPassword mapped from the currentPassword field value — PocketBase's native password-change rule enforces it (Pitfall 2); {password, passwordConfirm}-only payload returns 400"
  - "No authProvider.refresh() call in PasswordChangeSheet — identity unchanged after password update; Dio cookie manager persists new token automatically (Open Question 2)"
  - "State.mounted used (not context.mounted) for async BuildContext guards — IDE linter (use_build_context_synchronously) requires mounted check on the State object, not the BuildContext, in ConsumerState subclasses"

requirements-completed: [ACCT-03, ACCT-04]

duration: 12min
completed: "2026-06-20"
---

# Phase 08 Plan 02: EmailChangeSheet and PasswordChangeSheet Summary

**Two ConsumerStatefulWidget bottom-sheet forms for credential changes: EmailChangeSheet posts {email, currentPassword} to /user/$id/email and refreshes auth; PasswordChangeSheet posts {oldPassword, password, passwordConfirm} to /user/$id.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-06-20T12:53:00Z
- **Completed:** 2026-06-20T13:00:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `EmailChangeSheet` — FormBuilder sheet with email + current-password fields, posts to `/user/$userId/email`, toasts `email_updated` on success, calls `authProvider.refresh()` so the displayed email refreshes, surfaces DioException message on failure and keeps sheet open
- `PasswordChangeSheet` — FormBuilder sheet with current-password + new-password fields, posts `{oldPassword, password, passwordConfirm}` to `/user/$userId` (Pitfall 2: oldPassword required by PocketBase rule), toasts generic `error_updating_password` on failure (no server internals leaked), no refresh() needed
- Both sheets use `EdgeInsets.fromLTRB(16, 24, 16, 16 + MediaQuery.of(context).viewInsets.bottom)` so the keyboard never covers the submit button
- IDE linter warning caught and fixed: `State.mounted` used for async BuildContext guards (not `context.mounted`) per Dart lint `use_build_context_synchronously`

## Task Commits

1. **Task 1: EmailChangeSheet** - `8ff7798c` (feat)
2. **Task 2: PasswordChangeSheet** - `8973e875` (feat)

## Files Created/Modified

- `app/lib/components/settings/email_change_sheet.dart` — EmailChangeSheet ConsumerStatefulWidget (128 lines); posts to `/user/$userId/email`; refreshes auth on success
- `app/lib/components/settings/password_change_sheet.dart` — PasswordChangeSheet ConsumerStatefulWidget (115 lines); posts `{oldPassword, password, passwordConfirm}` to `/user/$userId`; no refresh() call

## Decisions Made

- `State.mounted` used instead of `context.mounted` in `ConsumerState` subclasses — IDE linter `use_build_context_synchronously` warns about BuildContext use across async gaps guarded by an unrelated mounted check; the fix is to use the State's own `mounted` getter
- Password errors use `l10n.error_updating_password` (generic) not the server's DioException message — T-08-07 disposition is mitigate; password auth failures should not leak server internals
- Email errors surface `ApiError.fromJson(error.response?.data).message` — T-08-07 allows the structured API message for email failures (server returns a user-facing message)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `context.mounted` → `State.mounted` for async BuildContext guards**
- **Found during:** Task 1 (EmailChangeSheet) immediately after Write
- **Issue:** IDE diagnostics flagged `use_build_context_synchronously` lint on two lines where `context.mounted` was used as the async guard. In a `ConsumerState`, the guard must use `mounted` (the State's own property) not `context.mounted` — the linter treats them as unrelated objects
- **Fix:** Changed `if (!context.mounted) return;` to `if (!mounted) return;` in both the success and error paths of `_submit()`. Applied same pattern proactively to Task 2.
- **Files modified:** app/lib/components/settings/email_change_sheet.dart
- **Verification:** `flutter analyze lib/components/settings/email_change_sheet.dart` — No issues found
- **Committed in:** 8ff7798c (Task 1 commit, already fixed before commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — Bug)
**Impact on plan:** Fix required for correct async safety. No scope creep.

## Issues Encountered

None beyond the auto-fixed mounted check above.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — both sheets are fully wired to the real API endpoints with real l10n keys and toast feedback. No placeholder data or hardcoded display strings.

## Threat Flags

None — the threat model mitigations are implemented:
- T-08-04: EmailChangeSheet requires `currentPassword` in the POST payload (server validates via `ValidatePassword`)
- T-08-05: PasswordChangeSheet includes `oldPassword` in the POST payload (Pitfall 2)
- T-08-06: Both sheets receive `userId` from the parent screen which reads it from `authProvider.value` (the authenticated user's own id)
- T-08-07: Password errors use generic `l10n.error_updating_password`; email errors use only `apiError.message` (no stack traces or internal details)
- T-08-08: `FormBuilderValidators.required()` + `.email()` block empty/malformed input client-side

## Next Phase Readiness

- Plan 03 (`settings_account_screen.dart`) can now wire `EmailChangeSheet` and `PasswordChangeSheet` via `showModalBottomSheet` — both are named, finished widgets with the `userId` parameter ready
- No blockers

## Self-Check: PASSED

- `app/lib/components/settings/email_change_sheet.dart`: FOUND
- `app/lib/components/settings/password_change_sheet.dart`: FOUND
- `class EmailChangeSheet extends ConsumerStatefulWidget`: FOUND
- `class PasswordChangeSheet extends ConsumerStatefulWidget`: FOUND
- `'oldPassword'` in password_change_sheet.dart: FOUND
- `'passwordConfirm'` in password_change_sheet.dart: FOUND
- `refresh()` absent from password_change_sheet.dart: CONFIRMED
- `flutter analyze lib/components/settings/` — No issues found: CONFIRMED
- Commits 8ff7798c, 8973e875: FOUND in git log

---
*Phase: 08-account-profile*
*Completed: 2026-06-20*
