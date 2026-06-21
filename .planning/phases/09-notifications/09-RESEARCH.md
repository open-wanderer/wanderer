# Phase 9: Notifications - Research

**Researched:** 2026-06-20
**Domain:** Flutter settings UI (SwitchListTile toggles), Riverpod synchronous-settings persistence, Flutter gen-l10n
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Two `SwitchListTile` rows per notification type (not a table with column headers). Each notification type has a section header showing its description, followed by a "Web" toggle row and an "Email" toggle row. 9 types × 2 rows = 18 `SwitchListTile`s total in a `ListView`.
- **D-02:** Section header style: `Padding + Text(titleSmall, color: onSurfaceVariant)` — identical to the Language & Units and Privacy screen section header pattern.
- **D-03:** `SwitchListTile` labels: "Web" row uses a new `"web": "Web"` ARB key (does not yet exist — must be added and regenerated). "Email" row uses the existing `"email": "Email"` ARB key.
- **D-04:** Section headers use the existing `settings_notification_*` ARB keys (e.g., `l10n.settings_notification_trail_comment` = "Someone left a comment on your trail"). These are the human-readable descriptions. The 9 keys all already exist in `app_en.arb`.
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
- **D-06:** `Settings.notifications` is `Map<String, NotificationPreference>?` where keys are snake_case JSON values of `NotificationType` (e.g., `"trail_comment"`) and values have `{web: bool, email: bool}`. Access pattern: `settings?.notifications?[NotificationType.trailComment.name]?.web ?? true`.
- **D-07:** Default both web and email to `true` when no saved preference exists — matches the web client's `?? true` fallback. Applied per toggle: `settings?.notifications?['trail_comment']?.web ?? true`.
- **D-08:** Auto-save on each toggle change — no Save button. Same pattern as Privacy and Language screens. Use `settingsProvider.saveToServer(settings.copyWith(notifications: updatedMap))` in a try/catch that surfaces `l10n.error_saving_settings` toast on failure.
- **D-09:** Build the updated notifications map by copying the existing map and updating only the changed key: `Map<String, NotificationPreference>.from(settings.notifications ?? {})..['trail_comment'] = existing.copyWith(web: value)`. Then `settings.copyWith(notifications: updatedMap)`.
- **D-10:** `ConsumerWidget` → `Scaffold` → `AppBar(back via context.pop(), title: l10n.notifications)` → `ListView` body. No section separators between notification groups other than the section-header `Padding + Text` widgets.
- **D-11:** Screen reads settings via `ref.watch(settingsProvider)`. Access `AsyncValue<Settings?>` and handle loading/error states (or use `.valueOrNull` / `.value` per project convention — use `.value` per the feedback memory from prior phases).

> **⚠️ RESEARCH CORRECTION TO D-11 (verified against code, HIGH confidence):** `settingsProvider` does NOT expose `AsyncValue<Settings?>`. `SettingsNotifier.build()` returns `Settings?` **synchronously** (`return box.getAll().firstOrNull?.toModel();`). Therefore `ref.watch(settingsProvider)` yields a plain `Settings?` — there is no loading/error state, no `.value`, no `.valueOrNull`. Follow the Privacy and Language screens exactly: `final settings = ref.watch(settingsProvider);` then guard with `if (settings == null)`. The planner MUST plan to the corrected reading, not the literal D-11 text. See Pitfall 1.

### Claude's Discretion
- (none explicitly enumerated in CONTEXT — the layout, save, and access patterns are all locked above; remaining discretion is limited to test breadth and helper extraction, both governed by existing screen precedent)

