# Phase 9: Notifications - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Fill the stub `SettingsNotificationsScreen` with two `SwitchListTile` rows per notification type — one for web delivery, one for email delivery — across all nine notification types. Each toggle auto-saves to the server via `settingsProvider.saveToServer()` on change.

Requirements covered: NOTIF-01, NOTIF-02, NOTIF-03, NOTIF-04, NOTIF-05, NOTIF-06, NOTIF-07, NOTIF-08, NOTIF-09

</domain>

<decisions>
## Implementation Decisions

### Row layout
- **D-01:** Two `SwitchListTile` rows per notification type (not a table with column headers). Each notification type has a section header showing its description, followed by a "Web" toggle row and an "Email" toggle row. 9 types × 2 rows = 18 `SwitchListTile`s total in a `ListView`.
- **D-02:** Section header style: `Padding + Text(titleSmall, color: onSurfaceVariant)` — identical to the Language & Units and Privacy screen section header pattern.
- **D-03:** `SwitchListTile` labels: "Web" row uses a new `"web": "Web"` ARB key (does not yet exist — must be added and regenerated). "Email" row uses the existing `"email": "Email"` ARB key.

### Notification descriptions (section headers)
- **D-04:** Section headers use the existing `settings_notification_*` ARB keys (e.g., `l10n.settings_notification_trail_comment` = "Someone left a comment on your trail"). These are the human-readable descriptions. The 9 keys all already exist in `app_en.arb`.

### Notification type ordering
- **D-05:** Follow the same display order as the web client:
  1. Trail comments (`trail_comment`)
  2. New followers (`new_follower`)
  3. Trail shares (`trail_share`)
  4. Trail likes (`trail_like`)
  5. List shares (`list_share`)
  6. Summit log creates (`summit_log_create`)
  7. Trail mentions (`trail_mention`)
  8. Comment mentions (`comment_mention`)
  9. Summit log mentions (`summit_log_mention`)

### Data model and defaults
- **D-06:** `Settings.notifications` is `Map<String, NotificationPreference>?` where keys are snake_case JSON values of `NotificationType` (e.g., `"trail_comment"`) and values have `{web: bool, email: bool}`. Access pattern: `settings?.notifications?[NotificationType.trailComment.name]?.web ?? true`.
- **D-07:** Default both web and email to `true` when no saved preference exists — matches the web client's `?? true` fallback. Applied per toggle: `settings?.notifications?['trail_comment']?.web ?? true`.

### Save pattern
- **D-08:** Auto-save on each toggle change — no Save button. Same pattern as Privacy and Language screens. Use `settingsProvider.saveToServer(settings.copyWith(notifications: updatedMap))` in a try/catch that surfaces `l10n.error_saving_settings` toast on failure.
- **D-09:** Build the updated notifications map by copying the existing map and updating only the changed key: `Map<String, NotificationPreference>.from(settings.notifications ?? {})..['trail_comment'] = existing.copyWith(web: value)`. Then `settings.copyWith(notifications: updatedMap)`.

### Screen structure
- **D-10:** `ConsumerWidget` → `Scaffold` → `AppBar(back via context.pop(), title: l10n.notifications)` → `ListView` body. No section separators between notification groups other than the section-header `Padding + Text` widgets.
- **D-11:** Screen reads settings via `ref.watch(settingsProvider)`. Access `AsyncValue<Settings?>` and handle loading/error states (or use `.valueOrNull` / `.value` per project convention — use `.value` per the feedback memory from prior phases).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Data model
- `app/lib/models/settings.dart` — `NotificationType` enum (9 values with `@JsonValue` snake_case annotations), `NotificationPreference` freezed class (`web: bool`, `email: bool`), `Settings` model with `Map<String, NotificationPreference>? notifications`
- `app/lib/models/settings.freezed.dart` — generated `.copyWith()` for `NotificationPreference`
- `app/lib/models/settings.g.dart` — JSON serialization for `NotificationPreference`

### Settings persistence
- `app/lib/provider/settings_provider.dart` — `SettingsNotifier.saveToServer(Settings)` and `updateFromServer()`; no changes needed here
- `app/lib/entities/settings_entity.dart` — `notificationsJson` field (JSON-encoded map); round-trip already implemented

### Screen pattern to follow
- `app/lib/routes/settings_privacy_screen.dart` — canonical model: `ConsumerWidget`, `_save` helper, section `Padding + Text(titleSmall)`, error toast pattern
- `app/lib/routes/settings_language_screen.dart` — `SwitchListTile` pattern for units toggle (simpler variant)

### Stub to fill
- `app/lib/routes/settings_notifications_screen.dart` — existing stub; Phase 9 replaces `SizedBox.shrink()` body

### Internationalisation
- `app/lib/i18n/app_en.arb` — all 9 `settings_notification_*` description keys exist; `"email": "Email"` exists; `"web"` key is MISSING and must be added
- Must run `flutter gen-l10n` after adding `"web"` key to regenerate `app_localizations.dart`

### Web reference (read for ordering and label parity)
- `web/src/routes/settings/notifications/+page.svelte` — web implementation; defines display order and uses `settings_notification_*` equivalent keys

### Requirements
- `.planning/REQUIREMENTS.md` — NOTIF-01 through NOTIF-09
- `.planning/ROADMAP.md` §Phase 9 — success criteria and phase goal

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `SwitchListTile` (Flutter Material) — used in `settings_language_screen.dart` for the metric/imperial toggle; same widget for web and email rows
- `settingsProvider` (`app/lib/provider/settings_provider.dart`) — already handles notifications field serialization through `settings_entity.dart`; no provider changes needed
- `_save` helper pattern from `settings_privacy_screen.dart` — copy verbatim, adapting for notifications map update

### Established Patterns
- Section header: `Padding(padding: EdgeInsets.fromLTRB(16, 16, 16, 8), child: Text(label, style: Theme.of(context).textTheme.titleSmall?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant)))` — from Language and Privacy screens
- Error toast on save failure: `ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(l10n.error_saving_settings)))` inside catch block
- AsyncValue access: use `.value` (not `.valueOrNull`) per project feedback

### Integration Points
- `settings_notifications_screen.dart` stub is already registered in go_router (`path: 'notifications'` under `/settings`) — no routing changes needed
- `app_en.arb` needs one new key: `"web": "Web"` — triggers `flutter gen-l10n` regeneration of all 14 localization files
- `NotificationType` enum's `.name` property gives camelCase (e.g., `trailComment`); the JSON value (`@JsonValue`) gives snake_case (`trail_comment`) — use the JSON value strings as map keys to match server expectations

</code_context>

<specifics>
## Specific Ideas

- User chose "two rows per notification type" layout explicitly — SwitchListTile pair per type, not a table or single inline row with two switches.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 9-Notifications*
*Context gathered: 2026-06-20*
