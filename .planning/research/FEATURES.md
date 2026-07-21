# Feature Research

**Domain:** Region-based offline map/tile management in a mobile hiking/navigation app
**Researched:** 2026-07-21
**Milestone:** v1.6 Offline Region Tile Repository
**Confidence:** MEDIUM (WebSearch-derived, cross-checked across multiple apps; no Context7/official API docs exist for competitor UX — these are product/UX conventions, not library APIs)

## Scope Note

This research is scoped to the **new** v1.6 capability only: app-wide predefined-region offline map management + the trail-download guard. It does not re-litigate what's already decided in PROJECT.md (bundled `regions.json`, `notDownloaded/downloading/downloaded/updateAvailable` status enum, session-scoped pause/resume, bbox-only regions, no remote manifest). Where research confirms or refines those decisions, it's called out explicitly.

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist in any "download regions for offline use" surface. Missing these makes the feature feel broken or untrustworthy.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Flat or lightly-grouped region list with search/filter | Users scan for their region by name, not by drilling a taxonomy — this is true even in apps with deep hierarchies (OsmAnd) where a search box at the top is the actual primary interaction | LOW | A hiking app manifest is small (tens, not thousands, of regions) — a flat list is the right complexity; no country→state→region tree needed. See Complexity Guidance below. |
| Per-region download button + progress indicator | Universal pattern (OsmAnd, Organic Maps, Gaia GPS, AllTrails) — tap to start, inline progress bar/percentage while downloading | LOW-MEDIUM | Already scoped: reuse `DownloadedTilePackage` status field to drive per-row UI state. |
| Size shown *before* download starts | Explicit UX-pattern guidance: "inform users of the anticipated package size before they start downloading it" (mapuipatterns.com) — users decide whether to download based on size vs available storage | LOW | `regions.json` already carries size fields per PROJECT.md — just needs to render pre-download. |
| Delete/remove downloaded region + reclaim storage | Every reviewed app (OsmAnd, Organic Maps, Gaia GPS) treats delete as a first-class action, usually via swipe or a per-region overflow/detail view | LOW | Already scoped as a v1.6 requirement. |
| Downloaded-region visual distinction in the list | Downloaded rows need a different affordance than not-downloaded rows (checkmark, filled icon, "Downloaded" label) so users don't re-trigger a full download by accident | LOW | Maps directly to the 4-state enum already decided — each status needs a distinct icon/label, not just a progress bar that disappears. |
| Pause/resume affordance during download | Long region downloads (vector + DEM can be 10s–100s of MB) on cellular need a pause control; users expect to stop and continue without restarting from zero | LOW-MEDIUM | Already scoped as session-only pause/resume — matches OsmAnd's actual behavior (resume works within a session; cross-restart resume is a known OsmAnd pain point per GitHub issues, so *not* supporting it in v1 is a reasonable, validated cut, not a gap). |
| Total disk usage summary | Users managing multiple regions want to see "X GB used" in one place, especially before deciding to delete something | LOW | Already scoped ("total disk usage" on the Settings page). |
| Guard/prompt when content needs an undownloaded region | Komoot explicitly surfaces "which region is needed" when a tour starts in a locked/undownloaded region rather than silently failing | MEDIUM | This is the "trail needs region X" flow — see Guard UX section below for the recommended shape. |

### Differentiators (Competitive Advantage)

