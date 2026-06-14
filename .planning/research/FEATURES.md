# Features Research

**Domain:** Offline trail navigation caching — pre-fetching Valhalla instructions at download time, silent fallback at runtime.
**Researched:** 2026-06-14
**Milestone:** v1.1 Offline Navigation

---

## Table Stakes

Features the user expects any offline-capable hiking app to provide. Missing one = the feature feels broken or untrustworthy.

| Feature | Why Expected | Complexity | Dependencies on Existing Code |
|---------|--------------|------------|-------------------------------|
| Cache Valhalla instructions during trail download | AllTrails, Komoot, Gaia GPS all bundle navigation data with the trail download. Users expect "download = ready to navigate offline." A downloaded trail that still requires connectivity to navigate is a broken promise. | Low | `TrailDownloadService.downloadTrail()` already chains multiple async fetches (photos → map tiles). Add a Valhalla POST step in the same chain before `box.put(entity)`. |
| Silent offline fallback in `launchNavigation` | Komoot's own support docs explain that offline navigation silently uses locally cached route data when there is no connection. Users don't want a modal — they want navigation to just start. | Low | `launchNavigation` in `navigation_launch_util.dart` already catches all errors and shows a toast. The change is: check `TrailEntity` for cached instructions before the HTTP POST. |
| Persist `NavigateResponse` (maneuvers + shape) to `TrailEntity` in ObjectBox | The `NavigateResponse` (maneuvers, shape) must survive app restarts. The download contract already persists GPX and map tiles to ObjectBox; navigation cache must follow the same contract. | Low–Medium | `TrailEntity` needs new fields: `navigateManeuversJson` (String?) and `navigateShapeJson` (String?) or a single `navigateResponseJson` blob. `TrailEntity.fromModel` / `toModel` must be updated. |
| Download success notification still reads "Saved for offline use" | `DownloadNotificationService.showSuccess()` already says this. Adding navigation caching must not change the user-visible success message — the Valhalla step is implementation detail. | Trivial | No change to `DownloadNotificationService`. |
| Graceful failure if Valhalla is unreachable at download time | If the navigation cache step fails (server down, GPX has < 2 points, etc.), the trail download must still complete successfully. Navigation simply won't be available offline for that trail — the user navigates online instead. Do NOT abort the entire download. | Low | Wrap the Valhalla POST in a try/catch inside `downloadTrail`. On failure, leave `navigateResponseJson` null and continue. |

---

## Differentiators

Features that are not strictly required but meaningfully improve the experience. Implement after table stakes are solid.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Offline badge / indicator on trail detail | AllTrails uses a filled circle icon; Gaia GPS fades incomplete downloads. A small "navigate offline ready" chip or badge on the trail detail screen tells the user they can hike without connectivity. | Low | `trail.isOffline` already set by `TrailEntity.toModel()`. Add check: `isOffline && navigateResponseJson != null` → show "offline navigation ready" badge. Requires no new provider. |
| "Offline" mode indicator in NavigationScreen | Google's offline UX guidelines recommend pairing an "offline pin" icon with the word "offline" when operating on cached data. A small banner or icon overlay on NavigationScreen when running from cache reassures the user and helps diagnose problems. | Low | Pass a `bool isOfflineCache` flag alongside `NavigateResponse` as `extra` to `context.push('/trail/:id/navigate')`. NavigationScreen reads it and shows a subtle indicator. |
| Re-cache navigation on next successful online launch | When a user navigates online (fresh Valhalla instructions) for a trail that is also downloaded, opportunistically update the cached `NavigateResponse` in ObjectBox. This keeps the cache fresh without a separate "refresh" gesture. | Low–Medium | In `launchNavigation`, after a successful HTTP POST and parse, if `trail.isOffline` is true, write the new `NavigateResponse` to ObjectBox in the background (fire-and-forget, no await). |

---

## Anti-Features

Features to explicitly not build in this milestone.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| "Stale cache" warning dialog or banner | Google Maps, Komoot, and OsmAnd all handle stale cached routes silently. PROJECT.md explicitly states: "Silently use cached version if trail updated since caching." A modal that asks "Route data is X days old — navigate anyway?" adds friction on a trailhead where the user just wants to go. The cache is for turn-by-turn instructions on a fixed trail, not live traffic data. Staleness risk is low. | Use the cache silently. If the trail was updated on the server, the user will get fresh instructions next time they're online. |
| Timestamp / "last cached" display in UI | web.dev offline guidelines recommend timestamps for high-volatility data (stocks, weather). Trail maneuvers are low-volatility — the trail geometry doesn't change week to week. Showing a "Cached 14 days ago" label adds noise without value. | Omit entirely. |
| User-initiated "refresh navigation cache" button | Adds a settings surface that most users will never touch. The re-cache-on-online-launch differentiator (above) achieves the same result automatically. | Automatic re-cache on next online navigation session. |
| Blocking progress step for Valhalla fetch during download | Gaia GPS shows download progress only for map tile cells (the slow step). The Valhalla POST for a 500-point shape returns in ~200–500 ms — too fast to warrant a named progress step. Adding "Step 3/3: Caching navigation…" to the notification would misrepresent the timing. | Fire the Valhalla POST silently within `downloadTrail`. Progress callback continues to reflect tile download progress only. |
| Offline routing (recalculation) | Computing a new route from the user's current position when they go off-trail requires embedding a Valhalla tile dataset on device (hundreds of MB). This is a v3+ concern. PROJECT.md Out of Scope: re-routing if user goes off-trail. | Stay on cached instructions; show the last known maneuver if the user drifts. |
| Partial / background download of navigation cache separately from trail | Komoot's approach (separate map vs route downloads) is appropriate for large regional datasets. Wanderer's per-trail model means the Valhalla call is a single POST that completes in under a second — no reason to split into a separate download step. | Bundle with the existing `downloadTrail` call. |

