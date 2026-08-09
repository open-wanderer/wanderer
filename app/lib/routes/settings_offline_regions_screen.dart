import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:skeletonizer/skeletonizer.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/region_hierarchy_row.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/models/region_tree_node.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/provider/region/tile_repository_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/format.dart';
import 'package:wanderer/util/region/disk_usage.dart';
import 'package:wanderer/util/region/tile_status.dart';
import 'package:wanderer/util/region/tree.dart';

/// One flattened, visible tree row: a node plus its indentation depth. Emitted
/// by `flattenVisible` and consumed by the region ListView.
typedef _RegionRow = ({RegionTreeNode node, int depth});

/// The "Offline Maps/Regions" Settings screen — a flat,
/// name-searchable, A-Z region list backed by `regionListNotifierProvider`,
/// with a live disk-usage summary and correctly-disabled `building`/`error`
/// catalog rows. Each ready region renders as two independent
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

  /// Set only for the genuine fresh-install edge case: zero cached regions
  /// AND the initial catalog fetch failed. Every
  /// other failure (a non-empty cached snapshot) surfaces a toast instead —
  /// see `_refreshCatalog`.
  Object? _freshInstallError;
  StackTrace? _freshInstallStackTrace;

  /// Tree shape, built once per genuine hierarchy fetch inside
  /// `_refreshCatalog` — NEVER rebuilt inside `build()`. `null`
  /// means no successful hierarchy fetch has completed this session. Combined
  /// with `_hierarchyLoadInFlight` it distinguishes "still loading" (skeleton)
  /// from "fetch resolved with no tree" (the offline empty state).
  List<RegionTreeNode>? _treeRoots;

  /// True while the initial `_refreshCatalog` round-trip is in flight. Seeded
  /// `true` because `initState` always fires the fetch — seeding here (rather
  /// than a `setState` at the top of `_refreshCatalog`) avoids calling
  /// `setState` during `initState`. Cleared at every resolution point of the
  /// fetch (success or either failure branch). While true AND `_treeRoots` is
  /// still `null`, `build()` renders a skeletonized tree instead of the
  /// offline empty state, so the "no connection" message no longer flashes on
  /// every screen open before the fetch resolves.
  bool _hierarchyLoadInFlight = true;

  /// Expand state, keyed by group node id. Reseeded (via
  /// `computeDefaultExpanded`) only when `_expandedSeeded` is false — i.e.
  /// right after a fresh `_treeRoots` assignment — so a user's manual
  /// expand/collapse survives unrelated rebuilds (download/progress ticks).
  Set<String> _expandedIds = {};
  bool _expandedSeeded = false;

  @override
  void initState() {
    super.initState();

    // The catalog only exists on the backend, so offline there is nothing to
    // fetch. Skipping the doomed round-trip suppresses the spurious error
    // toast `_refreshCatalog` raises whenever cached regions exist, and lets
    // the offline empty state render immediately rather than after a skeleton
    // waits out a request that cannot succeed. Assigning the field directly
    // is correct here — build has not run yet, so no setState is needed.
    if (!ref.read(onlineStatusProvider)) {
      _hierarchyLoadInFlight = false;
      return;
    }

    // Fire-and-forget — never blocks the list on a
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
      final hierarchyRows = await ref
          .read(regionRepositoryProvider)
          .refreshCatalogAndFetchHierarchy();
      if (!mounted) return;
      ref.invalidate(regionListNotifierProvider);
      setState(() {
        // The backend already prunes to enabled leaves + ancestor groups
        // (`?enabled=true`), so the tree is rendered as-received — no local
        // pruning pass.
        _treeRoots = buildRegionTree(hierarchyRows);
        // Reseed default-expand against the new tree shape on the next build.
        _expandedSeeded = false;
        _hierarchyLoadInFlight = false;
      });
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
        setState(() => _hierarchyLoadInFlight = false);
        return;
      }
      setState(() {
        _freshInstallError = e;
        _freshInstallStackTrace = st;
        _hierarchyLoadInFlight = false;
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

    // Render-time join — cheap on every build; the
    // tree shape itself (`_treeRoots`) is NEVER rebuilt here, only in
    // `_refreshCatalog` after a genuine fetch.
    //
    // Joined on `path`, never on the catalog's node id: the backend re-mints
    // region record ids on every rebuild (see `RegionEntity.id`), so an
    // id-keyed join silently stopped finding the downloaded row and rendered
    // every region as "not downloaded" while the disk-usage summary — which
    // iterates the same rows directly — still counted its bytes.
    final byPath = {for (final r in regions) r.path: r};

    if (_treeRoots != null && !_expandedSeeded) {
      _expandedIds = computeDefaultExpanded(_treeRoots!, (leaf) {
        final entity = byPath[leaf.path];
        if (entity == null) return false;
        final vectorStatus = entity.status;
        final hasVectorDownloadOrInProgress =
            vectorStatus == RegionStatus.downloaded ||
            vectorStatus == RegionStatus.downloading ||
            vectorStatus == RegionStatus.updateAvailable;
        final demStatus = entity.demPackage.target?.status;
        final hasDemDownloadOrInProgress =
            demStatus == PackageStatus.downloaded ||
            demStatus == PackageStatus.downloading;
        return hasVectorDownloadOrInProgress || hasDemDownloadOrInProgress;
      });
      _expandedSeeded = true;
    }

    final trimmedQuery = _searchQuery.trim();
    final filterMatches = trimmedQuery.isEmpty || _treeRoots == null
        ? null
        : computeFilterMatches(_treeRoots!, trimmedQuery);
    final visibleRows = _treeRoots == null
        ? const <_RegionRow>[]
        : flattenVisible(_treeRoots!, _expandedIds, filterMatches);

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
        title: WandererSearchBar(
          hintText: l10n.regions_search_hint,
          onChanged: (value) => setState(() => _searchQuery = value),
        ),
      ),
      body: showFullScreenError
          ? _buildFreshInstallError()
          : Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 24, 16, 24),
                  child: _buildDiskUsageSummary(l10n, regions),
                ),
                Expanded(
                  child: _buildRegionArea(
                    l10n,
                    visibleRows,
                    byPath,
                    hasSearchQuery: trimmedQuery.isNotEmpty,
                  ),
                ),
              ],
            ),
    );
  }

  /// Fresh-install edge case only (see `showFullScreenError` in `build`): zero
  /// cached regions AND the initial catalog fetch failed. Reuses
  /// AsyncLoader/WandererError verbatim per UI-SPEC's copy table.
  Widget _buildFreshInstallError() {
    return AsyncLoader<List<RegionEntity>>(
      asyncValue: AsyncValue<List<RegionEntity>>.error(
        _freshInstallError!,
        _freshInstallStackTrace ?? StackTrace.current,
      ),
      mockData: const [],
      builder: (_) => const SizedBox.shrink(),
    );
  }

  /// The scrollable area below the disk-usage summary. Resolves, in order, the
  /// three non-list states before falling through to the region list:
  ///   1. No hierarchy fetched yet (`_treeRoots == null`) — still loading
  ///      (skeleton) vs. resolved with no tree (offline empty state).
  ///   2. Tree fetched but nothing visible — empty catalog vs. no search match.
  ///   3. Otherwise, the region list itself.
  Widget _buildRegionArea(
    AppLocalizations l10n,
    List<_RegionRow> visibleRows,
    Map<String, RegionEntity> byPath, {
    required bool hasSearchQuery,
  }) {
    if (_treeRoots == null) {
      // `_hierarchyLoadInFlight` disambiguates "still loading" from "fetch
      // resolved with no tree": both share the `_treeRoots == null` condition,
      // so without this gate the offline "no connection" message flashed on
      // every screen open before the initial load resolved.
      if (_hierarchyLoadInFlight) return _buildSkeleton();
      return _buildEmptyState(
        title: l10n.regions_offline_unavailable_title,
        body: l10n.regions_offline_unavailable_body,
        // font_awesome_flutter's free icon set has no `wifiSlash` glyph
        // (Pro-only in upstream Font Awesome 6) — `linkSlash` ("broken link")
        // is the closest already-vetted disconnected/offline glyph in this
        // dependency (Rule 1 fix).
        icon: FontAwesomeIcons.linkSlash,
      );
    }

    if (visibleRows.isEmpty) {
      if (hasSearchQuery) {
        return _buildEmptyState(
          title: l10n.regions_empty_search_title,
          body: l10n.regions_empty_search_body,
        );
      }
      return _buildEmptyState(
        title: l10n.regions_empty_catalog_title,
        body: l10n.regions_empty_catalog_body,
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.only(bottom: 16),
      itemCount: visibleRows.length,
      itemBuilder: (context, index) =>
          _buildTreeRow(visibleRows[index], byPath),
    );
  }

  /// A single flattened tree row: either a group header or a leaf region card.
  Widget _buildTreeRow(_RegionRow row, Map<String, RegionEntity> byPath) {
    if (row.node.kind != RegionNodeKind.leaf) {
      return KeyedSubtree(
        key: ValueKey(row.node.id),
        child: _buildGroupRow(row.node, row.depth),
      );
    }

    final entity = byPath[row.node.path];
    if (entity == null) {
      // Pitfall 6: a hierarchy row with no matching RegionEntity (e.g. a
      // mid-refresh race) must never crash the render.
      return SizedBox.shrink(key: ValueKey(row.node.id));
    }
    return Padding(
      key: ValueKey(row.node.id),
      padding: EdgeInsets.only(left: _indentFor(row.depth), bottom: 16),
      child: _buildRegionRow(entity),
    );
  }

  /// Depth-based left indent for a tree row (UI-SPEC Spacing): 16px base,
  /// +16px per level, capped at depth 4 (80px total) — CoMaps hierarchy can
  /// nest several levels deep, and uncapped indentation would eventually
  /// leave no room for the row label on a narrow phone.
  double _indentFor(int depth) => 16.0 + (depth.clamp(0, 4) * 16.0);

  /// A group row: the entire row width toggles expand/collapse
  /// on tap (UI-SPEC Interaction Contract), with a chevron that swaps
  /// instantly (no rotation animation) and a `Semantics` label for screen
  /// readers. Unlike a leaf's bordered card, a group row has NO card/border —
  /// it sits directly on the screen background so it reads as structural
  /// chrome, not a peer of the download-action leaf cards.
  Widget _buildGroupRow(RegionTreeNode node, int depth) {
    final l10n = AppLocalizations.of(context)!;
    final isExpanded = _expandedIds.contains(node.id);
    final mutedColor = Theme.of(
      context,
    ).colorScheme.onSurface.withValues(alpha: 0.6);

    return Semantics(
      button: true,
      label: isExpanded
          ? l10n.regions_group_collapse_label(node.name)
          : l10n.regions_group_expand_label(node.name),
      child: InkWell(
        onTap: () {
          setState(() {
            final next = {..._expandedIds};
            if (isExpanded) {
              next.remove(node.id);
            } else {
              next.add(node.id);
            }
            _expandedIds = next;
          });
        },
        child: ConstrainedBox(
          constraints: const BoxConstraints(minHeight: 44),
          child: Padding(
            padding: EdgeInsets.only(left: _indentFor(depth), right: 16),
            child: Row(
              children: [
                FaIcon(
                  isExpanded
                      ? FontAwesomeIcons.angleDown
                      : FontAwesomeIcons.angleRight,
                  size: 16,
                  color: mutedColor,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    node.name,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodyLarge!.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// Disk-usage summary card: a Display-role headline byte
  /// figure over a Label-role sub-text stating how many regions contribute
  /// to it. `totalRegionDiskUsageBytes` reads real on-disk bytes (including
  /// `.part` partial files) — see `util/region/disk_usage.dart`.
  Widget _buildDiskUsageSummary(
    AppLocalizations l10n,
    List<RegionEntity> regions,
  ) {
    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).colorScheme.outline),
        borderRadius: BorderRadius.all(Radius.circular(8)),
        color: Theme.of(context).brightness == Brightness.light
            ? Colors.grey.shade100
            : Theme.of(context).colorScheme.secondaryContainer,
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
  /// real disk space.
  bool _hasAnyDiskUsage(RegionEntity region) {
    bool packageOccupiesDisk(PackageStatus? status) =>
        status == PackageStatus.downloaded ||
        status == PackageStatus.downloading;
    return packageOccupiesDisk(region.vectorPackage.target?.status) ||
        packageOccupiesDisk(region.demPackage.target?.status);
  }

  Widget _buildEmptyState({
    required String title,
    required String body,
    FaIconData? icon,
  }) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (icon != null) ...[
              FaIcon(
                icon,
                size: 32,
                color: Theme.of(
                  context,
                ).colorScheme.onSurface.withValues(alpha: 0.5),
              ),
              const SizedBox(height: 16),
            ],
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

  /// Shimmer placeholder shown while the initial catalog + hierarchy fetch is
  /// in flight (`_hierarchyLoadInFlight`). `skeletonizer` paints these fake
  /// group headers and leaf cards as animated "bones", so the screen reads as
  /// loading rather than flashing the offline empty state (both share the
  /// `_treeRoots == null` condition). The fake rows deliberately mirror the
  /// real `_buildGroupRow` / `_buildActiveRow` shapes so the bones land where
  /// the real content will — the literal placeholder strings are never visible
  /// (Skeletonizer replaces them), they only size the bones.
  Widget _buildSkeleton() {
    final outline = Theme.of(context).colorScheme.outline;
    return Skeletonizer(
      enabled: true,
      child: ListView.builder(
        padding: const EdgeInsets.only(bottom: 16),
        itemCount: 4,
        itemBuilder: (context, groupIndex) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Group header placeholder — mirrors `_buildGroupRow` (depth 0).
              Padding(
                padding: EdgeInsets.only(left: _indentFor(0), right: 16),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(minHeight: 44),
                  child: Row(
                    children: [
                      const FaIcon(FontAwesomeIcons.angleDown, size: 16),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          'Region group placeholder',
                          style: Theme.of(context).textTheme.bodyLarge!
                              .copyWith(fontWeight: FontWeight.bold),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              // Two leaf-card placeholders — mirror `_buildActiveRow` (depth 1).
              for (var i = 0; i < 2; i++)
                Padding(
                  padding: EdgeInsets.only(
                    left: _indentFor(1),
                    right: 16,
                    bottom: 16,
                  ),
                  child: Container(
                    decoration: BoxDecoration(
                      border: Border.all(color: outline),
                      borderRadius: const BorderRadius.all(Radius.circular(8)),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Padding(
                          padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                          child: Text(
                            'Region name placeholder',
                            style: Theme.of(context).textTheme.bodyLarge!
                                .copyWith(fontWeight: FontWeight.bold),
                          ),
                        ),
                        const ListTile(
                          dense: true,
                          minTileHeight: 12,
                          contentPadding: EdgeInsets.symmetric(horizontal: 16),
                          title: Text('Vector'),
                          subtitle: Text('00.0 MB'),
                          trailing: FaIcon(FontAwesomeIcons.download, size: 16),
                        ),
                        const ListTile(
                          dense: true,
                          minTileHeight: 12,
                          contentPadding: EdgeInsets.symmetric(horizontal: 16),
                          title: Text('Elevation data'),
                          subtitle: Text('00.0 MB'),
                          trailing: FaIcon(FontAwesomeIcons.download, size: 16),
                        ),
                        const SizedBox(height: 4),
                      ],
                    ),
                  ),
                ),
            ],
          );
        },
      ),
    );
  }

  /// CATALOG-STATUS PRECEDENCE GATE — must run
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
  /// deletable. Deleting the Vector tile cascades to delete DEM
  /// too; deleting the DEM tile removes only the DEM package.
  Widget _buildActiveRow(RegionEntity region) {
    final l10n = AppLocalizations.of(context)!;
    final downloadState = ref.watch(tileRepositoryStatusProvider)[region.path];
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
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  region.name,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(
                    context,
                  ).textTheme.bodyLarge!.copyWith(fontWeight: FontWeight.bold),
                ),
                IconButton(
                  style: IconButton.styleFrom(
                    backgroundColor: Colors.transparent,
                  ),
                  icon: FaIcon(
                    FontAwesomeIcons.mapLocationDot,
                    size: 18,
                    color: Theme.of(context).colorScheme.onSurface,
                  ),
                  onPressed: () => context.push(
                    '/settings/region/map?path=${Uri.encodeComponent(region.path)}',
                  ),
                ),
              ],
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
              vectorAvailable:
                  vectorStatus == RegionStatus.downloaded ||
                  vectorStatus == RegionStatus.updateAvailable,
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
      minTileHeight: 12,
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
              icon: const FaIcon(FontAwesomeIcons.arrowsRotate, size: 16),
              color: accentColor,
              tooltip: l10n.regions_update_action,
              onPressed: () => _onDownloadVector(region),
            ),
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.trash, size: 16),
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
              icon: const FaIcon(FontAwesomeIcons.arrowsRotate, size: 16),
              color: accentColor,
              tooltip: l10n.regions_retry,
              onPressed: () => _onDownloadVector(region),
            ),
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.trash, size: 16),
              color: Colors.redAccent,
              onPressed: () => _onDeleteRegion(region),
            ),
          ],
        );
    }
  }

  /// [vectorAvailable] gates the DEM tile on the Vector package being
  /// `downloaded`/`updateAvailable` — hillshading over a basemap that
  /// doesn't exist yet is meaningless, so the DEM download action stays
  /// disabled (with an explanatory subtitle) until Vector is present.
  Widget _buildDemTile(
    RegionEntity region,
    double? liveProgress,
    AppLocalizations l10n,
    Color accentColor, {
    required bool vectorAvailable,
  }) {
    final persisted =
        region.demPackage.target?.status ?? PackageStatus.notDownloaded;
    final status = resolveDemTileStatus(persisted, liveProgress);
    final isDownloading = status == PackageStatus.downloading;
    final isDone = status == PackageStatus.downloaded;
    final isError = status == PackageStatus.error;
    final isLocked = !vectorAvailable && !isDownloading && !isDone;
    final onDiskBytes = region.demPackage.target?.sizeBytesOnDisk;

    final String subtitleText;
    final Color? subtitleColor;
    if (isLocked) {
      subtitleText = l10n.regions_dem_locked_subtitle;
      subtitleColor = null;
    } else if (isError) {
      subtitleText = l10n.regions_download_failed;
      subtitleColor = Colors.redAccent;
    } else {
      subtitleText = formatBytes(
        (isDone ? onDiskBytes : null) ?? region.demSize ?? 0,
      );
      subtitleColor = null;
    }

    return ListTile(
      dense: true,
      minTileHeight: 12,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16),
      leading: _tileLeadingIcon(done: isDone, error: isError),
      title: Text(l10n.regions_dem_tile_title),
      subtitle: _tileSubtitle(
        downloading: isDownloading,
        progress: liveProgress,
        text: subtitleText,
        textColor: subtitleColor,
        accentColor: accentColor,
      ),
      trailing: isLocked
          ? IconButton(
              icon: const FaIcon(FontAwesomeIcons.download, size: 16),
              onPressed: null,
            )
          : _buildDemTrailing(region, status, l10n, accentColor),
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
  /// package is downloading (per-tile, not a combined average,
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
  /// `settings_categories_screen.dart`'s `_save` wrapper.
  ///
  /// Deliberately does NOT invalidate `regionListNotifierProvider` itself.
  /// That refresh is mandatory after every mutation (ObjectBox
  /// `ToOne.target` caches per-instance after first read), but a
  /// `mounted`-guarded invalidate here silently skipped it whenever the user
  /// left the screen mid-download, so a finished download came back rendering
  /// as `notDownloaded`. `TileRepositoryStatus` (keepAlive) now owns the
  /// refresh at every terminal point, screen or no screen.
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
    }
  }

  void _onDownloadVector(RegionEntity region) {
    _save(
      () => ref
          .read(tileRepositoryStatusProvider.notifier)
          .downloadVector(region.path),
    );
  }

  /// Cancel is synchronous: `cancelVector` clears the ephemeral progress
  /// (which Riverpod immediately re-renders from — the tile flips back to a
  /// Download button at once) and refreshes the region list itself, so the
  /// disk-usage summary and persisted reads follow. No pause/resume: the
  /// manager deletes the `.part` file, so a later download restarts from
  /// byte 0.
  void _onCancelVector(RegionEntity region) {
    ref.read(tileRepositoryStatusProvider.notifier).cancelVector(region.path);
  }

  void _onDownloadDem(RegionEntity region) {
    _save(
      () => ref
          .read(tileRepositoryStatusProvider.notifier)
          .downloadDem(region.path),
    );
  }

  /// See [_onCancelVector] — the DEM-side mirror, fully independent.
  void _onCancelDem(RegionEntity region) {
    ref.read(tileRepositoryStatusProvider.notifier).cancelDem(region.path);
  }

  /// The DEM tile's own delete action — removes ONLY the DEM
  /// package, IMMEDIATELY — deliberately no confirm dialog, asymmetric with
  /// the Vector tile's cascading [_onDeleteRegion].
  void _onDeleteDemPackage(RegionEntity region) {
    _save(
      () => ref
          .read(tileRepositoryStatusProvider.notifier)
          .deleteDemPackage(region.path),
    );
  }

  /// The Vector tile's delete action cascades to remove the DEM
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
      () => ref.read(tileRepositoryStatusProvider.notifier).delete(region.path),
    );
  }
}
