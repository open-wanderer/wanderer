# Phase 1: Backend API - Research

**Researched:** 2026-06-12
**Domain:** SvelteKit API endpoint + Valhalla turn-by-turn routing integration
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Flutter sends a waypoint lat/lon array — `{ waypoints: [{lat: number, lon: number}], costing?: 'pedestrian' | 'bicycle' }`. No GPX string or encoded polyline.
- **D-02:** Waypoints are lat/lon only — no elevation. Valhalla doesn't use elevation for routing.
- **D-03:** Accept all points the Flutter app sends. No server-side downsampling, no client-side pre-processing required.
- **D-04:** New dedicated `/navigate` endpoint — not an extension of the existing `/route` proxy. `/route` is a raw Valhalla proxy used by the web map editor; `/navigate` is a domain API that abstracts Valhalla's format behind a Flutter-friendly contract.
- **D-05:** Return maneuvers only — no Valhalla route polyline. The trail is already drawn on the map from GPX data via `TrailLayer`.
- **D-06:** Each maneuver object includes: `instruction` (string), `length` (number, km to next turn), `begin_shape_index` (integer, index into the decoded shape points array).
- **D-07:** Response also includes a `shape` array of decoded `[lat, lon]` points corresponding to Valhalla's route. Flutter uses `begin_shape_index` + `shape` together to locate maneuvers as the user moves. Flutter does not need its own polyline decoder.
- **D-08:** Flutter sends an explicit `costing` field (`'pedestrian'` | `'bicycle'`), derived from the trail's category. Costing is **optional** — defaults to `'pedestrian'` if omitted.
- **D-09:** Require authentication — check `event.locals.user` like other protected API endpoints. Unlike `/route` and `/height`, this endpoint is gated.

### Claude's Discretion

(None explicitly marked — implementation details such as Zod schema shape, error message wording, and internal type definitions are at planner discretion within the constraints above.)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| API-01 | New POST /api/v1/valhalla/navigate endpoint accepts a trail (waypoint array per D-01) and returns a structured maneuver list | Valhalla `/route` with `directions_type: "instructions"` returns `trip.legs[].maneuvers[]`. Existing `valhalla/route/+server.ts` is the file-structure template. `getValhallaBaseUrl()` provides the upstream URL. (Architecture Patterns, Code Examples) |
| API-02 | Each maneuver object includes instruction text, distance to next turn, and bearing | Valhalla maneuver provides `instruction` (string), `length` (km), `begin_shape_index` (int), and `bearing_before`/`bearing_after` (degrees from true north). **Note:** requirement says "bearing"; Valhalla has no single `bearing` field — see Open Questions Q1 and Pitfall 2. (Architecture Patterns, Common Pitfalls) |
</phase_requirements>

## Summary

This phase adds one new authenticated SvelteKit endpoint, `POST /api/v1/valhalla/navigate`, that transforms a Flutter-friendly request (`{ waypoints, costing? }`) into a Valhalla `/route` call and reshapes the upstream response into a compact maneuver + shape contract. The codebase already contains every primitive needed: the file-structure template (`valhalla/route/+server.ts`), the upstream URL resolver (`getValhallaBaseUrl()`), the error class (`APIError`), the polyline decoder (`decodePolyline`), Zod 3.25.76 for input validation, and an established `event.locals.user` auth pattern. **No new npm packages are required.**

The single most important technical subtlety: to get human-readable `instruction` text in the response, the Valhalla request must use `directions_type: "instructions"` (the default), **not** `"maneuvers"`. The `"maneuvers"` value returns maneuver objects *without* the `instruction` string, which would silently break the success criterion "Each maneuver object includes instruction text." The CONTEXT.md `code_context` block mentions `directions_type: "maneuvers"` — this is the one place where the locked context and the requirement (API-02 "instruction text") conflict, and the requirement wins.

A second subtlety: the existing `decodePolyline(str, precision = 6)` returns coordinate pairs as `[lng, lat]` (longitude first). D-07 requires the response `shape` array to be `[lat, lon]`. The implementation must flip each pair. Valhalla's default shape precision is 6 (1e6), which matches the decoder's default — so no precision argument change is needed, only the lat/lon flip.

