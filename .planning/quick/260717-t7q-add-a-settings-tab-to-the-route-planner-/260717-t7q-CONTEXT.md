# Quick Task 260717-t7q: Add a Settings tab to the Route Planner sheet - Context

**Gathered:** 2026-07-17
**Status:** Ready for planning

<domain>
## Task Boundary

Port the web route planner's activity/settings picker into the Flutter app, consolidated into a single picker. Add a third "Settings" tab to `RouteAnchorSheet` (own component, matching the existing Route Anchors / Elevation tab pattern), relocate the auto-routing toggle into it, and replace the manual-overrides settings panel with fixed sensible defaults per picked option. Do not implement "more settings" sliders or the "Reverse direction" button.

</domain>

<decisions>
## Implementation Decisions

### Unified picker (replaces web's Activity picker + Bike type picker)
- ONE picker, five options: **Hiking**, **Biking/Hybrid**, **Biking/Mountain**, **Biking/Cross**, **Biking/Road**.
- Picking a Biking option sets BOTH the Valhalla travel profile (`bicycle`) AND the `bicycle_type` costing option atomically, in one selection.
- Options are built from the operator's actual wanderer `Category` list (icons come from `Category.icon`, same resolution as `category_picker.dart`/`category_icon_util.dart`) — not hardcoded icons. A new name-substring heuristic (mirroring `costingForCategory`/`categoryForTravelProfile`'s existing bike/hike heuristic) maps operator categories to the 5 buckets. Exact heuristic wording, and the fallback icon/behavior when an instance has no category matching a given bucket, are Claude's Discretion (see below).
- This SAME 5-option picker appears in TWO places: the Phase-21 entry-point sheet (`travel_profile_sheet.dart`, currently a 2-card Hike/Bike sheet — expand it to 5 options) AND the new Settings tab inside the already-open planner. Both surfaces read/write the same underlying selection.

### Real routing-behavior change (not cosmetic)
- Picking an option adds `costing_options` to the existing `/valhalla/route` POST body (currently only sends `costing: travelProfile`, no options at all) — sensible fixed defaults per bucket, mirroring web's hardcoded per-type defaults in `route_editor.svelte` (bicycle_type, cycling_speed, use_hills, use_roads, avoid_bad_surfaces for biking; max_hiking_difficulty, walking_speed, use_hills for hiking). No sliders, no "more settings" — the value per bucket is fixed at implementation time.
- No `shortest` toggle exposed either (part of the "no manual overrides" simplification) — default `false` unless research suggests otherwise.

### Mid-session switching
- Switching between the 4 Biking sub-types (Hybrid/Mountain/Cross/Road) stays within the same `bicycle` travel-profile family — anchors are never at risk, only costing_options change.
- Switching Hiking ↔ any Biking option crosses `routeAnchorsProvider`'s family-provider boundary (currently keyed by `travelProfile`, a widget-immutable field). **Confirmed requirement: migrate existing anchors (same lat/lon) to the new profile's provider instance, and re-resolve ALL existing segments under the new costing** — this is new functionality; there is currently no "resolve all segments" bulk method (only `toggleAutoRouting()` flips a flag without touching existing straight segments, and `_resolveSegment` is per-pair). The planner must design this bulk re-resolve + cross-family anchor migration explicitly — it does not already exist anywhere in the codebase.
- ANY category/bucket switch (including within `bicycle`, e.g. Hybrid→Road) also triggers a full re-resolve of every existing segment under the new costing (not just cross-profile switches) — confirmed in discussion as the general rule, not a Hiking↔Biking-only special case.

### Finish handoff — explicitly OUT of scope
- Do NOT wire the picker's selection into `finishPlanning()`'s draft-Trail category pre-fill. Phase 21's `categoryForTravelProfile` heuristic stays exactly as-is. This is a deliberate scope boundary — the picker only drives Valhalla costing_options and (for the entry sheet) the fixed session travel profile, nothing else.

### Explicitly excluded
- No "more settings" sliders/manual override UI of any kind (cycling_speed, use_hills, use_roads, avoid_bad_surfaces, walking_speed, max_hiking_difficulty are all fixed per-bucket, not user-adjustable).
- No "Reverse direction" button/action.

