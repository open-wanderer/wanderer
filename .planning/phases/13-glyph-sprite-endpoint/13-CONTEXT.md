# Phase 13: Glyph & Sprite Endpoint - Context

**Gathered:** 2026-07-08
**Status:** Ready for planning
**Source:** Direct Q&A during /gsd-plan-phase 13 (discuss-phase was skipped; these are the load-bearing decisions that emerged while reviewing RESEARCH.md and PATTERNS.md)

<domain>
## Phase Boundary

This phase replaces `/api/v1/map/tileurl` with a single unified `/api/v1/map/config` endpoint that returns tile, glyph, and sprite URLs in one JSON object. It does **not** self-host or vendor the Protomaps glyph/sprite assets, does **not** touch `db/` (Go/PocketBase), and does **not** change the Docker build. It **does** touch the Flutter app: the existing `tile_url_provider.dart` (and its consumer `map_style_provider.dart`) must be updated to call the new merged endpoint instead of `/map/tileurl`.

This supersedes RESEARCH.md's primary recommendation (vendor `basemaps-assets` into the `wanderer-db` image, serve via Go `e.FileFS`, proxy through SvelteKit) and supersedes the original plan to add a *separate* `glyphurl`-style endpoint alongside `tileurl`. RESEARCH.md's facts about the MapLibre glyph/sprite URL contract, the 4-fontstack finding, and the sprite base-URL convention remain valid. PATTERNS.md's `tileurl` code-shape analog (auth guard, `$env/dynamic/private`, `json()`, `handleError`, `@swagger` JSDoc) remains the pattern to follow — it just now lives in one endpoint returning three fields instead of one.

</domain>

<decisions>
## Implementation Decisions