**Primary recommendation:** Create `web/src/routes/api/v1/valhalla/navigate/+server.ts` modeled on `route/+server.ts`, add a Zod schema validating `{ waypoints: [{lat, lon}]+, costing? }`, gate it behind `event.locals.user`, call Valhalla `/route` with `directions_type: "instructions"` and the chosen costing, then transform `trip.legs[*].maneuvers[*]` into `{ instruction, length, begin_shape_index, bearing }` plus a flipped `shape: [lat, lon][]` array. Return descriptive 4xx errors via the existing `handleError`/`APIError` pattern so invalid input never produces a 500.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Request validation (waypoints, costing) | Frontend Server (SSR) — SvelteKit `+server.ts` | — | Zod schema parse runs server-side before any upstream call; this is the API boundary. |
| Authentication / authorization | Frontend Server (SSR) | Database/Storage (PocketBase auth) | `event.locals.user` is populated per-request by PocketBase auth in `hooks.server.ts`; the endpoint reads it. |
| Routing computation (maneuvers, shape) | External service (Valhalla) | — | Valhalla owns turn-by-turn logic; the endpoint is an adapter, not a router. |
| Response transformation (Valhalla → Flutter contract) | Frontend Server (SSR) | — | The SvelteKit endpoint reshapes upstream JSON; this is its core domain responsibility (D-04). |
| Polyline decoding (shape string → `[lat,lon][]`) | Frontend Server (SSR) | — | Server decodes once so Flutter needs no decoder (D-07). Reuses `decodePolyline`. |
| Trail line rendering on map | Mobile / Client (Flutter `TrailLayer`) | — | Out of this phase; informs why the endpoint returns no polyline (D-05). |
| HTTP transport (Flutter → endpoint) | Mobile / Client (Dio) | — | Phase 2 concern; listed for boundary clarity. |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@sveltejs/kit` | 2.60.1 | `RequestEvent`, `json()`, `error()` for the endpoint | Already the framework for all `web/src/routes/api/v1/` endpoints. [VERIFIED: node require @sveltejs/kit/package.json → 2.60.1] |
| `zod` | 3.25.76 | Request body validation (`{ waypoints, costing? }`) | Project convention: "Input validation via Zod schemas on all API endpoints" (CLAUDE.md). [VERIFIED: node require zod/package.json → 3.25.76] |

### Supporting (existing project utilities — reuse, do not reinstall)

| Module | Symbol | Purpose | When to Use |
|--------|--------|---------|-------------|
| `$lib/server/valhalla` | `getValhallaBaseUrl()` | Resolve `VALHALLA_URL` from env | Always — gives the upstream base URL. [CITED: web/src/lib/server/valhalla.ts] |
| `$lib/util/api_util` | `APIError`, `handleError(e)` | Structured error responses; `handleError` already maps `ZodError`→400, `SyntaxError`→400 | Use `handleError(e)` in catch; throw `APIError(status, msg)` for domain errors. [CITED: web/src/lib/util/api_util.ts:6-17,158-168] |
| `$lib/util/polyline_util` | `decodePolyline(str, precision=6)` | Decode Valhalla leg shape to coordinate pairs | Decode `leg.shape` then flip `[lng,lat]`→`[lat,lon]` for the response `shape`. [CITED: web/src/lib/util/polyline_util.ts:26-68] |
| `$lib/models/valhalla` | `Trip`, `Leg`, `Location`, `Summary` | Existing Valhalla response types | Extend: `Leg` currently has only `summary` + `shape`; add a `maneuvers` field or a navigate-specific type (per CONTEXT canonical_refs). [CITED: web/src/lib/models/valhalla.ts:106-123] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing `decodePolyline` | A new dedicated decoder | Pointless — the existing one is correct at precision 6; only a lat/lon flip is needed. |
| `directions_type: "instructions"` | `"maneuvers"` | `"maneuvers"` omits the `instruction` string → fails API-02. Do not use. (See Pitfall 2.) |
| New navigate type | Reusing raw `proxyJsonResponse` | This endpoint *transforms* the response (D-04/D-05), so the raw proxy helper is explicitly NOT used (per CONTEXT canonical_refs). |

**Installation:**
```bash
# None — all dependencies already present in web/package.json.
```

**Version verification:**
```bash
node -e "console.log(require('zod/package.json').version)"           # → 3.25.76 (verified 2026-06-12)
node -e "console.log(require('@sveltejs/kit/package.json').version)" # → 2.60.1  (verified 2026-06-12)
```

## Package Legitimacy Audit

This phase installs **no external packages**. All code reuses existing project dependencies and utilities. The Package Legitimacy Gate is therefore not applicable.

| Package | Registry | Disposition |
|---------|----------|-------------|
| (none) | — | No installs in this phase |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
Flutter app (Phase 2, Dio)
        │  POST /api/v1/valhalla/navigate
        │  body: { waypoints:[{lat,lon}...], costing?:"pedestrian"|"bicycle" }
        ▼
┌─────────────────────────────────────────────────────────────┐
│ SvelteKit endpoint: navigate/+server.ts  (this phase)        │
│                                                              │
│  1. auth gate ──── event.locals.user? ──no──► 401 Unauthorized
│         │ yes                                                │
│  2. parse body ── Zod NavigateSchema ──fail──► 400 invalid_params (handleError)
│         │ ok                                                 │
│  3. build Valhalla request:                                  │
│        { locations: waypoints,                               │
│          costing: costing ?? "pedestrian",                   │
│          directions_type: "instructions" }   ◄── KEY DETAIL  │
│         │                                                    │
│  4. event.fetch(getValhallaBaseUrl() + "/route") ──► Valhalla
│         │                              (network err)──► 502  │
│  5. transform trip.legs[*]:                                  │
│        maneuvers → { instruction, length,                    │
│                      begin_shape_index, bearing }            │
│        decodePolyline(leg.shape) → flip → shape:[lat,lon][]  │
│         ▼                                                    │
│  6. json({ maneuvers, shape })  ──► 200                      │
└─────────────────────────────────────────────────────────────┘
        ▲
Valhalla routing service (external, VALHALLA_URL)
```

