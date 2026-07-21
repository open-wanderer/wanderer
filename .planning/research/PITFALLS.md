# Pitfalls Research

**Domain:** App-wide, region-based offline map tile downloading/storage in a Flutter app (pre-1.0 `maplibre` native GL binding + ObjectBox), replacing an existing trail-scoped single-file-per-cell system.
**Researched:** 2026-07-21
**Confidence:** MEDIUM (mix of HIGH-confidence platform/OS behavior, MEDIUM-confidence library behavior verified against docs/discussions, and LOW-confidence maplibre-native-specific claims that need an in-repo spike — flagged individually below)

## Critical Pitfalls

### Pitfall 1: Treating region downloads like trail downloads — full-file, no resumability

**What goes wrong:**
`trail_download_service.dart`'s `_downloadTracked` wraps a single `Dio.download()` call per cell file with no `Range` header / partial-file resume. That's fine at trail scale (each `.pmtiles` cell is small, per-cell, cheap to retry). At region scale, `regions.json` ships one vector `.pmtiles` (and optionally one DEM `.pmtiles`) per region — files that can be tens to hundreds of MB. If the connection drops at 95%, the existing pattern re-downloads the whole file from byte 0. PMTiles is also immutable-by-design at the format level: there's no tile-level patch/append story, so a half-written file is not a partial map, it's a corrupt one and must never be treated as "good enough."

**Why it happens:**
The trail-scoped code's retry-the-whole-file behavior was invisible because cell files are small. Porting `_downloadTracked` verbatim to region-sized files silently inherits a UX (and bandwidth) cliff: a 300MB region download that fails at 99% costs the user another 300MB, every time, on a route that may be flaky (rural/mobile users are the exact audience for offline maps).

**How to avoid:**
- Download to a `.part`/`.tmp` sibling path, only rename to the final `.pmtiles` path after a completed download (this also naturally fixes "half-downloaded file treated as cached" bugs — the existing `File(localPath).exists()` cache check would otherwise treat a truncated file as complete).
- Add `Range: bytes=<offset>-` support in the Dio download call (Dio supports resuming via `options: Options(headers: {'range': 'bytes=$downloaded-'})` plus append-mode file writes) so a dropped connection mid-download resumes rather than restarts, within the single app session the milestone already scopes for.
- Validate the finished file before rename — PMTiles has a fixed-size header with a magic number and declared length; a cheap header check (magic bytes + file size sanity) catches truncation before the file is registered as `downloaded`.

**Warning signs:**
Users on flaky connections repeatedly failing to complete a large region; support reports of "download restarts at 0% every time"; a `downloaded` region that fails to render (corrupt/truncated pmtiles opened by the native reader).

**Phase to address:**
The phase that builds `TileRepositoryManager`'s download path (before Settings UI ships) — resumability is a download-primitive concern, not a UI concern, and is much harder to retrofit once `DownloadedTilePackage` status semantics exist.

---

### Pitfall 2: No pre-flight disk-space check, and platform-inconsistent "free space" reads

**What goes wrong:**
Nothing in the current trail-scoped path checks available storage before writing. At trail scale a missing check is low-risk (a few MB per trail). At region scale — potentially several regions of 100s of MB to low GB each, per the `regions.json` model — a user can start a download that fills the device mid-write. The failure mode is worse than "download fails cleanly": `Directory.create`/`File.writeAsBytes` on a full disk can throw partway through a multi-file batch (vector done, DEM half-written, or vice versa), leaving an inconsistent `DownloadedTilePackage` on disk with no signal to the UI other than a generic Dio/IO exception.

Even when a check is added, iOS and Android disagree about what "free space" means: iOS's reported "available" capacity for opportunistic storage (`volumeAvailableCapacityForImportantUsage`) already accounts for purgeable/cached data the OS may reclaim, while Android's `StatFs`/`getFreeSpace()` reports raw filesystem free bytes without accounting for the reserve (often several hundred MB to ~1GB) the OS keeps for system stability. A naive "does `freeBytes > downloadSize`" check using the wrong API on either platform will pass when it shouldn't (Android) or fail defensively more than necessary (iOS).

**Why it happens:**
Disk space checks are easy to skip in a codebase that has never needed them (trail cells are small enough that "just try the download" was an acceptable strategy). Region-scale files change the risk profile.