### Deferred Ideas (OUT OF SCOPE)
- None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NOTIF-01 | Toggle web + email for trail comments | `trail_comment` key; `NotificationType.trailComment`; ARB `settings_notification_trail_comment` exists |
| NOTIF-02 | Toggle web + email for new followers | `new_follower`; `NotificationType.newFollower`; ARB `settings_notification_new_follower` exists |
| NOTIF-03 | Toggle web + email for trail shares | `trail_share`; `NotificationType.trailShare`; ARB `settings_notification_trail_share` exists |
| NOTIF-04 | Toggle web + email for trail likes | `trail_like`; `NotificationType.trailLike`; ARB `settings_notification_trail_like` exists |
| NOTIF-05 | Toggle web + email for list shares | `list_share`; `NotificationType.listShare`; ARB `settings_notification_list_share` exists |
| NOTIF-06 | Toggle web + email for summit log creates | `summit_log_create`; `NotificationType.summitLogCreate`; ARB `settings_notification_summit_log_create` exists |
| NOTIF-07 | Toggle web + email for trail mentions | `trail_mention`; `NotificationType.trailMention`; ARB `settings_notification_trail_mention` exists |
| NOTIF-08 | Toggle web + email for comment mentions | `comment_mention`; `NotificationType.commentMention`; ARB `settings_notification_comment_mention` exists |
| NOTIF-09 | Toggle web + email for summit log mentions | `summit_log_mention`; `NotificationType.summitLogMention`; ARB `settings_notification_summit_log_mention` exists |

All 9 requirements are satisfied by the same single-screen implementation. Each maps to one section header + two SwitchListTile rows. The persistence layer (`Map<String, NotificationPreference>`) already round-trips all 9 keys through `settings_entity.dart` and `saveToServer`.
</phase_requirements>

## Summary

Phase 9 is a **pure UI fill-in on fully pre-existing infrastructure**. The data model (`NotificationType` enum, `NotificationPreference` freezed class, `Settings.notifications` map), the persistence path (`settingsProvider.saveToServer` → POST `/settings/{id}` → `updateFromServer` → ObjectBox round-trip via `notificationsJson`), and the route registration (`/settings/notifications`) all already exist and are verified working. The phase replaces the `SizedBox.shrink()` body of the existing `SettingsNotificationsScreen` stub with a `ListView` of 9 section headers, each followed by two `SwitchListTile` rows (Web + Email).

The only non-UI work is adding **one** ARB key — `"web": "Web"` — to `app_en.arb` and running `flutter gen-l10n` to regenerate `app_localizations.dart` across all 14 locale files. Every other label key already exists (`email`, `notifications`, `error_saving_settings`, and all nine `settings_notification_*` description keys).

There are **no new packages, no migrations, no security surface, and no external dependencies** beyond the Flutter SDK already in use. The dominant risk is a documentation inaccuracy: CONTEXT D-11 describes the settings provider as `AsyncValue<Settings?>`, but the provider is synchronous (`Settings?`). The planner must follow the Privacy/Language screen pattern (`settings == null` guards), not the AsyncValue text.

**Primary recommendation:** Convert the stub to a `ConsumerWidget`, copy the `_save` + `_sectionHeader` helpers verbatim from `settings_privacy_screen.dart`, drive 9 rows from an ordered list of `(NotificationType, descriptionKey)` pairs, and use `SwitchListTile` pairs that read `settings?.notifications?[key]?.web ?? true` and write via the D-09 map-copy pattern. Add the `"web"` ARB key and regenerate l10n as the first task.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Render 18 toggle rows | Mobile Frontend (`app/lib/routes/`) | — | UI composition; Material `SwitchListTile` |
| Read current preference state | Mobile Providers (`settingsProvider`) | Models (`Settings`) | Riverpod synchronous provider already exposes `Settings?` |
| Persist a toggle change | Mobile Providers (`saveToServer`) | SvelteKit API (POST `/settings/{id}`) | Existing path; no changes to provider or server needed |
| Localized labels/descriptions | i18n (`app_en.arb` + gen-l10n) | — | All keys exist except `"web"`; regenerate after add |
| Local persistence round-trip | Mobile Models/Entities (`settings_entity.dart`) | ObjectBox | `notificationsJson` JSON-encode already implemented |

**Key check for the planner:** No task should touch `settings_provider.dart`, `settings_entity.dart`, `settings.dart`, or the go_router config. They are complete. The only files that change are `settings_notifications_screen.dart`, `app_en.arb` (+ generated l10n files), and a new test file.

## Standard Stack