### Recommended Project Structure

```
web/src/routes/api/v1/valhalla/
├── route/+server.ts          # existing — raw proxy (template, do NOT modify)
├── height/+server.ts         # existing — raw proxy
└── navigate/+server.ts       # NEW — this phase

web/src/lib/models/
└── valhalla.ts               # add ValhallaManeuver type + maneuvers field on Leg,
                              #   and the navigate request/response types

web/src/lib/models/api/
└── valhalla_navigate_schema.ts   # NEW (suggested) — Zod NavigateRequestSchema,
                                  #   matching the existing api/*_schema.ts convention
```

### Pattern 1: Authenticated transforming Valhalla endpoint

**What:** A POST handler that auth-gates, Zod-validates, calls Valhalla, and reshapes the response.
**When to use:** This is the endpoint.
**Example:**
```typescript
// Source: synthesised from web/src/routes/api/v1/valhalla/route/+server.ts (proxy shape)
//         + web/src/routes/api/v1/search/actor/+server.ts (auth gate)
//         + web/src/lib/util/api_util.ts (handleError)
import { getValhallaBaseUrl } from '$lib/server/valhalla';
import { handleError } from '$lib/util/api_util';
import { decodePolyline } from '$lib/util/polyline_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { z } from 'zod';

const NavigateSchema = z.object({
    waypoints: z.array(z.object({
        lat: z.number().min(-90).max(90),
        lon: z.number().min(-180).max(180),
    })).min(2, 'at_least_two_waypoints'),
    costing: z.enum(['pedestrian', 'bicycle']).default('pedestrian'),
});

export async function POST(event: RequestEvent) {
    if (!event.locals.user) {
        return error(401, 'Unauthorized');           // matches search/actor pattern
    }
    try {
        const baseUrl = getValhallaBaseUrl();
        if (!baseUrl) return json({ message: 'VALHALLA_URL not set' }, { status: 400 });

        const body = NavigateSchema.parse(await event.request.json()); // ZodError → handleError → 400

        const valhallaReq = {
            locations: body.waypoints,                 // [{lat,lon}] is already Valhalla's shape
            costing: body.costing,
            directions_type: 'instructions',           // ◄── REQUIRED for instruction text
        };

        const res = await event.fetch(baseUrl + '/route', {
            method: 'POST',
            body: JSON.stringify(valhallaReq),
        });
        if (!res.ok) {
            const detail = await res.text();
            return json({ message: 'valhalla_error', detail }, { status: 502 });
        }

        const trip = (await res.json()).trip;

        // shape: decode each leg, flip [lng,lat] -> [lat,lon], concatenate
        const shape: [number, number][] = [];
        const maneuvers: { instruction: string; length: number; begin_shape_index: number; bearing: number }[] = [];
        for (const leg of trip.legs) {
            const offset = shape.length;               // account for multi-leg concatenation
            for (const [lng, lat] of decodePolyline(leg.shape)) {
                shape.push([lat, lng]);                // FLIP — decodePolyline returns [lng,lat]
            }
            for (const m of leg.maneuvers ?? []) {
                maneuvers.push({
                    instruction: m.instruction,
                    length: m.length,
                    begin_shape_index: offset + m.begin_shape_index,
                    bearing: m.bearing_after ?? m.bearing_before ?? 0,  // see Open Question Q1
                });
            }
        }
        return json({ maneuvers, shape });
    } catch (e) {
        return handleError(e);                          // ZodError→400, SyntaxError→400, else 500
    }
}
```
*This is a reference skeleton, not a finished implementation. The planner/executor decides final type placement and whether `bearing` is one field or `bearing_before`/`bearing_after` (Open Question Q1).*

