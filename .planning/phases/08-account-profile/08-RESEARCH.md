# Phase 8: Account & Profile - Research

**Researched:** 2026-06-20
**Domain:** Flutter settings UI (avatar upload, form bottom sheets, account credential changes) on top of an existing Riverpod + Dio + PocketBase stack
**Confidence:** HIGH

## Summary

Phase 8 fills the stub `SettingsAccountScreen` with five capabilities (avatar, bio, email, password, delete) that all bind to existing infrastructure. There is exactly **one new dependency** (`image_picker`); everything else — form widget (`WandererTextField`), validators (`FormBuilderValidators`), toast feedback (`toastProvider`), settings persistence (`settingsProvider.saveToServer`), auth state (`authProvider`), avatar URL construction (`RecordFunctions.getFileUrl`), and every API endpoint — already exists and is in active use elsewhere in the app. The screen mirrors the canonical `SettingsPrivacyScreen` shell and the `LoginScreen`/`RegisterScreen` form pattern.

Research **confirmed all five API endpoints against the live SvelteKit + Go source**, and surfaced **three concrete corrections** to the locked CONTEXT.md decisions that the planner must reconcile (detailed in the Pitfalls and Open Questions sections): (1) bio is NOT saved via the user endpoint — it persists through `settingsProvider` to `POST /api/v1/settings/{id}`; (2) password change very likely requires `oldPassword` in the payload (the web client sends it and PocketBase's native password-change rule enforces it) — D-11 omits it; (3) refreshing the displayed email/avatar after a change cannot be done by deserializing the POST response, because `User.toEntity()` throws without an expanded `actor` — the screen must trigger an `authProvider` re-fetch instead.

**Primary recommendation:** Build a `ConsumerWidget` → `Scaffold` → `ListView` exactly like `SettingsPrivacyScreen`, reuse `WandererTextField` + `FormBuilderValidators` for the two `showModalBottomSheet` forms, add `image_picker: ^1.2.2` (Flutter-team verified publisher), construct the avatar URL with `userEntity.getFileUrl(serverUrl, avatar)` and the dicebear fallback, and add a public `refresh()` method to the `Auth` notifier so email/avatar changes can re-fetch the expanded user.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Avatar upload (ACCT-01)**
- **D-01:** Add `image_picker` to `pubspec.yaml` — the only new dependency this phase.
- **D-02:** Gallery-only source (`ImageSource.gallery`). No camera picker.
- **D-03:** Auto-upload on pick. The moment the user selects an image, call `POST /api/v1/user/{id}/file` (multipart) and refresh the displayed avatar. No staging or Save button for avatar.

**Bio editing (ACCT-02)**
- **D-04:** Plain-text `TextField` (multiline).
- **D-05:** Explicit Save button, enabled only when the current text differs from the persisted value.

**Email change (ACCT-03)**
- **D-06:** Modal bottom sheet triggered by a `'Change email'` `ListTile` tap. No new route needed.
- **D-07:** Form uses `FormBuilder` + `WandererTextField`. Two fields: new email address and current password. Validate both non-empty; validate email format.
- **D-08:** On submit, call `POST /api/v1/user/{id}/email` with `{ email, currentPassword }`. On success, show success toast using existing `l10n.email_updated`. On error, surface the API error message in a toast.

**Password change (ACCT-04)**
- **D-09:** Same modal bottom sheet pattern as email change. Triggered by a `'Change password'` `ListTile` tap.
- **D-10:** Two `WandererTextField` fields: current password and new password (both `isPassword: true`). Validate both non-empty.
- **D-11:** On submit, call `POST /api/v1/user/{id}` with `{ password, passwordConfirm }` (the server re-auths with the current password). On success, show `l10n.new_password_success` toast. On error, show `l10n.error_updating_password`.

**Account deletion (ACCT-05)**
- **D-12:** Simple `AlertDialog` confirm. Body text uses `l10n.account_delete_confirm`. Two actions: Cancel and Delete (destructive color).
- **D-13:** After successful deletion, call `authProvider.notifier.logout()` then let go_router's redirect guard push the user to `/login`.

**Screen structure**
- **D-14:** `ConsumerWidget` → `Scaffold` → `AppBar` (back nav via `context.pop()`) → `ListView`. Sections: avatar row at top, bio text field with Save button, then two `ListTile` rows for Change email / Change password, then a Delete account `ListTile` (destructive color).
- **D-15:** `authProvider` is the source of truth for the current user's `UserEntity` (id, email, avatar). Bio lives in `settingsProvider` (`Settings.bio` via the settings API).

### Claude's Discretion
- `image_picker` exact version (CONTEXT says "latest stable" / "check pubspec for version constraints").
- Bottom-sheet implementation detail (`StatefulWidget` vs `ConsumerStatefulWidget`).
- Inline submit-button progress affordance (UI-SPEC marks it optional).

### Deferred Ideas (OUT OF SCOPE)
- None — discussion stayed within phase scope. (API token management ACCT-F01 is deferred per REQUIREMENTS.md; do NOT port the web account screen's API-token UI.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ACCT-01 | View and update avatar | `image_picker` (gallery) → `POST /api/v1/user/{id}/file` multipart field `avatar` (confirmed in `user_store.ts` + `file/+server.ts`); display via `UserEntity.getFileUrl(serverUrl, avatar)` (confirmed in `record.dart`) with dicebear fallback (pattern in `profile_screen.dart`). Refresh via `authProvider` re-fetch (see Pitfall 3). |
| ACCT-02 | View and edit bio | `Settings.bio` field exists (`settings.dart`); save via `settingsProvider.notifier.saveToServer(settings.copyWith(bio: ...))` → `POST /api/v1/settings/{id}` (confirmed in `settings_provider.dart` + `settings/[id]/+server.ts`). NOT the user endpoint — see Pitfall 1. |
| ACCT-03 | Change email | `POST /api/v1/user/{id}/email` body `{ email, currentPassword }` (confirmed in `email/+server.ts` + Go `user_email_change.go`; current password IS validated server-side via `e.Auth.ValidatePassword`). |
| ACCT-04 | Change password | `POST /api/v1/user/{id}` (confirmed in `user/[id]/+server.ts`). Schema accepts `oldPassword`, `password`, `passwordConfirm` (`user_schema.ts`); web sends all three. D-11 omits `oldPassword` — see Pitfall 2. |
| ACCT-05 | Delete account with confirmation | `DELETE /api/v1/user/{id}` (confirmed in `user/[id]/+server.ts`) → `authProvider.notifier.logout()` → go_router redirect to `/login`. `AlertDialog` confirm gate. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Avatar pick (gallery) | Browser/Client (Flutter device) | — | OS image picker is a device-local capability (`image_picker`) |
| Avatar upload | API / Backend | Client (multipart build) | Multipart POST to SvelteKit which proxies to PocketBase file storage |
| Avatar display URL | Client | API (file serving) | URL built client-side via `getFileUrl`; bytes served by `/api/v1/files/...` |
| Bio persistence | API / Backend (settings collection) | Client state (`settingsProvider` + ObjectBox cache) | Bio is a `Settings` field, saved via the settings endpoint, cached in ObjectBox |
| Email change | API / Backend (Go custom route) | Client form | Credential change with server-side password re-validation; must stay server-side |
| Password change | API / Backend (PocketBase update) | Client form | Same — credential mutation, server-authoritative |
| Account deletion | API / Backend | Client (confirm + post-delete logout) | Destructive server mutation; client only confirms + clears local auth |
| Post-change state refresh | Client state (`authProvider`) | API (re-fetch with expand) | Displayed user must re-fetch expanded record after a credential/avatar change |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `image_picker` | `^1.2.2` | Pick an image from the gallery for avatar upload | Official Flutter-team package (flutter.dev verified publisher); the de-facto standard for gallery/camera selection [CITED: pub.dev/packages/image_picker] |
| `flutter_form_builder` | `^10.3.0+2` (already installed) | Form state + validation host for the email/password sheets | Already used by `LoginScreen`/`RegisterScreen` [VERIFIED: app/pubspec.yaml] |
| `form_builder_validators` | `^11.3.0` (already installed) | `.required()`, `.email()`, `.compose()` validators | Already used by `LoginScreen`/`RegisterScreen` [VERIFIED: app/lib/routes/register_screen.dart] |
| `flutter_riverpod` | `3.3.1` (already installed) | State (`authProvider`, `settingsProvider`, `toastProvider`, `apiProvider`) | Project standard [VERIFIED: CLAUDE.md] |
| `dio` | `5.9.2` (already installed) | HTTP client + multipart `FormData` | Project standard [VERIFIED: CLAUDE.md] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `font_awesome_flutter` | `11.0.0` (already installed) | `FaIcon` for `WandererTextField` icons + toast icons | ListTile leading icons, toast `FontAwesomeIcons.circleExclamation` |
| `go_router` | `17.2.1` (already installed) | `context.pop()` back nav; redirect guard pushes `/login` after delete | AppBar back button + post-delete redirect |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `image_picker` | `file_picker`, `wechat_assets_picker` | `image_picker` is the official, minimal-permission, gallery-or-camera package; alternatives add scope (multi-select, custom UI) not needed here. CONTEXT D-01 locks `image_picker`. |
| `WandererTextField` (sheets) | raw `TextField` | CONTEXT D-07/D-10 + UI-SPEC mandate `WandererTextField` for sheet fields (built-in error display, theme styling). Bio uses raw multiline `TextField` per D-04. |

**Installation:**
```bash
cd app
flutter pub add image_picker
flutter pub get
```

**Version verification:** `image_picker` latest stable is **1.2.2** (published ~52 days before 2026-06-20), iOS 13+, Android SDK 24+ [CITED: pub.dev/packages/image_picker]. The app's `environment: sdk: ^3.11.5` and existing Flutter Material baseline satisfy these constraints. Run `flutter pub add image_picker` to let pub resolve the exact compatible constraint rather than pinning by hand.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| `image_picker` | pub.dev | mature (years) | very high (Flutter-team package) | github.com/flutter/packages | n/a (Dart ecosystem; flutter.dev verified publisher) | Approved |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

> `slopcheck` is oriented at the npm/PyPI hallucination surface and does not cover pub.dev; legitimacy here is established by the **flutter.dev verified publisher** badge on pub.dev and the canonical source repo `github.com/flutter/packages` [CITED: pub.dev/packages/image_picker]. This is the single most widely used image-selection package in the Flutter ecosystem. Tag: `[VERIFIED: pub.dev — flutter.dev verified publisher]`.

## Architecture Patterns

### System Architecture Diagram

```
SettingsAccountScreen (ConsumerWidget)
  │
  ├─ watch(authProvider) ──► UserEntity (id, email, avatar, serverUrl, preferredUsername)
  ├─ watch(settingsProvider) ──► Settings (bio)
  │
  ├─[Avatar row]
  │   tap ─► ImagePicker.pickImage(gallery)
  │         └─ XFile ─► Dio FormData{'avatar': MultipartFile} ─► POST /api/v1/user/{id}/file
  │               success ─► authProvider.notifier.refresh() ─► GET /user/{id}?expand=...
  │                            └─ re-watch ─► CircleAvatar(getFileUrl(serverUrl, avatar))
  │               error   ─► toastProvider.add(error)
  │
  ├─[Bio section]
  │   TextField(multiline) ─► local controller text
  │     Save (enabled iff text != Settings.bio)
  │       └─ settingsProvider.saveToServer(settings.copyWith(bio: text))
  │            └─ POST /api/v1/settings/{id} ─► ObjectBox + state update
  │            success/error ─► toast
  │
  ├─[ListTile "Change email"] ─► showModalBottomSheet
  │     FormBuilder{ WandererTextField(email), WandererTextField(currentPassword, isPassword) }
  │       saveAndValidate() ─► POST /api/v1/user/{id}/email {email, currentPassword}
  │         (Go: ValidatePassword(currentPassword) → Auth.Set email → re-token)
  │         success ─► pop sheet + toast(email_updated) + authProvider.refresh()
  │         error   ─► toast(api message), sheet stays open
  │
  ├─[ListTile "Change password"] ─► showModalBottomSheet
  │     FormBuilder{ WandererTextField(oldPassword, isPassword), WandererTextField(password, isPassword) }
  │       saveAndValidate() ─► POST /api/v1/user/{id} {oldPassword, password, passwordConfirm}
  │         success ─► pop sheet + toast(new_password_success)
  │         error   ─► toast(error_updating_password), sheet stays open
  │
  └─[ListTile "Delete account" (error color)] ─► AlertDialog(confirm)
        Delete ─► DELETE /api/v1/user/{id}
          success ─► authProvider.notifier.logout() ─► go_router redirect ─► /login
```

### Recommended Project Structure
```
app/lib/
├── routes/
│   └── settings_account_screen.dart   # fill the existing stub (ConsumerWidget)
├── provider/
│   └── auth_provider.dart             # ADD a public refresh() method (see Pattern 4)
└── components/
    └── settings/                      # OPTIONAL: extract email/password sheet widgets here
        ├── email_change_sheet.dart    #   (mirrors web/src/lib/components/settings/email_modal.svelte)
        └── password_change_sheet.dart
```
> Extracting the two sheets into their own widget files is discretionary but matches the web layout (`web/src/lib/components/settings/email_modal.svelte`, `password_modal.svelte`) and keeps `settings_account_screen.dart` readable.

### Pattern 1: Canonical settings screen shell (copy from Privacy screen)
**What:** `ConsumerWidget` → `Scaffold` → `AppBar(leading: back → context.pop())` → `ListView`, with a `_sectionHeader(context, label)` helper.
**When to use:** The whole screen skeleton.
**Example:**
```dart
// Source: app/lib/routes/settings_privacy_screen.dart [VERIFIED: codebase]
Widget _sectionHeader(BuildContext context, String label) {
  final colorScheme = Theme.of(context).colorScheme;
  return Padding(
    padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
    child: Text(label,
      style: Theme.of(context).textTheme.titleSmall
        ?.copyWith(color: colorScheme.onSurfaceVariant)),
  );
}
// Scaffold(appBar: AppBar(leading: IconButton(Icons.arrow_back, onPressed: () => context.pop()),
//   title: Text(l10n.account)), body: ListView(children: [...]))
```

### Pattern 2: Toast feedback (copy from Privacy `_save`)
**What:** Provider calls (`saveToServer`, raw Dio) have NO internal error handling; wrap in try/catch and surface a `ToastMessage`.
**Example:**
```dart
// Source: app/lib/routes/settings_privacy_screen.dart [VERIFIED: codebase]
try {
  await ref.read(settingsProvider.notifier).saveToServer(updated);
} catch (_) {
  ref.read(toastProvider.notifier).add(ToastMessage(
    type: ToastType.error,
    icon: FontAwesomeIcons.circleExclamation,
    text: l10n.error_saving_settings, // or a specific key per action
  ));
}
```

### Pattern 3: FormBuilder + WandererTextField + validators (copy from Login/Register)
**What:** `FormBuilder(key: _formKey, child: Column([WandererTextField(...), ...]))`, submit via `_formKey.currentState!.saveAndValidate()`, read values from `_formKey.currentState!.value['fieldName']`.
**Example:**
```dart
// Source: app/lib/routes/register_screen.dart [VERIFIED: codebase]
WandererTextField(
  name: 'email',
  label: l10n.email,
  isPassword: false,
  validator: FormBuilderValidators.compose([
    FormBuilderValidators.required(),
    FormBuilderValidators.email(),
  ]),
),
WandererTextField(
  name: 'currentPassword',
  label: l10n.current_password,
  isPassword: true,
  validator: FormBuilderValidators.required(),
),
// on submit:
if (_formKey.currentState?.saveAndValidate() ?? false) {
  final v = _formKey.currentState!.value;
  // v['email'], v['currentPassword']
}
```

### Pattern 4: Public auth refresh after credential/avatar change (NEW — required)
**What:** After email or avatar change, the displayed `UserEntity` must be re-fetched WITH the expanded actor + settings. The existing fetch method `_updateUserEntity` is **private**, and you cannot build a `UserEntity` from a bare POST response because `User.toEntity()` throws without `expand.actor`.
**When to use:** After successful email change and avatar upload.
**Example (add to `Auth` notifier):**
```dart
// Source: derived from app/lib/provider/auth_provider.dart [VERIFIED: codebase]
// Option A (simplest): expose a public refresh that re-runs build()
Future<void> refresh() async {
  final id = state.value?.id;
  if (id == null) return;
  state = await AsyncValue.guard(() => _updateUserEntity(id));
}
// Option B: ref.invalidate(authProvider) from the screen — but build() reads from
//           ObjectBox first, so a network re-fetch (Option A) is more reliable for
//           reflecting the just-changed email/avatar immediately.
```
> Note: `_updateUserEntity` already calls `GET /user/{id}?expand=activitypub_actors_via_user,settings_via_user`, puts the entity in ObjectBox, and syncs settings. Reusing it is the correct path.

### Pattern 5: Avatar URL + dicebear fallback
**What:** `UserEntity` mixes in `RecordFunctions`; build the avatar URL from the stored filename + server URL. Fall back to dicebear initials.
**Example:**
```dart
// Source: app/lib/models/record.dart + app/lib/routes/profile_screen.dart [VERIFIED: codebase]
final avatarUrl = user.getFileUrl(user.serverUrl, user.avatar); // null if no avatar
CircleAvatar(
  radius: 40,
  backgroundColor: Colors.grey.shade300,
  backgroundImage: NetworkImage(
    avatarUrl ??
      'https://api.dicebear.com/7.x/initials/png?seed=${user.preferredUsername}&backgroundType=gradientLinear',
  ),
  onBackgroundImageError: (e, _) {},
)
// getFileUrl returns: '$serverUrl/api/v1/files/$collectionId/$id/$filename'
```

### Pattern 6: Multipart avatar upload via Dio
**What:** Build `FormData` with the `'avatar'` field (multipart field name confirmed in `file/+server.ts` upload + web `users_update`).
**Example:**
```dart
// Source: web user_store.ts field name + Dio docs [VERIFIED: codebase for field name]
final picker = ImagePicker();
final XFile? picked = await picker.pickImage(source: ImageSource.gallery);
if (picked == null) return;
final formData = FormData.fromMap({
  'avatar': await MultipartFile.fromFile(picked.path),
});
await ref.read(apiProvider).post('/user/$userId/file', data: formData);
await ref.read(authProvider.notifier).refresh();
```

### Anti-Patterns to Avoid
- **Saving bio through the user endpoint.** Bio is a `Settings` field — use `settingsProvider`, not `/user/{id}`. (See Pitfall 1.)
- **Rebuilding `UserEntity` from a POST response.** `User.toEntity()` throws without `expand.actor`. Re-fetch instead. (See Pitfall 3.)
- **Hardcoding hex colors / font sizes.** UI-SPEC: everything derives from `Theme.of(context)`; the only allowed explicit color is the destructive red on the delete row.
- **Using `Column` for the body.** Use `ListView` so it scrolls with the keyboard open (matches Privacy/Language screens).
- **Adding API-token UI.** The web account screen has API-token modals; ACCT-F01 is deferred/out-of-scope. Do not port it.
- **Camera permission.** D-02 is gallery-only; do NOT request camera. (iOS still needs the photo-library plist key — see Environment Availability.)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Form validation + error display | Custom validators / manual error text | `FormBuilderValidators` + `WandererTextField` (built-in error UI) | Already standardized; `.email()` handles RFC edge cases |
| Toast / snackbar feedback | Raw `ScaffoldMessenger` everywhere | `toastProvider` + `ToastMessage` | App-wide overlay already wired (`toast_overlay.dart`) |
| Image selection from gallery | Platform channels / intents | `image_picker` | Handles iOS/Android permission flows, scoped storage, EXIF |
| Avatar URL construction | String-concatenated PocketBase URL | `RecordFunctions.getFileUrl` | Handles absolute-URL passthrough + thumb param |
| Settings persistence + cache | New persistence layer | `settingsProvider.saveToServer` | Already POSTs + writes ObjectBox + updates state |
| Post-change user re-fetch | Manual JSON → entity | `Auth._updateUserEntity` (via new public `refresh()`) | Already requests the correct `expand` and writes ObjectBox |

**Key insight:** This phase is ~90% wiring of existing primitives. The only genuinely new code is: the screen layout, the two bottom-sheet forms, the `image_picker` call, and a small public `refresh()` on the `Auth` notifier.

## Runtime State Inventory

> This is a UI-build phase, not a rename/migration. The categories below are checked for completeness because the phase mutates stored user state (avatar, email, bio).

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | ObjectBox `UserEntity` (avatar, email) and `SettingsEntity` (bio) caches. After a change, the cached copy is stale until re-fetched. | `authProvider.refresh()` re-writes `UserEntity`; `settingsProvider.saveToServer` re-writes `SettingsEntity`. Both already handle ObjectBox writes. |
| Live service config | None — no external service config carries account state. | None — verified by reviewing endpoints (SvelteKit + Go only). |
| OS-registered state | iOS `Info.plist` must gain `NSPhotoLibraryUsageDescription` for `image_picker` gallery access (App Store policy). | Add the plist key (see Environment Availability). Android needs no manifest change (scoped storage). |
| Secrets/env vars | None new. | None. |
| Build artifacts | Adding `image_picker` regenerates `pubspec.lock` and may add native iOS pods (`pod install`). | Run `flutter pub get`; on iOS run pod install (usually automatic via Flutter build). |

**Nothing found in category:** Live service config and Secrets — verified by reading all five `/api/v1/user*` and `/api/v1/settings*` endpoints; none reference external service configuration or secrets beyond PocketBase auth already in place.

## Common Pitfalls

### Pitfall 1: Bio saved through the wrong endpoint
**What goes wrong:** D-11/canonical_refs in CONTEXT.md describe `POST /api/v1/user/{id}` as "update user (bio, password)". Sending `bio` there will NOT persist it — the user `UserUpdateSchema` only accepts `username/email/password/oldPassword/passwordConfirm`, and bio lives in the `settings` collection.
**Why it happens:** On the web, the *User model* has a `bio` field too, blurring the distinction; on mobile, bio is exclusively a `Settings` field.
**How to avoid:** Save bio via `settingsProvider.notifier.saveToServer(settings.copyWith(bio: text))` → `POST /api/v1/settings/{id}`. D-15 already states bio lives in `settingsProvider` — follow D-15, not the canonical_refs note.
**Warning signs:** Bio appears to save (no error) but is gone after reopening the screen.

### Pitfall 2: Password change missing `oldPassword`
**What goes wrong:** D-11 says send `{ password, passwordConfirm }` and claims "the server re-auths with the current password." The SvelteKit handler re-auths with the *new* password (not a verification of the old one). PocketBase's native users-collection password-change rule requires `oldPassword`; without it the update is rejected (400) for a non-superuser. The web client sends `oldPassword` and the schema accepts it.
**Why it happens:** D-11 conflated the email-change server validation (which DOES check current password) with password change.
**How to avoid:** Send `{ oldPassword, password, passwordConfirm }`. Collect the current password in the password sheet (D-10 already specifies a "current password" field — wire it to `oldPassword`). Map the new-password field to both `password` and `passwordConfirm`.
**Warning signs:** Password change returns 400 "Missing old password" / validation error even with valid input.

### Pitfall 3: Refreshing displayed email/avatar after a change
**What goes wrong:** Trying to update `authProvider` by deserializing the email/avatar POST response throws — `User.toEntity()` requires `expand.actor`, which the POST response lacks.
**Why it happens:** The entity model couples `UserEntity` creation to an expanded ActivityPub actor.
**How to avoid:** Add a public `refresh()` to the `Auth` notifier that calls the existing private `_updateUserEntity(id)` (which fetches with the correct `expand`). Call it after successful avatar upload and email change. (Bio uses `settingsProvider`, which refreshes itself.)
**Warning signs:** Crash/exception in toEntity, or the new email/avatar not showing until app restart.

### Pitfall 4: iOS photo-library permission crash
**What goes wrong:** On iOS, calling `pickImage(gallery)` without `NSPhotoLibraryUsageDescription` in `Info.plist` crashes the app at runtime.
**Why it happens:** iOS requires a usage-description string for photo access; App Store policy mandates the key even when `requestFullMetadata: false`.
**How to avoid:** Add `NSPhotoLibraryUsageDescription` to `app/ios/Runner/Info.plist` with a user-facing reason. (Android needs no change — scoped storage.)
**Warning signs:** App terminates the moment the gallery picker is invoked on iOS.

### Pitfall 5: Bio Save button "changed" detection
**What goes wrong:** Comparing a `TextEditingController` value to `Settings.bio` naively (e.g. `null` vs `''`) leaves Save enabled when nothing changed, or disabled when it should be enabled.
**Why it happens:** `Settings.bio` is nullable; an empty field and a null bio are equivalent for "unchanged."
**How to avoid:** Normalize: treat `null` and `''` as equal; enable Save iff `controller.text != (settings.bio ?? '')`. Re-evaluate on each `onChanged`.
**Warning signs:** Save stays greyed after editing, or stays active after a successful save.

### Pitfall 6: Bottom sheet hidden behind keyboard
**What goes wrong:** The submit button in the email/password sheet is covered by the on-screen keyboard.
**How to avoid:** Use `isScrollControlled: true` on `showModalBottomSheet` and add `MediaQuery.of(context).viewInsets.bottom` to the sheet's bottom padding (UI-SPEC spacing rule).
**Warning signs:** User can't reach the submit button when typing.

## Code Examples

### Account-deletion confirm dialog (ACCT-05)
```dart
// Source: web ConfirmModal usage + Flutter AlertDialog [CITED: D-12 + Flutter docs]
final confirmed = await showDialog<bool>(
  context: context,
  builder: (ctx) => AlertDialog(
    title: Text(l10n.confirm_deletion),
    content: Text(l10n.account_delete_confirm),
    actions: [
      TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.cancel)),
      TextButton(
        onPressed: () => Navigator.pop(ctx, true),
        child: Text(l10n.delete_account,
          style: TextStyle(color: Colors.red.shade400)),
      ),
    ],
  ),
);
if (confirmed == true) {
  await ref.read(apiProvider).delete('/user/$userId');
  await ref.read(authProvider.notifier).logout(); // go_router redirect → /login
}
```

### Destructive delete ListTile
```dart
// Source: UI-SPEC color rule [CITED: D-12/D-14]
ListTile(
  leading: Icon(Icons.delete_outline, color: Colors.red.shade400),
  title: Text(l10n.delete_account, style: TextStyle(color: Colors.red.shade400)),
  onTap: () { /* show confirm dialog */ },
)
```
> The theme maps `colorScheme.error` to a light-red background token (`#FEF2F2`) that is illegible as foreground text. Use a saturated red foreground (`Colors.red.shade400`, matching `WandererTextField`'s error color) and verify contrast in light + dark mode.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `image_picker` requiring `android:requestLegacyExternalStorage` | Scoped storage, no Android manifest change needed | image_picker 0.8.x+ | Android setup is zero-config [CITED: pub.dev/packages/image_picker] |
| iOS min 11/12 | image_picker 1.x requires iOS 13+ | image_picker 1.x | Fine — app already targets iOS 12+ per CLAUDE.md; confirm deployment target ≥13 if pub resolves 1.x |

**Deprecated/outdated:**
- The web account screen's API-token feature: out of scope (ACCT-F01 deferred). Ignore the `ApiTokenModal` references in `web/src/routes/settings/account/+page.svelte`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Password change requires `oldPassword` (PocketBase native rule), so D-11's `{password, passwordConfirm}`-only payload will fail | Pitfall 2 | If wrong, sending `oldPassword` is still harmless (schema accepts it). If right (likely), omitting it breaks ACCT-04. The web client sends it [VERIFIED: password_modal.svelte], strongly supporting the assumption. |
| A2 | iOS deployment target must be ≥13 if pub resolves `image_picker` 1.x | Environment Availability | Build/runtime failure on iOS if target < 13. Verify `ios/Podfile` / Xcode target during execution. |
| A3 | `image_picker` 1.2.2 is the version pub will resolve under Dart `^3.11.5` | Standard Stack | Low — `flutter pub add` resolves the correct constraint regardless; the pinned version is informational. |

## Open Questions

1. **Does `oldPassword` need to be in the password-change payload?** (drives ACCT-04)
   - What we know: web sends `oldPassword`; `UserUpdateSchema` accepts it; PocketBase normally enforces it for password changes; the SvelteKit handler itself does not check it but PocketBase's collection rule does.
   - What's unclear: whether this PocketBase instance has relaxed the password-change rule.
   - Recommendation: Include `oldPassword` (collect it via the D-10 "current password" field). It is harmless if unused and required if enforced.

2. **Does the password sheet's submit also need to re-auth/refresh state?** 
   - What we know: the SvelteKit handler re-auths and returns the record + new token; cookies update via the Dio cookie manager automatically.
   - Recommendation: No explicit `authProvider.refresh()` needed for password change (email/identity unchanged); rely on the cookie manager. Verify the user stays logged in after change during testing.

3. **iOS deployment target.** 
   - Recommendation: During execution, confirm `app/ios` deployment target ≥ 13; bump if pub resolves image_picker 1.x and the target is lower.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `image_picker` (pub.dev) | Avatar pick (ACCT-01) | ✗ (not yet added) | `^1.2.2` target | None — required; `flutter pub add image_picker` |
| `flutter_form_builder` | Email/password sheets | ✓ | `^10.3.0+2` | — |
| `form_builder_validators` | Form validation | ✓ | `^11.3.0` | — |
| `dio` (multipart) | Avatar upload, all API calls | ✓ | 5.9.2 | — |
| `toastProvider` | All feedback | ✓ | — | — |
| iOS `NSPhotoLibraryUsageDescription` | image_picker on iOS | ✗ (must add) | — | None — required for iOS gallery access |
| Backend `/api/v1/user/{id}`, `/email`, `/file`, `/settings/{id}`, `DELETE /user/{id}` | all five requirements | ✓ (confirmed in source) | — | — |

**Missing dependencies with no fallback:**
- `image_picker` package — add via `flutter pub add image_picker`.
- iOS `NSPhotoLibraryUsageDescription` in `app/ios/Runner/Info.plist` — required or the app crashes on gallery access.

**Missing dependencies with fallback:**
- None.

## Project Constraints (from CLAUDE.md)

- **Tech stack lock:** Flutter + Riverpod (`riverpod_annotation` codegen) + go_router + flutter_map + freezed — follow existing patterns. (This phase: Riverpod + go_router + freezed `Settings`/`UserEntity`.)
- **HTTP via existing Dio client** (`apiProvider`) — do not introduce a new HTTP client.
- **No breaking changes** to existing screens, bottom nav, or routes. (Adding a public `refresh()` to `Auth` is additive; verify it doesn't alter `build()` behavior.)
- **Naming:** Flutter screens PascalCase (`SettingsAccountScreen`); provider files `_provider.dart` → `.g.dart`; Dart camelCase; private members `_`-prefixed.
- **Riverpod:** `@riverpod` annotation; `AsyncValue.guard()` for exception wrapping; access via `ref.watch/read`.
- **Error handling:** wrap async in try/catch; surface via toast (no throwing to UI). Use `.value` (not `.valueOrNull`) for nullable `AsyncValue` access (per user MEMORY).
- **i18n:** every user-visible string is an `l10n.*` key; no hardcoded display strings. All required keys confirmed present (see below).
- **Code generation:** after editing `auth_provider.dart` (annotated), run `dart run build_runner build` to regenerate `.g.dart`. (Note: `refresh()` is a plain method on the existing class — codegen still must run if any annotation surface changes; safe to run regardless.)
- **GSD workflow:** edits must go through a GSD command (this is the planned-phase path).

## Security Domain

`security_enforcement: true`, `security_asvs_level: 1`, `security_block_on: high`.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Email/password changes go through PocketBase auth; current password validated server-side (email: Go `ValidatePassword`; password: PocketBase `oldPassword` rule). Do NOT add client-side-only credential logic. |
| V3 Session Management | yes | Email change returns a new auth token; Dio cookie manager persists `pb_auth` cookie automatically. Verify session remains valid post-change; account delete must call `logout()` to clear the cookie jar + ObjectBox. |
| V4 Access Control | yes | Endpoints enforce `id === authStore.record.id` (403 otherwise) server-side. Client must send the authenticated user's own id (`authProvider.value.id`). |
| V5 Input Validation | yes | `FormBuilderValidators.email()` + `.required()` client-side; Zod (`UserUpdateSchema`) + Go validation server-side. Client validation is UX-only; server is authoritative. |
| V6 Cryptography | no (delegated) | Password hashing/token signing handled entirely by PocketBase (`JWT Go`, bcrypt). Never handle raw password storage client-side. |
| V7 Error Handling/Logging | yes | Surface server error messages via toast WITHOUT leaking internals; for email failure show the API message, for password show generic `error_updating_password` (D-11). |

### Known Threat Patterns for Flutter + PocketBase account management

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Account takeover via email change without password | Spoofing/Elevation | Require + server-validate `currentPassword` (confirmed enforced in Go `user_email_change.go`) |
| Password change without proving current password | Elevation | Send `oldPassword`; PocketBase rule enforces it (see Pitfall 2) |
| IDOR — editing another user's account | Tampering | Server checks `id === authStore.record.id` (403); send own id only |
| Stale session after credential change | Session | Persist new token via cookie manager; on delete, `logout()` clears jar + cache |
| Accidental/unauthorized deletion | Repudiation/Tampering | Explicit `AlertDialog` confirm gate before `DELETE` (ACCT-05) |
| Sensitive data in logs | Info Disclosure | Never `print()` passwords/tokens; toast shows safe messages only |

**Block-on-high check:** No high-severity gaps if the three pitfalls (bio endpoint, `oldPassword`, server-side id enforcement) are honored. The server already enforces access control and password validation — the client must not weaken these.

## Sources

### Primary (HIGH confidence)
- Codebase (read this session) — `settings_account_screen.dart` (stub), `settings_privacy_screen.dart` (canonical shell + toast pattern), `login_screen.dart` / `register_screen.dart` (FormBuilder + validators), `wanderer_text_field.dart`, `auth_provider.dart`, `settings_provider.dart`, `api_provider.dart`, `record.dart` (`getFileUrl`), `user_entity.dart`, `user.dart` (`toEntity` actor requirement), `settings.dart` (`bio`), `profile_screen.dart` (dicebear fallback), `toast_provider.dart`, `app_en.arb` (ARB keys), `pubspec.yaml`.
- SvelteKit API source — `user/[id]/+server.ts`, `user/[id]/email/+server.ts`, `user/[id]/file/+server.ts`, `settings/[id]/+server.ts`, `user_schema.ts`, `web/src/lib/stores/user_store.ts`, `password_modal.svelte`.
- Go backend — `db/routes/user_email_change.go` (current-password validation confirmed).

### Secondary (MEDIUM confidence)
- pub.dev/packages/image_picker — version 1.2.2, iOS 13+/Android SDK 24+, Info.plist requirements, verified publisher.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every package except `image_picker` is already installed and in use; `image_picker` verified on pub.dev.
- Architecture: HIGH — screen shell, form, toast, and persistence patterns all copied from existing, verified screens.
- Endpoints: HIGH — all five verified against live SvelteKit + Go source.
- Pitfalls: HIGH (1, 3, 4, 5, 6) / MEDIUM (2 — `oldPassword` requirement inferred from web client + PocketBase norms, not directly executed).

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable; the only volatile item is the `image_picker` version, which `flutter pub add` resolves automatically)
