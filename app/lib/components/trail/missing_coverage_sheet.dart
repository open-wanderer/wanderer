import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/region/tile_repository_provider.dart';
import 'package:wanderer/util/byte_format_util.dart';
import 'package:wanderer/util/region_tile_status_util.dart';

/// The user's checkbox selection from [showMissingCoverageSheet] — the
/// regions whose Vector package the user checked, and the regions whose DEM
/// package the user checked. Both lists can be empty (D-08's trail-only
/// escape hatch). Plain class (not freezed): no build_runner run needed for
/// this self-contained contract.
class MissingCoverageSelection {
  final List<RegionEntity> vectorRegions;
  final List<RegionEntity> demRegions;

  const MissingCoverageSelection({
    required this.vectorRegions,
    required this.demRegions,
  });
}

/// GUARD-02/GUARD-03's user surface: a bottom modal sheet listing every
/// region overlapping a trail's bbox that isn't yet `downloaded`/
/// `updateAvailable`, with independent Vector/DEM checkboxes per region
/// (D-06/D-07) and a single always-enabled Download button (D-08). Mirrors
/// `track_save_options_sheet.dart`'s shape; the region rows mirror
/// `settings_offline_regions_screen.dart`'s styling.
///
/// Returns `null` when dismissed (swipe/tap-outside) — the caller must abort
/// the download entirely in that case. Returns a non-null
/// [MissingCoverageSelection] (even with both lists empty) when the user
/// taps Download.
///
/// This sheet is purely presentational: it never triggers a region package
/// download itself or imports the trail-download provider — starting the
/// selected downloads is the caller's job (Plan 03).
Future<MissingCoverageSelection?> showMissingCoverageSheet(
  BuildContext context,
  Trail trail,
  List<RegionEntity> missingRegions,
) {
  return showModalBottomSheet<MissingCoverageSelection>(
    context: context,
    isDismissible: true,
    enableDrag: true,
    isScrollControlled: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => _MissingCoverageSheetContent(
      trail: trail,
      missingRegions: missingRegions,
    ),
  );
}

/// Per-region live state, resolved once per [build] from
/// [tileRepositoryStatusProvider] via `resolveVectorTileStatus`/
/// `resolveDemTileStatus` — captures whether each package is already
/// downloading/erroring right now, independent of local checkbox state.
class _RegionRowState {
  final RegionEntity region;
  final RegionStatus vectorStatus;
  final PackageStatus? demStatus; // null when the region has no DEM archive

  const _RegionRowState({
    required this.region,
    required this.vectorStatus,
    required this.demStatus,
  });
}

class _MissingCoverageSheetContent extends ConsumerStatefulWidget {
  final Trail trail;
  final List<RegionEntity> missingRegions;

  const _MissingCoverageSheetContent({
    required this.trail,
    required this.missingRegions,
  });

  @override
  ConsumerState<_MissingCoverageSheetContent> createState() =>
      _MissingCoverageSheetContentState();
}