### Pattern 2: Multi-leg shape index offset

**What:** Valhalla returns one `leg` per pair of locations. `begin_shape_index` is leg-local. If you concatenate all legs into one `shape` array, you must add a running offset to each maneuver's `begin_shape_index`.
**When to use:** Whenever `waypoints.length > 2` (more than one leg). Hiking trails sent as full waypoint arrays (D-03) will be multi-leg.
**Why it matters:** Without the offset, Flutter would index maneuvers from leg 2+ into the wrong shape coordinates. See Pitfall 1.

### Anti-Patterns to Avoid

- **Using `proxyJsonResponse`:** That helper returns the raw upstream body. This endpoint must transform (D-04/D-05). Reusing it defeats the purpose.
- **Re-encoding then returning a polyline:** D-05 forbids returning a route polyline; return only the decoded `shape` array for indexing.
- **Server-side category→costing inference:** D-08 explicitly moves this to Flutter because Category is free-form. Do not parse category names server-side.
- **Throwing raw errors / letting Valhalla 5xx propagate as 500:** Success criterion 3 requires descriptive non-500 errors for bad input. Wrap upstream failures as 502 and validation failures as 400.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Decoding Valhalla shape | A new polyline decoder | `decodePolyline` (`$lib/util/polyline_util.ts`) | Already correct at precision 6; battle-tested in the web editor. Only flip coords. |
| Request validation | Manual `if (typeof x !== ...)` checks | Zod `NavigateSchema` | Project convention; gives free 400 mapping via `handleError`. |
| Error responses | Ad-hoc `json({error}, {status})` | `APIError` + `handleError` | Consistent error shape (`{message, detail}`) across the API; already maps ZodError/SyntaxError. |
| Resolving Valhalla URL | Reading `env.VALHALLA_URL` directly | `getValhallaBaseUrl()` | Handles `PUBLIC_` fallback + URL normalization (`$lib/server/url.ts`). |
| Auth check | Custom token parsing | `if (!event.locals.user)` | Populated per-request by PocketBase hooks; the established pattern. |

**Key insight:** This phase is almost entirely *composition of existing utilities*. The only genuinely new logic is the Valhalla→Flutter response transform and the multi-leg shape-index offset.

## Common Pitfalls

