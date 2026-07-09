---
phase: 13-glyph-sprite-endpoint
plan: 01
subsystem: map-config
tags: [sveltekit, maplibre, glyphs, sprites, flutter-provider]
provides:
  - Unified /api/v1/map/style-sources SvelteKit endpoint returning tileUrl, glyphUrl, spriteUrl
  - Operator override via MAP_ASSETS_URL (glyphs+sprite), independent of existing TILE_SERVER_URL
  - Flutter MapStyleSources model + mapStyleSourcesProvider consuming the endpoint
tech-stack:
  added: []
  patterns: ["SvelteKit config route mirroring an existing precedent (tileurl)", "Riverpod keepAlive provider + freezed model for server config"]
key-files:
  created:
    - web/src/routes/api/v1/map/style-sources/+server.ts
    - app/lib/models/map_style_sources.dart
    - app/lib/provider/map_style_sources_provider.dart
  modified:
    - app/lib/provider/map_style_provider.dart
  deleted:
    - web/src/routes/api/v1/map/tileurl/+server.ts
    - app/lib/provider/tile_url_provider.dart
key-decisions:
  - "Executed manually outside /gsd-execute-phase — user built and committed the endpoint directly (commit 4c540b32 'Add MapStyleSources'), then iterated on scope live in conversation rather than via the 13-01-PLAN.md flow."
  - "Route/model naming deviated from 13-01-PLAN.md: shipped as /api/v1/map/style-sources / MapStyleSources / mapStyleSourcesProvider, not /api/v1/map/config / MapConfig / mapConfigProvider. Same contract (tileUrl, glyph template, sprite base), different names."
  - "Sprite override initially shipped without an override var and hardcoded to /v4/dark (GLYPH-03 gap); patched after review to derive both glyphUrl and spriteUrl from one MAP_ASSETS_URL base, restoring single-env-var override for both and un-hardcoding the theme."
duration: n/a (interactive session, not timed agent execution)
completed: 2026-07-08
---

# Phase 13: Glyph & Sprite Endpoint Summary

**A single SvelteKit endpoint (`/api/v1/map/style-sources`) now resolves tile, glyph, and sprite URLs in one call, replacing `/api/v1/map/tileurl`, with one new operator override env var covering glyphs+sprite.**

## Accomplishments
- Consolidated three prospective config endpoints (tileurl/glyphurl/spriteurl) into one, per direct user redirect during planning — `web/src/routes/api/v1/map/style-sources/+server.ts`.
- Glyph/sprite origin defaults to Protomaps' public `basemaps-assets` host (no self-hosting/vendoring — a scope reduction from the original RESEARCH.md recommendation, explicitly chosen by the user since the app already fetched these directly from Protomaps with zero server involvement).
- `MAP_ASSETS_URL` operator env var overrides both glyph and sprite origin together, independent of the pre-existing `TILE_SERVER_URL`; `spriteUrl` is returned as a theme-agnostic base (`.../sprites/v4`) so both light and dark variants resolve.
- Flutter app repointed from `tileUrlProvider`/`tile_url_provider.dart` to a typed `MapStyleSources` model + `mapStyleSourcesProvider`; `map_style_provider.dart` reads `.tileUrl` off it with otherwise-identical rendering behavior.

## Decisions & Deviations
- This phase's actual execution bypassed `/gsd-execute-phase` — the user implemented and committed the endpoint directly, then the scope was corrected live (self-hosting → Protomaps-default, three routes → one unified route) via conversation rather than a replan cycle. ROADMAP.md/REQUIREMENTS.md were updated to match before the (superseded) 13-01-PLAN.md was ever executed as written.
- Naming differs from the plan (`style-sources`/`MapStyleSources` vs. the plan's `config`/`MapConfig`) — functionally equivalent, not reconciled back into the plan doc.
- A real GLYPH-03 gap (sprite had no override, was hardcoded to `/v4/dark`) was caught and fixed post-hoc directly in `+server.ts`.

## Next Phase Readiness
Phase 15 (GLYPH-04) can consume `mapStyleSourcesProvider`'s `glyphUrl`/`spriteUrl` fields to wire glyphs/sprite into the MapLibre style — those fields are fetched and held today but not yet used in rendering.