This phase introduces **no new dependencies**. It uses only what is already in `pubspec.yaml` and the Flutter SDK.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Flutter Material | SDK 3.11.5+ | `SwitchListTile`, `ListView`, `Scaffold`, `AppBar` | Built-in; canonical toggle widget |
| flutter_riverpod | 3.3.1 | `ConsumerWidget`, `ref.watch`/`ref.read` | Project's state-management standard |
| go_router | 17.2.1 | `context.pop()` back navigation | Project navigation standard; route already registered |
| flutter gen-l10n | SDK | Regenerate `app_localizations.dart` after ARB edit | Project i18n standard (`l10n.yaml` present) |
| freezed | 3.2.5 | `NotificationPreference.copyWith(...)` | Already generated; no codegen run needed for Phase 9 (model unchanged) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| font_awesome_flutter | 11.0.0 | `FontAwesomeIcons.circleExclamation` in error toast | Inside the `_save` catch block (copied verbatim) |
| flutter_test + flutter_localizations | SDK | Widget test with `ProviderScope` override | Wave 0 test, mirroring `settings_privacy_screen_test.dart` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `SwitchListTile` | `ListTile` + trailing `Switch` | More boilerplate, no benefit; CONTEXT D-01 locks `SwitchListTile` |
| Plain `Settings?` read | `AsyncNotifierProvider` rewrite | Out of scope and unnecessary — provider is already synchronous |

**Installation:** None. No `pubspec.yaml` changes. No `flutter pub add`. No `build_runner` run (the model is unchanged; only `flutter gen-l10n` for the new ARB key).

## Package Legitimacy Audit

**Not applicable — this phase installs no external packages.** No `flutter pub add`, no new `pubspec.yaml` entries, no transitive additions. All libraries used are already locked in `pubspec.lock` from prior phases. slopcheck / registry verification is therefore moot.

## Architecture Patterns

### System Architecture Diagram

```
User taps a toggle on Notifications screen
        │
        ▼
SwitchListTile.onChanged(bool value)
        │
        ├─ guard: if (settings == null) → error toast, return
        │
        ▼
Build updated map (D-09):
  existing = settings.notifications?[key]          // NotificationPreference?
  pref     = (existing ?? NotificationPreference(web:true,email:true))
                .copyWith(web: value)  // or .copyWith(email: value)
  updated  = Map.from(settings.notifications ?? {})..[key] = pref
        │
        ▼
_save(ref, l10n, settings.copyWith(notifications: updated))
        │
        ▼
settingsProvider.notifier.saveToServer(updated)
        │
        ├─ POST /settings/{id}  (dio apiProvider)  ──► SvelteKit API ──► PocketBase
        │         │
        │         └─ on DioException ─► catch ─► error toast (l10n.error_saving_settings)
        │
        ▼ (on success)
Settings.fromJson(response) ─► updateFromServer
        │
        ▼
ObjectBox put (notificationsJson) + ref.invalidateSelf()
        │
        ▼
ref.watch(settingsProvider) re-emits Settings? ─► toggle reflects persisted state
```

The save is effectively optimistic-after-confirm: the watched provider only updates once the server response is decoded and written to ObjectBox. The toggle's displayed value is derived from the watched `Settings?` on each build, so a failed save leaves the persisted (old) value intact after rebuild.

### Recommended Project Structure
```
app/lib/routes/settings_notifications_screen.dart   # REPLACE stub body (the only screen file)
app/lib/i18n/app_en.arb                              # ADD "web": "Web"
app/lib/i18n/app_localizations*.dart                 # REGENERATED by gen-l10n (do not hand-edit)
app/test/routes/settings_notifications_screen_test.dart  # NEW widget test
```