Not required for v1, but observed in mature apps as what separates a good offline experience from a merely functional one.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Layer/size breakdown per region (base map vs DEM) | OsmAnd treats contour lines/hillshade as a visibly separate download line from the base map, with its own size — lets users skip terrain shading on a small phone to save space | LOW | Already scoped as "size breakdown (vector vs DEM)" — research confirms this is the right level of granularity (2 layers, not N). Don't expose more granular layer toggles (e.g., separate roads/POIs/water) — no reviewed hiking app does this at the region level. |
| Update-available indicator when source tiles change | OsmAnd flags regions as outdated when newer extracts exist; a distinct 4th status. Real-world OsmAnd GitHub issues show this is *also* a common source of user confusion (false "outdated" flags, blocking navigation on stale-but-usable maps) | MEDIUM | Already scoped in the status enum. Key lesson from OsmAnd's pitfalls: **never block existing offline functionality (rendering/routing) just because `updateAvailable` is true** — it should be an optional nudge, not a functional gate. Stale-but-present tiles must keep working. |
| Bulk actions ("download all," "delete all") | Useful once a user has 5+ regions downloaded, but not essential for a v1 with a handful of regions bundled | LOW | Defer unless the initial `regions.json` ships with many small regions (e.g., per-country splits) — with few, large regions, bulk actions add little value. |
| Auto-download the region containing current GPS location on first launch | Organic Maps auto-downloads the map for the user's current location without a prompt on first open, reducing time-to-value | MEDIUM | Genuinely nice but requires location-permission-before-onboarding sequencing; worth flagging as a fast-follow, not v1 — it interacts with permission flow ordering already established in the app (geolocator + auto-geolocate setting, see project memory on `allowAutoGeolocate`). |
| Highlight downloaded region boundaries on the map | Recent Organic Maps addition — helps users visually confirm coverage on the map itself, not just in a settings list | LOW-MEDIUM | Nice differentiator once the region registry exists; a bbox rectangle overlay is cheap to add given regions are already bbox-only. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that look good on a competitor screenshot but create disproportionate complexity for a v1 predefined-region hiking app. All of the following are already correctly excluded in PROJECT.md's Out of Scope — this research validates those cuts with concrete evidence from the ecosystem.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| User-drawn custom download areas (Gaia GPS / AllTrails "Custom Areas" style) | "Let me download exactly the bbox I want" feels more flexible | Requires a map-based draw-and-confirm UI, tile-count/size estimation for an arbitrary polygon, and dedup logic against already-downloaded regions (Gaia GPS explicitly dedupes overlapping custom boxes to avoid double storage) — none of which exists yet, and it's orthogonal to a curated hiking-region manifest | Predefined bbox regions from `regions.json`, sized to hiking-relevant areas (parks, ranges, states) — matches Organic Maps' "whole state, not custom area" model, which is the right complexity match for this app |
| Country → state/province → sub-region hierarchical tree navigation | OsmAnd does this because its manifest has thousands of entries worldwide | For a bundled, curated manifest (tens of regions, hiking-relevant), a tree adds navigation depth with no payoff — users would tap through 2-3 screens to reach a region a flat searchable list would show in one | Flat list, optionally grouped by a single level (e.g., country) only if the manifest later grows past ~30-40 entries; not needed for v1's bundled regions.json |
| Full polygon region boundaries (non-bbox) | Bboxes "waste" space by including area outside the actual region of interest | Polygon clipping requires geometry processing beyond what `regions.json` + PocketBase's `generator.go` pipeline supports today, and it doesn't change what's functionally downloadable (bbox tiles still fully cover the region) | Bbox-only regions (already decided) — oversized coverage is a acceptable tradeoff for a v1, revisit only if disk usage complaints emerge |
| Cross-app-restart resumable/background downloads | Feels like table stakes coming from OS-level download managers | OsmAnd's own GitHub issues show this is a persistent source of bugs (stalled downloads that can't restart, corrupted partial state) even for a mature app; implementing it correctly requires a background task/foreground-service architecture that doesn't exist in this app yet | Session-scoped pause/resume (already decided) — if the app is killed mid-download, the user restarts the download; acceptable given regions aren't enormous and this mirors many apps' actual (if not marketed) behavior |
| Remote/dynamically-fetched region manifest with per-user region unlocking (Komoot-style paid region packs) | Komoot's "unlock this region" flow is well-known and looks like a natural fit for a "which region do I need" guard | Adds an entitlement/purchase layer entirely irrelevant to Wanderer (no paywall model), and a remote manifest means handling manifest versioning/staleness — unnecessary complexity for a v1 bundled-asset approach | Bundled `regions.json` app asset (already decided); the guard dialog borrows Komoot's *messaging* pattern ("this needs region X, download it") without any unlock/entitlement logic |
| Granular per-layer toggles (roads, POIs, contour intervals, water) beyond vector-vs-DEM | Power users may want finer control | No reviewed hiking app exposes this at the region-download level — OsmAnd's contour plugin is the único example of a second toggle, and even that's a single on/off, not granular | Two-way toggle only: base vector map (always) + optional DEM (toggle) — matches what's already scoped |

