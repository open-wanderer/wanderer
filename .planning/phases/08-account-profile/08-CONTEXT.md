# Phase 8: Account & Profile - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Fill the stub `SettingsAccountScreen` with a complete account management UI: avatar display/upload, bio editing, email change, password change, and account deletion. All operations call existing web API endpoints via the Dio client. Adding `image_picker` is the only new dependency.

Requirements covered: ACCT-01, ACCT-02, ACCT-03, ACCT-04, ACCT-05

</domain>

<decisions>
## Implementation Decisions

### Avatar upload (ACCT-01)
- **D-01:** Add `image_picker` to `pubspec.yaml` — the only new dependency this phase. No other image packages are needed.
- **D-02:** Gallery-only source (`ImageSource.gallery`). No camera picker. This minimises the permission surface.
- **D-03:** Auto-upload on pick. The moment the user selects an image, call `POST /api/v1/user/{id}/file` (multipart) and refresh the displayed avatar. No staging or Save button for avatar.

### Bio editing (ACCT-02)
- **D-04:** Plain-text `TextField` (multiline). The web uses TipTap rich text but no rich editor package is in pubspec — plain text is idiomatic Flutter.
- **D-05:** Explicit Save button, enabled only when the current text differs from the persisted value. Matches the web client's Save-only-when-changed pattern.

### Email change (ACCT-03)
- **D-06:** Modal bottom sheet triggered by an `'Change email'` `ListTile` tap. No new route needed.
- **D-07:** Form uses `FormBuilder` + `WandererTextField` (extends `FormBuilderField<String>` — already used in `LoginScreen` and `RegisterScreen`). Two fields: new email address and current password. Validate both non-empty; validate email format.
- **D-08:** On submit, call `POST /api/v1/user/{id}/email` with `{ email, currentPassword }`. On success, show success toast using existing `l10n.email_updated`. On error, surface the API error message in a toast.

### Password change (ACCT-04)
- **D-09:** Same modal bottom sheet pattern as email change. Triggered by a `'Change password'` `ListTile` tap.
- **D-10:** Two `WandererTextField` fields: current password and new password (both `isPassword: true`). Validate both non-empty.
- **D-11:** On submit, call `POST /api/v1/user/{id}` with `{ password, passwordConfirm }` (the server re-auths with the current password). On success, show `l10n.new_password_success` toast. On error, show `l10n.error_updating_password`.

### Account deletion (ACCT-05)
- **D-12:** Simple `AlertDialog` confirm, matching the web client. Body text uses `l10n.account_delete_confirm` ("You are about to delete your account. All your trails will also be deleted. Do you want to proceed?"). Two actions: Cancel and Delete (destructive color).
- **D-13:** After successful deletion, call `authProvider.notifier.logout()` then let go_router's redirect guard push the user to `/login`. No extra route needed.

### Screen structure
- **D-14:** `ConsumerWidget` → `Scaffold` → `AppBar` (back nav via `context.pop()`) → `ListView`. Sections: avatar row at top, bio text field with Save button, then two `ListTile` rows for Change email / Change password, then a Delete account `ListTile` (destructive color).
- **D-15:** The `authProvider` (not `settingsProvider`) is the source of truth for the current user's `UserEntity` (id, email, avatar, etc.). Bio lives in `settingsProvider` (Settings.bio field via the settings API).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing screen patterns (must follow)
- `app/lib/routes/settings_privacy_screen.dart` — canonical ConsumerWidget/AppBar/ListView structure for settings screens
- `app/lib/routes/login_screen.dart` — FormBuilder + WandererTextField usage pattern
- `app/lib/routes/register_screen.dart` — FormBuilder validation + saveAndValidate() pattern

### Custom widget
- `app/lib/components/base/wanderer_text_field.dart` — `WandererTextField extends FormBuilderField<String>`; params: `name` (required), `validator`, `initialValue`, `label`, `placeholder`, `icon`, `disabled`, `isPassword`

### State providers
- `app/lib/provider/auth_provider.dart` — `authProvider` — source of `UserEntity` (id, email, avatar, serverUrl, username)
- `app/lib/provider/settings_provider.dart` — `settingsProvider` — source of `Settings` (bio field lives here)
- `app/lib/provider/api_provider.dart` — `apiProvider` — Dio client for HTTP calls

### Models & entities
- `app/lib/entities/user_entity.dart` — `UserEntity`: id, username, email, avatar, serverUrl
- `app/lib/models/settings.dart` — `Settings` freezed model with `bio` field

### API endpoints (all confirmed on web side)
- `POST /api/v1/user/{id}` — update user (bio, password). Password update also re-auths.
- `POST /api/v1/user/{id}/file` — avatar upload (multipart form, field name `"avatar"`)
- `POST /api/v1/user/{id}/email` — email change body: `{ email, currentPassword }`
- `DELETE /api/v1/user/{id}` — delete account
- `web/src/lib/stores/user_store.ts` — reference for request shapes: `users_update()`, `users_update_email()`, `users_delete()`

### Stub to fill
- `app/lib/routes/settings_account_screen.dart` — current stub: `ConsumerWidget` with empty `Container()` body

### Internationalisation
- `app/lib/i18n/app_en.arb` — all required keys confirmed present: `avatar`, `change_email`, `change_password`, `current_password`, `new_password`, `email`, `email_updated`, `error_updating_password`, `new_password_success`, `new_password_error`, `delete_account`, `account_delete_confirm`, `confirm_deletion`, `save`, `cancel`
- `bio` key may need to be verified; `add_bio` exists. Check that a plain `bio` label key exists or use `add_bio`.

### Requirements
- `.planning/REQUIREMENTS.md` — ACCT-01 through ACCT-05
- `.planning/ROADMAP.md` §Phase 8 — success criteria and phase goal

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `WandererTextField` — drop-in styled form field with built-in error display; use `isPassword: true` for credential fields
- `FormBuilder` + `_formKey.currentState!.saveAndValidate()` — existing form validation pattern from LoginScreen and RegisterScreen
- `authProvider.notifier.logout()` — existing logout that clears cookie jar and resets state
- `settingsProvider.notifier.saveToServer(settings)` — existing server save with ObjectBox persistence

### Established Patterns
- Bottom sheets: use `showModalBottomSheet()` from `material.dart` with a `StatefulWidget` or `ConsumerStatefulWidget` form inside
- Error toasts: try/catch with `ScaffoldMessenger.of(context).showSnackBar(...)` or the existing toast provider pattern used in Phase 6/7
- Auto-upload multipart: `FormData.fromMap({'avatar': await MultipartFile.fromFile(path)})` via Dio

### Integration Points
- Avatar display: `UserEntity.avatar` is a filename; construct the full URL using the existing `serverUrl` + PocketBase file URL pattern. Check `profile_screen.dart` for how avatar URLs are currently constructed.
- Bio persists via `settingsProvider` (not directly on UserEntity)
- After email change, re-fetch `UserEntity` so the displayed email refreshes (or update `authProvider` state directly)

</code_context>

<specifics>
## Specific Details

- Use `WandererTextField` (not raw `TextField`) for all form fields in the email and password change bottom sheets
- Validate both email and password change forms using `FormBuilder` validation before submitting
- Avatar upload: `image_picker` version to add — check pubspec for version constraints; use latest stable
- Bottom sheet dismiss on success: call `Navigator.of(context).pop()` after showing the success toast
- Bio `TextField` should be multiline (`maxLines: null` or a fixed count like 4–5)
- Delete account `ListTile` leading icon and text color should use the error/destructive color: `Theme.of(context).colorScheme.error`

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 8-Account & Profile*
*Context gathered: 2026-06-20*
