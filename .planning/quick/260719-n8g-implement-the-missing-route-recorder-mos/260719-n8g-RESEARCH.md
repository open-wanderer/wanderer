# Quick Task 260719-n8g: Implement the missing route recorder — Research

**Researched:** 2026-07-19
**Domain:** Flutter mobile (Riverpod, go_router, MapLibre, ObjectBox) — reuse of `navigation_screen.dart` as a trail-less "recording mode"
**Confidence:** HIGH (all findings verified against the actual source files in this session)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Entry point:** Wire the existing `trail_source_select_screen.dart` "Record trail" card (`_comingSoon` stub) as the real entry. No new UI surface.
- **Screen reuse:** Reuse `NavigationScreen` in an empty-response "recording mode" (stub/empty `NavigateResponse`, sentinel id). Do NOT build a separate `RecordScreen`.
- **Button row (recording, left→right):** (1) Pause/unpause toggle (same `togglePause`), (2) **Stop recording** — red, square icon, center/dominant slot, opens the existing 3-option dialog (Cancel / Exit without saving / Save), (3) Elevation profile toggle (unchanged). Order changes from nav's `[exit, pause, elevation]` to `[pause, stop, elevation]`.
- **Finish path:** Recording mode's ONLY finish trigger is the 3-option dialog (`_confirmExit`/`_NavExitChoice`), reused verbatim. No auto-arrival banner (`isArrived` can never be true with empty maneuvers).
- **Resume-after-kill:** Wire `ActiveSessionType.rec` resume in `main.dart`'s `_maybeResume` alongside the `.nav` path. Do NOT defer.

### Claude's Discretion
- Exact sentinel/placeholder value for `widget.id` and shape of the "empty" `NavigateResponse`.
- Whether recording mode needs its own title/copy (note: **there is no AppBar** — see Pitfall 3).
- Resume-dialog copy differences between `.nav` and `.rec` — mirror the `.nav` dialog unless `rec` needs different wording.

### Deferred Ideas (OUT OF SCOPE)
- None stated. Maximum reuse; recording mode is a thin variant, not a parallel implementation.
</user_constraints>

## Summary

Everything needed already exists. `NavigationScreen` is already almost entirely GPS-driven and null-guards every maneuver/route/trail dependency to nothing when they are empty. The task is to add a single `bool isRecording` constructor flag, thread it through four small branch points (button row, persist, entry route, resume), pass an empty `NavigateResponse` + empty-string `id`, and add a handful of i18n keys. No new provider, no schema migration (the `ActiveSessionType.rec` enum and all session-agnostic entity fields already exist), and no changes to the breadcrumb/stats plumbing.

**Primary recommendation:** Add `final bool isRecording` (default `false`) to `NavigationScreen`. Add a new top-level `/record` GoRoute that builds `NavigationScreen(id: '', response: const NavigateResponse(maneuvers: [], shape: []), isRecording: true, resumeSession: <extra if present>)`. Branch `_buildButtonRow`, `_persistNow` (sessionType/trailId), and `main.dart`'s `_maybeResume` on the flag/session type. Wire the "Record trail" card to request location permission then `context.push('/record')`.

## Point-by-Point Findings

### 1. The "empty" NavigateResponse (minimal diff)

`NavigateResponse` is a freezed class with two `required` list fields and no other required data (`navigate_response.dart:23-33`):
```dart
const factory NavigateResponse({
  required List<NavigateManeuver> maneuvers,
  required List<List<double>> shape,
}) = _NavigateResponse;
```
**Construct `const NavigateResponse(maneuvers: [], shape: [])`.** Verified downstream tolerance:
- `Navigation.build` (`navigation_provider.dart:58-108`) explicitly handles `shape.isEmpty` → empty cumulative arrays; `onPosition` (`:129-130`) early-returns after appending the breadcrumb when `shape.isEmpty`. Breadcrumb collection still works.
- `NavigationStatsNotifier.build(response, {resume})` uses `response` only as a family cache key; stats are 100% GPS-driven.
- `shapeAsGeographic` returns `[]` → map `initCenter` falls back to `Geographic(lat: 0, lon: 0)` (`navigation_screen.dart:957-959`). See Pitfall 4.
- `maneuvers.isEmpty` → `_buildActiveBannerContent` returns `SizedBox.shrink()` (`:1229-1231`); `isArrived` is `currentIndex >= maneuvers.length - 1 && maneuvers.isNotEmpty` = **always false** (`:930-931`) so the completion banner never renders. **No banner-suppression code is needed.** [VERIFIED: source]