### Pitfall 1: Leg-local shape indices break for multi-waypoint trails
**What goes wrong:** Each Valhalla `leg.maneuvers[].begin_shape_index` is relative to *that leg's* shape. Trails sent as full waypoint arrays (D-03) produce many legs. Concatenating shapes without offsetting indices makes Flutter point maneuvers at the wrong coordinates.
**Why it happens:** Valhalla's API is leg-centric; the Flutter contract is a single flat `shape` array.
**How to avoid:** Maintain a running `offset = shape.length` before appending each leg, and add it to every `begin_shape_index` in that leg (see Pattern 2).
**Warning signs:** Maneuvers appear "ahead of" the user; off-by-N coordinate jumps that grow with leg count.

### Pitfall 2: `directions_type: "maneuvers"` returns no instruction text
**What goes wrong:** Setting `directions_type: "maneuvers"` (mentioned in CONTEXT.md code_context) returns maneuver geometry/length but omits the human-readable `instruction` string — directly failing success criterion 2 / API-02.
**Why it happens:** Valhalla distinguishes `none` (no maneuvers), `maneuvers` (maneuvers, no narrative), and `instructions` (maneuvers **with** narrative, the default).
**How to avoid:** Use `directions_type: "instructions"` (or omit it — it's the default). [CITED: valhalla.github.io/valhalla/api/turn-by-turn/api-reference/]
**Warning signs:** Response maneuvers have empty/absent `instruction`.

### Pitfall 3: Coordinate order flip (`[lng,lat]` vs `[lat,lon]`)
**What goes wrong:** `decodePolyline` returns pairs as `[lng, lat]` (longitude first — line 64 of `polyline_util.ts`). D-07 requires `shape` as `[lat, lon]`. Forgetting the flip puts every point in the ocean.
**Why it happens:** GeoJSON convention (lng-first) vs. the lat-first contract chosen for Flutter.
**How to avoid:** Push `[lat, lng]` explicitly when building the response `shape`.
**Warning signs:** Map renders points in a mirrored/wrong hemisphere; coordinates near `(lon, lat)` swapped.

### Pitfall 4: Shape precision mismatch
**What goes wrong:** Valhalla's default polyline precision is 6 (1e6). Some Valhalla deployments / OSRM-style output use precision 5. Decoding at the wrong precision scales all coordinates by 10x.
**Why it happens:** Two precisions exist in the wild; `decodePolyline` defaults to 6 (correct for Valhalla's native `/route` response).
**How to avoid:** Call `decodePolyline(leg.shape)` with the default precision 6 (matches Valhalla native response). Do **not** pass `5`. [CITED: valhalla docs — shape uses 6 digits]
**Warning signs:** Coordinates off by a factor of 10.

### Pitfall 5: Valhalla upstream errors surfacing as 500
**What goes wrong:** A Valhalla routing failure (e.g., unroutable points) returns a non-2xx that, if unhandled, becomes a generic 500 — violating success criterion 3.
**Why it happens:** `event.fetch` doesn't throw on HTTP error status; only on network failure.
**How to avoid:** Check `res.ok` and return a descriptive 502 with Valhalla's message; catch network failures and return 502 too.
**Warning signs:** Bad/sparse waypoints produce opaque 500s instead of helpful messages.

## Code Examples

### Building the Valhalla request body
```typescript
// Source: web/src/lib/stores/valhalla_store.svelte.ts:75-81 (existing web caller shape)
//         + valhalla.github.io/valhalla/api/turn-by-turn/api-reference (directions_type)
const valhallaReq = {
    locations: body.waypoints,            // [{lat, lon}, ...]  — Valhalla's native location shape
    costing: body.costing,                // "pedestrian" | "bicycle"
    directions_type: 'instructions',      // default; REQUIRED for instruction text
};
const res = await event.fetch(getValhallaBaseUrl() + '/route', {
    method: 'POST', body: JSON.stringify(valhallaReq),
});
```

### Auth gate (return 401, not 500)
```typescript
// Source: web/src/routes/api/v1/search/actor/+server.ts:98-100
if (!event.locals.user) {
    return error(401, 'Unauthorized');
}
```

### Decode + flip shape
```typescript
// Source: web/src/lib/util/polyline_util.ts:26-68 (decodePolyline returns [lng,lat])
const flatShape: [number, number][] = [];
for (const [lng, lat] of decodePolyline(leg.shape)) {  // default precision 6
    flatShape.push([lat, lng]);                         // -> [lat, lon] for Flutter (D-07)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct Flutter → Valhalla calls | SvelteKit endpoint proxies Valhalla (keeps creds server-side) | Project decision (STATE.md) | This phase follows that decision — `/navigate` lives in SvelteKit. |
| Raw Valhalla proxy (`/route`, `/height`) | Domain-shaped, authed endpoint (`/navigate`) | This phase (D-04) | New endpoint transforms + auth-gates rather than passing through. |

**Deprecated/outdated:** None relevant. The Valhalla turn-by-turn API and the SvelteKit `RequestEvent` API are both stable at the pinned versions.

## Runtime State Inventory

Not applicable — this is a greenfield additive phase (one new file + type additions). No rename/refactor/migration. No stored data, live-service config, OS-registered state, secrets, or build artifacts are altered.

- **Stored data:** None — verified, no datastore writes (endpoint is read/transform only).
- **Live service config:** None — verified, no external service configuration changes.
- **OS-registered state:** None.
- **Secrets/env vars:** Uses existing `VALHALLA_URL` (already configured); no new secrets.
- **Build artifacts:** None.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `VALHALLA_URL` is configured in the target environment (it powers existing `/route` and `/height`). | Environment Availability | If unset, endpoint returns 400 "VALHALLA_URL not set" — handled gracefully, not a hard block. |
| A2 | Flutter waypoint arrays will frequently exceed 2 points (multi-leg), making the shape-index offset mandatory. | Pitfall 1 | If always 2 points, offset is a no-op (still correct). Low risk. |
| A3 | The Valhalla deployment returns shape at precision 6 (its documented default). | Pitfall 4 | If the deployment is configured for precision 5, coordinates scale 10×. Verify against a live response during execution. |

**All other claims are VERIFIED (codebase grep / node require) or CITED (official Valhalla docs / source files).**

## Open Questions

1. **API-02 says "bearing" (singular); Valhalla provides `bearing_before` and `bearing_after`.**
   - What we know: Valhalla maneuvers expose `bearing_before` and `bearing_after` (clockwise degrees from true north). D-06 lists `instruction`, `length`, `begin_shape_index` but does *not* list a bearing field, while API-02 *requires* bearing.
   - What's unclear: Whether the Flutter contract wants a single `bearing` (e.g., `bearing_after`, the direction of travel after the turn) or both fields.
   - Recommendation: Include a single `bearing` = `bearing_after` (most useful for "which way am I now heading after this maneuver"), and optionally pass through both. Confirm with the planner; cheap to adjust. This is the one spot where D-06 and API-02 differ — flag for the planner to reconcile.

2. **`directions_type` conflict between CONTEXT and requirements.**
   - What we know: CONTEXT.md code_context says `directions_type: "maneuvers"`; API-02 requires instruction text, which only `"instructions"` provides.
   - What's unclear: Whether the CONTEXT note was shorthand or an intentional choice.
   - Recommendation: Use `"instructions"`. Document this in the plan so the executor doesn't follow the CONTEXT shorthand literally.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Valhalla service (`VALHALLA_URL`) | The endpoint's upstream `/route` call | Assumed ✓ (powers existing `/route`, `/height`) | runtime env | Endpoint returns 400 "VALHALLA_URL not set" if absent |
| `zod` | Request validation | ✓ | 3.25.76 | — |
| `@sveltejs/kit` | Endpoint runtime | ✓ | 2.60.1 | — |
| Node.js | SvelteKit server | ✓ | 22 (per CLAUDE.md) | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** `VALHALLA_URL` not being set is handled in-code (graceful 400), not a build blocker.

## Validation Architecture

`workflow.nyquist_validation` is `false` in `.planning/config.json` — section skipped per instructions.

For reference only (not a phase gate): the project uses **Vitest 4.1.4** (`vite.config.ts` → `test.include: ['src/**/*.{test,spec}.{js,ts}']`). If the planner chooses to add tests, an endpoint unit test at `web/src/routes/api/v1/valhalla/navigate/server.test.ts` could mock `event.fetch` and assert the transform (instruction passthrough, shape flip, index offset, 400 on bad input). No existing tests cover the API layer.

## Security Domain

`security_enforcement: true`, `security_asvs_level: 1`, `security_block_on: high`.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | `event.locals.user` gate (D-09) — populated by PocketBase auth hooks. Return 401 when absent. |
| V3 Session Management | no (delegated) | Session handled upstream by PocketBase auth store / cookie jar; endpoint only reads `locals.user`. |
| V4 Access Control | partial | Authenticated-only access. No per-resource ownership check needed — routing a waypoint list isn't a user-owned resource. |
| V5 Input Validation | yes | Zod `NavigateSchema`: bounded lat/lon, `costing` enum, min waypoint count. Rejects malformed JSON via `handleError` → 400. |
| V6 Cryptography | no | No crypto in this endpoint. |

### Known Threat Patterns for SvelteKit + external routing service

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthenticated abuse of a paid/heavy routing backend | Spoofing / DoS | Auth gate (D-09) restricts to logged-in users. |
| Oversized / unbounded waypoint payload exhausting Valhalla or memory | Denial of Service | `BODY_SIZE_LIMIT` (SvelteKit env) caps request size; consider a Zod `.max()` on waypoint count. Flag to planner — D-03 says accept all points, so cap should be generous, not absent. |
| Injection of arbitrary Valhalla parameters via passthrough | Tampering | Build the Valhalla request body explicitly from validated fields (`locations`, `costing`, `directions_type`) — do **not** spread the raw client body into the upstream request. |
| Coordinate values out of range causing upstream errors → 500 | Tampering / availability | Zod `.min(-90).max(90)` / `.min(-180).max(180)` on lat/lon; upstream failures wrapped as 502. |
| SSRF via `VALHALLA_URL` | — | URL comes from server env via `getValhallaBaseUrl()`, never from the client. No SSRF surface. |

**Highest-severity item:** the unbounded-payload DoS vector (medium). With `security_block_on: high`, it does not block, but the planner should add a generous Zod `.max()` waypoint cap and rely on `BODY_SIZE_LIMIT`. Explicitly constructing the upstream body (not spreading client input) closes the parameter-injection vector.

## Sources

### Primary (HIGH confidence)
- Codebase (verified via Read/grep): `web/src/routes/api/v1/valhalla/route/+server.ts`, `.../height/+server.ts`, `.../search/actor/+server.ts`, `.../trail/+server.ts`, `web/src/lib/server/valhalla.ts`, `web/src/lib/server/http.ts`, `web/src/lib/server/url.ts`, `web/src/lib/util/api_util.ts`, `web/src/lib/util/polyline_util.ts`, `web/src/lib/models/valhalla.ts`, `web/src/lib/stores/valhalla_store.svelte.ts`, `web/src/lib/models/api/trail_schema.ts`, `web/src/app.d.ts`, `web/vite.config.ts`.
- Valhalla turn-by-turn API reference — maneuver fields, `directions_type`, shape precision, length units: https://valhalla.github.io/valhalla/api/turn-by-turn/api-reference/
- Version verification: `node -e require(...)/package.json.version` for zod (3.25.76) and @sveltejs/kit (2.60.1).

### Secondary (MEDIUM confidence)
- Valhalla turn-by-turn overview & narrative instruction behavior (verbal/written instruction generation under `directions_type: instructions`): https://valhalla.github.io/valhalla/api/turn-by-turn/overview/ ; corroborated by https://deepwiki.com/valhalla/valhalla/7-turn-by-turn-directions

### Tertiary (LOW confidence)
- None relied upon for any claim.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all deps verified present; no new installs.
- Architecture: HIGH — derived directly from existing endpoint source + official Valhalla docs.
- Pitfalls: HIGH — index offset, coord flip, and precision verified against actual `decodePolyline` source + Valhalla docs; `directions_type` behavior confirmed via official reference.

**Research date:** 2026-06-12
**Valid until:** ~2026-07-12 (stable stack; Valhalla API and pinned framework versions are not fast-moving)