### One unified endpoint, not three
- `/api/v1/map/tileurl` is **replaced** by `/api/v1/map/config` (exact final segment name is Claude's discretion, but it must be a single route, not `tileurl` + `glyphurl` + `spriteurl`).
- Response shape: one JSON object with all three concerns, e.g. `{ tileUrl, glyphs, sprite }` (exact key names Claude's discretion, but must be self-describing and distinct — do not reuse the old `{ url }` shape since it's now ambiguous).
- The old `/api/v1/map/tileurl/+server.ts` route file is deleted/replaced by the new route, not kept alongside it.

### Architecture — config-only, no self-hosting for glyphs/sprite
- No new Go/PocketBase routes. No `db/routes/map_fonts.go` or `map_sprite.go`. No Docker build-stage changes to `db/Dockerfile`.
- No asset vendoring — the ~1000 pre-baked font/sprite files are NOT downloaded into any image or committed to git.
- **Reasoning:** the app's current theme already fetches glyphs/sprites directly from `protomaps.github.io` with zero Wanderer server involvement. Self-hosting a copy was judged redundant. The value this phase adds is the operator override capability (GLYPH-03) plus consolidating the app's server-config fetch into one call, not taking over byte-serving.

### Default behavior — point at Protomaps, not a Wanderer-hosted copy
- Glyph/sprite fields default to the existing public URLs: `https://protomaps.github.io/basemaps-assets/fonts/{fontstack}/{range}.pbf` and `https://protomaps.github.io/basemaps-assets/sprites/v4/{light|dark}`, when the override env var is unset.
- Tile URL field keeps its existing default/override behavior exactly as `tileurl` has it today (Protomaps API fallback using `PROTOMAPS_API_KEY`, override via `TILE_SERVER_URL`) — this is unchanged, just relocated into the merged response.
- One **new** env var covers glyph+sprite override (per original success criterion language "one environment variable" for glyphs/sprite). It is separate from the existing `TILE_SERVER_URL` — tiles and glyphs/sprite are overridden independently, both surfaced through the one `/map/config` response. Exact new env var name is Claude's discretion (e.g. `GLYPH_SERVER_URL`).

### Flutter app changes — parse the unified response
- `app/lib/provider/tile_url_provider.dart` currently does `GET /map/tileurl` → `response.data['url'] as String` → exposes a `Future<String>` via `tileUrlProvider`. This must change to call `/map/config` and parse the full object.
- Decide (Claude's discretion) whether to: (a) keep `tileUrlProvider` as the sole consumer-facing provider but have it internally call a new shared "map config" fetch and extract just the tile URL field (minimal-diff, keeps `map_style_provider.dart` untouched), or (b) introduce a new `map_config_provider.dart` exposing a typed config object (`tileUrl`, `glyphs`, `sprite`) and update `map_style_provider.dart` to read `.tileUrl` from it. Prefer whichever keeps `map_style_provider.dart`'s behavior byte-identical today, since GLYPH-04 (Phase 15) is the phase that actually wires `glyphs`/`sprite` into the rendered style — this phase only needs the app to fetch and hold those two fields, not use them yet.
- Keep the `@Riverpod(keepAlive: true)` caching behavior — config should still be fetched once and cached, matching today's `tileUrlProvider`.
- Regenerate the Riverpod `.g.dart` file for any renamed/new provider (`build_runner`), matching how `tile_url_provider.g.dart` was generated.

### Fontstacks — bundle/reference all 4, not 3
- REQUIREMENTS.md and ROADMAP.md have been corrected: the style's theme references **4** fontstacks (`Noto Sans Regular`, `Noto Sans Medium`, `Noto Sans Italic`, and `Noto Sans Devanagari Regular v1` via a data-driven `text-font` case on `script == "Devanagari"`), not 3. The glyph template is fontstack-agnostic (`{fontstack}/{range}.pbf}`), so this mainly matters for verification: confirm requests for the Devanagari fontstack also resolve against the Protomaps default.

### Sprite variants — light and dark
- Both `light` and `dark` sprite variants must resolve (`.json`/`.png`/`@2x` for each), since the dark theme references `sprites/v4/dark` and Phase 15 (CORE-02) switches themes live.

### Auth posture
- Mirror `tileurl`: the config endpoint requires an authenticated `event.locals.pb.authStore.record`, returning 401 if absent — same as today's `tileurl` route.

### Claude's Discretion
- Exact route path/name for the unified config endpoint (must not be `tileurl`).
- Exact response JSON key names.
- Exact new env var name for the glyph/sprite override.
- Whether the Flutter-side change introduces a new provider file or extends the existing one (see above) — pick the smaller diff that keeps `map_style_provider.dart` behavior unchanged.
- Whether to validate/allowlist the override env var's URL shape at all (low risk — operator-controlled config, not user input).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Precedent pattern (being replaced, not extended)
- `web/src/routes/api/v1/map/tileurl/+server.ts` — the exact code shape to carry forward into the new unified route: `$env/dynamic/private`, auth check, JSON response, `try/catch` → `handleError`, `@swagger` JSDoc. This file is deleted/replaced, not kept.
- `app/lib/provider/tile_url_provider.dart` and `app/lib/provider/tile_url_provider.g.dart` — the Flutter-side consumer that must be updated to call the new endpoint.
- `app/lib/provider/map_style_provider.dart` — consumes `tileUrlProvider`; must still receive a tile URL string with identical behavior after the change.
- `app/lib/provider/api_provider.dart` — the shared Dio client (`baseUrl/api/v1`) used for the request.

### Research (partially superseded — see Phase Boundary above)
- `.planning/phases/13-glyph-sprite-endpoint/13-RESEARCH.md` — MapLibre glyph/sprite URL contract (still valid), 4-fontstack finding (still valid), sprite base-URL convention pitfall (still valid). Ignore its Go/vendoring/Docker recommendations and its assumption of a separate `glyphurl` route — both superseded.
- `.planning/phases/13-glyph-sprite-endpoint/13-PATTERNS.md` — `tileurl` code-shape analog; still the pattern to follow, now applied to a merged 3-field response.

</canonical_refs>

<specifics>
## Specific Ideas

- User's explicit direction: "Rather than having 3 different API endpoints for tileurl, glyphurl and spriteurl, design one endpoint `/map/config` that returns all three in one object. Parse accordingly on the flutter side."

</specifics>

<deferred>
## Deferred Ideas

- Self-hosting/vendoring the glyph/sprite assets (considered and explicitly rejected for this phase — may be revisited later if a hard dependency on `protomaps.github.io` becomes a problem, but out of scope now).
- Actually wiring `glyphs`/`sprite` fields into the rendered map style, caching them, and `file://` rewriting for offline — Phase 15 (GLYPH-04, OFFL-01/02). Phase 13 only needs the app to fetch and hold these fields.

</deferred>

---

*Phase: 13-glyph-sprite-endpoint*
*Context gathered: 2026-07-08 via direct Q&A (discuss-phase skipped)*