## Feature Dependencies

```
Region list UI (browse/search)
    └──requires──> Bundled regions.json manifest
    └──requires──> ObjectBox Region entity (status, size, paths)

Per-region download/pause/resume/delete
    └──requires──> Region list UI
    └──requires──> TileRepositoryManager (lifecycle)
    └──requires──> DownloadedTilePackage entity

DEM toggle per region
    └──requires──> Per-region download lifecycle
    └──requires──> Existing Mapterhorn DEM pipeline (generator.go), re-keyed to regions

Total disk usage summary
    └──requires──> DownloadedTilePackage entity (size + status tracking)

Trail download guard ("region X needed")
    └──requires──> Region list UI + download lifecycle (must be able to trigger a download from the guard dialog)
    └──requires──> Region coverage check (trail bbox ⊂ region bbox, or overlap)

Update-available indicator ──enhances──> Region list UI (does not block trail rendering/download of already-downloaded regions)

Map rendering from global region registry ──requires──> TileRepositoryManager
    └──conflicts with──> legacy trail-scoped tile cache (must be fully removed, not dual-run, per PROJECT.md's "no migration" decision)
```

### Dependency Notes

- **Region list UI requires the manifest and entity model to exist first:** the roadmap phase that builds `regions.json` + `Region`/`DownloadedTilePackage` ObjectBox entities must precede any Settings screen work — there's nothing to list otherwise.
- **DEM toggle requires per-region download lifecycle to exist first:** the toggle is a modifier on an existing per-region download action, not a standalone feature; sequencing DEM after core download/delete avoids building UI for a lifecycle that doesn't exist yet.
- **Trail guard requires the region list/download lifecycle to be functional before it can offer a "download it now" CTA** — a guard dialog that can only say "go to Settings" (rather than triggering the download inline) is a degraded but valid fallback if sequencing forces the guard earlier; recommend sequencing guard *after* core region download UI so the CTA can be a direct, in-dialog download trigger (see Guard UX below).
- **Update-available enhances but must not gate** the list/rendering — this is a correctness constraint distilled from OsmAnd's real-world bug reports, not just a UX nicety: an `updateAvailable` region must still render and route exactly like a `downloaded` region.
- **Map rendering from the global registry conflicts with (replaces) the legacy trail-scoped cache** — per PROJECT.md this is an outright deletion, not a parallel/fallback path, so this phase should be sequenced after the region registry is proven functional (region UI + at least one successful download/render round-trip), not before.

## MVP Definition

### Launch With (v1)

Minimum viable region-management surface — matches and refines what PROJECT.md's Active requirements already specify.

- [ ] Flat, searchable region list (Settings → Offline Maps/Regions) — no hierarchical tree; manifest size doesn't justify one
- [ ] Per-region row showing: name, status (not-downloaded / downloading+progress / downloaded / update-available), size (vector + DEM breakdown)
- [ ] Download / pause / resume / delete actions per region, matching the existing list-item + sub-route Settings pattern
- [ ] Per-region DEM toggle (default off or on — decide based on typical vector-vs-DEM size ratio; DEM should be clearly the "optional, adds size" choice)
- [ ] Total disk usage summary at the top or bottom of the region list
- [ ] Trail download guard: on trail download tap, check region coverage; if missing, show an informative dialog naming the region(s) needed with a direct "Download region" CTA (not a silent block, not a generic "download maps first" message)
- [ ] Partial-coverage handling: if a trail spans two regions, guard dialog lists both missing regions and lets the user download either/both before proceeding (see Guard UX below)

