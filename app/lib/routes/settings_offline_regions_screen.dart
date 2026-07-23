import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/provider/region/tile_repository_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/byte_format_util.dart';
import 'package:wanderer/util/region_disk_usage_util.dart';
import 'package:wanderer/util/region_tile_status_util.dart';

/// SETUI-01..06: the "Offline Maps/Regions" Settings screen — a flat,
/// name-searchable, A-Z region list backed by `regionListNotifierProvider`,
/// with a live disk-usage summary and correctly-disabled `building`/`error`
/// catalog rows (D-09). Each ready region renders as two independent
/// list tiles — Vector and Elevation data (DEM) — each with its own
/// download/cancel/delete action; see `_buildActiveRow`.
///
/// A `ConsumerStatefulWidget` because it holds the local search-query state
/// and the disk-usage `FutureBuilder`'s future (recreated only when the
/// region-list snapshot identity changes, not on every unrelated rebuild).
class SettingsOfflineRegionsScreen extends ConsumerStatefulWidget {
  const SettingsOfflineRegionsScreen({super.key});

  @override
  ConsumerState<SettingsOfflineRegionsScreen> createState() =>
      _SettingsOfflineRegionsScreenState();
}

class _SettingsOfflineRegionsScreenState
    extends ConsumerState<SettingsOfflineRegionsScreen> {
  String _searchQuery = '';

  List<RegionEntity>? _diskUsageRegions;
  Future<int>? _diskUsageFuture;

  /// Set only for the genuine fresh-install edge case (RESEARCH.md Pitfall
  /// 4): zero cached regions AND the initial catalog fetch failed. Every
  /// other failure (a non-empty cached snapshot) surfaces a toast instead —
  /// see `_refreshCatalog`.
  Object? _freshInstallError;
  StackTrace? _freshInstallStackTrace;

  @override
  void initState() {
    super.initState();
    // Fire-and-forget (RESEARCH.md Pitfall 4) — never blocks the list on a
    // network round-trip; the list renders the synchronous provider
    // snapshot unconditionally below.
    _refreshCatalog();
  }

  /// Refreshes the local catalog from the backend, then re-reads the
  /// snapshot provider so the list reflects any upserted rows. On failure,
  /// surfaces a toast ONLY if the current snapshot already has cached data
  /// — an already-downloaded, usable-offline region must never disappear
  /// just because a catalog refresh failed while offline. Reserves the
  /// full-screen error treatment for the one case with nothing to show.
  Future<void> _refreshCatalog() async {
    try {
      await ref.read(regionRepositoryProvider).refreshCatalog();
      if (!mounted) return;
      ref.invalidate(regionListNotifierProvider);
    } catch (e, st) {
      if (!mounted) return;
      final cached = ref.read(regionListNotifierProvider);
      if (cached.isNotEmpty) {
        final l10n = AppLocalizations.of(context)!;
        ref
            .read(toastProvider.notifier)
            .add(
              ToastMessage(
                type: ToastType.error,
                icon: FontAwesomeIcons.circleExclamation,
                text: l10n.error_saving_settings,
              ),
            );
        return;
      }
      setState(() {
        _freshInstallError = e;
        _freshInstallStackTrace = st;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final regions = ref.watch(regionListNotifierProvider);

    // Recreate the disk-usage future only when the region-list snapshot
    // identity changes (e.g. after ref.invalidate post-mutation) — not on
    // every unrelated rebuild (search typing, etc).
    if (!identical(_diskUsageRegions, regions)) {
      _diskUsageRegions = regions;
      _diskUsageFuture = totalRegionDiskUsageBytes(regions);
    }

    final query = _searchQuery.trim().toLowerCase();
    final filtered = query.isEmpty
        ? regions
        : regions.where((r) => r.name.toLowerCase().contains(query)).toList();

    // Fresh-install edge case only: zero cached regions AND the initial
    // catalog fetch failed. Reuse AsyncLoader/WandererError verbatim per
    // UI-SPEC's copy table — every other case renders the synchronous
    // snapshot unconditionally.
    final showFullScreenError = regions.isEmpty && _freshInstallError != null;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.settings_offline_regions_title),
      ),
      body: showFullScreenError
          ? AsyncLoader<List<RegionEntity>>(
              asyncValue: AsyncValue<List<RegionEntity>>.error(
                _freshInstallError!,
                _freshInstallStackTrace ?? StackTrace.current,
              ),
              mockData: const [],
              builder: (_) => const SizedBox.shrink(),
            )
          : Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
                  child: WandererSearchBar(
                    hintText: l10n.regions_search_hint,
                    onChanged: (value) => setState(() => _searchQuery = value),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                  child: _buildDiskUsageSummary(l10n, regions),
                ),

                Expanded(
                  child: regions.isEmpty
                      ? _buildEmptyState(
                          title: l10n.regions_empty_catalog_title,
                          body: l10n.regions_empty_catalog_body,
                        )
                      : filtered.isEmpty
                      ? _buildEmptyState(
                          title: l10n.regions_empty_search_title,
                          body: l10n.regions_empty_search_body,
                        )
                      : ListView.separated(
                          padding: const EdgeInsets.only(bottom: 16),
                          itemCount: filtered.length,
                          separatorBuilder: (_, _) => Divider(
                            height: 1,
                            color: Theme.of(context).colorScheme.outline,
                          ),
                          itemBuilder: (context, index) =>
                              _buildRegionRow(filtered[index]),
                        ),
                ),
              ],
            ),
    );
  }

  /// Disk-usage summary card (SETUI-05, D-06): a Display-role headline byte
  /// figure over a Label-role sub-text stating how many regions contribute
  /// to it. `totalRegionDiskUsageBytes` reads real on-disk bytes (including
  /// `.part` partial files) — see `region_disk_usage_util.dart`.
  Widget _buildDiskUsageSummary(
    AppLocalizations l10n,
    List<RegionEntity> regions,
  ) {
    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).colorScheme.outline),
        borderRadius: BorderRadius.all(Radius.circular(8)),
        color: Colors.grey.shade100,
      ),
      padding: const EdgeInsets.all(16),
      child: FutureBuilder<int>(
        future: _diskUsageFuture,
        builder: (context, snapshot) {
          final totalBytes = snapshot.data ?? 0;
          final count = regions.where(_hasAnyDiskUsage).length;
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                formatBytes(totalBytes),
                style: Theme.of(context).textTheme.headlineSmall!.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                l10n.regions_disk_usage_summary(formatBytes(totalBytes), count),
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          );
        },
      ),
    );
  }

  /// A region contributes to the disk-usage region COUNT (distinct from the
  /// byte sum itself) when it has at least one package that is downloaded
  /// or mid-flight (downloading) — a partial `.part` file still occupies
  /// real disk space, D-06.
  bool _hasAnyDiskUsage(RegionEntity region) {
    bool packageOccupiesDisk(PackageStatus? status) =>
        status == PackageStatus.downloaded ||
        status == PackageStatus.downloading;
    return packageOccupiesDisk(region.vectorPackage.target?.status) ||
        packageOccupiesDisk(region.demPackage.target?.status);
  }

  Widget _buildEmptyState({required String title, required String body}) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              title,
              style: Theme.of(context).textTheme.bodyLarge,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              body,
              style: Theme.of(context).textTheme.bodySmall,
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }

  /// CATALOG-STATUS PRECEDENCE GATE (RESEARCH.md Pattern 2, D-09) — must run
  /// BEFORE any RegionStatus/download-action rendering. A `building`/`error`
  /// catalog region always renders disabled with a caption and NO download
  /// action, regardless of what the local `status` getter would otherwise
  /// compute (always `notDownloaded`, since no package row exists yet).
  Widget _buildRegionRow(RegionEntity region) {
    if (region.catalogStatus == CatalogStatus.building) {
      return _buildDisabledRow(
        region,
        caption: AppLocalizations.of(context)!.regions_not_yet_available,
      );
    }
    if (region.catalogStatus == CatalogStatus.error) {
      return _buildDisabledRow(
        region,
        caption: AppLocalizations.of(context)!.regions_build_failed,
      );
    }
    return _buildActiveRow(region);
  }

  Widget _buildDisabledRow(RegionEntity region, {required String caption}) {
    return Opacity(
      opacity: 0.6,
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    region.name,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodyLarge!.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(caption, style: Theme.of(context).textTheme.bodySmall),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// Reached only when `catalogStatus == CatalogStatus.ready`. Renders the
  /// region name, then a Vector tile and (when `region.demUrl != null`) an
  /// Elevation data tile — each independently downloadable/cancellable/
  /// deletable (SETUI-04). Deleting the Vector tile cascades to delete DEM
  /// too (D-02); deleting the DEM tile removes only the DEM package (D-01).
  Widget _buildActiveRow(RegionEntity region) {
    final l10n = AppLocalizations.of(context)!;
    final downloadState = ref.watch(tileRepositoryStatusProvider)[region.id];
    final colorScheme = Theme.of(context).colorScheme;
    final accentColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;

    final vectorStatus = resolveVectorTileStatus(
      region.status,
      downloadState?.vectorProgress,
    );

    return Container(
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).colorScheme.outline),
        borderRadius: const BorderRadius.all(Radius.circular(8)),
      ),
      margin: const EdgeInsets.symmetric(horizontal: 16.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
            child: Text(
              region.name,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(
                context,
              ).textTheme.bodyLarge!.copyWith(fontWeight: FontWeight.bold),
            ),
          ),
          _buildVectorTile(
            region,
            vectorStatus,
            downloadState?.vectorProgress,
            l10n,
            accentColor,
          ),
          if (region.demUrl != null)
            _buildDemTile(
              region,
              downloadState?.demProgress,
              l10n,
              accentColor,
            ),
          const SizedBox(height: 4),
        ],
      ),
    );
  }

  Widget _buildVectorTile(
    RegionEntity region,
    RegionStatus status,
    double? liveProgress,
    AppLocalizations l10n,
    Color accentColor,
  ) {
    final isDownloading = status == RegionStatus.downloading;
    final isDone =
        status == RegionStatus.downloaded ||
        status == RegionStatus.updateAvailable;
    final isError = status == RegionStatus.error;
    final onDiskBytes = region.vectorPackage.target?.sizeBytesOnDisk;

    final String subtitleText;
    final Color? subtitleColor;
    if (isError) {
      subtitleText = l10n.regions_download_failed;
      subtitleColor = Colors.redAccent;
    } else if (status == RegionStatus.updateAvailable) {
      subtitleText = l10n.regions_update_available;
      subtitleColor = Colors.orange;
    } else {
      subtitleText = formatBytes(
        (status == RegionStatus.downloaded ? onDiskBytes : null) ??
            region.vectorSize ??
            0,
      );
      subtitleColor = null;
    }

    return ListTile(
      dense: true,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16),
      leading: _tileLeadingIcon(done: isDone, error: isError),
      title: Text(l10n.regions_vector_tile_title),
      subtitle: _tileSubtitle(
        downloading: isDownloading,
        progress: liveProgress,
        text: subtitleText,
        textColor: subtitleColor,
        accentColor: accentColor,
      ),
      trailing: _buildVectorTrailing(region, status, l10n, accentColor),
    );
  }

  Widget _buildVectorTrailing(
    RegionEntity region,
    RegionStatus status,
    AppLocalizations l10n,
    Color accentColor,
  ) {
    switch (status) {
      case RegionStatus.notDownloaded:
        return IconButton(
          icon: const FaIcon(FontAwesomeIcons.download, size: 16),
          color: accentColor,
          tooltip: l10n.download,
          onPressed: () => _onDownloadVector(region),
        );
      case RegionStatus.downloading:
        return IconButton(
          icon: const FaIcon(FontAwesomeIcons.xmark, size: 16),
          tooltip: l10n.cancel,
          onPressed: () => _onCancelVector(region),
        );
      case RegionStatus.downloaded:
        return IconButton(
          icon: const FaIcon(FontAwesomeIcons.trash, size: 16),
          color: Colors.redAccent,
          onPressed: () => _onDeleteRegion(region),
        );
      case RegionStatus.updateAvailable:
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.arrowsRotate),
              color: accentColor,
              tooltip: l10n.regions_update_action,
              onPressed: () => _onDownloadVector(region),
            ),
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.trash),
              color: Colors.redAccent,
              onPressed: () => _onDeleteRegion(region),
            ),
          ],
        );
      case RegionStatus.error:
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.arrowsRotate),
              color: accentColor,
              tooltip: l10n.regions_retry,
              onPressed: () => _onDownloadVector(region),
            ),
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.trash),
              color: Colors.redAccent,
              onPressed: () => _onDeleteRegion(region),
            ),
          ],
        );
    }
  }

  Widget _buildDemTile(
    RegionEntity region,
    double? liveProgress,
    AppLocalizations l10n,
    Color accentColor,
  ) {
    final persisted =
        region.demPackage.target?.status ?? PackageStatus.notDownloaded;
    final status = resolveDemTileStatus(persisted, liveProgress);
    final isDownloading = status == PackageStatus.downloading;
    final isDone = status == PackageStatus.downloaded;
    final isError = status == PackageStatus.error;
    final onDiskBytes = region.demPackage.target?.sizeBytesOnDisk;

    final subtitleText = isError
        ? l10n.regions_download_failed
        : formatBytes((isDone ? onDiskBytes : null) ?? region.demSize ?? 0);

    return ListTile(
      dense: true,

      contentPadding: const EdgeInsets.symmetric(horizontal: 16),
      leading: _tileLeadingIcon(done: isDone, error: isError),
      title: Text(l10n.regions_dem_tile_title),
      subtitle: _tileSubtitle(
        downloading: isDownloading,
        progress: liveProgress,
        text: subtitleText,
        textColor: isError ? Colors.redAccent : null,
        accentColor: accentColor,
      ),
      trailing: _buildDemTrailing(region, status, l10n, accentColor),
    );
  }

  Widget _buildDemTrailing(
    RegionEntity region,
    PackageStatus status,
    AppLocalizations l10n,
    Color accentColor,
  ) {
    switch (status) {
      case PackageStatus.notDownloaded:
        return IconButton(
          icon: const FaIcon(FontAwesomeIcons.download, size: 16),
          color: accentColor,
          tooltip: l10n.download,
          onPressed: () => _onDownloadDem(region),
        );
      case PackageStatus.downloading:
        return IconButton(
          icon: const FaIcon(FontAwesomeIcons.xmark, size: 16),
          tooltip: l10n.cancel,
          onPressed: () => _onCancelDem(region),
        );
      case PackageStatus.downloaded:
        return IconButton(
          icon: const FaIcon(FontAwesomeIcons.trash, size: 16),
          color: Colors.redAccent,
          onPressed: () => _onDeleteDemPackage(region),
        );
      case PackageStatus.error:
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.arrowsRotate),
              color: accentColor,
              tooltip: l10n.regions_retry,
              onPressed: () => _onDownloadDem(region),
            ),
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.trash),
              color: Colors.redAccent,
              onPressed: () => _onDeleteDemPackage(region),
            ),
          ],
        );
    }
  }

  /// Shared leading status glyph for both tiles: a green check once that
  /// package is downloaded (or stale-but-downloaded, for Vector), a red
  /// exclamation on error, otherwise nothing (the notDownloaded/downloading
  /// states read entirely from the subtitle + trailing action).
  Widget? _tileLeadingIcon({required bool done, required bool error}) {
    if (error) {
      return const FaIcon(
        FontAwesomeIcons.circleExclamation,
        color: Colors.redAccent,
      );
    }
    if (done) {
      return const FaIcon(FontAwesomeIcons.circleCheck, color: Colors.green);
    }
    return null;
  }

  /// Shared subtitle for both tiles: a progress bar while that specific
  /// package is downloading (D-07 — now per-tile, not a combined average,
  /// since vector/DEM downloads are fully independent), otherwise [text]
  /// (byte size, "Update available", or "Download failed").
  Widget _tileSubtitle({
    required bool downloading,
    required double? progress,
    required String text,
    required Color? textColor,
    required Color accentColor,
  }) {
    if (downloading) {
      return Padding(
        padding: const EdgeInsets.only(top: 4),
        child: LinearProgressIndicator(value: progress, color: accentColor),
      );
    }
    final mutedColor = Theme.of(
      context,
    ).colorScheme.onSurface.withValues(alpha: 0.5);
    return Text(
      text,
      style: Theme.of(
        context,
      ).textTheme.bodySmall?.copyWith(color: textColor ?? mutedColor),
    );
  }

  /// Persists [op] and surfaces only an error toast on failure — mirrors
  /// `settings_categories_screen.dart`'s `_save` wrapper. Additionally
  /// ALWAYS invalidates `regionListNotifierProvider` in a `finally` block
  /// (RESEARCH.md Pitfall 2 — ObjectBox `ToOne.target` caches per-instance
  /// after first read) so every tile reflects the true post-action state
  /// regardless of success or failure.
  Future<void> _save(Future<void> Function() op) async {
    try {
      await op();
    } catch (_) {
      if (!mounted) return;
      final l10n = AppLocalizations.of(context)!;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_settings,
            ),
          );
    } finally {
      if (mounted) ref.invalidate(regionListNotifierProvider);
    }
  }

  void _onDownloadVector(RegionEntity region) {
    _save(
      () => ref
          .read(tileRepositoryStatusProvider.notifier)
          .downloadVector(region.id),
    );
  }

  /// Cancel is synchronous: `cancelVector` clears the ephemeral progress
  /// (which Riverpod immediately re-renders from — the tile flips back to a
  /// Download button at once), then we invalidate the region list so the
  /// disk-usage summary and persisted reads refresh too. No pause/resume:
  /// the manager deletes the `.part` file, so a later download restarts from
  /// byte 0.
  void _onCancelVector(RegionEntity region) {
    ref.read(tileRepositoryStatusProvider.notifier).cancelVector(region.id);
    ref.invalidate(regionListNotifierProvider);
  }

  void _onDownloadDem(RegionEntity region) {
    _save(
      () => ref
          .read(tileRepositoryStatusProvider.notifier)
          .downloadDem(region.id),
    );
  }

  /// See [_onCancelVector] — the DEM-side mirror, fully independent.
  void _onCancelDem(RegionEntity region) {
    ref.read(tileRepositoryStatusProvider.notifier).cancelDem(region.id);
    ref.invalidate(regionListNotifierProvider);
  }

  /// SETUI-04/D-01: the DEM tile's own delete action — removes ONLY the DEM
  /// package, IMMEDIATELY — deliberately no confirm dialog, asymmetric with
  /// the Vector tile's cascading [_onDeleteRegion].
  void _onDeleteDemPackage(RegionEntity region) {
    _save(
      () => ref
          .read(tileRepositoryStatusProvider.notifier)
          .deleteDemPackage(region.id),
    );
  }

  /// D-02: the Vector tile's delete action cascades to remove the DEM
  /// package too (one on-device region has one storage directory), so it
  /// requires a confirm dialog first — mirrors
  /// `settings_categories_screen.dart`'s confirm-before-disable dialog shape
  /// (2 actions: Cancel/Delete, no middle "view detail" action).
  Future<void> _onDeleteRegion(RegionEntity region) async {
    final l10n = AppLocalizations.of(context)!;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.regions_delete_confirm_title(region.name)),
        content: Text(l10n.regions_delete_confirm_body),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(
              l10n.regions_delete_confirm_action,
              style: TextStyle(color: Colors.red.shade400),
            ),
          ),
        ],
      ),
    );

    if (confirmed != true) return;
    if (!mounted) return;

    await _save(
      () => ref.read(tileRepositoryStatusProvider.notifier).delete(region.id),
    );
  }
}