**Provider-identity note:** use a single shared `const` instance (or `widget.response`) at every call site — the `navigationProvider`/`navigationStatsProvider` families are keyed on the response identity, and the screen already passes `widget.response` uniformly. A `const` empty response is stable across rebuilds. Since only one session is ever active, sharing the const across sessions is harmless.

### 2. Sentinel `widget.id`

`widget.id` feeds `ref.watch/read(trailProvider(widget.id))` at ~6 sites, all already null-guarded (`trailAsync.value?...`, `?? []`, `.value?.pmTiles`). `TrailNotifier.build(id)` (`trail_provider.dart:16-63`) does `api.get("/trail/$id")` then on failure falls back to an ObjectBox lookup and rethrows → the provider resolves to `AsyncError`, and `.value` is `null`, which every guard handles.

**Recommendation: use empty-string `id: ''` (smallest diff).** Do NOT refactor `id` to nullable (touches every call site + the `/trail/:id/navigate` route). Two consequences to handle:
- `trailProvider('')` fires a wasted `GET /trail/` that 404s. Harmless (cached AsyncError), but if you want to avoid it, guard the single watch at `:908`: `final trailAsync = widget.isRecording ? const AsyncValue<Trail>.loading() : ref.watch(trailProvider(widget.id));` (optional polish).
- `_buildElevationPage` (`:1516-1538`) renders `error_reading_file` text on `trailAsync.error`. In recording mode there is no trail GPX to profile. See Pitfall 2 for the fix.

### 3. Adding the `isRecording` branch

Add to the constructor (`navigation_screen.dart:51-63`):
```dart
final bool isRecording;
const NavigationScreen({ ..., this.isRecording = false, ... });
```
Thread to exactly these points:
- **(a) Button row** — `_buildButtonRow` (`:1541-1642`): branch on `widget.isRecording` to render `[pause, stop, elevation]`. Reuse the existing pause FAB body (`:1568-1591`) as the left button, add a red center FAB (`backgroundColor: Colors.red`/`colorScheme.error`, square/stop icon — **no stop icon is imported yet**; use `Icons.stop` or add `FontAwesomeIcons.stop`) whose `onPressed` calls `_confirmExit(context, localizations)`, and keep the elevation FAB (`:1594-1638`) unchanged.
- **(b) Completion banner** — **no change required**; `isArrived` is structurally always false and the active banner already collapses to `SizedBox.shrink()` for empty maneuvers.
- **(c) App-bar title** — **there is no AppBar.** `build()` returns `Scaffold(body: Stack(...))` with no `appBar` (verified: `grep appBar` → none). Any "Recording" label would have to go in the top banner `Card` (`:1085-1097`), which currently renders nothing in recording mode. Optional: in recording mode, render a small "Recording…" card there instead of `SizedBox.shrink()`. Discretionary.

### 4. Reusing the 3-option exit dialog as the finish trigger

`_confirmExit` (`:1127-1165`) already shows Cancel / `exit_navigation` / (`save_track` if `_hasSavableTrack()`), and routes `saveTrack` → `_saveRecordedTrack()`, `exit` → `active_nav.clear` + `context.pop()`. **Reuse verbatim** — both the new red Stop button (3a) and the existing `PopScope`/back gesture (`:933-937`) call it. The only recording-specific concern is the dialog **content string**: `stop_navigation_confirm` = "Stop navigation and return to the trail?" is wrong for recording. Branch the content text on `widget.isRecording` to a new key (see i18n below). `_saveRecordedTrack` (`:657-699`) already builds the GPX via `buildGpxFromPoints` + `buildDraftTrail` and `pushReplacement('/trail/create/edit')` — this is exactly the required save path; the `originalTrail?.categoryId` read (`:670`) resolves to `null` in recording mode (no trail), which `buildDraftTrail(category: null)` accepts.

### 5. Wiring the entry card + the route

**Entry** (`trail_source_select_screen.dart:142-150`): replace `onTap: () => _comingSoon(l10n)` with a new `_openRecorder(l10n)`. Recording needs GPS permission — nav gets it in `launchNavigation` (`navigation_launch_util.dart:87-110`) **before** pushing; `NavigationScreen` itself does not request permission. **Mirror that permission block (or extract a shared helper)** in `_openRecorder`, then `context.push('/record')`.

