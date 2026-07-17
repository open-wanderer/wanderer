# Phase 21: Route Planner Handoff & Entry Point - Context

**Gathered:** 2026-07-17
**Status:** Ready for planning

<domain>
## Phase Boundary

A user reaches the Route Planner from the trail-source-select flow via a "Plan a route" entry point, chooses an initial hike/bike travel profile up front (fixed for the session), and hands off a finished plan as a draft Trail to the existing create/edit screen. This phase covers HANDOFF-01/02/03.

</domain>

<decisions>
## Implementation Decisions

### Entry-point flow
- **D-01:** The hike/bike selection dialog is a **modal bottom sheet with two tappable cards** (icon + title + description), visually consistent with `trail_source_select_screen.dart`'s existing `_SourceActionCard` pattern the user just came from — not a plain `AlertDialog` or a segmented control.
- **D-02:** The dialog is **dismissible** — back button / tap-outside closes it with no navigation, returning the user to `TrailSourceSelectScreen`. Not forced-choice.
- **D-03 (SCOPE ADD):** Port `allowAutoGeolocate` to the Flutter `Settings` model (`app/lib/models/settings.dart`), mirroring the web app's `Settings.behavior.allowAutoGeolocate` (`web/src/lib/models/settings.ts:56`). The Route Planner's initial map center gates a GPS fix behind this setting — GPS only if `allowAutoGeolocate` is true, otherwise fall back to a non-GPS default. This closes the gap flagged in the 2026-07-16 memory note. Scope is limited to the schema/model field + the gate check at planner entry — **no new settings-screen toggle UI required for this phase** (the field can default the same way web defaults it; exposing a toggle in `SettingsScreen` is optional/out of scope unless planning decides otherwise).
- The existing `/route-planner` route registration (`app/lib/provider/router_provider.dart:254-260`, currently hardcoded `travelProfile: 'pedestrian'` + `Geographic(lat:0, lon:0)`, marked TEMPORARY) is replaced by the real entry point: `TrailSourceSelectScreen`'s existing "Plan a route" card (`trail_source_select_screen.dart:72-82`, also marked TEMPORARY — currently pushes `/route-planner` directly) opens the hike/bike bottom sheet first, then pushes to the planner with the chosen `travelProfile` and the resolved `initialCenter`.

### Handoff trigger & validation
- **D-04:** The "Finish planning" handoff action is an **app bar action** on `RoutePlannerScreen`. As part of this change, **undo/redo move out of the app bar into the top-right map controls column** (the same `controls` Column slot used by the auto-routing toggle (D-06, 19-CONTEXT.md) and the location-search button (D-04, 20-CONTEXT.md)) — freeing the app bar for the Finish action.
- **D-05:** Handoff requires **≥2 route anchors**. Below that, the Finish action is disabled/blocked (mirrors Phase 20 D-13's <2-anchor empty-state precedent for the elevation tab) — a route needs at least a start and end point to become a draft Trail.
- **D-06:** If the one-time `/api/v1/valhalla/height` elevation fetch fails at handoff time, **proceed without elevation and hand off silently** — no error dialog/snackbar, no retry UI. Matches Phase 20 D-11's established best-effort-elevation precedent. The draft Trail hands off with ele-less GPX points; the user can still review on the create/edit screen.

### Draft Trail contents (SCOPE CHANGE from original PRD wording)
- **D-07 (SCOPE CHANGE):** Route anchors **never become `Waypoint` records**. They stay strictly internal to the Route Planner's in-memory state (`RouteAnchor`, `route_anchor_provider.dart`). The draft Trail handed off to `trail_create_screen` carries **only the synthesized GPX track** (`plannedGpxProvider`'s `Gpx`, with elevation merged in at handoff per D-06) — `waypoints: []`, same shape as a plain GPX-file import with no named points. `.planning/REQUIREMENTS.md`'s HANDOFF-01 and `.planning/ROADMAP.md`'s Phase 21 success criterion 3 were amended in this session to drop "+ named waypoints" and state explicitly that no `Waypoint` records are created.
- **D-08:** The hike/bike choice from the entry dialog **pre-fills the draft Trail's category** — Hike → a hiking category, Bike → a biking category (the reverse mapping of `costingForCategory`'s existing `'bike'`-substring heuristic in `gpx_util.dart:22-24`). Saves the user a step on the create/edit screen. Exact category ID/matching logic against the app's category model is implementer's discretion (see below) — no existing reverse-lookup helper was found during scouting.

