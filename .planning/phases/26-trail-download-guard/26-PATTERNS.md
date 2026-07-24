# Phase 26: Trail Download Guard - Pattern Map

**Mapped:** 2026-07-24
**Files analyzed:** 5 (2 new, 3 modified)
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `app/lib/util/trail_coverage_util.dart` | utility | transform (pure fn) | `app/lib/util/byte_format_util.dart` (pure fn style) / `app/lib/util/region_tile_status_util.dart` (test-style precedent) | role-match |
| `app/lib/components/trail/missing_coverage_sheet.dart` | component (bottom sheet) | request-response (user selection → return value) | `app/lib/components/navigation/track_save_options_sheet.dart` | exact |
| `app/lib/provider/trail/trail_download_state_provider.dart` (MODIFIED) | provider/service (Notifier) | event-driven / CRUD (download orchestration) | itself (existing `DownloadingTrailIds.download`) + `app/lib/provider/region/tile_repository_provider.dart` (`TileRepositoryStatus`) for the parallel-trigger pattern | exact (self) |
| `app/lib/services/download_notification_service.dart` (MODIFIED) | service | event-driven (notification side-effects) | itself (existing `showProgress`/`showSuccess`/`showError`) | exact (self) |
| `test/util/trail_coverage_util_test.dart` | test | transform | `app/test/util/region_tile_status_util_test.dart` | role-match |

## Pattern Assignments

### `app/lib/util/trail_coverage_util.dart` (utility, transform)

**Analog:** `app/lib/util/byte_format_util.dart` (pure top-level function, `library;` doc header, no class wrapper) + region/trail bbox field shapes from `app/lib/entities/region_entity.dart` and `app/lib/models/trail.dart`.

**Module shape pattern** (byte_format_util.dart, full file, lines 1-19):
```dart
/// Human-readable byte formatting for the region tile repository's disk
/// usage displays (SETUI-05).
library;

const int _kb = 1024;

String formatBytes(int bytes) {
  if (bytes >= _gb) return '${(bytes / _gb).toStringAsFixed(1)} GB';
  ...
}
```
Copy this shape: a `library;` doc comment header, no class wrapper, plain top-level functions — exactly what `missingCoverageRegions`/`bboxesOverlap` should look like.

**Bbox field shapes to consume** — `RegionEntity` (region_entity.dart:30-33):
```dart
double minLon;
double minLat;
double maxLon;
double maxLat;
```
and its computed status getter (region_entity.dart:103-121, `RegionStatus get status`) — compare by enum value, never `.code`/`.index`:
```dart
switch (vectorTarget.status) {
  case PackageStatus.downloaded:
    final isStale = catalogStatus == CatalogStatus.ready &&
        version != null && version != lastDownloadedVersion;
    return isStale ? RegionStatus.updateAvailable : RegionStatus.downloaded;
  ...
}
```

`Trail` bbox fields (trail.dart:80-83, freezed, JSON-keyed but the same double fields at runtime):
```dart
@JsonKey(name: 'max_lat') @Default(0) double maxLat,
@JsonKey(name: 'max_lon') @Default(0) double maxLon,
@JsonKey(name: 'min_lat') @Default(0) double minLat,
@JsonKey(name: 'min_lon') @Default(0) double minLon,
```