### Pattern 1: Ordered notification metadata table
**What:** Drive the 9 sections from a single ordered list of `(NotificationType, String description)` so the loop emits sections in D-05 order and uses the JSON-value string as the map key.
**When to use:** The body of `build`, iterated with a `for` loop inside the `ListView` children.
**Example:**
```dart
// Source: pattern derived from web reference web/src/routes/settings/notifications/+page.svelte
// (notificationItems list) + Flutter loop idiom from settings_language_screen.dart.
// NOTE: NotificationType.trailComment.name == "trailComment" (camelCase).
// The MAP KEY must be the snake_case @JsonValue ("trail_comment"), NOT .name.
// Use an explicit (key, l10nGetter) tuple list — do NOT call .name for the map key.
final items = <({String key, String label})>[
  (key: 'trail_comment',      label: l10n.settings_notification_trail_comment),
  (key: 'new_follower',       label: l10n.settings_notification_new_follower),
  (key: 'trail_share',        label: l10n.settings_notification_trail_share),
  (key: 'trail_like',         label: l10n.settings_notification_trail_like),
  (key: 'list_share',         label: l10n.settings_notification_list_share),
  (key: 'summit_log_create',  label: l10n.settings_notification_summit_log_create),
  (key: 'trail_mention',      label: l10n.settings_notification_trail_mention),
  (key: 'comment_mention',    label: l10n.settings_notification_comment_mention),
  (key: 'summit_log_mention', label: l10n.settings_notification_summit_log_mention),
];
```

### Pattern 2: Per-toggle save (D-08 / D-09)
**What:** Each `SwitchListTile.onChanged` builds an updated map touching only the changed key and the changed field, then calls the shared `_save`.
**When to use:** Every Web and Email row.
**Example:**
```dart
// Source: composition of settings_privacy_screen.dart _save helper + CONTEXT D-09.
void onToggle(String key, {required bool isWeb, required bool value}) {
  if (settings == null) {
    // same error-toast guard the Privacy/Language screens use
    return;
  }
  final existing = settings.notifications?[key]
      ?? const NotificationPreference(web: true, email: true);
  final pref = isWeb
      ? existing.copyWith(web: value)
      : existing.copyWith(email: value);
  final updated = Map<String, NotificationPreference>.from(
    settings.notifications ?? {},
  )..[key] = pref;
  _save(ref, l10n, settings.copyWith(notifications: updated));
}
```

### Pattern 3: Reuse `_save` and `_sectionHeader` verbatim
**What:** Copy the two helpers from `settings_privacy_screen.dart` unchanged. `_save` already wraps `saveToServer` in try/catch and surfaces `l10n.error_saving_settings` via the toast provider.
**Example:**
```dart
// Source: app/lib/routes/settings_privacy_screen.dart lines 16-49 (copy verbatim)
Future<void> _save(WidgetRef ref, AppLocalizations l10n, Settings updated) async {
  try {
    await ref.read(settingsProvider.notifier).saveToServer(updated);
  } catch (_) {
    ref.read(toastProvider.notifier).add(ToastMessage(
      type: ToastType.error,
      icon: FontAwesomeIcons.circleExclamation,
      text: l10n.error_saving_settings,
    ));
  }
}
```

### Anti-Patterns to Avoid
- **Using `NotificationType.trailComment.name` as the map key.** `.name` returns camelCase (`"trailComment"`), but the server and `@JsonValue` use snake_case (`"trail_comment"`). The persisted map keys and the web client both use snake_case. Use the literal snake_case strings (or a `@JsonValue`-aware helper), never `.name`. — verified against `settings.dart` enum annotations and the web `+page.svelte`.
- **Treating the provider as async.** Do not wrap reads in `.when(...)`, `.value`, or `.valueOrNull`. The provider returns `Settings?` directly. (Corrects D-11.)
- **Mutating `settings.notifications` in place.** It is a freezed-owned map; build a new `Map.from(...)` (D-09). In-place mutation would not trigger a rebuild and may throw on an unmodifiable map.
- **Adding a Save button.** CONTEXT D-08 locks auto-save-on-change.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Toggle row with label | Custom `Row` + `Switch` + `Text` | `SwitchListTile(title:..., value:..., onChanged:...)` | Material handles tap target, a11y, theming; matches Language screen |
| Settings persistence | New HTTP call / store | `settingsProvider.saveToServer` | Already implements POST + decode + ObjectBox write + invalidate |
| Local cache of toggles | New ObjectBox field | `notificationsJson` on `SettingsEntity` | Round-trip already implemented and tested |
| Error feedback | Custom SnackBar | `_save`'s toast path (`toastProvider`) | Consistent with Privacy/Language/Account screens |
| Localized strings | Hardcoded "Web"/"Email"/descriptions | ARB keys + gen-l10n | All keys exist except `"web"`; 14 locales auto-generated |

