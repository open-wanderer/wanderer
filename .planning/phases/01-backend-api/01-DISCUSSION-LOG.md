# Phase 1: Backend API - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-12
**Phase:** 1-Backend API
**Areas discussed:** Input format, Response contract, Costing profile, Auth requirement

---

## Input Format

| Option | Description | Selected |
|--------|-------------|----------|
| Waypoint lat/lon array | Flutter sends `[{lat, lon}, ...]`. Simple to parse, compact, Valhalla expects lat/lon anyway. | ✓ |
| Raw GPX string | Flutter sends full GPX XML. Server must parse before calling Valhalla. | |
| Encoded polyline | Pre-encoded polyline. Server decodes before calling Valhalla. | |

**User's choice:** Waypoint lat/lon array

| Option | Description | Selected |
|--------|-------------|----------|
| Lat/lon only | Valhalla doesn't use elevation for routing. Minimal payload. | ✓ |
| Lat/lon + elevation | Include ele for completeness. Valhalla ignores it. | |

**User's choice:** Lat/lon only

| Option | Description | Selected |
|--------|-------------|----------|
| Accept all points, server decides | Flutter sends full list. No client downsampling. | ✓ |
| Flutter downsamples first | Flutter reduces to 50–100 key points. Smaller payload. | |
| Document a max-point limit | Define max (e.g. 500), validate server-side, return 400 if exceeded. | |

**User's choice:** Accept all points, server decides

| Option | Description | Selected |
|--------|-------------|----------|
| New /navigate endpoint | Clean domain API, Flutter sends simple waypoints, server handles Valhalla format. | ✓ |
| Extend /route with a navigate mode | Add ?mode=navigate param. Reuses infrastructure, mixes concerns. | |
| Flutter calls /route directly | No server changes. Flutter tightly coupled to Valhalla format. | |

**User's choice:** New /navigate endpoint
**Notes:** User questioned why a new endpoint rather than reusing /route. Explanation: /route is a raw Valhalla proxy used by the web map editor; /navigate is a domain API for Flutter. The user confirmed the new endpoint approach after the explanation.

---

## Response Contract

| Option | Description | Selected |
|--------|-------------|----------|
| Maneuvers + route shape | Return both maneuver array and encoded route polyline. Phase 2 needs shape for map. | |
| Maneuvers only | Return just maneuver list. Flutter draws trail from existing GPX. | ✓ |
| You decide | Leave shape decision to planner/researcher. | |

**User's choice:** Maneuvers only
**Notes:** User noted "The trail is already displayed on the map by encoding the GPX into a polyline. See trail_layer.dart." — confirmed the trail is rendered from GPX, not from Valhalla's returned shape.

| Option | Description | Selected |
|--------|-------------|----------|
| instruction + length + begin_shape_index | Valhalla fields. begin_shape_index lets Flutter locate maneuver on route. Matches API-02. | ✓ |
| instruction + length + bearing_after | Swap shape_index for bearing. Simpler but no GPS-to-maneuver mapping. | |
| Full Valhalla maneuver object | Pass all fields through. Flutter picks what it needs. | |

**User's choice:** instruction + length + begin_shape_index

| Option | Description | Selected |
|--------|-------------|----------|
| Decoded shape points in response | Return full decoded lat/lon array. Flutter doesn't need a polyline decoder. | ✓ |
| begin_shape_index only, Flutter decodes | Flutter receives encoded polyline and decodes it. | |
| Skip shape points entirely | Flutter uses GPX coordinates, not Valhalla shape. | |

**User's choice:** Decoded shape points in response

---

## Costing Profile

**User's input:** "Make the costing model dependent on the trail's category"

After investigating — Category is a free-form `{id, name}` record with user-defined names, not an enum.

| Option | Description | Selected |
|--------|-------------|----------|
| Flutter sends costing type explicitly | Request includes `costing: 'pedestrian' \| 'bicycle'`. Flutter derives from category. | ✓ |
| Server maps category name to costing | Server pattern-matches category name. Fragile with user-defined strings. | |
| Hardcode pedestrian for v1 | Ignore category. All trails navigate as pedestrian. | |

**User's choice:** Flutter sends costing type explicitly

| Option | Description | Selected |
|--------|-------------|----------|
| Optional, default pedestrian | Flutter omits costing for hiking (most cases), includes for bike trails. | ✓ |
| Required | Flutter must always send costing. Explicit contract, fails fast if omitted. | |

**User's choice:** Optional, default pedestrian

---

## Auth Requirement

| Option | Description | Selected |
|--------|-------------|----------|
| Require auth | Check event.locals.user. Consistent with other protected endpoints. | ✓ |
| Open, no auth required | Same as existing /route and /height endpoints. | |

**User's choice:** Require auth

---

## Claude's Discretion

None — user made explicit choices for all gray areas.

## Deferred Ideas

None — discussion stayed within phase scope.