class _MissingCoverageSheetContentState
    extends ConsumerState<_MissingCoverageSheetContent> {
  // D-07: Vector defaults checked, DEM defaults unchecked, for every listed
  // region. Toggling only mutates this local state (D-06) — no download
  // starts from this widget.
  late final Map<String, bool> _vectorChecked = {
    for (final region in widget.missingRegions) region.id: true,
  };
  late final Map<String, bool> _demChecked = {
    for (final region in widget.missingRegions) region.id: false,
  };

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final downloadStates = ref.watch(tileRepositoryStatusProvider);

    final rows = widget.missingRegions.map((region) {
      final downloadState = downloadStates[region.id];
      final vectorStatus = resolveVectorTileStatus(
        region.status,
        downloadState?.vectorProgress,
      );
      final demStatus = region.demUrl != null
          ? resolveDemTileStatus(
              region.demPackage.target?.status ?? PackageStatus.notDownloaded,
              downloadState?.demProgress,
            )
          : null;
      return _RegionRowState(
        region: region,
        vectorStatus: vectorStatus,
        demStatus: demStatus,
      );
    }).toList();

    var combinedBytes = 0;
    var selectedCount = 0;
    for (final row in rows) {
      final vectorSelected =
          row.vectorStatus != RegionStatus.downloading &&
          (_vectorChecked[row.region.id] ?? false);
      final demSelected =
          row.demStatus != null &&
          row.demStatus != PackageStatus.downloading &&
          (_demChecked[row.region.id] ?? false);
      if (vectorSelected) combinedBytes += row.region.vectorSize ?? 0;
      if (demSelected) combinedBytes += row.region.demSize ?? 0;
      if (vectorSelected || demSelected) selectedCount++;
    }

    final summaryText = selectedCount == 0
        ? 'Downloading trail only'
        : '$selectedCount region${selectedCount == 1 ? '' : 's'} selected '
              '· ${formatBytes(combinedBytes)}';

    return SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        16,
        16,
        16,
        24 + MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8.0),
            child: Center(
              child: Container(
                width: 30,
                height: 5,
                decoration: BoxDecoration(
                  borderRadius: const BorderRadius.all(Radius.circular(24)),
                  color: theme.colorScheme.secondaryContainer,
                ),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8.0),
            child: Text(
              'Missing offline map coverage',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          const SizedBox(height: 4),
          Text(
            "${widget.trail.name} isn't fully covered by your downloaded "
            'regions yet. Select any regions below to download them '
            'together with the trail — or tap Download to get the '
            'trail without them.',
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 16),
          for (final row in rows) ...[
            _buildRegionCard(context, row, l10n),
            const SizedBox(height: 8),
          ],
          Text(summaryText, style: theme.textTheme.bodySmall),
          const SizedBox(height: 8),
          FilledButton(
            onPressed: () {
              final vectorRegions = rows
                  .where(
                    (row) =>
                        row.vectorStatus != RegionStatus.downloading &&
                        (_vectorChecked[row.region.id] ?? false),
                  )
                  .map((row) => row.region)
                  .toList();
              final demRegions = rows
                  .where(
                    (row) =>
                        row.demStatus != null &&
                        row.demStatus != PackageStatus.downloading &&
                        (_demChecked[row.region.id] ?? false),
                  )
                  .map((row) => row.region)
                  .toList();
              Navigator.pop(
                context,
                MissingCoverageSelection(
                  vectorRegions: vectorRegions,
                  demRegions: demRegions,
                ),
              );
            },
            child: Text(l10n.download),
          ),
        ],
      ),
    );
  }

  /// A flat, bordered "Terrain Log" card per missing region — matches
  /// `_ToggleCard`'s `elevation: 0` + outline-border doctrine and
  /// `settings_offline_regions_screen.dart`'s region-row shape.
  Widget _buildRegionCard(
    BuildContext context,
    _RegionRowState row,
    AppLocalizations l10n,
  ) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outline),
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: Text(
              row.region.name,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.bodyLarge?.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          _buildVectorRow(context, row, l10n),
          if (row.demStatus != null) _buildDemRow(context, row, l10n),
          const SizedBox(height: 4),
        ],
      ),
    );
  }

  Widget _buildVectorRow(
    BuildContext context,
    _RegionRowState row,
    AppLocalizations l10n,
  ) {
    if (row.vectorStatus == RegionStatus.downloading) {
      return Padding(
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.regions_vector_tile_title),
            const SizedBox(height: 4),
            const LinearProgressIndicator(),
            const SizedBox(height: 4),
            Text('Downloading…', style: Theme.of(context).textTheme.bodySmall),
          ],
        ),
      );
    }

    final isError = row.vectorStatus == RegionStatus.error;
    return CheckboxListTile(
      dense: true,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16),
      title: Text(l10n.regions_vector_tile_title),
      subtitle: isError
          ? Text(
              l10n.regions_download_failed,
              style: const TextStyle(color: Colors.redAccent),
            )
          : Text(formatBytes(row.region.vectorSize ?? 0)),
      value: _vectorChecked[row.region.id] ?? true,
      onChanged: (checked) => setState(() {
        _vectorChecked[row.region.id] = checked ?? false;
      }),
    );
  }

  Widget _buildDemRow(
    BuildContext context,
    _RegionRowState row,
    AppLocalizations l10n,
  ) {
    if (row.demStatus == PackageStatus.downloading) {
      return Padding(
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.regions_dem_tile_title),
            const SizedBox(height: 4),
            const LinearProgressIndicator(),
            const SizedBox(height: 4),
            Text('Downloading…', style: Theme.of(context).textTheme.bodySmall),
          ],
        ),
      );
    }

    final isError = row.demStatus == PackageStatus.error;
    return CheckboxListTile(
      dense: true,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16),
      title: Text(l10n.regions_dem_tile_title),
      subtitle: isError
          ? Text(
              l10n.regions_download_failed,
              style: const TextStyle(color: Colors.redAccent),
            )
          : Text(formatBytes(row.region.demSize ?? 0)),
      value: _demChecked[row.region.id] ?? false,
      onChanged: (checked) => setState(() {
        _demChecked[row.region.id] = checked ?? false;
      }),
    );
  }
}