**Key insight:** Phase 9's entire model + persistence + routing stack was deliberately built in Phases 6–8. The only genuinely new artifact is one ARB key and ~120 lines of declarative widget code mirroring an existing screen. Resist any temptation to "improve" the provider or model.

## Runtime State Inventory

> This is **not** a rename/refactor/migration phase — it is additive UI. Included briefly for completeness; the canonical question ("what runtime state holds an old string?") does not apply.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `SettingsEntity.notificationsJson` in ObjectBox already stores the `Map<String, NotificationPreference>` round-trip. No key/schema change. | None — verified in `settings_entity.dart` |
| Live service config | None — verified, no external service config references this screen | None |
| OS-registered state | None — verified, no OS registration involved | None |
| Secrets/env vars | None — verified, no secrets referenced | None |
| Build artifacts | `app_localizations*.dart` (14 files) are generated and WILL change after adding the `"web"` ARB key | Run `flutter gen-l10n`; commit regenerated files |

## Common Pitfalls

### Pitfall 1: Provider is synchronous, not AsyncValue (contradicts D-11)
**What goes wrong:** Following D-11 literally, the implementer wraps `ref.watch(settingsProvider)` in `.when()` / `.value`, which does not compile (the value is `Settings?`, which has no `.value`).
**Why it happens:** CONTEXT D-11 was written assuming an AsyncNotifier; the actual `SettingsNotifier.build()` returns `Settings?` synchronously.
**How to avoid:** Mirror `settings_privacy_screen.dart`: `final settings = ref.watch(settingsProvider);` then guard `if (settings == null)`. No async handling.
**Warning signs:** Any reference to `AsyncValue`, `.when`, `.value`, or `.valueOrNull` for `settingsProvider` in the plan.

### Pitfall 2: `.name` vs `@JsonValue` key mismatch
**What goes wrong:** Toggles appear to work in-session but never persist (or read back as defaults) because the map is keyed by camelCase `"trailComment"` while the server stores `"trail_comment"`.
**Why it happens:** `enum.name` is the Dart identifier (camelCase); `@JsonValue` is snake_case. They differ for every multi-word type.
**How to avoid:** Use snake_case literal strings as map keys (Pattern 1 tuple list). Never `NotificationType.x.name`.
**Warning signs:** `.name` appearing anywhere near the notifications map.

### Pitfall 3: Forgetting `flutter gen-l10n` after adding `"web"`
**What goes wrong:** `l10n.web` does not exist → compile error, or the implementer hardcodes `"Web"` (violates D-03 + project i18n convention).
**Why it happens:** ARB edits don't auto-regenerate unless `flutter pub get`/build triggers it.
**How to avoid:** First task adds the key to `app_en.arb` AND runs `flutter gen-l10n`; verify `l10n.web` resolves before writing the screen.
**Warning signs:** Hardcoded `"Web"` Text widget; missing-getter compile error on `l10n.web`.

### Pitfall 4: Lazy ListView mounting fewer tiles than asserted in tests
**What goes wrong:** Widget test expects `findsNWidgets(18)` for `SwitchListTile` but the lazy `ListView` only mounts on-screen rows.
**Why it happens:** `ListView` is lazy; long subtitles + many rows exceed the default test viewport.
**How to avoid:** Set a tall viewport in the test (`tester.view.physicalSize = const Size(1080, 8000)`), exactly as `settings_privacy_screen_test.dart` does (it uses 1080×4000 for 6 tiles; 18 tiles need more height).
**Warning signs:** Flaky `findsNWidgets` counts in the new test.

## Code Examples

### Reading a toggle value with web-parity default (D-07)
```dart
// Source: CONTEXT D-07 + web reference (s?.trail_comment?.web ?? true)
final webValue   = settings?.notifications?['trail_comment']?.web   ?? true;
final emailValue = settings?.notifications?['trail_comment']?.email ?? true;
```

