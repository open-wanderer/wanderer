# Milestones

## v1.4 MapLibre Migration (Shipped: 2026-07-10)

**Phases completed:** 6 phases, 17 plans, 40 requirements

**Key accomplishments:**

- Native GL map rendering — `WandererMap` and all 6 map screens now run on `maplibre` (`MapLibreMap`) instead of `flutter_map`, with live light/dark style swapping, scale bar, and ODbL attribution.
- Self-hosted glyph & sprite serving — new unified `/api/v1/map/style-sources` endpoint resolves tile, glyph, and sprite URLs under operator override, fixing missing place-name labels and route-shield icons that silently failed to render before.
- Offline parity preserved — downloaded trails render basemap via native `pmtiles://` and place labels via cached `file://` glyphs/sprites in airplane mode, including multi-cell trails.
- Server-side clustering — the map screen now renders `POST /search/trails/cluster` results as native circle/symbol layers matching web's `ClusterLayer`, replacing client-side rendering.
- Turn-by-turn navigation migrated — heading-up follow, compass reset, and live location puck all run on maplibre-native APIs; offline navigation from the ObjectBox cache is unregressed.
- Both `flomp/*` forks retired — `flutter_map` + 4 plugins, `vector_map_tiles`, and `vector_tile_renderer` are gone from `pubspec.yaml`; `maplibre` is pinned to an exact version (0.3.5).

**Known deferred items at close:** 15 (see STATE.md Deferred Items — 14 are completed quick tasks the audit tool couldn't classify; 1, dark mode for the Flutter app, was planned but never executed and remains open for a future milestone).

---

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