---

## UX Notes

### 1. Caching at download time — does the user see progress?

**No new progress step needed.** The existing download UX is:
1. Toast: "Downloading [trail name]…" (in-app, immediate)
2. System notification: "Preparing download…" → "Downloading map tiles (N / M)" (ongoing, per-tile)
3. System notification: "[trail name] — Saved for offline use" (completion)

The Valhalla POST happens in under 500 ms and does not need its own progress step. Insert it after photo downloads and before (or after) map tile downloads. If it fails, swallow the error and continue — the tile download result determines the success notification.

**Implementation hook:** `TrailDownloadService.downloadTrail()` calls `_downloadMapTiles()` last, then calls `box.put(entity)`. The navigation cache POST should be added just before `box.put(entity)` so the entity is written with the cached response in a single transaction.

### 2. Navigating offline — UI indicators needed?

**Minimal indicator is table stakes; rich indicator is a differentiator.**

The minimal requirement: navigation starts without error. The user already knows they're offline because they see no signal bars. A subtle visual signal — an icon in the AppBar or a chip on the maneuver card — reassures them the app is working from cache, not silently failing.

Pattern from AllTrails: filled black circle = downloaded. Pattern from Gaia GPS: progress bar appears in the downloads menu. For NavigationScreen, the right pattern is a non-blocking icon in the top-right AppBar area (e.g., `Icons.wifi_off` or `FontAwesomeIcons.towerBroadcast` with a slash). Do not use a persistent banner — it competes with the maneuver instruction text at the top.

### 3. Stale cache — how do good apps handle this?

AllTrails explicitly warns that iOS may delete cached data to free space and recommends checking before leaving. Komoot says "content must be downloaded in advance." Neither app surfaces a "this is X days old" warning in-session.

For Wanderer v1.1, the right pattern is:
- **Silent use**: load cached `NavigateResponse` from ObjectBox, push to NavigationScreen, show offline icon, done.
- **Opportunistic refresh**: when the user navigates online for a trail that is also downloaded, update the ObjectBox record in the background (fire-and-forget).
- **No user-visible staleness UI**: trail turn-by-turn instructions are low-volatility data. The trail geometry changes only if the author re-uploads the GPX, which is rare.

### 4. Feature dependencies on existing download workflow

```
Trail download (existing)
  └─ _downloadPhotos()           ← unchanged
  └─ _downloadMapTiles()         ← unchanged (drives progress callback)
  └─ [NEW] _fetchNavigationCache()  ← POST /valhalla/navigate, swallow errors
  └─ box.put(entity)             ← entity now includes navigateResponseJson field

launchNavigation (existing)
  └─ [NEW] check TrailEntity for navigateResponseJson if trail.isOffline
  └─ if found AND offline → decode + push NavigationScreen (isOfflineCache: true)
  └─ if online (regardless) → POST /valhalla/navigate (existing path)
  └─ if online + trail.isOffline → opportunistic background re-cache
```

The key invariant: **online sessions always fetch fresh instructions** (no behavior change). Cache is only consumed when there is no network.

---

## Sources

- [Komoot: Download routes and maps for offline use](https://support.komoot.com/hc/en-us/articles/10356476920986-Download-routes-and-maps-for-offline-use) — confirmed download-then-navigate workflow; no separate "cache navigation" step visible to user
- [Gaia GPS: Check if a Map is Downloaded — offline indicator patterns](https://help.gaiagps.com/hc/en-us/articles/115003523707-Check-if-a-Map-is-Downloaded-and-Available-Offline-on-the-iOS-app) — progress bar in downloads menu, faded icon for in-progress download (HIGH confidence)
- [Google Design: UX Design for Offline](https://design.google/library/offline-design) — offline pin icon + word "offline", cloud-off icon for no connectivity, file size transparency (HIGH confidence)
- [web.dev: Offline UX Design Guidelines](https://web.dev/articles/offline-ux-design-guidelines) — timestamp for high-volatility data, skeleton loaders, stale-while-revalidate pattern (HIGH confidence)
- [AllTrails: Offline maps navigation help](https://support.alltrails.com/hc/en-us/articles/37213942810132) — filled circle = downloaded, hollow circle = not downloaded (MEDIUM confidence, HTTP 403 on direct fetch; inferred from search result snippets)
- [Valhalla GitHub Discussion: Mobile offline routing](https://github.com/valhalla/valhalla/issues/1530) — tile-based offline routing for mobile; confirms caching instructions is the right approach for v1 (no on-device Valhalla engine needed) (HIGH confidence)
- Existing codebase analysis: `TrailDownloadService`, `DownloadNotificationService`, `TrailEntity`, `navigation_launch_util.dart`, `trail_dropdown.dart` — confirmed download chain structure and ObjectBox entity schema
