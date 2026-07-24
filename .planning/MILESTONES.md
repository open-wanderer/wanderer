# Milestones

## v1.5 Route Planner (Shipped: 2026-07-24)

**Phases completed:** 3 phases, 13 plans, 23 tasks

**Key accomplishments:**

- Precision-parameterized Google-encoded-polyline codec (default 5, Valhalla decodes at 6) plus a new freezed `RouteAnchor`/`RouteSegment`/`SegmentState`/`RouteAnchorsSnapshot` in-memory route model that never reuses the persisted `Waypoint` type
- Class-based `RouteAnchors` `@riverpod` family notifier: per-segment Valhalla routing engine with a CancelToken + generation-counter race-guard, append/drag/insert anchor mutations, geometric segment-split for plain taps, and an immutable-snapshot undo/redo stack
- Native-map rendering surfaces for the route planner: `RouteAnchorLayer` (numbered, draggable `WidgetLayer` markers) and `RouteSegmentLayer` (GeoJSON-backed, state-filtered `LineStyleLayer` segment renderer with an invisible wide hit-test layer), plus a unit-tested `buildSegmentsGeoJson` builder
- `RoutePlannerScreen`: the screen that hosts the native map, disambiguates marker/segment/empty-map taps into the correct 19-02 mutation, exposes the auto-routing toggle, and puts undo/redo + blocked-segment/retry toast copy in the app bar — closing the goal-backward reachability chain for all of Phase 19's requirements
- Added deleteAnchor/reorderAnchors mutators to RouteAnchors (segment-collapse-on-delete, adjacency-diff reuse-on-reorder) plus a buildGpxFromPoints helper and plannedGpxProvider that derives a live pre-elevation Gpx by walking the anchor-id chain.
- LocationSearchScreen — a locations-only mirror of GlobalSearchScreen that reuses the existing debounced globalSearchProvider and pops its result back to the caller via `/location-search`, a new pushable go_router route.
- ElevationProfile now accepts a null Trail with gpx.getTotals()-derived stats and no anchor icons; new ElevationTab fetches /valhalla/height only while visible, debounced 500ms, with index-aligned ele-merging and a <2-anchor empty state.
- RouteAnchorListTab — a ReorderableListView.builder of route anchors with a numbered accent badge, coordinate subtitle, immediate no-confirmation delete, and long-press-drag reorder wired directly to the sheet's ScrollController.
- RouteAnchorSheet — a docked, tabbed DraggableScrollableSheet (Route Anchors + Elevation) wired into route_planner_screen.dart alongside a magnifying-glass search control that pans the map to a searched location at zoom 13.
- `finishPlanning` orchestration util that turns the in-progress planner route into a draft Trail (GPX track only, no Waypoint records) with a silent one-time elevation merge and hike/bike category pre-fill, handed off via the existing `pendingImportedTrail` mechanism.
- Ported the `Behavior` nested type (`allowAutoGeolocate`, `mapClusteringMaxZoom`, `showTrailStartMarker`) from web's `Settings.behavior` onto the Flutter `Settings` freezed model, with `SettingsEntity.behaviorJson` persisting it via the same JSON-blob strategy as `privacyJson` — closing the D-03 gap so Plan 03's GPS gate at planner entry has a real field to read.
- Real HANDOFF-02/03 entry point: the "Plan a route" card now opens a dismissible hike/bike bottom sheet, resolves a GPS-gated (or fallback) initial map center via `Settings.behavior?.allowAutoGeolocate`, and pushes `/route-planner` with the chosen travel profile — replacing both TEMPORARY Phase-19 stubs (the hardcoded route registration and the card's direct push).
- App-bar "Finish" action on `RoutePlannerScreen`, gated on >=2 route anchors, wired to Plan 01's `finishPlanning`; undo/redo relocated into the top-right map controls Column to free the app-bar slot Finish now occupies.

---

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
