# Quick Task 260721-eob: Add a bottom modal sheet on track save from navigation screen with 'Recalculate heights' and 'Follow roads' toggle options - Context

**Gathered:** 2026-07-21
**Status:** Ready for planning

<domain>
## Task Boundary

When a user decides to save a track from the navigation screen (recording mode), a bottom modal sheet opens first with two togglable options:
1. **Recalculate heights** — subtitle explains this replaces recorded GPS heights with Valhalla-derived heights (more accurate).
2. **Follow roads** — snaps the recorded path to the road network using Valhalla's `trace_route` action. Requires a new SvelteKit API route (`web/`) that proxies the request to Valhalla — no such route exists yet (only `/valhalla/route`, `/valhalla/height`, `/valhalla/navigate`).

After the user confirms and all Valhalla cleanup operations conclude, the (potentially transformed) trail is handed off to `trail_create_screen` exactly as today's flow already does (via `pendingImportedTrail` + `context.pushReplacement('/trail/create/edit', extra: trail)`).

**Existing entry points that must show this sheet (both call `_saveRecordedTrack` in `app/lib/routes/navigation_screen.dart`):**
- `_confirmExit`'s exit dialog → `_NavExitChoice.saveTrack` (line ~1190)
- The arrival completion banner's save button (`_buildCompletionBannerContent`, line ~1352)

</domain>

<decisions>
## Implementation Decisions

### Toggle defaults
- Both "Recalculate heights" and "Follow roads" default to **off** when the sheet opens. The user opts in explicitly to either transformation; the raw GPS recording is preserved unless requested otherwise.

### Operation order (when both toggles are on)
- **Snap roads first, then recalculate heights on the snapped points.** Heights must reflect the final (possibly road-moved) coordinates — running height lookup on the pre-snap trace first would produce elevation values misaligned with the saved path.
- Pipeline for `_saveRecordedTrack`: breadcrumb → (if Follow roads) Valhalla `trace_route` snap → (if Recalculate heights) Valhalla `/height` on the resulting shape → merge → `buildDraftTrail`.

### Follow-roads costing profile
- Derive the Valhalla costing profile via the existing `costingForCategory()` helper (`app/lib/util/valhalla_util.dart`), using the same category-derivation the app already applies in `launchNavigation` and the route planner — not a hardcoded `pedestrian`.

### Failure handling
- Both the new trace_route proxy call and the existing `/valhalla/height` call are **best-effort with silent fallback** — on error/timeout, proceed with the pre-transformation track rather than blocking the save or showing an error toast. This matches the existing precedent in `buildFinalPlannedGpx` (`app/lib/util/route_planner_handoff_util.dart`), which already silently falls back on a `/valhalla/height` failure.

### Claude's Discretion
- Exact modal sheet visual layout/spacing — follow existing bottom-sheet patterns in the app (e.g. `waypoint_sheet.dart`, travel-profile sheet) and the project's "Terrain Log" design language (flat, bordered, IBM Plex Sans).
- New SvelteKit route naming/path — follow sibling convention: `web/src/routes/api/v1/valhalla/trace-route/+server.ts` (or `trace_route`, matching whichever casing convention the planner finds cleanest against existing siblings `route`, `height`, `navigate`).
- New env var name for the trace_route upstream URL — follow the existing per-action pattern (`VALHALLA_ROUTE_URL`, `VALHALLA_HEIGHT_URL`, `VALHALLA_NAVIGATE_URL`) with a new `VALHALLA_TRACE_ROUTE_URL` (or similar), added to `ExternalServiceUrlKey` in `web/src/lib/server/url.ts` and a getter in `web/src/lib/server/valhalla.ts`. Note: none of these vars are set in the repo's own `docker-compose*.yml` (only `VALHALLA_URL` is) — this is a pre-existing discrepancy for the other three actions too, not something this task needs to fix.
- i18n strings for the new sheet's copy — add new keys to `app_en.arb` only (per this repo's recent convention of extracting hard-coded Dart literals into l10n keys); other locale ARB files are synced via Crowdin separately, not hand-edited.
- Whether the new trace_route SvelteKit route requires `event.locals.user` auth like `/valhalla/navigate` does, or is unauthenticated like `/valhalla/route` and `/valhalla/height` — planner's call, but leaning toward matching `/valhalla/navigate`'s authenticated pattern since it's Flutter-app-only just like navigate.

</decisions>

<specifics>
## Specific Ideas

- Reuse `mergeHeightsIntoGpx` (`app/lib/util/route_planner_handoff_util.dart`) for the heights-merge step — it already does exactly this (zips a `heights` array onto a `shape` array of `{lat,lon}` points).
- Reuse `buildNavShape` (`app/lib/util/gpx_util.dart`) to build the Valhalla-shape array from breadcrumb points, same as `launchNavigation` and `buildFinalPlannedGpx` already do.
- The new trace_route call's response shape should mirror how `/valhalla/navigate`'s `+server.ts` already decodes `trip.legs[].shape` via `decodePolyline` — only the shape/coordinates are needed here (no maneuvers), since this is track-cleanup, not turn-by-turn guidance.
- `costingForCategory` / `costingToCategory` already exist in `app/lib/util/valhalla_util.dart` for the category ↔ costing mapping.

</specifics>

<canonical_refs>
## Canonical References

No external specs — requirements fully captured in decisions above. Existing sibling code (`web/src/routes/api/v1/valhalla/{route,height,navigate}/+server.ts`, `app/lib/util/route_planner_handoff_util.dart`) serves as the pattern reference.

</canonical_refs>