**Core pattern (from RESEARCH.md, verified against real field names above):**
```dart
bool bboxesOverlap({
  required double aMinLon, required double aMinLat,
  required double aMaxLon, required double aMaxLat,
  required double bMinLon, required double bMinLat,
  required double bMaxLon, required double bMaxLat,
}) {
  return aMinLon <= bMaxLon && aMaxLon >= bMinLon &&
      aMinLat <= bMaxLat && aMaxLat >= bMinLat;
}
```
Compare `RegionStatus` values by identity (`!=  RegionStatus.downloaded && != RegionStatus.updateAvailable`), never by `.code` int (Pitfall 3 — `RegionStatus.error` code is `5`, not `2` as CONTEXT.md's table mis-transcribes; never hand-transcribe codes).

Per D-04/Pitfall 4: compute **two** sets, not one — `overlappingRegions` (any status) and `missing` (subset not downloaded/updateAvailable) — so "fully covered" and "no region at all" are distinguishable by the caller.

---

### `app/lib/components/trail/missing_coverage_sheet.dart` (component, request-response)

**Analog:** `app/lib/components/navigation/track_save_options_sheet.dart` (full file read, 155 lines) — copy this shape near-verbatim per RESEARCH.md Pattern 3 and UI-SPEC's explicit mandate.

**Show-function signature pattern** (track_save_options_sheet.dart:15-27):
```dart
Future<(bool recalcHeights, bool followRoads)?> showTrackSaveOptionsSheet(
  BuildContext context,
) {
  return showModalBottomSheet<(bool, bool)>(
    context: context,
    isDismissible: true,
    enableDrag: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (context) => const _TrackSaveOptionsSheetContent(),
  );
}
```
For `missing_coverage_sheet.dart`, add `isScrollControlled: true` (per RESEARCH.md Pattern 3 — the region list can be longer than this precedent's fixed 2-toggle sheet) and return a typed `MissingCoverageSelection?` instead of a raw tuple.

**Drag handle + title + scroll body pattern** (track_save_options_sheet.dart:47-80):
```dart
return SingleChildScrollView(
  padding: EdgeInsets.fromLTRB(16, 16, 16, 24 + MediaQuery.of(context).viewInsets.bottom),
  child: Column(
    mainAxisSize: MainAxisSize.min,
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      Padding(
        padding: const EdgeInsets.symmetric(vertical: 8.0),
        child: Center(
          child: Container(
            width: 30, height: 5,
            decoration: BoxDecoration(
              borderRadius: const BorderRadius.all(Radius.circular(24)),
              color: theme.colorScheme.secondaryContainer,
            ),
          ),
        ),
      ),
      Padding(
        padding: const EdgeInsets.symmetric(vertical: 8.0),
        child: Text(l10n.save_recording_options,
          style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
      ),
      const SizedBox(height: 8),
      // ... per-item cards ...
      const SizedBox(height: 16),
      FilledButton(
        onPressed: () => Navigator.pop(context, (_recalcHeights, _followRoads)),
        child: Text(l10n.save),
      ),
    ],
  ),
);
```
Note the sheet title uses UI-SPEC copy "Missing offline map coverage"; the button label is "Download" (always enabled — never gate `onPressed` on a selection, per Pitfall 5/D-08).

**Bordered flat card pattern** (`_ToggleCard`, track_save_options_sheet.dart:112-154 — "Terrain Log" doctrine, elevation 0, border only):
```dart
return Card(
  elevation: 0,
  shape: RoundedRectangleBorder(
    borderRadius: BorderRadius.circular(16),
    side: BorderSide(color: theme.colorScheme.outline),
  ),
  clipBehavior: Clip.antiAlias,
  child: SwitchListTile(...),
);
```
For the region block, replace `SwitchListTile` with a `Column` containing the region name (bold, `bodyLarge`) + one or two `CheckboxListTile`s (Vector always, DEM only if `region.demUrl != null`) — see the row-gating pattern below.

**Region row / DEM-gating pattern** (`settings_offline_regions_screen.dart:307-407`, `_buildActiveRow`/`_buildVectorTile`):
```dart
Widget _buildActiveRow(RegionEntity region) {
  ...
  return Container(
    decoration: BoxDecoration(
      border: Border.all(color: Theme.of(context).colorScheme.outline),
      borderRadius: const BorderRadius.all(Radius.circular(8)),
    ),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
          child: Text(region.name,
            style: Theme.of(context).textTheme.bodyLarge!.copyWith(fontWeight: FontWeight.bold)),
        ),
        _buildVectorTile(...),
        if (region.demUrl != null) _buildDemTile(...),
      ],
    ),
  );
}
```
```dart
// lines 393-406 — ListTile shape for a package row, size in subtitle
return ListTile(
  dense: true,
  contentPadding: const EdgeInsets.symmetric(horizontal: 16),
  leading: _tileLeadingIcon(done: isDone, error: isError),
  title: Text(l10n.regions_vector_tile_title),
  subtitle: _tileSubtitle(downloading: isDownloading, progress: liveProgress,
      text: subtitleText, textColor: subtitleColor, accentColor: accentColor),
  trailing: _buildVectorTrailing(region, status, l10n, accentColor),
);
```
`formatBytes()` (byte_format_util.dart) is the size-string source; reuse verbatim, no new formatter (Don't-Hand-Roll table).

**Checkbox default state (D-07):** Vector pre-checked, DEM unchecked — initialize local `State` fields per region on sheet build, mirroring `_TrackSaveOptionsSheetContentState`'s `bool _recalcHeights = false;` field-init pattern (track_save_options_sheet.dart:39-40), just with per-region maps instead of two flat bools.

---

### `app/lib/provider/trail/trail_download_state_provider.dart` (MODIFIED — provider/Notifier, event-driven)

**Analog:** itself (full file, 88 lines) — the guard is inserted at the top of the existing `download()` method; do not restructure the rest.

**Full existing method** (trail_download_state_provider.dart:22-86):
```dart
Future<void> download(Trail trail) async {
  if (state.contains(trail.id)) return;
  state = {...state, trail.id};

  final trailDownloadService = ref.read(trailDownloadServiceProvider);
  final notificationService = ref.read(downloadNotificationServiceProvider);
  final toastNotifier = ref.read(toastProvider.notifier);

  final glyphCacheWarm = ref.read(glyphSpriteCacheProvider.future);

  toastNotifier.add(ToastMessage(
    type: ToastType.info,
    icon: FontAwesomeIcons.download,
    text: 'Downloading ${trail.name}...',
  ));
  await notificationService.showProgress(trail.name, 0, 0);

  try {
    await trailDownloadService.downloadTrail(
      trail,
      onGeneratingChanged: (isGenerating) {
        if (isGenerating) notificationService.showGenerating(trail.name);
      },
      onProgress: (done, total) =>
          notificationService.showProgress(trail.name, done, total),
    );
    await notificationService.showSuccess(trail.name);
    ref.invalidate(trailLibraryProvider);
    toastNotifier.add(ToastMessage(
      type: ToastType.success, icon: FontAwesomeIcons.circleCheck,
      text: 'Trail saved for offline use',
    ));
  } catch (e) {
    await notificationService.showError(trail.name);
    toastNotifier.add(ToastMessage(
      type: ToastType.error, icon: FontAwesomeIcons.xmark,
      text: 'Error saving trail',
    ));
  } finally {
    state = {...state}..remove(trail.id);
  }

  try {
    await glyphCacheWarm;
  } catch (_) {}
}
```
Insert the guard between `state = {...state, trail.id};` and the `trailDownloadService`/`notificationService`/`toastNotifier` reads — i.e. compute `overlappingRegions`/`missing` synchronously via `ref.read(regionListNotifierProvider)` (region_provider.dart:165-174) BEFORE the re-entry guard's early return matters, then conditionally await `showMissingCoverageSheet(navigatorKey.currentContext!, ...)`. If the sheet is dismissed (`null`), abort entirely (`state = {...state}..remove(trail.id); return;`) — do not run the existing body.

**Non-widget UI trigger pattern** (`navigatorKey.currentContext`, `app/lib/main.dart:202-214`, existing resume-navigation dialog — reuse verbatim):
```dart
final ctx = navigatorKey.currentContext;
if (ctx == null) return;
...
showDialog<bool>(
  context: ctx,
  builder: (dialogCtx) => AlertDialog(
    content: Text(AppLocalizations.of(dialogCtx)!.resume_navigation_prompt(trailName)),
    ...
  ),
);
```
Same null-check-then-use shape applies to `showModalBottomSheet` from inside `download()`.

**Region download trigger** (fire-and-forget, D-09 — never `await` before the trail download starts) — `app/lib/provider/region/tile_repository_provider.dart:42-81` (`downloadVector`) shows the idempotent-reentry-guarded shape to call, unmodified, from the sheet's selection:
```dart
ref.read(tileRepositoryStatusProvider.notifier).downloadVector(region.id);
if (demChecked) {
  ref.read(tileRepositoryStatusProvider.notifier).downloadDem(region.id);
}
```
Note `TileRepositoryStatus.downloadVector`/`downloadDem` are themselves `Future<void>` and already idempotent/re-entry-guarded (`if (state[regionId]?.vectorProgress != null) return;`) — do not add a second guard on top.

**Local catalog read (D-11 — never call `refreshCatalog()`):** `region_provider.dart:165-174`:
```dart
@Riverpod(name: 'regionListNotifierProvider')
class RegionListNotifier extends _$RegionListNotifier {
  @override
  List<RegionEntity> build() {
    final store = ref.watch(objectBoxProvider);
    final regions = store.box<RegionEntity>().getAll();
    regions.sort((a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()));
    return regions;
  }
}
```
Read via `ref.read(regionListNotifierProvider)` — plain synchronous list, no I/O, no `RegionRepository` (that class only exposes `fetchCatalog`/`upsertCatalog`/`refreshCatalog`, all network-touching — do not use it here per RESEARCH.md's Anti-Patterns correction to CONTEXT.md).

**Error handling pattern:** identical try/catch/finally shape as the existing method above — keep the guard's own sheet-await outside the `try` (a dismissed sheet is not an "error", it's a clean abort with no notification/toast at all).

---

### `app/lib/services/download_notification_service.dart` (MODIFIED — service, event-driven)

**Analog:** itself (full file, 155 lines) — extend `showProgress`, do not replace it.

**Existing signature to extend** (download_notification_service.dart:69-100):
```dart
Future<void> showProgress(String trailName, int done, int total) async {
  await _ensureInitialized();
  final androidDetails = AndroidNotificationDetails(
    _channelId, _channelName,
    channelDescription: 'Trail tile download progress',
    importance: Importance.low, priority: Priority.low,
    showProgress: true,
    maxProgress: total > 0 ? total : 1,
    progress: done,
    indeterminate: total == 0,
    ongoing: true, autoCancel: false, onlyAlertOnce: true,
  );
  const iosDetails = DarwinNotificationDetails(
    presentAlert: false, presentBadge: false, presentSound: false,
  );
  await _plugin.show(
    _notificationId,
    trailName,
    total > 0
        ? 'Downloading trail... ${((done / total) * 100).clamp(0, 100).round()}%'
        : 'Preparing download...',
    NotificationDetails(android: androidDetails, iOS: iosDetails),
  );
}
```
**Pitfall 1 (RESEARCH.md):** this hardcodes title=`trailName`, body=`'Downloading trail... {pct}%'`. Add optional nullable `String? title`/`String? body` params (default to today's exact strings when null, so the 0-region GUARD-01 path stays byte-for-byte unchanged) — OR add a sibling method `showAggregateProgress(String title, String body, int done, int total)` reusing the same `AndroidNotificationDetails`/`_plugin.show(_notificationId, ...)` body shape verbatim, changing only the two text arguments. Either approach must keep the single fixed `_notificationId = 42` (no new notification channel/id).

Same extension shape applies to `showSuccess`/`showError` only if D-10's aggregate copy needs distinct terminal-state text (UI-SPEC's copy table only specifies aggregate title/body for the in-progress state, so `showSuccess`/`showError` likely need no change — confirm against UI-SPEC's Copywriting Contract row before touching them).

---

## Shared Patterns

### Enum comparison — explicit `.code`, never `.index`
**Source:** `app/lib/models/region_status.dart` (full file) — `RegionStatus`/`CatalogStatus`/`PackageStatus` all use explicit `.code` ints with an `orElse` fallback decode; `error` was appended out of numeric order (`RegionStatus.error(5)`, not `2`).
**Apply to:** `trail_coverage_util.dart`'s coverage filter — compare by enum value (`region.status == RegionStatus.downloaded`), never transcribe or compare `.code` ints.

### Non-widget → UI trigger via `navigatorKey`
**Source:** `app/lib/main.dart:202-214`, `navigatorKey` declared in `app/lib/provider/router_provider.dart:56` (`final navigatorKey = GlobalKey<NavigatorState>();`)
**Apply to:** `trail_download_state_provider.dart`'s `download()` method — the only place in this phase a widget-less Notifier needs to show UI. Read `navigatorKey.currentContext` transiently at point of use; never store it on the Notifier instance (anti-pattern flagged in RESEARCH.md).

### Fire-and-forget parallel background downloads
**Source:** `app/lib/provider/region/tile_repository_provider.dart` (`downloadVector`/`downloadDem`) + existing `trailDownloadService.downloadTrail` call in `trail_download_state_provider.dart`.
**Apply to:** the sheet's Download button handler — never `await` region downloads before starting/returning from the trail download; all run concurrently, un-awaited by the sheet itself (D-09).

### Bordered flat "Terrain Log" card, no shadows
**Source:** `_ToggleCard` in `track_save_options_sheet.dart:112-154`, and the region-row `Container` in `settings_offline_regions_screen.dart:320-324` — both use `elevation: 0` + `BorderSide(color: colorScheme.outline)`, never `Card`'s default shadow.
**Apply to:** `missing_coverage_sheet.dart`'s per-region blocks.

### Size formatting
**Source:** `app/lib/util/byte_format_util.dart` — `formatBytes(int bytes)`.
**Apply to:** every size string in the new sheet (per-package size, combined-size summary).

## No Analog Found

None — every file in scope has a strong (exact or role-match) existing analog; this phase is explicitly a composition of already-shipped patterns (RESEARCH.md's own conclusion).

## Metadata

**Analog search scope:** `app/lib/util/`, `app/lib/components/navigation/`, `app/lib/components/trail/`, `app/lib/provider/trail/`, `app/lib/provider/region/`, `app/lib/services/`, `app/lib/entities/`, `app/lib/models/`, `app/lib/routes/settings_offline_regions_screen.dart`, `app/lib/main.dart`, `app/test/util/`
**Files scanned:** 12 read directly (byte_format_util.dart, track_save_options_sheet.dart, trail_download_state_provider.dart, download_notification_service.dart, region_provider.dart, tile_repository_provider.dart, region_entity.dart, region_status.dart, settings_offline_regions_screen.dart excerpt, main.dart excerpt, trail.dart excerpt) plus RESEARCH.md/CONTEXT.md/UI-SPEC.md
**Pattern extraction date:** 2026-07-24