**Route:** add a **new top-level GoRoute `/record`** (not a child of `/trail/:id` — there is no real id). Add alongside the other top-level routes in `router_provider.dart` (e.g. near `:276`):
```dart
GoRoute(
  path: '/record',
  builder: (context, state) {
    final resume = state.extra is ActiveNavigationEntity
        ? state.extra as ActiveNavigationEntity : null;
    return NavigationScreen(
      id: '',
      response: const NavigateResponse(maneuvers: [], shape: []),
      isRecording: true,
      resumeSession: resume,
    );
  },
),
```
Do NOT overload `/trail/:id/navigate` (`:307-324`) with a fake id — its extra is a `(NavigateResponse, bool, ActiveNavigationEntity?)` tuple and there is no trail to key on. A dedicated `/record` route is the cleaner, smaller diff and keeps the resume push trivial.

### 6. Persisting & resuming an `ActiveSessionType.rec` row

**Persist** — `_persistNow` (`:576-622`) hardcodes `sessionType: ActiveSessionType.nav` and `trailId: widget.id`. Branch both:
```dart
sessionType: widget.isRecording ? ActiveSessionType.rec : ActiveSessionType.nav,
trailId: widget.isRecording ? null : widget.id,
```
All other fields are session-agnostic (`active_navigation_entity.dart:38-57` documents `breadcrumbPolyline`/`elevations`/`timestampsUtc`/stats as shared; `currentManeuverIndex` is nullable and irrelevant for rec — leave it, it'll persist `0`). `sessionType` is `@Transient()` but round-trips via `dbSessionType` (index) getter/setter (`:21-27`), so a `.rec` row reads back correctly.

**Resume** — `main.dart:_maybeResume` (`:172-232`) currently bails on any non-`nav` row (`:180-183`). Add a `.rec` branch **before** that guard:
- If `row.sessionType == ActiveSessionType.rec`: **skip `readCachedNav`** (no trail). Show a resume dialog (mirror the `.nav` one at `:205-221`, but with no trail name — use a generic `resume_recording_prompt`). On accept: `navigatorKey.currentContext?.push('/record', extra: row)`. On decline: `active_nav.clear(store)`.
- "Resume" for rec = rehydrate breadcrumb + stats only. `initState` already does this generically from `resumeSession.breadcrumbPolyline`/`elevations`/`timestampsUtc` and the `NavigationStatsSeed` (`:254-288`). `currentManeuverIndex` null → `_resumeManeuverIndex` null → no maneuver restore. **Works unchanged for rec.** [VERIFIED: source]

### 7. Pitfalls

**Pitfall 1 — Existing tests do NOT break.** The only tests touching this area are `test/provider/navigation_provider_test.dart`, `test/models/navigate_response_test.dart`, `test/provider/navigation_stats_provider_test.dart`. **None construct the `NavigationScreen` widget or assert route paths** (grep for `NavigationScreen(` in `test/` → 0 hits). Adding a defaulted `isRecording` param and a `/record` route is non-breaking. Still run `flutter test` after. [VERIFIED: grep]

**Pitfall 2 — Elevation page shows an error in recording mode.** `_buildElevationPage` (`:1516-1538`) has an `error:` branch that renders `error_reading_file`; `trailProvider('')` errors. Fix: guard `_buildElevationPage` (or the elevation toggle) on `widget.isRecording` to return `SizedBox.shrink()` (no trail = nothing to profile). Live elevation-of-breadcrumb is out of scope (`ElevationProfile` wants a `Trail`+`Gpx`).

**Pitfall 3 — No AppBar exists.** Focus point 3(c)'s "app-bar title" is moot; the screen is a full-bleed `Stack`. Put any "Recording" affordance in the top banner Card slot if desired.

**Pitfall 4 — Map opens at (0,0) until first fix.** Empty shape → `initCenter (0,0)` (`:957-959`). Follow mode recenters on the first GPS fix (`_pushCamera`, `:526-535`), so it self-corrects within a second. Acceptable; centering on last-known location is optional polish, out of scope.

**Pitfall 5 — `coming_soon` may become unused.** After wiring the card, `_comingSoon` (`trail_source_select_screen.dart:30-40`) has no remaining caller (the two other cards use their own handlers). Remove it (and the now-unused import if any) to avoid an analyzer `unused_element` warning.

## i18n Keys

**Reuse as-is:** `trail_source_record` ("Record trail"), `exit_navigation` ("Exit"), `save_track` ("Save track"), `cancel`, `resume`, `pause`, `location_tracking_notification_title`/`_text`, `new_trail`, `error_saving_trail`.

**Add (to `app_en.arb` + all `app_*.arb`):**
- `stop_recording` — Stop button tooltip, e.g. "Stop recording".
- `stop_recording_confirm` — dialog content in recording mode, e.g. "Stop recording?" (replaces `stop_navigation_confirm` when `isRecording`).
- `resume_recording_prompt` — rec resume dialog, e.g. "Resume recording?" (no trail-name placeholder, unlike `resume_navigation_prompt(trail)`).
- (Optional, discretionary) `recording_in_progress` / "Recording…" if you add a banner label.

**Remove:** the `_comingSoon` call site (see Pitfall 5); `coming_soon` key stays if used elsewhere — verify before deleting.

## Environment Availability

Pure Flutter/Dart code changes against existing packages (Riverpod, go_router, maplibre, objectbox, gpx, geolocator, font_awesome_flutter) — all already in `pubspec.yaml` and imported by the touched files. No new dependencies. Step 2.6 external-dependency audit: **N/A** (no new tools/services). No package legitimacy audit needed (no installs).

## Validation Architecture

**Framework:** `flutter_test` (existing; `test/` dir with provider/model tests). Quick run: `cd app && flutter test`.

**Requirement → test map (Wave 0 gaps):**
| Behavior | Test type | Command | Exists? |
|----------|-----------|---------|---------|
| Empty `NavigateResponse` yields empty nav state, breadcrumb still appends | unit | `flutter test test/provider/navigation_provider_test.dart` | ✅ extend with an empty-response case |
| `_persistNow` writes a `.rec` row with `trailId == null` when recording | unit (via a store helper test) | new `test/util/active_navigation_store_test.dart` | ❌ optional |
| Widget-level recording button row / dialog | widget | new `test/routes/navigation_screen_test.dart` | ❌ likely out of scope for a quick task (heavy MapLibre/native deps) |

Recommendation: add/extend the provider unit test for the empty-response path (cheap, high value); widget tests for `NavigationScreen` are impractical due to native MapLibre/tracelet/sensor dependencies — verify the UI manually. Run `flutter analyze` + `flutter test` as the gate.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `const NavigateResponse(maneuvers: [], shape: [])` compiles as a const (freezed const factory with only list literals) | 1 | Low — if const fails, use a non-const `final _emptyResponse`; provider families still key on identity, so hold it in a field. |
| A2 | `trailProvider('')` resolves to `AsyncError` (not an unhandled throw) and every `.value?` guard tolerates it | 2 / Pitfall 2 | Low — all reads are null-guarded in source; worst case is the elevation error text (Pitfall 2 fix covers it). |
| A3 | Recording requires the same location-permission flow as `launchNavigation`; `NavigationScreen` does not self-request it | 5 | Medium — if omitted, recording silently gets no fixes on a fresh permission state. Mirror `navigation_launch_util.dart:87-110`. |

## Sources

### Primary (HIGH — read this session)
- `app/lib/routes/navigation_screen.dart` — constructor, `_confirmExit`, `_buildButtonRow`, `_persistNow`, `_saveRecordedTrack`, banner/isArrived logic, no-AppBar Scaffold.
- `app/lib/models/navigate_response.dart` — freezed factory, required fields, `shapeAsGeographic`.
- `app/lib/entities/active_navigation_entity.dart` — `ActiveSessionType.rec`, session-agnostic fields, transient sessionType round-trip.
- `app/lib/provider/navigation_provider.dart`, `navigation_stats_provider.dart` — empty-shape handling, family keying on `response`.
- `app/lib/provider/router_provider.dart` — route table, `/trail/:id/navigate` builder.
- `app/lib/main.dart` — `_maybeResume` nav-only guard.
- `app/lib/routes/trail_source_select_screen.dart` — `_comingSoon` stub card.
- `app/lib/util/route_planner_handoff_util.dart` — `buildDraftTrail`.
- `app/lib/util/navigation_launch_util.dart` — permission flow + `/navigate` push pattern.
- `app/lib/util/active_navigation_store.dart`, `app/lib/provider/trail/trail_provider.dart` — persistence + trail fetch fallback.
- `app/lib/i18n/app_en.arb` — existing keys (grep-verified).
- `app/test/` — grep confirmed no widget/route tests for NavigationScreen.

## Metadata

**Confidence breakdown:**
- Reuse strategy / branch points: HIGH — verified against source line-by-line.
- Empty-response / sentinel-id tolerance: HIGH — every downstream guard confirmed in code.
- Permission requirement (A3): MEDIUM — inferred from `launchNavigation` precedent; confirm during implementation.

**Research date:** 2026-07-19
**Valid until:** ~30 days (stable local codebase; no external/version-sensitive claims).
