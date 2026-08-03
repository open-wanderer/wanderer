import 'package:freezed_annotation/freezed_annotation.dart';

part 'region_download_state.freezed.dart';

/// Immutable per-region UI download state, keyed by region id in
/// `TileRepositoryStatus`'s state map (Settings/Regions screen subscribes to
/// this). UI-only, ephemeral state — the authoritative download lifecycle
/// lives on `RegionEntity.status`/`DownloadedTilePackageEntity`, so no JSON
/// serialization is needed here.
///
/// Vector and DEM downloads are fully independent (separate `CancelToken`s,
/// separate progress streams), so there is no single combined `status` —
/// each progress field's presence (non-null) IS that package's live
/// "downloading" signal; see `util/region/tile_status.dart`'s
/// `resolveVectorTileStatus`/`resolveDemTileStatus`.
@freezed
abstract class RegionDownloadState with _$RegionDownloadState {
  const factory RegionDownloadState({
    double? vectorProgress,
    double? demProgress,
  }) = _RegionDownloadState;
}
