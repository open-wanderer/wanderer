# Milestones

## v1.2 Settings Screens (Shipped: 2026-06-29)

**Phases completed:** 4 phases, 9 plans, 12 tasks

**Key accomplishments:**

- Five-row settings list (Account/Privacy/Language/Notifications/Appearance) wired to /settings sub-routes, with a full 14-locale RadioGroup<Language> + metric/imperial switch screen that auto-saves to the server, plus two themed stub screens.
- Every format_util call site (14 files, ~50 calls) now reads the live metric/imperial preference from unitProvider, so toggling units in settings re-renders distances, elevations, and speeds across trail cards, lists, navigation, and the elevation profile — backed by new imperial conversion tests.
- 1. [Rule 3 - Blocking] Plural ARB getters require positional, not named, arguments
- Added image_picker ^1.2.2 dependency, iOS photo-library plist key, `AppLocalizations.account` l10n getter, and `Auth.refresh()` Riverpod notifier method as prerequisites for Plans 02 and 03.
- Two ConsumerStatefulWidget bottom-sheet forms for credential changes: EmailChangeSheet posts {email, currentPassword} to /user/$id/email and refreshes auth; PasswordChangeSheet posts {oldPassword, password, passwordConfirm} to /user/$id.
- Filled SettingsAccountScreen stub with all five ACCT sections: CircleAvatar gallery upload with multipart POST, change-aware bio TextField, email/password modal sheets, and AlertDialog-gated account deletion with logout.
- Filled the stub SettingsNotificationsScreen with 9 sections of independent Web/Email SwitchListTile toggles that auto-save to the server via the D-09 map-copy pattern, plus a new l10n.web ARB key and a widget test.

---
