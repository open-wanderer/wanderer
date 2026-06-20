# Phase 8: Account & Profile - Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 5 (1 fill-stub, 1 modify, 2 optional new, 1 config)
**Analogs found:** 5 / 5 (every file has a strong in-repo analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `app/lib/routes/settings_account_screen.dart` (fill stub) | screen (ConsumerWidget) | request-response (CRUD over Dio) | `app/lib/routes/settings_privacy_screen.dart` | exact (settings screen shell + toast) |
| `app/lib/provider/auth_provider.dart` (modify: add public `refresh()`) | provider (Riverpod notifier) | request-response (fetch + ObjectBox) | self — extend existing `_updateUserEntity` | exact (same file) |
| `app/lib/components/settings/email_change_sheet.dart` (optional new) | component (form sheet) | request-response (POST) | `app/lib/routes/register_screen.dart` | role-match (FormBuilder + WandererTextField) |
| `app/lib/components/settings/password_change_sheet.dart` (optional new) | component (form sheet) | request-response (POST) | `app/lib/routes/register_screen.dart` | role-match (FormBuilder + WandererTextField) |
| `app/pubspec.yaml` + `app/ios/Runner/Info.plist` (modify) | config | n/a | existing pubspec / plist | config-edit |

> Sheet extraction (rows 3–4) is discretionary per RESEARCH §Recommended Project Structure. If kept inline in the screen, they still copy the `register_screen.dart` form pattern below.

## Pattern Assignments

### `app/lib/routes/settings_account_screen.dart` (screen, request-response)

**Analog:** `app/lib/routes/settings_privacy_screen.dart`

**Imports pattern** (`settings_privacy_screen.dart` lines 1-8) — copy and extend with `auth_provider`, `api_provider`, `wanderer_text_field`, `flutter_form_builder`, `form_builder_validators`, `image_picker`, `dio`, `record.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
```

**Scaffold + AppBar + ListView shell** (`settings_privacy_screen.dart` lines 60-68) — copy verbatim, swap title to `l10n.account`:
```dart
return Scaffold(
  appBar: AppBar(
    leading: IconButton(
      icon: const Icon(Icons.arrow_back),
      onPressed: () => context.pop(),
    ),
    title: Text(l10n.account),
  ),
  body: ListView(children: [ /* avatar row, bio, ListTiles */ ]),
);
```

**Section header helper** (`settings_privacy_screen.dart` lines 38-49) — copy verbatim:
```dart
Widget _sectionHeader(BuildContext context, String label) {
  final colorScheme = Theme.of(context).colorScheme;
  return Padding(
    padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
    child: Text(label,
      style: Theme.of(context).textTheme.titleSmall
        ?.copyWith(color: colorScheme.onSurfaceVariant)),
  );
}
```

**Toast-wrapped save / error pattern** (`settings_privacy_screen.dart` lines 16-34) — copy structure for bio save and every API call:
```dart
try {
  await ref.read(settingsProvider.notifier).saveToServer(updated);
} catch (_) {
  ref.read(toastProvider.notifier).add(ToastMessage(
    type: ToastType.error,
    icon: FontAwesomeIcons.circleExclamation,
    text: l10n.error_saving_settings,
  ));
}
```

**Avatar display (CircleAvatar + dicebear fallback)** — from `profile_screen.dart` lines 124-133 + `record.dart` `getFileUrl`:
```dart
final user = ref.watch(authProvider).value;
final avatarUrl = user?.getFileUrl(user.serverUrl, user.avatar); // null if none
CircleAvatar(
  radius: 40,
  backgroundColor: Colors.grey.shade300,
  backgroundImage: NetworkImage(
    avatarUrl ??
      'https://api.dicebear.com/7.x/initials/png?seed=${user?.preferredUsername}&backgroundType=gradientLinear',
  ),
  onBackgroundImageError: (e, _) {},
)
```
> `getFileUrl` (`record.dart` lines 13-27) returns `'$baseUrl/api/v1/files/$collectionId/$id/$filename'` or null. `UserEntity` mixes in `RecordFunctions` (`user_entity.dart` line 6), so call it directly on the entity.

**Avatar multipart upload (Dio)** — `apiProvider` is the Dio client (`api_provider.dart`); auto-upload on pick (D-03):
```dart
final picked = await ImagePicker().pickImage(source: ImageSource.gallery);
if (picked == null) return;
final formData = FormData.fromMap({
  'avatar': await MultipartFile.fromFile(picked.path),
});
await ref.read(apiProvider).post('/user/${user.id}/file', data: formData);
await ref.read(authProvider.notifier).refresh(); // re-fetch expanded user
```

**Bio save** — bio lives in `Settings`, not `/user` (Pitfall 1). Use `settingsProvider.saveToServer` (`settings_provider.dart` lines 33-39):
```dart
await ref.read(settingsProvider.notifier)
    .saveToServer(settings.copyWith(bio: controller.text));
// Save enabled iff controller.text != (settings.bio ?? '')  -- Pitfall 5
```

**Delete account (AlertDialog + logout)** — from RESEARCH §Code Examples + `auth_provider.logout` (lines 112-118):
```dart
final confirmed = await showDialog<bool>(context: context, builder: (ctx) =>
  AlertDialog(
    title: Text(l10n.confirm_deletion),
    content: Text(l10n.account_delete_confirm),
    actions: [
      TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.cancel)),
      TextButton(onPressed: () => Navigator.pop(ctx, true),
        child: Text(l10n.delete_account, style: TextStyle(color: Colors.red.shade400))),
    ],
  ));
if (confirmed == true) {
  await ref.read(apiProvider).delete('/user/${user.id}');
  await ref.read(authProvider.notifier).logout(); // go_router redirect → /login
}
```

---

### `app/lib/provider/auth_provider.dart` (provider — MODIFY: add public `refresh()`)

**Analog:** self. Reuse the existing private `_updateUserEntity` (lines 120-142), which already GETs `/user/$id?expand=activitypub_actors_via_user, settings_via_user`, builds the `UserEntity`, syncs settings, and writes ObjectBox.

**Add this public method** (Pattern 4 / Pitfall 3 — do NOT rebuild `UserEntity` from a bare POST response; `User.toEntity()` throws without `expand.actor`):
```dart
Future<void> refresh() async {
  final id = state.value?.id;
  if (id == null) return;
  state = await AsyncValue.guard(() => _updateUserEntity(id));
}
```
> Class is `@Riverpod(keepAlive: true) class Auth extends _$Auth` (lines 14-15). `refresh()` is a plain method — additive, no `build()` change. Run `dart run build_runner build` after editing per CLAUDE.md (codegen-annotated file).
> Existing `AsyncValue.guard` pattern already used in `register`/`login` (lines 67, 96). Access nullable value with `.value` (per user MEMORY, not `.valueOrNull`).

---

### `app/lib/components/settings/{email,password}_change_sheet.dart` (component, request-response)

**Analog:** `app/lib/routes/register_screen.dart`

**FormBuilder + WandererTextField + validators** (`register_screen.dart` lines 20, 69-100):
```dart
final _formKey = GlobalKey<FormBuilderState>();
// ...
FormBuilder(
  key: _formKey,
  autovalidateMode: AutovalidateMode.onUnfocus,
  child: Column(spacing: 12, children: [
    WandererTextField(
      name: 'email',
      label: l10n.email,
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
  ]),
)
```
> `WandererTextField` (`wanderer_text_field.dart` lines 5-22) extends `FormBuilderField<String>`; params: `name` (required), `validator`, `initialValue`, `label`, `placeholder`, `icon`, `disabled`, `isPassword`. Built-in error display (lines 106-113).

**Submit / read values** (`register_screen.dart` lines 118-128):
```dart
if (_formKey.currentState?.saveAndValidate() ?? false) {
  final v = _formKey.currentState!.value;
  // email sheet:    v['email'], v['currentPassword']
  // password sheet: v['currentPassword'] (-> oldPassword), v['newPassword']
}
```

**Email change submit (D-08)** — `POST /api/v1/user/{id}/email`:
```dart
await ref.read(apiProvider).post('/user/$userId/email',
    data: {'email': v['email'], 'currentPassword': v['currentPassword']});
Navigator.of(context).pop();
ref.read(toastProvider.notifier).add(ToastMessage(
  type: ToastType.success, icon: FontAwesomeIcons.check, text: l10n.email_updated));
await ref.read(authProvider.notifier).refresh(); // refresh displayed email
```

**Password change submit (D-11 + Pitfall 2)** — include `oldPassword`; `POST /api/v1/user/{id}`:
```dart
await ref.read(apiProvider).post('/user/$userId', data: {
  'oldPassword': v['currentPassword'],
  'password': v['newPassword'],
  'passwordConfirm': v['newPassword'],
});
// success -> pop + toast(new_password_success); error -> toast(error_updating_password)
```
> The web client sends `oldPassword` and PocketBase's native rule enforces it. D-11's `{password, passwordConfirm}`-only payload will 400. No `authProvider.refresh()` needed for password change (identity unchanged; cookie manager handles the new token).

**Bottom sheet host (Pitfall 6)** — `showModalBottomSheet(isScrollControlled: true, ...)` and pad with `MediaQuery.of(context).viewInsets.bottom`. Use a `ConsumerStatefulWidget` for the sheet (holds `_formKey`). Error toast surfaces the API message and keeps the sheet open (do not pop on error).

---

## Shared Patterns

### Toast feedback (all actions)
**Source:** `settings_privacy_screen.dart` lines 16-34; `toast_provider.dart` lines 8-33
**Apply to:** every API call (avatar, bio, email, password, delete)
```dart
ref.read(toastProvider.notifier).add(ToastMessage(
  type: ToastType.error,            // or .success
  icon: FontAwesomeIcons.circleExclamation,
  text: l10n.<key>,
));
```

### Provider calls have no internal error handling
**Source:** `settings_provider.dart` (saveToServer throws), `api_provider.dart` (raw Dio)
**Apply to:** all five operations — wrap each in try/catch at the screen/sheet layer.

### Riverpod access conventions
**Source:** CLAUDE.md + `auth_provider.dart`
**Apply to:** screen + sheets — `ref.watch(authProvider)` / `ref.watch(settingsProvider)` for display; `ref.read(...notifier)` for mutations; `AsyncValue.guard` inside notifiers; `.value` for nullable AsyncValue (per MEMORY).

### Current-user id source of truth
**Source:** D-15 — `ref.watch(authProvider).value` → `UserEntity` (id, email, avatar, serverUrl, preferredUsername). Always send the authenticated user's own id (V4 access control; server returns 403 otherwise).

## No Analog Found

None. Every file maps to an existing in-repo pattern.

| File | Notes |
|------|-------|
| `app/ios/Runner/Info.plist` | Add `NSPhotoLibraryUsageDescription` (Pitfall 4) — config edit, no code analog needed. |

## Metadata

**Analog search scope:** `app/lib/routes/`, `app/lib/provider/`, `app/lib/components/base/`, `app/lib/models/`, `app/lib/entities/`
**Files scanned:** settings_privacy_screen.dart, register_screen.dart, settings_account_screen.dart (stub), auth_provider.dart, settings_provider.dart, api_provider.dart, toast_provider.dart, wanderer_text_field.dart, record.dart, user_entity.dart, profile_screen.dart
**Pattern extraction date:** 2026-06-20