### A complete section (one notification type → header + two SwitchListTiles)
```dart
// Source: composition of settings_privacy_screen.dart (_sectionHeader, _save, activeColor)
// + settings_language_screen.dart (SwitchListTile is the units variant) + CONTEXT D-01..D-09.
_sectionHeader(context, item.label),
SwitchListTile(
  title: Text(l10n.web),                       // NEW ARB key
  value: settings?.notifications?[item.key]?.web ?? true,
  activeColor: activeColor,                     // dark: onSurface, light: primary
  onChanged: (value) => onToggle(item.key, isWeb: true, value: value),
),
SwitchListTile(
  title: Text(l10n.email),                      // existing ARB key
  value: settings?.notifications?[item.key]?.email ?? true,
  activeColor: activeColor,
  onChanged: (value) => onToggle(item.key, isWeb: false, value: value),
),
```

### The new ARB entry
```jsonc
// app/lib/i18n/app_en.arb — add alphabetically near existing keys
"web": "Web",
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Stub `SizedBox.shrink()` body | `ListView` of 18 `SwitchListTile`s | Phase 9 | This phase |
| `Switch` + manual `ListTile` | `SwitchListTile` | Material standard | Use `SwitchListTile` per D-01 |

**Deprecated/outdated:** None relevant. `SwitchListTile.activeColor` is still valid in the Flutter SDK pinned here (3.11.5+); the existing Privacy/Language screens use the same `activeColor` pattern, so following them guarantees API parity. If a lint flags `activeColor` as deprecated in favor of `activeThumbColor`/`WidgetStateProperty`, match whatever the sibling screens currently use to stay consistent — do not introduce a divergent API in one screen.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `SwitchListTile.activeColor` is non-deprecated in the pinned Flutter SDK | State of the Art / Code Examples | Low — sibling screens use it; if deprecated, copy their exact API. Implementer will see a lint, not a failure. |

**All other claims in this research are VERIFIED against the codebase** (model, provider, entity, ARB, web reference, and test files were all read directly this session). Only A1 depends on SDK behavior not directly probed.

## Open Questions (RESOLVED)

1. **Does the server reject a notifications map that omits some of the 9 keys?**
   - What we know: The web client always sends a complete 9-key `notifications` object on every save. The mobile D-09 pattern sends a partial-then-merged map (only previously-saved keys + the just-changed key).
   - What's unclear: Whether POST `/settings/{id}` merges server-side or replaces wholesale. If it replaces, the first-ever toggle on a fresh account would persist only one key (others fall back to `?? true` defaults on next read, so UX is still correct).
   - RESOLVED: Functionally safe either way — reads default to `?? true`, so UX is correct regardless of server merge behavior. No action needed for v1; D-09 partial-map pattern is accepted as-is.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK + `flutter gen-l10n` | ARB regeneration | ✓ (project uses it; `l10n.yaml` present) | 3.11.5+ | — |
| Existing `pubspec.lock` deps | All widgets/providers | ✓ | locked | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None.

This phase has no external services, no network tools beyond the already-configured dio client, and no new packages. (Step 2.6 effectively trivial — all deps pre-existing.)

## Validation Architecture

> nyquist_validation not explicitly disabled in config — section included.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `flutter_test` (Flutter SDK) |
| Config file | none — Flutter convention; tests in `app/test/` |
| Quick run command | `cd app && flutter test test/routes/settings_notifications_screen_test.dart` |
| Full suite command | `cd app && flutter test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NOTIF-01..09 | 9 section headers render in D-05 order | widget | `flutter test test/routes/settings_notifications_screen_test.dart` | ❌ Wave 0 |
| NOTIF-01..09 | 18 `SwitchListTile`s present (Web+Email × 9) | widget | same | ❌ Wave 0 |
| NOTIF-01..09 | null notifications → all toggles default to `true` (D-07) | widget | same | ❌ Wave 0 |
| NOTIF-01..09 | "Web" and "Email" labels render (D-03) | widget | same | ❌ Wave 0 |

