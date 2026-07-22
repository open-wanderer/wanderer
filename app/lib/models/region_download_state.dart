import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/region_status.dart';

part 'region_download_state.freezed.dart';

/// Immutable per-region UI download state, keyed by region id in
/// `TileRepositoryStatus`'s state map (Phase 24's Settings/Regions screen
/// subscribes to this). UI-only, ephemeral state — the authoritative
/// download lifecycle lives on `RegionEntity.status`/`DownloadedTilePackageEntity`,
/// so no JSON serialization is needed here.
@freezed
abstract class RegionDownloadState with _$RegionDownloadState {
  const factory RegionDownloadState({
    @Default(RegionStatus.notDownloaded) RegionStatus status,
    double? vectorProgress,
    double? demProgress,
  }) = _RegionDownloadState;
}