### Add After Validation (v1.x)

- [ ] Update-available → user-triggered re-download flow (non-blocking nudge, e.g., a badge + "update" action, not a modal)
- [ ] Bulk download/delete actions — add once the manifest grows past a size where per-region tapping becomes tedious
- [ ] Map-based region boundary highlight overlay — nice visual confirmation once the region registry is stable

### Future Consideration (v2+)

- [ ] Auto-download the region containing the user's current location on first launch — depends on permission-flow sequencing decisions beyond this milestone's scope
- [ ] Remote/updatable region manifest — explicitly deferred in PROJECT.md; revisit only if bundled-asset manifest staleness becomes a real problem
- [ ] User-drawn custom download areas — explicitly deferred; only revisit if predefined regions prove too coarse for real usage patterns
- [ ] Offline search within downloaded regions — not part of this milestone's scope (map rendering + trail download only); would be a separate, larger feature touching search infrastructure

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Flat searchable region list w/ 4-state status | HIGH | LOW | P1 |
| Download/pause/resume/delete per region | HIGH | MEDIUM | P1 |
| Size shown pre-download (vector + DEM breakdown) | HIGH | LOW | P1 |
| Total disk usage summary | MEDIUM | LOW | P1 |
| Trail guard dialog with direct download CTA | HIGH | MEDIUM | P1 |
| Partial-coverage (multi-region) guard handling | MEDIUM | MEDIUM | P1 (trails near region borders will hit this; skipping it means a confusing dead-end guard) |
| DEM toggle per region | MEDIUM | LOW | P1 |
| Update-available non-blocking indicator | LOW-MEDIUM | LOW-MEDIUM | P2 |
| Map boundary highlight overlay | LOW | LOW | P2 |
| Bulk download/delete | LOW | LOW | P3 |
| Auto-download region on first launch | MEDIUM | MEDIUM-HIGH | P3 |
| Hierarchical region tree | LOW (for this manifest size) | MEDIUM | P3 (not recommended — see Anti-Features) |

**Priority key:**
- P1: Must have for v1.6 launch
- P2: Should have, add when possible within v1.6 or immediate follow-up
- P3: Nice to have, future consideration — not v1.6

## Complexity Guidance: Region Presentation

**Recommendation: flat list, not hierarchical tree, not map-based selection.**

Evidence:
- OsmAnd's continent → country → sub-region tree exists because its manifest has thousands of entries; the app's *actual* primary interaction is still a search box at the top of the download tab, not tree navigation.
- Organic Maps downloads "whole states" — one level, no sub-hierarchy for most of the world — and this is considered a reasonable, well-liked tradeoff (efficient vector format keeps sizes manageable even at state granularity).
- A bundled `regions.json` manifest for a single hiking app is realistically tens of entries (specific trail-dense areas/parks/ranges), not a global country database. A tree adds navigation depth (2-3 taps to reach a leaf) with zero payoff at this scale.
- Map-based region selection (tap-a-region-on-the-map) is a plausible v1.x enhancement (pairs well with the "highlight downloaded regions on the map" differentiator) but is not needed for v1 discovery — a list with a search field is faster to reach a specific known region name.

**If the manifest grows** (e.g., splits into many small per-park regions later), add a single grouping level (e.g., group by country/macro-region) with a search box, not a multi-level tree.

## Status/State UX Guidance

The 4-state enum (`notDownloaded` / `downloading` / `downloaded` / `updateAvailable`) already decided in PROJECT.md maps cleanly to observed patterns, with these per-state UI affordances:

