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
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/byte_format_util.dart';
import 'package:wanderer/util/region_disk_usage_util.dart';

/// SETUI-01..06: the "Offline Maps/Regions" Settings screen — a flat,
/// name-searchable, A-Z region list backed by Plan 01's synchronous
/// `regionListNotifierProvider`, with a live disk-usage summary and
/// correctly-disabled `building`/`error` catalog rows (D-09). Per-region
/// download/pause/resume/delete/DEM-toggle actions are completed in Task 2's
/// `_buildActiveRow`.
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
                _buildDiskUsageSummary(l10n, regions),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                  child: WandererSearchBar(
                    hintText: l10n.regions_search_hint,
                    onChanged: (value) =>
                        setState(() => _searchQuery = value),
                  ),
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
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 24, 16, 0),
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
                l10n.regions_disk_usage_summary(
                  formatBytes(totalBytes),
                  count,
                ),
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
  /// or mid-flight (downloading/paused — a partial `.part` file still
  /// occupies real disk space, D-06).
  bool _hasAnyDiskUsage(RegionEntity region) {
    bool packageOccupiesDisk(PackageStatus? status) =>
        status == PackageStatus.downloaded ||
        status == PackageStatus.downloading ||
        status == PackageStatus.paused;
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

  /// Reached only when `catalogStatus == CatalogStatus.ready`. Task 1's
  /// stub renders the name + vector/DEM size breakdown only (SETUI-02's
  /// before-download requirement); Task 2 completes this with the six
  /// `RegionStatus` states, combined progress, DEM toggle, and
  /// delete/retry/update actions.
  Widget _buildActiveRow(RegionEntity region) {
    return Padding(
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
                Text(
                  region.demSize != null
                      ? '${formatBytes(region.vectorSize ?? 0)} vector · '
                            '${formatBytes(region.demSize!)} DEM'
                      : '${formatBytes(region.vectorSize ?? 0)} vector',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
