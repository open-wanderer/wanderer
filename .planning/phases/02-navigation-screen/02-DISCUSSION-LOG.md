# Phase 2: Navigation Screen — Discussion Log

**Date:** 2026-06-12
**Areas discussed:** 4 of 4

---

## Area 1: Navigate Button Entry

**Q: Where should the Navigate button live on each screen?**
A: `trail_detail_screen`: fixed full-width button at the bottom in front of (above) the scrollable content. `trail_detail_map_screen`: big button floating over the elevation profile when it is open.

**Q: Navigate button style?**
A: Filled primary color, full-width `ElevatedButton.icon`. (Not an icon-only FAB.)

**Q: go_router path?**
A: `/trail/:id/navigate` (sub-route, keeps trail ID in context, avoids nav bar interference).

---

## Area 2: Map Camera Follow

**Q: Camera follow mode?**
A: Auto-follow with free-pan. Map auto-centers on GPS position; recenter button appears when user pans away.

**Q: Zoom level?**
A: Inherit / user-adjustable. Pinch-zoom is respected; zoom is not locked.

**Q: Orientation toggle placement?**
A: Compass icon button top-right, reusing the existing `MapCompass` widget pattern.

---

## Area 3: Maneuver Advancement

**Q: Advancement detection method?**
A: Distance threshold (~30 m) to the `begin_shape_index` point of the next maneuver. (Left to Claude to decide.)

**Q: What happens at trail end?**
A: Show a completion banner in the maneuver instruction area. Future iteration will offer saving as a summit log (deferred).

**Q: GPS stream source?**
A: `flutter_map_location_marker` stream — same as `CurrentLocationLayer`, reuses existing permission and accuracy setup.

---

## Area 4: API Call Timing

**Q: When does the navigate API call happen?**
A: Before navigating — on the trail detail screen. Navigate button shows a loading spinner while the Dio call is in flight. Rest of screen content still displays.

**Q: On success?**
A: Navigate to `/trail/:id/navigate` passing the `NavigateResponse` as go_router `extra`.

**Q: On failure?**
A: Show a toast message and cancel navigation (stay on trail detail screen).

**Q: What data does NavigationScreen receive?**
A: Trail ID only (via route param). Trail data fetched from existing `trailProvider(id)`. `NavigateResponse` passed as go_router `extra` to avoid a second API call.

---

## Deferred Ideas

- Save completed trail as summit log — user mentioned at trail-end; future iteration