| Status | Row affordance | Primary action | Secondary |
|--------|----------------|-----------------|-----------|
| `notDownloaded` | Size shown (vector + DEM breakdown), no icon/muted download icon | "Download" button | — |
| `downloading` | Progress bar/percentage, size shown | "Pause" | Cancel (removes partial download) |
| Paused (within session — a UI sub-state of `downloading`, not necessarily a distinct enum value) | Progress bar frozen at last percentage, distinct visual (e.g., outlined vs filled) | "Resume" | Cancel |
| `downloaded` | Checkmark/filled icon, size on disk shown | "Delete" (or overflow menu) | DEM toggle if applicable |
| `updateAvailable` | `downloaded` styling + a small badge/label ("Update available") — must NOT look broken or block use | "Update" (optional, user-triggered) | "Delete" still available |

Key lesson pulled directly from OsmAnd's real-world issue tracker: **do not let `updateAvailable` degrade functionality.** Multiple OsmAnd GitHub issues describe navigation/routing breaking or refusing to proceed because the app decided a map was "outdated," even fully offline with no way to update. For Wanderer, `updateAvailable` should be purely informational — the region keeps rendering and the trail guard should treat `updateAvailable` identically to `downloaded` (i.e., it satisfies the coverage check, no guard dialog fires).

## Storage Communication Guidance

- Show size **before** download starts (all reviewed apps agree this is non-negotiable) — already possible since `regions.json` is bundled with size fields.
- Break size into exactly two components: base vector map size, optional DEM size — matches OsmAnd's contour-lines-as-separate-download precedent and matches what's already scoped. Don't go finer-grained (no reviewed app exposes per-layer sizes beyond this split at the region level).
- Show a running total disk usage figure somewhere on the region list screen (not just per-row) — this is what lets users decide *which* region to delete when storage is tight, mirroring Gaia GPS/OsmAnd's storage-management framing.
- Because sizes are pre-computed and bundled in `regions.json` (not estimated live from tile counts), there's no need for the "estimate vs actual size mismatch" handling that Gaia GPS users report as a pain point (their sizes are dynamically computed from a live tile count against a moving map selection — not applicable here since regions are fixed, pre-packaged bboxes).

## Trail-Guard UX Guidance

**Recommended shape: informative dialog with a direct download CTA, never a silent block.**

Evidence base: Komoot's pattern — when a tour starts in an undownloaded/unlocked region, the app explicitly surfaces *which* region is needed rather than a generic failure. This directly matches what PROJECT.md's guard requirement already specifies ("checks region coverage before a trail download, prompts to download the covering region if missing").