**How to avoid:**
- Check available space before starting a region download (vector + DEM combined), with a safety margin (e.g. require 1.5-2x the manifest's declared size to account for the temp `.part` file coexisting with prior data during resume, plus OS reserve).
- Use platform-appropriate space APIs — for iOS prefer `volumeAvailableCapacityForImportantUsage` (via a plugin or platform channel) over generic free-space calls; on Android budget in a fixed reserve (don't treat 100% of `getFreeSpace()` as usable).
- On low-space failure mid-download, delete the partial `.part` file and surface a specific "not enough storage" error distinct from a generic network failure, so the Settings → Offline Maps UI can show an actionable message instead of a silent retry loop.
- Re-check space before each file in a multi-file region download (vector, then DEM) — space available at download start may not hold by the time the DEM starts if other downloads are running concurrently.

**Warning signs:**
`OSError`/`FileSystemException` with `ENOSPC`/"No space left on device" surfacing as an unhandled Dio/IO exception; regions stuck in a `downloading` state forever after a full-disk failure; users reporting successful iOS downloads that Android rejects (or vice versa) at similar free-space levels.

**Phase to address:**
Same phase as `TileRepositoryManager`'s download path — this is a companion concern to Pitfall 1 (both are about "the download primitive must be defensive," not about the region model itself).

---

### Pitfall 3: Multi-file / multi-region downloads racing against iOS background suspension and Android Doze

**What goes wrong:**
The milestone explicitly scopes pause/resume to "within a single app session" (no cross-restart background download persistence) — so a full `background_downloader`-style native background-session rewrite is out of scope. But "single session" does not mean "app stays foregrounded": a user can background the app (switch to another app, lock the screen) mid-download without force-quitting it, and both OSes will still interfere:
- **iOS**: a backgrounded app gets a short grace period (historically ~30s, platform-controlled and not guaranteed) before the OS suspends its process; any in-flight `Dio` request without an explicit background task assertion (`UIApplication.beginBackgroundTask` or a proper `URLSession` background configuration) is frozen mid-stream, not cleanly cancelled — leaving the connection in limbo until it either completes on the next foreground or times out.
- **Android**: for downloads that run long enough (Doze/App Standby, or simply losing the CPU/network to background restrictions), a plain in-process HTTP download can be throttled or killed outright unless run from a foreground service with an active notification.

Downloading multiple region files back-to-back (vector + DEM, potentially concurrently across sibling downloads like the trail-scoped code's `Future.wait` pattern) multiplies the exposure: a longer total download window makes a backgrounding event during the download much more likely than it was for a single small trail cell.

**Why it happens:**
The trail-scoped code was fast enough (small files) that users rarely backgrounded the app mid-download, and the milestone's explicit scoping decision to skip cross-restart background sessions creates an easy trap: "no cross-restart persistence" gets read as "no background handling needed at all," when in-session backgrounding (not app restart) is still a near-certain occurrence for a multi-hundred-MB download.

**How to avoid:**
- Treat "app backgrounded mid-download" as a first-class, expected failure mode even without implementing true OS background downloading: detect app lifecycle transitions (`AppLifecycleState.paused`) and pause/cancel the active `CancelToken` deliberately rather than letting the OS freeze the socket unpredictably, then offer a clear "resume" affordance on foreground return (reusing the Pitfall 1 resume mechanism).
- On Android, if region downloads are expected to run for more than a minute or two, at minimum surface this as a documented constraint ("keep the app open while downloading") in the Settings UI copy — don't silently let long downloads run backgrounded and fail unexplained.
- Do not conflate "cancelled by network/OS" with "cancelled by user" in the `DownloadedTilePackage` status — the recovery path (silent auto-resume on next foreground vs. an error state requiring user action) differs.

**Warning signs:**
Downloads that "hang" indefinitely with no progress and no error when the app was backgrounded during them; regions stuck in `downloading` status after the user manually force-quit and reopened the app; bug reports correlating with screen-lock timing.

**Phase to address:**
The phase implementing the download engine and its status-tracking contract (same phase as Pitfalls 1-2) should define lifecycle-aware cancel/pause semantics; the Settings UI phase should surface the resulting states (not silently retry).

---

### Pitfall 4: Layer-clone explosion when the existing multi-cell offline-style pattern is reused at region scale

**What goes wrong:**
`offline_style_rewriter.dart`'s documented strategy for combining multiple `.pmtiles` archives into one style is to emit **N duplicated sources + N duplicated layer sets**, one per archive (`<source>-cell-<i>` / `<layerId>__cell<i>`), because the installed `pmtiles` Dart package and the native `pmtiles://` protocol have no merge/union capability. The doc comment is explicit that this is "bounded by `cellPaths.length`" and sized for a realistic trail (1-4 cells).

Region-based downloads change that bound. Even though each region ships as a single pre-merged `.pmtiles` file (unlike trail-scoped per-cell extraction), rendering **multiple downloaded regions simultaneously** (the whole point of "region-based, not trail-scoped" — a user should be able to pan/record across two adjacent downloaded regions without a gap) still requires the same N-source/N-layer-clone technique, just with N = number of *simultaneously relevant downloaded regions* instead of N = trail cell count. A user who downloads 5-10 regions over time (a realistic app-wide usage pattern, vs. a realistic single-trail cell count) could push layer counts into the hundreds once the style needs to reference every downloaded region that could be visible. Community reports (maplibre-gl-js discussion on 8000+ dynamically added PMTiles layers) confirm MapLibre's renderer degrades — slower style parse, slower startup, increased memory — as source/layer count grows, independent of PMTiles' own per-file efficiency. **Confidence: MEDIUM** — the layer-count-degrades-performance claim is corroborated by public MapLibre GL JS discussion, not maplibre-native 0.3.5 specifically; the exact threshold on this native binding is unverified and should be spiked.

**Why it happens:**
The multi-cell technique was designed and validated for "a handful of cells covering one trail," a bound that quietly stops holding once "how many archives could be active in the style at once" is driven by "how many regions has this user ever downloaded" instead of "how big is one trail."

**How to avoid:**
- Do not extend the trail-scoped N-clone technique naively to "clone into the style every downloaded region, always." Only include sources/layers for regions that intersect the *current viewport* (or a small buffer around it), and add/remove them as the user pans — mirrors the "lazy-load sources only when bounds intersect viewport" pattern seen in the broader MapLibre ecosystem for large multi-source setups.
- Budget and cap the maximum number of simultaneously active region sources in the style; if a user has downloaded more regions than that, only the ones near the viewport get materialized as style sources.
- Spike this early (before committing to the N-clone pattern at region scale): build a style with 10-20 duplicated source/layer sets against the pinned `maplibre` 0.3.5 binding and measure style-load time and memory on a mid-tier Android device, the same way the v1.4 15-01 spike validated `file://` sprite behavior before committing to it.

**Warning signs:**
Map screen startup time growing measurably as more regions are downloaded; frame drops or stutters when panning near a region boundary with several regions active; native crash reports correlating with users who have many downloaded regions.

**Phase to address:**
This must be resolved in the phase that designs `TileRepositoryManager`'s viewport-based tile-reading pipeline (explicitly called out in `PROJECT.md` as "global viewport-based tile-reading pipeline") — it is the architectural crux of "app-wide, region-based" and should not be deferred to a later polish phase. Recommend an explicit research/spike phase or spike task before the pipeline is finalized.

---

### Pitfall 5: Every region add/remove triggers a full style reload, colliding with known `maplibre` 0.3.5 lifecycle quirks

**What goes wrong:**
Because the offline style is a full JSON document rewritten by `rewriteStyleForOffline` and applied wholesale (not incremental `addSource`/`addLayer` calls — unconfirmed at the API level for this package, flagged below), each time a region is downloaded, deleted, or the viewport crosses into newly-relevant regions (per Pitfall 4's mitigation), the whole style may need to be rebuilt and reapplied. In the trail-scoped system this happened once, at trail-download time. In the region system, per `PROJECT.md`'s described UX (download/pause/resume/delete from a Settings page, plus live map re-rendering as the user pans across region boundaries), style rewrites could become a much more frequent, possibly per-pan-event operation.

This directly collides with two already-known `maplibre` 0.3.5 quirks called out in project context: `onStyleLoaded` can fire before `onMapCreated`, requiring a buffered-replay pattern, and `file://` sprite resolution is unreliable on-device. Both were worked around for a rare, one-time style application (trail open). If style reloads become frequent (region boundary crossings, download completion while the map screen is open), the buffered-replay workaround and the sprite self-registration workaround both need to be correct on *every* reload, not just the first — a much larger surface for the same known bugs to resurface, plus new failure modes (camera jump/flicker on reload, in-flight tile requests dropped mid-fetch when the style swaps).
**Confidence: LOW/MEDIUM on the "incremental add/remove API" question** — pub.dev/docs search did not confirm whether `maplibre` 0.3.5 exposes `addSource`/`removeSource` at runtime versus requiring a full style replace; this needs direct verification against the installed package's API before the phase that implements it.

**Why it happens:**
The trail-scoped design's "rewrite style once, on download" pattern was correct for its use case and got baked into `offline_style_rewriter.dart`'s architecture. Region-based rendering has a fundamentally different cadence (style composition changes as the user moves, not just once at a content-creation event), and that mismatch isn't visible until the region UI actually drives frequent reloads.

**How to avoid:**
- Verify directly against the pinned `maplibre` 0.3.5 API whether sources/layers can be added or removed incrementally (`MapLibreMap`/`MapController`-level methods) without a full style-document reapplication. If incremental APIs exist, prefer them for viewport-driven region activation over repeated full-style rewrites.
- If only whole-style replacement is available, debounce/coalesce region-activation changes (don't reload on every pan frame — reload only when the set of "regions needed for the current viewport" actually changes, and debounce rapid pans).
- Re-run the buffered-replay and sprite self-registration workarounds through an explicit test pass once reload frequency increases, rather than assuming the one-time-reload validation still holds.

**Warning signs:**
Visible flicker/flash or camera reset when panning across a region boundary; sprite icons (route shields, trail arrows) intermittently failing to render after the *second or later* style reload in a session (vs. only the first, which is the already-known/worked-around case); dropped tile requests visible as blank tiles briefly after a reload.

**Phase to address:**
Same phase as Pitfall 4 (viewport-based pipeline design) — pair the spike there with a direct API check of `maplibre` 0.3.5's incremental source/layer methods before committing to a reload strategy.

---

### Pitfall 6: ObjectBox enum-backed download-status field breaks silently on enum reordering/insertion

**What goes wrong:**
ObjectBox Dart has no native enum column type — the standard, documented pattern is `@Transient()` on the enum field with an `int` getter/setter backed by `.index` (or an explicit int mapping) for the persisted value. `PROJECT.md` specifies a `Region`/`DownloadedTilePackage` status enum (`notDownloaded/downloading/downloaded/updateAvailable`). If this is implemented as `.index`-backed and a later phase inserts a new status value (e.g. adding `paused` or `failed` between existing values, which is a very likely addition once Pitfall 1-3's failure modes are handled), every already-persisted `int` on-device silently reinterprets as the *wrong* enum value — no exception, no migration warning, just wrong status displayed to real users on real devices (e.g. a `downloaded` region silently becomes `updateAvailable` because a new value was inserted before it in the enum declaration).

**Why it happens:**
`.index`-backed enum persistence looks correct and passes all tests during initial development (enum never changes during a single dev cycle), and the bug only manifests across an app version upgrade on a device that already has persisted data — exactly the scenario unit tests don't cover, and the "no migration path" project constraint makes it easy to dismiss as "not our problem" when it actually is a data-integrity problem, not a migration problem.

**How to avoid:**
- Never back the status enum with `.index`. Assign explicit, stable integer constants per status (e.g. `notDownloaded = 0, downloading = 1, downloaded = 2, updateAvailable = 3`, with new values always appended with new numbers, never inserted) and map explicitly in the getter/setter, not via `Enum.values[index]`/`.index`.
- Document the enum's stable-int contract directly next to the enum declaration (a comment stating "append only, never reorder, never reuse a retired value") so future phases (which will likely add `paused`/`failed`/`corrupted` per Pitfalls 1-3) don't reintroduce the bug.
- Add a defensive fallback in the int→enum decode path (unrecognized int → a safe default like `notDownloaded`, not a crash and not silent misinterpretation) so an unexpected value degrades safely.

**Warning signs:**
Region statuses that appear wrong only after an app update on a device with pre-existing downloads; QA passing on fresh installs but failing on upgraded installs (a gap easy to miss if the team primarily tests fresh installs, which is likely given this is pre-production).

**Phase to address:**
The phase that defines the ObjectBox `Region`/`DownloadedTilePackage` entities, before any status enum ships to a real device — this is much cheaper to get right on the first schema than to fix after values are already persisted on dev/QA devices.

---

### Pitfall 7: No cascade delete — deleting a Region can orphan `DownloadedTilePackage` rows and dangling `Trail` references

**What goes wrong:**
ObjectBox does not cascade-delete related entities: removing a `ToOne`/`ToMany` relation, or even deleting the entity a relation points to, does not automatically delete or null out the other side. Two concrete risks given this milestone's model:
1. Deleting a `Region` (Settings → Offline Maps → delete) can leave its `DownloadedTilePackage` row orphaned in the database, or leave a `Trail`'s reference to "the region that covers it" pointing at a deleted object — reading that relation later either returns a stale/incorrect object or throws, depending on how the relation is modeled (`ToOne` vs. a stored id field).
2. The reverse direction is just as real: `PROJECT.md` specifies a "trail download guard: checks region coverage before a trail download" — if that guard caches or stores which region covers a trail, deleting the region out from under an already-downloaded trail needs an explicit decision (does the trail's offline data become unusable? does the guard silently re-prompt for redownload?) that ObjectBox will not make for you.

**Why it happens:**
Relational-database habits (assuming `ON DELETE CASCADE` exists) don't transfer to ObjectBox, and this is easy to miss in initial development because deletion is usually tested in isolation (delete a region with nothing else referencing it) rather than under the realistic condition of "a trail was downloaded because this region covered it, and now the region is being deleted while the trail is still cached."

**How to avoid:**
- Model the delete path explicitly: when a `Region` is deleted, write the cleanup transaction by hand — delete or update every `DownloadedTilePackage` for that region, and decide (and implement) what happens to trails whose coverage guard pointed at it (most defensible default: mark affected trails' offline data as stale/needs-redownload rather than leaving a dangling reference).
- Prefer storing the region relationship as an explicit foreign-key-style id field plus an application-level integrity check, rather than relying on `ToOne` semantics to "just work" on delete — makes the missing-cascade behavior visible in code review rather than implicit.
- Add a startup/maintenance sweep (cheap, since it only runs against a small `Region`/`DownloadedTilePackage` table) that detects and cleans any orphaned rows left by a delete that didn't complete its full cleanup transaction (e.g. app killed mid-delete).

**Warning signs:**
Crashes or null-object errors when reading a trail's covering region after that region was deleted; disk usage reported in the Settings page not matching what `du`/actual file listing shows (a symptom of orphaned `DownloadedTilePackage` rows whose files were deleted but whose DB rows weren't, or vice versa).

**Phase to address:**
The phase that implements region deletion in `TileRepositoryManager` — write the full cleanup transaction (DB rows + files, see Pitfall 9) as a single unit from the start, not as a follow-up fix.

---

### Pitfall 8: Race between download-progress writes and UI reads on the same ObjectBox entity

**What goes wrong:**
The trail-scoped download reports progress via in-memory callbacks (`onProgress`, `onGeneratingChanged`) directly to the calling widget — no ObjectBox write happens mid-download. A region-based system, per `PROJECT.md`'s Settings page requirements (list regions with live download/pause/resume state, survive navigating away and back), more plausibly needs download progress and status persisted to ObjectBox so the Settings page can reflect state after a navigation round-trip or app relaunch. That introduces two race conditions the trail-scoped code never had to handle:
1. A background download loop writing frequent progress updates to a `DownloadedTilePackage` row while the Settings UI has a live `Store` query/watch subscribed to the same entity — if writes aren't batched/transactional, the UI can observe a torn/half-updated state (e.g. `bytesDownloaded` updated but `status` not yet flipped to `downloading`, or vice versa).
2. ObjectBox Dart objects and the `Store` are not safe to share arbitrarily across isolates without explicit `Store.attach()` — if the download loop is moved to a separate isolate for the multi-file, potentially-long-running region downloads (a very likely refactor once file sizes grow per Pitfall 1-3), naive object-passing across the isolate boundary rather than re-attaching a `Store` per isolate will corrupt or silently drop writes.

**Why it happens:**
The trail-scoped design's callback-only progress model sidestepped this entirely by never persisting progress mid-download. Once progress needs to survive a UI navigation (a real region-management UX requirement, unlike the trail-scoped modal-dialog download flow), the natural next step — "just write progress to the entity on every tick" — introduces write frequency and threading concerns that didn't previously exist in this codebase.

**How to avoid:**
- Batch progress writes (e.g. update the persisted entity at a coarser interval — every N% or every second — rather than on every byte-progress callback) and always update `status` + `bytesDownloaded`/`totalBytes` in the same `runInTransaction` write, mirroring the existing `_store.runInTransaction(TxMode.write, () { box.put(entity); })` pattern already used for trail entities.
- If download work moves to a separate isolate, explicitly `Store.attach()` a new `Store` instance inside that isolate rather than passing ObjectBox objects/queries across the isolate boundary directly.
- Keep the fast-moving, per-tick progress signal (the existing points-based fractional progress pattern) as an in-memory stream/callback for the actively-open Settings screen, and use the persisted ObjectBox row only for coarse-grained state that must survive navigation/relaunch — don't force every progress tick through the database.

**Warning signs:**
Settings page showing a progress bar that visibly stutters or shows stale percentages; `status` and `bytesDownloaded` observed out of sync (e.g. `status: downloaded` but `bytesDownloaded < totalBytes`); crashes or silently-lost writes after moving download work to a background isolate.

**Phase to address:**
The phase implementing the Settings → Offline Maps/Regions page's live state binding, in coordination with whichever phase decides whether downloads run on the main isolate (simpler, matches existing trail-scoped pattern) or a background isolate (needed only if region file sizes make main-isolate I/O visibly janky — validate before committing to the isolate refactor's added complexity).

---

### Pitfall 9: Ripping out trail-scoped tile code leaves orphaned files on existing dev/test devices with no cleanup path

**What goes wrong:**
`PROJECT.md` explicitly scopes this as "no migration — app is pre-production," and correctly so for schema/code. But that scoping only covers *code*, not *files already on disk*. Every dev/test device that has downloaded a trail under the current system has real files at `<app-docs>/library/<trailId>/tiles/*.pmtiles` (and `*_dem.pmtiles`) sitting outside ObjectBox entirely — they're just files, referenced only by the `TrailEntity.pmTiles`/`demPmTiles` string-list fields. If the phase that deletes `trail_download_service.dart`, `TrailEntity.pmTiles`/`demPmTiles`, and the trail-scoped download UI simply deletes the *code*, those directories become permanently orphaned: no code path references them anymore, they don't show up in the new region-based disk-usage total (which only knows about `DownloadedTilePackage`), and they silently continue consuming device storage forever. On a QA/dev device that has downloaded many trails over the v1.0-v1.5 milestones, this could be a meaningful chunk of storage that the new Settings → Offline Maps "total disk usage" figure will under-report relative to what `du` on the actual filesystem shows — undermining the exact feature (accurate disk usage reporting) this milestone is building.

**Why it happens:**
"No migration path" is true and correct for the *data model* (there's no `TrailEntity.pmTiles` value that needs converting into a `Region`), but it's easy to conflate "no data migration needed" with "no cleanup needed" — the old files aren't being migrated into anything, they just need to be deleted, which is a different (and still necessary) operation.

**How to avoid:**
- Add an explicit one-time cleanup step to the removal phase: before (or as part of) deleting the trail-scoped download code, walk `<app-docs>/library/*/tiles/` (or wherever the old per-trail tile directories live) and delete every such directory, independent of whether any `TrailEntity` still references it — this cleanup should run once on app startup after the update ships, gated so it doesn't re-run every launch (e.g. a `SharedPreferences`/settings flag, or simply checking whether the legacy directory still exists).
- Do this cleanup *before* wiring the new Settings page's disk-usage total, so the first number a user/QA ever sees is accurate, not artificially low.
- Explicitly verify the `library/<trailId>/photos/` and `library/<trailId>/waypoints/` subdirectories (trail photos, not tiles) are *not* accidentally swept by an overly broad cleanup — only the `tiles/` subtree (and the top-level `demPmTiles` paths) is legacy region-model territory; trail photo caching is unrelated and stays.

**Warning signs:**
Settings → Offline Maps disk usage total not matching actual device storage used by the app (visible via OS-level app storage inspector); QA devices that predate v1.6 showing unexpectedly high "other" app storage after upgrading.

**Phase to address:**
The same phase that deletes the legacy trail-scoped tile download/cache code (per `PROJECT.md`'s Active requirement "legacy trail-scoped tile download/cache code removed") — the cleanup sweep and the code deletion should ship together, not as a follow-up.

---

### Pitfall 10: iCloud/device backup bloat from un-excluded large region files

**What goes wrong:**
Files written under the app's documents directory (via `path_provider`'s `getApplicationDocumentsDirectory()`, the same directory the trail-scoped code already uses) are backed up to iCloud by default on iOS unless explicitly marked with `NSURLIsExcludedFromBackupKey`/`isExcludedFromBackup`. At trail scale this was a minor issue (a few small `.pmtiles` cells per trail). At region scale — potentially multiple regions of 100s of MB to low-GB each — un-excluded offline map data can make a user's iCloud backup balloon by gigabytes of easily-re-downloadable data, which Apple's own guidance explicitly discourages backing up, and which can cause slow/failed backups and unhappy users hitting their iCloud storage cap because of this app specifically.

**Why it happens:**
`path_provider`'s Flutter API has no built-in "exclude from backup" flag — it requires a native platform-channel call (Swift/Kotlin) that's easy to skip entirely if nobody on the team has hit it before, and the trail-scoped system's small file sizes never made the problem visible.

**How to avoid:**
- Add a native iOS call (via a small platform channel, or an existing plugin if one is already a dependency) to set `isExcludedFromBackup = true` on the region tile storage directory once, at directory-creation time — not per-file, so new files written into an already-excluded directory inherit the exclusion automatically on APFS.
- Verify Android doesn't need the equivalent treatment for this app's backup configuration (Android's auto-backup is opt-in via `android:allowBackup`/`backup_rules.xml`; check whether the app's manifest already excludes large data directories, or extend the existing exclusion rules to cover the new region storage path).

**Warning signs:**
User reports of unexpectedly large iCloud backups after using the app; App Store review feedback flagging backup size (Apple has rejected/flagged apps for this in the past for large re-downloadable caches).

**Phase to address:**
The phase implementing region file storage in `TileRepositoryManager` (same phase as directory-structure decisions) — cheapest to add when the storage directory is first created, not retrofitted after files already exist unexcluded on real devices.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Reuse `_downloadTracked`/`downloadTrail` structure verbatim for regions, full-file-only, no resume | Fast to ship, proven code path | Full re-download on any interruption for files 10-100x larger than trail cells (Pitfall 1) | Never past a first internal spike — fix before any real device testing on cellular |
| Skip disk-space pre-check for v1.6's first cut | Simpler download path | Partial/corrupt state on full-disk failures, confusing errors (Pitfall 2) | Only acceptable if downloads are gated to Wi-Fi-only and manifest sizes stay small in initial region set — revisit before adding more/larger regions |
| Full style rewrite/reapply on every region add/remove instead of incremental source/layer APIs | No new maplibre API surface to learn/verify | Style-reload frequency multiplies exposure to known `onStyleLoaded`/sprite quirks (Pitfall 5) | Acceptable for an initial spike with 1-2 regions active; must be revisited before shipping multi-region viewport panning |
| `.index`-backed enum for download status | Zero extra code | Silent data corruption on any future enum reorder/insert (Pitfall 6) | Never — the explicit-int pattern costs almost nothing extra and removes the risk entirely |
| Defer the legacy trail-tile-file cleanup sweep to "later" | Smaller removal-phase diff | Permanently orphaned files on every device that predates the cleanup, undermining the new disk-usage feature's accuracy (Pitfall 9) | Never — ship cleanup in the same phase as code removal |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| `maplibre` 0.3.5 native `pmtiles://file://` protocol | Assuming multiple region archives merge into one seamless source the way a single trail's few cells did | Explicitly design for N-source/N-layer duplication at region scale (Pitfall 4), scoped to viewport, not "all downloaded regions always" |
| Dio `.download()` | Assuming Dio supports resume/append out of the box for large files | Implement `Range`-header resume manually with a `.part` file, as Dio has no built-in append-to-existing-file support |
| `path_provider` documents directory | Assuming it's backup-safe by default | Explicitly exclude the region tile directory from iCloud backup on iOS (Pitfall 10) |
| ObjectBox relations (`Region` ↔ `DownloadedTilePackage` ↔ `Trail`) | Assuming delete cascades like a relational DB foreign key | Write explicit cleanup transactions on every delete path (Pitfall 7) |
| Mapterhorn DEM pipeline (`generator.go`/download-dem endpoint) reused at region scale | Assuming region-sized DEM extracts behave like trail-cell-sized ones for best-effort-failure handling | Re-validate the "DEM failure never blocks vector basemap" contract still holds when DEM files are region-sized (larger, more likely to hit Pitfall 1-3 failure modes independently of the vector file) |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Style rewritten with every downloaded region always included | Slower map screen startup, increased memory, possible native-side slowdown per public MapLibre reports on high source/layer counts | Viewport-scoped source/layer inclusion (Pitfall 4) | Roughly once total downloaded regions exceeds what a trail-scale design (1-4 cells) was ever validated against — exact native-binding threshold unverified, spike recommended |
| Frequent full style reapplication as the user pans across region boundaries | Visible flicker, dropped in-flight tile requests, sprite/glyph re-resolution quirks resurfacing (Pitfall 5) | Debounce/coalesce reloads to only when the active region set actually changes | Any region-panning session once reload cadence exceeds "once per screen open," the case the original design was validated for |
| Per-tick ObjectBox writes for download progress | UI stutter, database write amplification | Coarse-grained batched writes (Pitfall 8) | Once progress reporting moves from in-memory-only (trail-scoped model) to persisted (needed for Settings page navigation round-trips) |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Trusting manifest-declared file sizes/URLs from `regions.json` without validating downloaded content | A compromised/misconfigured CDN entry could serve an oversized or malformed file that silently exhausts device storage or crashes the native PMTiles reader | Validate PMTiles header/magic bytes and declared size sanity after download, before promoting a `.part` file to the final path (ties into Pitfall 1's finished-file validation) |
| Path construction for region files using manifest-provided `id`/`name` values without sanitization | Path traversal if a manifest entry ever contains `../` segments (even though `regions.json` is bundled today, the "remote manifest" possibility is explicitly named as a deferred-not-impossible future feature in `PROJECT.md`) | Reuse the existing `_assertSafePath` pattern from `offline_style_rewriter.dart` for any region-derived path, exactly as it's already applied to trail cell paths |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| Generic error message for every download failure (network, disk-full, cancelled, corrupt-on-verify) | Users can't tell whether retrying will help, freeing space will help, or the download is simply broken | Distinct, actionable status/error states per failure category (ties into Pitfall 2 and 6's status-enum design) |
| Silent full-restart on resume instead of a visible "resuming from X%" | Users perceive the download as buggy/stuck when it appears to restart progress after backgrounding | Surface resume state explicitly once Pitfall 1's resume mechanism exists |
| No warning before starting a large region download on cellular | Unexpected data usage, possible carrier data cap surprises | Surface manifest-declared size before download starts and consider a Wi-Fi-only default/toggle (not explicitly scoped in `PROJECT.md` — worth flagging as a roadmap question, not assumed) |

## "Looks Done But Isn't" Checklist

- [ ] **Region download completes:** Often missing validation that the finished file isn't truncated/corrupt — verify PMTiles header/magic bytes post-download, not just "file exists" (Pitfall 1).
- [ ] **Disk usage total in Settings:** Often missing legacy trail-tile file cleanup — verify the reported total matches actual on-device storage via OS-level inspection, not just the sum of `DownloadedTilePackage` rows (Pitfall 9).
- [ ] **Region deletion:** Often missing cascade cleanup of dependent rows/files — verify no orphaned `DownloadedTilePackage` rows or dangling trail-coverage references remain after deleting a region that a trail's coverage guard once pointed at (Pitfall 7).
- [ ] **Multi-region map rendering:** Often "looks done" with 1-2 test regions but untested at realistic scale — verify style-load time and memory with 10+ downloaded regions before considering the viewport pipeline complete (Pitfall 4).
- [ ] **DEM toggle per region:** Often missing verification that a region can be `downloaded` (vector) while its DEM independently fails/is disabled — verify the status model can represent "vector-only" distinctly from "fully downloaded," matching the trail-scoped precedent that DEM failures are always best-effort.
- [ ] **iOS backup exclusion:** Often invisible until a real device's iCloud backup size is inspected — verify the region storage directory is actually excluded, not just assumed from a code comment (Pitfall 10).

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|-----------------|------------------|
| Full-file-only downloads shipped without resume (Pitfall 1) | MEDIUM | Add `.part` + Range-resume behind the existing download function's signature; no schema change needed since it's purely a download-mechanics fix |
| `.index`-backed enum already shipped and some devices have persisted values (Pitfall 6) | HIGH | Requires a one-time int-remap migration keyed to the exact enum declaration order at time of shipping — expensive and error-prone; strongly prefer prevention over recovery here |
| Orphaned legacy trail-tile files discovered post-release (Pitfall 9) | LOW | Ship the cleanup sweep as a follow-up patch release; it's a pure file-deletion operation with no data-model implications |
| Layer-clone explosion discovered after multiple regions ship (Pitfall 4) | HIGH | Requires reworking the viewport-scoping logic in `TileRepositoryManager` and `offline_style_rewriter`-equivalent code after the fact, likely touching the same code multiple future phases already depend on — strongly prefer the pre-ship spike |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| 1. No resume, full-file-only downloads | `TileRepositoryManager` download engine phase | Kill network mid-download on a real device; confirm resume from last byte, not byte 0 |
| 2. No disk-space pre-check, platform-inconsistent free-space reads | `TileRepositoryManager` download engine phase | Fill a test device/simulator near capacity; confirm a clear pre-download error, not a partial-write crash |
| 3. Backgrounding mid-download (iOS suspension / Android Doze) | `TileRepositoryManager` download engine phase | Background the app mid-download on both iOS and Android; confirm a deliberate pause state, not a silent hang |
| 4. Layer-clone explosion at region scale | Viewport-based tile-reading pipeline design phase (spike recommended) | Load 10+ region archives in the style; measure load time/memory against a defined budget on a mid-tier Android device |
| 5. Frequent style reloads colliding with known `maplibre` 0.3.5 quirks | Same phase as Pitfall 4 | Pan across multiple region boundaries repeatedly; confirm no flicker, no sprite/glyph regression on 2nd+ reload |
| 6. `.index`-backed enum status field | ObjectBox `Region`/`DownloadedTilePackage` schema phase | Code review checklist item: enum persistence must use explicit int constants, never `.index` |
| 7. Missing cascade delete for `Region`/`DownloadedTilePackage`/`Trail` coverage | Region deletion implementation (`TileRepositoryManager`) | Delete a region that a downloaded trail's coverage guard references; confirm no crash and a defined (not dangling) trail state |
| 8. Progress-write/UI-read races | Settings → Offline Maps live state binding phase | Watch the Settings page during an active download from a second observer (e.g. two navigation entries to the page); confirm no torn/inconsistent reads |
| 9. Orphaned legacy trail-tile files after code removal | Legacy trail-scoped code removal phase | Run the cleanup sweep against a dev device with pre-v1.6 trail downloads; verify `library/*/tiles/` is empty and disk-usage total matches OS-reported app storage |
| 10. iCloud backup bloat | Region storage directory setup (`TileRepositoryManager`) | Inspect iCloud backup settings on a real iOS device after downloading a region; confirm the region storage directory is excluded |

## Sources

- [MapLibre PMTiles for MapLibre GL — Protomaps Docs](https://docs.protomaps.com/pmtiles/maplibre) — MEDIUM confidence, official docs
- [PMTiles area download mechanism · Issue #2352 · maplibre/maplibre-native](https://github.com/maplibre/maplibre-native/issues/2352) — MEDIUM confidence, maintainer-engaged GitHub issue
- [Supporting multiple offline pmtile maps · maplibre/maplibre-native · Discussion #3764](https://github.com/maplibre/maplibre-native/discussions/3764) — MEDIUM confidence, direct maintainer commentary confirming no first-class multi-file offline story exists
- [Performance bottleneck when adding 8000+ PMTiles layers dynamically · maplibre/maplibre-gl-js · Discussion #5988](https://github.com/maplibre/maplibre-gl-js/discussions/5988) — MEDIUM confidence (JS binding, not native — directionally relevant, exact native threshold unverified)
- [ObjectBox Relations docs](https://docs.objectbox.io/relations) and [ObjectBox Data Model Updates docs](https://docs.objectbox.io/advanced/data-model-updates) — HIGH confidence, official docs
- [Removing a entity with ToOne relation keeps related entity · Issue #547 · objectbox/objectbox-dart](https://github.com/objectbox/objectbox-dart/issues/547) — MEDIUM confidence, confirms no-cascade-delete behavior on the Dart binding specifically
- [background_downloader | Flutter package](https://pub.dev/packages/background_downloader) — HIGH confidence, official pub.dev docs describing iOS/Android background download termination behavior that motivated Pitfall 3
- [isExcludedFromBackupKey | Apple Developer Documentation](https://developer.apple.com/documentation/foundation/nsurlisexcludedfrombackupkey) — HIGH confidence, official Apple docs
- Android Doze/App Standby background execution constraints — MEDIUM confidence, corroborated across multiple community sources (Android Developers Blog, community write-ups) rather than a single primary source
- In-repo: `app/lib/services/trail_download_service.dart`, `app/lib/util/offline_style_rewriter.dart`, `app/pubspec.yaml`, `.planning/PROJECT.md` — HIGH confidence, direct inspection of the existing system this milestone replaces

---
*Pitfalls research for: App-wide, region-based offline map tile management in Flutter (maplibre 0.3.5 + ObjectBox), replacing trail-scoped offline tiles*
*Researched: 2026-07-21*