> Tap-to-save assertion is intentionally out of scope for the widget test: `settings_privacy_screen_test.dart` documents that the save path needs an `apiProvider`/HTTP override the harness doesn't provide. Mirror that precedent — assert render + defaults, leave save to acceptance criteria / manual verification.

### Sampling Rate
- **Per task commit:** `cd app && flutter test test/routes/settings_notifications_screen_test.dart`
- **Per wave merge:** `cd app && flutter test test/routes/`
- **Phase gate:** `cd app && flutter test` green + `flutter analyze` clean before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `app/test/routes/settings_notifications_screen_test.dart` — covers NOTIF-01..09 render + defaults. Model on `settings_privacy_screen_test.dart`: `ProviderScope(overrides: [settingsProvider.overrideWithValue(const Settings(id: '1', notifications: {...}))])`, tall viewport (use ≥ 8000 height for 18 tiles), full localization delegates, `locale: Locale('en')`.
- [ ] No shared fixture/conftest needed — the override-with-value pattern is self-contained per the Privacy test.
- [ ] No framework install — `flutter_test` already present.

## Security Domain

> `security_enforcement` not disabled in config — section included. This phase has a minimal security surface (it reuses existing authenticated persistence).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Reuses existing PocketBase session via `apiProvider`; no auth code added |
| V3 Session Management | no | Unchanged; cookie jar handled by existing dio client |
| V4 Access Control | partial | Server enforces that a user can only POST their own `/settings/{id}`; unchanged from prior phases |
| V5 Input Validation | yes (server-side) | Toggle values are booleans from `SwitchListTile`; server validates settings payload as in existing screens. No free-text input. |
| V6 Cryptography | no | None — no crypto in this phase |

### Known Threat Patterns for Flutter settings toggle screen

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Tampering with another user's settings | Tampering | Server-side ownership check on `/settings/{id}` (existing, unchanged) |
| Sensitive data in logs | Information Disclosure | Don't `print()` settings payloads; `_save` surfaces only generic `error_saving_settings` (no server internals) |

**Net assessment:** No new security-relevant code. The boolean-only input, generic error messaging, and reuse of the already-audited `saveToServer` path mean Phase 9 adds no new attack surface beyond what Phases 6–8 already shipped.

## Sources

### Primary (HIGH confidence)
- `app/lib/models/settings.dart` — `NotificationType` enum (9 `@JsonValue` snake_case values), `NotificationPreference` freezed class, `Settings.notifications` map type
- `app/lib/provider/settings_provider.dart` — confirmed `build()` returns `Settings?` **synchronously**; `saveToServer` POSTs then `updateFromServer`
- `app/lib/entities/settings_entity.dart` — `notificationsJson` JSON round-trip already implemented
- `app/lib/routes/settings_privacy_screen.dart` — canonical `_save` + `_sectionHeader` + `activeColor` pattern; synchronous `settings == null` guard
- `app/lib/routes/settings_language_screen.dart` — `SwitchListTile`/toggle precedent
- `app/lib/routes/settings_notifications_screen.dart` — the stub to fill
- `app/lib/i18n/app_en.arb` — verified: `web` ABSENT; `email`, `notifications`, `error_saving_settings`, and all nine `settings_notification_*` keys PRESENT; 14 locale ARB files; `l10n.yaml` config confirmed
- `app/test/routes/settings_privacy_screen_test.dart` — canonical widget-test pattern (ProviderScope override, tall viewport, localization delegates)
- `web/src/routes/settings/notifications/+page.svelte` — confirms D-05 display order and `?? true` default parity

### Secondary (MEDIUM confidence)
- None needed — all claims verified against source.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; all libs already locked and in use
- Architecture: HIGH — read provider, model, entity, both sibling screens, and web reference directly
- Pitfalls: HIGH — Pitfall 1 (sync provider) and Pitfall 2 (.name vs @JsonValue) verified against actual enum + provider source

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable — pure UI on frozen infrastructure; the only external variable is a possible Flutter SDK `activeColor` deprecation lint, tracked as A1)