Concrete recommendation for the dialog:
1. **Trigger point:** on trail download tap (matches PROJECT.md — checked before, not after, the trail download begins).
2. **Content:** name the specific region(s) that cover the trail's bbox, plus their download size. Don't just say "map data required" — name it, the way Komoot names the specific region.
3. **Primary CTA:** "Download [Region Name]" — triggers the region download directly from the dialog (reusing the same download lifecycle as the Settings region list), not just a link that dumps the user into Settings to find it themselves. This requires the guard to depend on the core region download lifecycle already existing (see Feature Dependencies).
4. **Partial coverage (trail spans two regions):** list all missing regions with their individual sizes and a combined total; allow downloading one, both, or dismissing. Do not force downloading all covering regions before allowing the trail download to proceed at all if only one region is missing — trail download should proceed for whatever regions the user has now chosen, with the guard simply not re-blocking once minimum coverage exists. (Coverage rule to formalize during planning: does "covered" mean 100% of trail bbox inside downloaded region(s) union, or just the trail's own start point? Recommend bbox-union coverage since regions are bbox-only and the app already computes trail bboxes for the existing per-trail grid-cell download.)
5. **Dismissal:** user can cancel out of the dialog without downloading anything; the underlying trail download action should then simply not proceed (not silently degrade to a partial/broken download) — consistent with the existing PopScope-guard conventions already used elsewhere in trail creation flows (see recent commit `955c3cdf` "Adds popscope guard for trail_create_screen" for the app's existing guard-dialog idiom to match visually/behaviorally).

## Competitor Feature Analysis

| Feature | OsmAnd | Organic Maps | Gaia GPS | Komoot | Our Approach |
|---------|--------|---------------|----------|--------|--------------|
| Region granularity | Country/state hierarchy, thousands of entries | State/whole-region, one level | User-drawn custom bbox (not predefined) | Predefined regions, paywalled | Predefined, bbox-only, flat list — tens of entries in bundled manifest |
| Status states | Downloaded / downloading / **outdated** (buggy blocking behavior reported) | Downloaded / downloading | Downloaded / downloading (no update concept — static export) | Locked / unlocked / downloaded | 4-state enum (already decided); `updateAvailable` explicitly non-blocking, unlike OsmAnd |
| Size breakdown | Base map separate from contour-lines/hillshade plugin download | Single size (vector map only) | Live tile-count estimate (can mismatch actual) | Not surfaced prominently | Vector vs DEM, both pre-computed in bundled manifest (no estimate drift) |
| Trail/route-needs-region guard | Routing continues/fails inconsistently on outdated maps (issue-tracker complaints) | N/A (search-first onboarding nudge, not a hard guard) | N/A (no route-vs-download-area linkage) | Explicit: names the needed region on tour start | Explicit dialog naming region(s), direct in-dialog download CTA, non-blocking dismissal |
| Resume across app restart | Attempted, frequently buggy per GitHub issues | Not prominent | Not applicable (fast small downloads) | N/A | Explicitly out of scope for v1 — session-only pause/resume, matching the safer, lower-complexity end of the spectrum |

## Sources

- [OsmAnd — Maps & Resources docs](https://osmand.net/docs/user/personal/maps-resources/)
- [OsmAnd — Download Maps docs](https://osmand.net/docs/user/start-with/download-maps/)
- [OsmAnd — Topography (contour lines) plugin docs](https://osmand.net/docs/user/plugins/topography/)
- [OsmAnd GitHub — Improve Downloads to monitor/resume stalled downloads (#4467)](https://github.com/osmandapp/OsmAnd/issues/4467)
- [OsmAnd GitHub — Don't stop routing because of outdated maps (#21092)](https://github.com/osmandapp/OsmAnd/issues/21092)
- [OsmAnd GitHub — Offline navigation stopped while driving, asked to download maps (#19498)](https://github.com/osmandapp/OsmAnd/issues/19498)
- [Organic Maps official site](https://organicmaps.app/)
- [Organic Maps GitHub — First app open, explain offline + map download (#7210)](https://github.com/organicmaps/organicmaps/issues/7210)
- [MobileMaplets — How to Download Maps for Offline Use](https://www.mobilemaplets.com/blog/download-maps-offline)
- [Gaia GPS Help — Download Maps for Offline Use](https://help.gaiagps.com/hc/en-us/articles/360047131513-Download-Maps-for-Offline-Use)
- [Gaia GPS Help — Individual Offline Map Tile Limits](https://help.gaiagps.com/hc/en-us/articles/360000915488-Individual-Offline-Map-Tile-Limits)
- [Komoot Support — Offline gebruiken (offline usage)](https://support.komoot.com/hc/articles/360023078891-Komoot-offline-gebruiken)
- [Komoot Support — Regio niet beschikbaar om te downloaden (region not available to download)](https://support.komoot.com/hc/articles/360024969212-Regio-niet-beschikbaar-om-te-downloaden)
- [Map UI Patterns — Offline maps](https://mapuipatterns.com/offline-maps/)
- [Material Design (m1) — Offline states pattern](https://m1.material.io/patterns/offline-states.html)
- [AllTrails Help — Download custom areas for offline use](https://support.alltrails.com/hc/en-us/articles/37758009767444-Download-custom-areas-for-offline-use)
- Existing codebase context: `app/lib/services/trail_download_service.dart`, `app/lib/routes/settings_screen.dart` family, `db/services/tiles/generator.go` (read for continuity, not web-searched)

---
*Feature research for: region-based offline map management, hiking/navigation app domain*
*Researched: 2026-07-21*
