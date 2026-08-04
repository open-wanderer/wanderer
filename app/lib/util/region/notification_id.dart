/// Stable per-package notification id for offline map region downloads.
///
/// Vector and DEM downloads are fully independent — each owns its own
/// `CancelToken` and progress stream — so each needs its OWN notification
/// rather than sharing one that the two would overwrite in turn. Deriving the
/// id from `(regionPath, dem)` rather than handing out a counter means the
/// same package always maps to the same notification across its whole
/// lifecycle (progress -> success/error/dismiss) with no allocation table to
/// keep in sync, and a cancel-then-redownload correctly reuses the same slot.
///
/// Keyed on `path`, never the region record id: the backend re-mints that id
/// on every catalog rebuild (see `RegionEntity.id`), which would silently
/// strand the in-flight notification under an id nothing looks up again.
///
/// The `_idBase` offset guarantees the result can never collide with the
/// fixed id 42 that `DownloadNotificationService`'s trail methods own, so a
/// concurrent trail download and region download never overwrite each other.
/// The output stays well inside a signed 32-bit int, which is what both the
/// Android and iOS notification ids require.
///
/// Two different paths colliding on one id is possible in principle (any hash
/// into a finite range collides), and harmless in practice: the loser's
/// notification is overwritten, the downloads themselves are untouched.
library;

const int _idBase = 1000;
const int _idRange = 100000000;

int regionNotificationId(String regionPath, {required bool dem}) {
  final key = '$regionPath:${dem ? 'dem' : 'vector'}';
  // FNV-1a, masked to 30 bits each round so the running value can never reach
  // the 2^53 float boundary on the JS/web target or go negative on native.
  var hash = 0x811c9dc5;
  for (final unit in key.codeUnits) {
    hash = (hash ^ unit) & 0x3fffffff;
    hash = (hash * 0x01000193) & 0x3fffffff;
  }
  return _idBase + (hash % _idRange);
}
