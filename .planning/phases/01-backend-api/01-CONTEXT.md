# Phase 1: Backend API - Context

**Gathered:** 2026-06-12
**Status:** Ready for planning

<domain>
## Phase Boundary

A new SvelteKit endpoint `POST /api/v1/valhalla/navigate` that accepts trail waypoints and an optional costing type from Flutter, calls Valhalla for turn-by-turn maneuvers, and returns a structured response containing the maneuver list and decoded route shape points.

</domain>

<decisions>
## Implementation Decisions

### Input Format
- **D-01:** Flutter sends a waypoint lat/lon array — `{ waypoints: [{lat: number, lon: number}], costing?: 'pedestrian' | 'bicycle' }`. No GPX string or encoded polyline.
- **D-02:** Waypoints are lat/lon only — no elevation. Valhalla doesn't use elevation for routing.
- **D-03:** Accept all points the Flutter app sends. No server-side downsampling, no client-side pre-processing required.
- **D-04:** New dedicated `/navigate` endpoint — not an extension of the existing `/route` proxy. Reason: `/route` is a raw Valhalla proxy used by the web map editor. `/navigate` is a domain API that abstracts Valhalla's format behind a Flutter-friendly contract.

### Response Contract
- **D-05:** Return maneuvers only — no Valhalla route polyline. The trail is already drawn on the map from GPX data via `TrailLayer` (see `app/lib/components/map/trail_layer.dart`).
- **D-06:** Each maneuver object includes: `instruction` (string), `length` (number, km to next turn), `begin_shape_index` (integer, index into the decoded shape points array).
- **D-07:** Response also includes a `shape` array of decoded `[lat, lon]` points corresponding to Valhalla's route. Flutter uses `begin_shape_index` + `shape` together to locate maneuvers as the user moves. Flutter does not need its own polyline decoder.

### Costing Profile
- **D-08:** Flutter sends an explicit `costing` field (`'pedestrian'` | `'bicycle'`), derived from the trail's category. Costing is **optional** — defaults to `'pedestrian'` if omitted. This avoids fragile server-side category name matching (Category is a free-form `{id, name}` record, not an enum).

### Authentication
- **D-09:** Require authentication — check `event.locals.user` like other protected API endpoints (trail endpoints, etc.). Unlike the existing thin Valhalla proxies (`/route`, `/height`), this endpoint is gated.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing Valhalla API Layer
- `web/src/routes/api/v1/valhalla/route/+server.ts` — Existing thin proxy pattern. Navigate endpoint follows same file structure but adds input parsing, costing, and response transformation.
- `web/src/lib/server/valhalla.ts` — `getValhallaBaseUrl()` utility — reuse this.
- `web/src/lib/server/http.ts` — `proxyJsonResponse()` helper — the navigate endpoint does NOT use this (it transforms the response); shown here for context on what NOT to reuse.
- `web/src/lib/models/valhalla.ts` — Existing Valhalla types (`ValhallaRouteResponse`, `Leg`, etc.). The `Leg` type only has `shape` — `maneuvers` will need to be added or a new type defined for the navigate endpoint.

### Trail Map Rendering (informs response contract)
- `app/lib/components/map/trail_layer.dart` — Trail is drawn from `trail.expand?.gpx`, not from Valhalla shape. Confirms maneuvers-only response is correct (no need to return polyline from navigate endpoint).

### API Conventions
- `web/src/routes/api/v1/trail/+server.ts` — Example of an authenticated SvelteKit endpoint with Zod validation. Follow this pattern for auth check and error handling.
- `web/src/lib/util/api_util.ts` — `APIError` class for error responses.

### Requirements
- `.planning/REQUIREMENTS.md` — API-01 and API-02 define the contract: `POST /api/v1/valhalla/navigate` accepts trail, returns maneuver list with `instruction`, `distance`, `bearing`.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/lib/server/valhalla.ts` `getValhallaBaseUrl()` — reuse to get Valhalla base URL from env
- `web/src/lib/util/api_util.ts` `APIError` — use for structured error responses
- `web/src/routes/api/v1/valhalla/route/+server.ts` — file structure template for the new endpoint

### Established Patterns
- Zod schema for request body validation — used on all API endpoints (`RecordListOptionsSchema`, `TrailCreateSchema`, etc.)
- `event.locals.user` auth check — present in protected endpoints, not in existing Valhalla proxies
- JSDoc `@swagger` comment block — present on all API routes for OpenAPI docs

### Integration Points
- New file: `web/src/routes/api/v1/valhalla/navigate/+server.ts`
- Valhalla call: `POST ${getValhallaBaseUrl()}/route` with `directions_type: "maneuvers"` and the appropriate costing model
- Flutter caller (Phase 2): will POST to `/api/v1/valhalla/navigate` via the existing Dio client at app startup of navigation

</code_context>

<specifics>
## Specific Ideas

- The user confirmed the trail is already displayed on the map by encoding the GPX into a polyline (via `TrailLayer`), so the navigate response must NOT redundantly return the route shape as a polyline — only the decoded shape points array needed for maneuver indexing.
- Costing type is derived in Flutter from the trail's category (a free-form string), and passed explicitly to the API rather than inferred server-side.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 1-Backend API*
*Context gathered: 2026-06-12*