### Claude's Discretion
- Exact category ID/lookup used for D-08's hike→category / bike→category mapping (no existing helper does this reverse mapping; `costingForCategory` only goes category→profile string).
- Bottom sheet card copy/icons for the hike/bike dialog (D-01) — follow `_SourceActionCard`'s existing visual conventions.
- Exact fallback value for `initialCenter` when `allowAutoGeolocate` is false or GPS fails (D-03) — e.g. last map camera position (`mapCameraProvider`, already used elsewhere) vs. a fixed default. Not discussed in depth; implementer should pick the simplest existing precedent.
- Whether `allowAutoGeolocate` needs a settings-screen toggle in this phase, or ships model-only until a future settings phase surfaces it (D-03 leaves this open, defaulting to "no new toggle UI required").

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap (amended this session)
- `.planning/REQUIREMENTS.md` — HANDOFF-01 marked SCOPE CHANGE and reworded: "synthesized GPX + named waypoints" → "synthesized GPX track only, no Waypoint records."
- `.planning/ROADMAP.md` — Phase 21 success criterion 3 reworded to match; explicit SCOPE CHANGE note added after the success criteria (mirrors Phase 20's precedent for documenting scope changes inline).

### Prior phase context
- `.planning/phases/20-route-planner-views-waypoint-list-elevation-location-search/20-CONTEXT.md` — D-04 (top-right map controls Column, search button joins it), D-09/D-10 (`plannedGpxProvider` incremental Gpx synthesis, pre-elevation by design — this phase adds the one-time elevation merge at handoff), D-11 (best-effort elevation fetch precedent reused for D-06 above).
- `.planning/phases/19-route-planner-core-waypoint-editing-routing-engine/19-CONTEXT.md` — D-01 (route anchor terminology — reinforces D-07's "anchors never become waypoints"), D-06 (top-right map controls Column pattern), D-10 (undo/redo as app-bar icon buttons — this phase relocates them per D-04).

### Project-level (memory, still open as of this session)
- `.planning/STATE.md` §Blockers/Concerns has no open item specific to Phase 21 beyond what's captured above.
- User's 2026-07-16 note on `allowAutoGeolocate` (captured project memory, not a file in this repo): Phase 21 must gate the planner's initial-camera-on-user-location behavior behind this setting rather than resolving GPS unconditionally — directly resolved by D-03 above.

No other external specs/ADRs — requirements fully captured in decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/routes/trail_source_select_screen.dart:1-211` (`TrailSourceSelectScreen`): `ListView` of `_SourceActionCard` widgets (icon-badge + title + description, `Card`/`InkWell`). "Plan a route" card already exists at lines 72-82 (marked TEMPORARY, currently pushes `/route-planner` directly with no dialog) — this phase replaces its `onTap` to open the hike/bike bottom sheet (D-01) first.
- `app/lib/util/trail_import_util.dart:28` (`Trail? pendingImportedTrail`): bare mutable global (not a provider) used because go_router's `extra` can be silently dropped on a same-process router refresh. Set right before `navContext.push('/trail/create/edit', extra: trail)` (line 111). This phase's handoff should follow the identical mechanism: build the draft `Trail`, set `pendingImportedTrail = draftTrail`, then push `/trail/create/edit` with `extra: draftTrail`.
- `app/lib/provider/router_provider.dart:261-278`: the `/trail/create/edit` route builder — checks `state.extra is Trail` first, clears `pendingImportedTrail` on use (line 266), falls back to the global if `extra` isn't a `Trail` (line 274). No changes needed here; this phase is a producer of the same contract the GPX-import flow already satisfies.
- `app/lib/routes/navigation_screen.dart:975-977`: closest existing `showDialog<bool>(... AlertDialog ...)` precedent, though D-01 chose a bottom sheet instead — reference `settings_categories_screen.dart` / `settings_subcategories_screen.dart` for modal bottom sheet chrome instead.
- `app/lib/provider/planned_gpx_provider.dart:22-42` (`plannedGpxProvider`, `@riverpod Gpx plannedGpx(Ref ref, String travelProfile)`): returns the synthesized route Gpx, points-only, no `ele` (deliberately, per Phase 20 D-10). This phase's handoff reads this provider's output and merges in a one-time elevation fetch (D-06) before building the draft Trail.
- `app/lib/provider/route_anchor_provider.dart` (`RouteAnchors` / `RouteAnchorsState`): `anchors: List<RouteAnchor>`, `segments: List<RouteSegment>` — read for the ≥2-anchor validation (D-05); anchors themselves are NOT converted to Waypoints (D-07).
- `app/lib/util/gpx_util.dart:22-24` (`costingForCategory`): existing category→`'pedestrian'`/`'bicycle'` heuristic (`.contains('bike')`). D-08 needs the reverse direction (profile→category); no existing helper, Claude's Discretion.

### Established Patterns
- Riverpod 3.x + `riverpod_annotation` codegen; `AsyncValue` listener closures must be explicitly typed.
- Top-right map `controls` Column slot (established Phase 19 D-06, extended Phase 20 D-04) is the shared home for auto-routing toggle, location-search button, and now (D-04) undo/redo.
- App bar `actions` list on `RoutePlannerScreen` (`route_planner_screen.dart:144-162`) currently holds undo/redo — this phase's Finish action takes that slot instead.

### Integration Points
- `app/lib/provider/router_provider.dart:254-260`: `/route-planner` route registration — currently hardcodes `travelProfile`/`initialCenter` and is marked TEMPORARY; this phase wires the real values from the entry-point flow (hike/bike choice + D-03's gated GPS/fallback).
- `app/lib/models/settings.dart:94-104` (`Settings` freezed class): needs a new `behavior`/`allowAutoGeolocate` field per D-03 — confirmed absent as of this session (grep across `settings.dart` and `settings.freezed.dart` returned zero matches for either term). Mirror `web/src/lib/models/settings.ts:56` and `web/src/lib/models/api/settings_schema.ts` for shape.
- `app/lib/routes/route_planner_screen.dart:29-48`: constructor already accepts `required String travelProfile` and `required ml.Geographic initialCenter` with no defaults, by design — this phase is what finally supplies real values instead of the TEMPORARY hardcoded ones.

</code_context>

<specifics>
## Specific Ideas

- The user explicitly corrected the "named waypoints" framing during discussion: "Route anchors do not become trail waypoints. They stay strictly in the route planner" and, when asked what the handoff's waypoints list should then contain, confirmed "No Waypoint records at all — just the GPX track." This directly overrides HANDOFF-01's original wording (D-07).
- Undo/redo relocation (D-04) was the user's own suggestion in response to being asked where Finish should live: "Put it in the appbar. Move the undo/redo buttons into the map actions instead."

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. D-03's `allowAutoGeolocate` port and D-07's waypoint-scope correction are both implementation-mechanism/requirement-wording clarifications of already-scoped HANDOFF requirements, not new capabilities, and were reconciled directly in REQUIREMENTS.md/ROADMAP.md rather than left as loose ideas.

</deferred>

---

*Phase: 21-route-planner-handoff-entry-point*
*Context gathered: 2026-07-17*