### Claude's Discretion
- Exact substring/keyword heuristic for mapping operator category names to the 5 buckets (mirror the existing `bike`/`cycling`/`bicycle` vs `hik`/`walk`/`foot` pattern in `gpx_util.dart`, extended with road/mountain/cross/gravel/hybrid/city keywords for the 4 bike sub-buckets).
- Fallback behavior when an instance's category list has no category matching a given bucket (e.g., no "Cross"-ish category exists) — whether to still show all 5 options with a generic fallback icon, or omit unmatched buckets. Lean toward always showing all 5 (Valhalla's bicycle_type enum is fixed regardless of what categories the operator happens to have) with a sensible default icon fallback.
- Exact fixed costing_options values per bucket — use web's `route_editor.svelte` hardcoded defaults as the source of truth (bicycle_type-specific `cycling_speed` already varies by type there per `adjustSpeeddependingOnBikeType`).
- Visual layout of the entry-point sheet with 5 options instead of 2 (list vs grid, matching `_TravelProfileCard`'s existing shape vs a denser list mirroring `category_picker.dart`).
- Where the unified selection state lives (a new provider vs. plumbing through existing `routeAnchorsProvider`/screen state) — a research/planning decision informed by the cross-family-migration requirement above.

### Architecture (resolved after research)
- **Rec B chosen**: convert `RouteAnchors` to `@Riverpod(keepAlive: true)` with NO family argument — `travelProfile` and `costingOptions` move fully into `RouteAnchorsState`. A single `switchProfile(profile, costingOptions)` notifier method handles BOTH within-bicycle bucket switches and cross-profile (Hiking↔Biking) switches uniformly — no anchor migration needed since anchors never leave the single provider instance, and no Riverpod autoDispose seeding race exists. This requires updating all 6 call sites that currently pass `travelProfile` as a family argument: `route_planner_screen.dart`, `route_anchor_sheet.dart`, `route_anchor_list_tab.dart`, `elevation_tab.dart`, `route_anchor_layer.dart`, `planned_gpx_provider.dart` (also drop its family arg), plus `route_planner_handoff_util.dart`'s read of the current profile.
- A profile switch is a **fresh undo baseline** — it does not push an undo/redo snapshot; the switch itself is not undoable (matches the prior "travelProfile fixed for lifetime" invariant, keeps the notifier method simple).

</decisions>

<specifics>
## Specific Ideas

Web reference implementation (source of truth for defaults and general shape, NOT for UI structure — the app consolidates to one picker where web has two):
- `web/src/lib/components/trail/route_editor.svelte` — activity picker, bike-type picker, all costing_options defaults, the settings panel this task deliberately does NOT replicate (no sliders).
- Icons: reuse `app/lib/util/category_icon_util.dart` resolution (`Category.icon` → FontAwesome), same as `app/lib/components/trail/category_picker.dart`.
- Existing bike/hike substring heuristic to extend: `app/lib/util/gpx_util.dart`'s `costingForCategory`/`categoryForTravelProfile`.
- Current Valhalla request (needs `costing_options` added): `app/lib/provider/route_anchor_provider.dart` `_resolveSegment`, POSTs to `/valhalla/route` with only `costing: state.travelProfile` today.
- Auto-routing toggle to relocate: `_buildAutoRoutingToggle` in `app/lib/routes/route_planner_screen.dart` (currently in the top-right map controls Column), backed by `routeAnchorsProvider(travelProfile).notifier.toggleAutoRouting()`.
- Tab host to extend to 3 tabs: `app/lib/components/route_planner/route_anchor_sheet.dart` (currently 2 tabs: Route Anchors, Elevation — note this file has an unrelated uncommitted change in the working tree from the user, touching the same TabBar; the plan must account for that current state, not the pre-edit version).
- Entry-point sheet to expand from 2 cards to 5 options: `app/lib/components/route_planner/travel_profile_sheet.dart`.

</specifics>

<canonical_refs>
## Canonical References

No external specs — requirements fully captured in decisions above. Web source files listed under Specific Ideas are the canonical behavioral reference.

</canonical_refs>
