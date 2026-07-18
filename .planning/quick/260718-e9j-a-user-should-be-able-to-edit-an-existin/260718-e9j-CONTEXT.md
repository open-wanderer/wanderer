# Quick Task 260718-e9j: A user should be able to edit an existing route in the trail planner - Context

**Gathered:** 2026-07-18
**Status:** Ready for planning

<domain>
## Task Boundary

A user should be able to edit an existing route in the trail planner. The `trail_create_screen` gives the option to open a trail in the route planner. This navigates to `route_planner_screen`. Anchors are prepopulated (look at the web version to see how this is done). Saving the trail in the route planner returns the edited trail to the awaiting `trail_create_screen`.

</domain>

<decisions>
## Implementation Decisions

### Return flow (route_planner_screen -> trail_create_screen)
- Use pop-with-result: `route_planner_screen` is pushed via `context.push(...)` and awaited by `trail_create_screen`; on Save it calls `context.pop(editedTrail)` (or equivalent result payload) instead of the current one-way `finishPlanning()` forward push to `/trail/create/edit`.
- `trail_create_screen` merges the returned route/track data into its existing in-memory `Trail`, preserving title/description/photos/other fields already entered by the user. No new screen instance is created and no navigation stack push of a duplicate create screen happens.
- The existing GPX-import `finishPlanning()` forward-push flow (`route_planner_handoff_util.dart`) is for the import case and is out of scope to remove/change unless required to share code cleanly — the new edit entry path should use its own return-based mechanism.

### Anchor prepopulation
- Mirror the web app's logic exactly: `web/src/routes/trail/edit/[id]/+page.svelte:initRouteAnchors` — one anchor per track-segment boundary (first point of each `trkseg`, plus the last point of the final segment). No denser interior sampling.
- No reverse-geocoding performed at load time for prepopulated anchors (matches web behavior).

### Entry point
- An `IconButton` placed to the left of the "Save" button in `trail_create_screen`'s app bar, which opens the route planner for the current trail.
- Enabled only when the trail has a recorded/existing track (GPX data) to seed anchors from.

### Claude's Discretion
- Exact icon choice for the new app bar button (e.g. route/map/edit-route icon) consistent with existing Material icon usage in the app.
- Whether `RouteAnchors` provider needs a new "seed from trail" initializer method vs. extending `resetForSession`, and how anchors reverse-map back into the trail's track/waypoints on save (data shape reconciliation) — implementer's call based on `RouteAnchor`/`RouteAnchorsState` shapes found during exploration (`app/lib/provider/route_anchor_provider.dart`, `app/lib/models/route_anchor.dart`).
- Router registration changes needed in `router_provider.dart` to support passing the existing trail into `/route-planner` and returning a result.

</decisions>

<specifics>
## Specific Ideas

Exploration findings that inform implementation (from a prior codebase scan, not yet re-verified by the planner):
- `trail_create_screen.dart` (`app/lib/routes/trail_create_screen.dart`) currently has no navigation to `route_planner_screen` at all — this is new.
- `route_planner_screen.dart` constructor (lines ~50-55) currently takes `travelProfile`, `initialCostingOptions`, `initialCenter` — no existing-trail parameter yet.
- `routeAnchorsProvider.resetForSession` (`app/lib/provider/route_anchor_provider.dart:340`) always resets anchors to empty; there is no current path to seed anchors from an existing trail's track.
- Existing "Finish" flow: `_onFinish` (route_planner_screen.dart:451) -> `finishPlanning()` (`app/lib/util/route_planner_handoff_util.dart:103-132`) builds a draft `Trail` and does a forward `push('/trail/create/edit', extra: draftTrail)` — this is the GPX-import pattern, kept separate from the new edit-return flow per the decision above.
- Web anchor prepopulation reference: `web/src/routes/trail/edit/[id]/+page.svelte:553-577` (`initRouteAnchors`).

</specifics>

<canonical_refs>
## Canonical References

No external specs — requirements fully captured in decisions above. Web app source (`web/src/routes/trail/edit/[id]/+page.svelte`) serves as the behavioral reference for anchor prepopulation, per the task description.

</canonical_refs>
