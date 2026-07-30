/// Path-safety helpers for the app-wide region tile repository.
///
/// A catalog region `path` (e.g. `"canada.alberta.south"`) flows from a
/// fetched `GET /api/v1/regions` manifest into an on-disk directory name. Even
/// though the catalog is admin-controlled and already validated server-side
/// against the same allow-list (`regionIDPattern` in
/// `db/services/regions/config.go`), this file re-validates every value on the
/// client before it ever reaches `package:path` — defense in depth against a
/// corrupted local catalog row or a future non-catalog caller (RESEARCH
/// Pitfall 4 / T-23-01).
///
/// The value is NEVER string-concatenated into a path; every path builder
/// routes it through [assertValidRegionPath] first, then joins via
/// `package:path`.
library;

import 'package:path/path.dart' as p;

// NOTE: there is deliberately NO region-*id* validator here. A region's server
// record id is re-minted on every backend rebuild (see `RegionEntity.id`) and
// must never be used to build a path, key a row, or name a directory — keying
// on it is what silently detached every downloaded region. Validating an id
// would only make that mistake look sanctioned. Use the path allow-list below.

/// Byte-for-byte the same shape as the backend's region-`path` allow-list
/// (`regionIDPattern` in `db/services/regions/config.go`): lowercase
/// alphanumerics plus `_`, `.`, `'`, `-`; must start with an alphanumeric.
/// Unlike [regionIdPattern] this permits the `.` separator that a
/// materialized region path carries (e.g. `canada.alberta.south`). A path
/// containing `..` is additionally rejected as a path-traversal guard,
/// mirroring the backend's `IsValidRegionID`.
final RegExp regionPathPattern = RegExp(r"^[a-z0-9][a-z0-9_.'-]*$");

/// Whether [path] matches the region-path allow-list and carries no `..`.
bool isValidRegionPath(String path) =>
    regionPathPattern.hasMatch(path) && !path.contains('..');

/// Returns [path] unchanged if valid; otherwise throws [ArgumentError] — no
/// download URL may be built from a path that fails this check (T-23-01).
String assertValidRegionPath(String path) {
  if (!isValidRegionPath(path)) {
    throw ArgumentError.value(
      path,
      'regionPath',
      r"must match ^[a-z0-9][a-z0-9_.'-]*$ and contain no '..'",
    );
  }
  return path;
}

/// Build the on-disk storage directory for a region under [root]:
/// `<root>/regions/<path>/`. Throws [ArgumentError] if [path] is invalid.
///
/// Keyed on the region's materialized `path`, NOT on its server record id:
/// the backend re-mints that id whenever it rebuilds a region (see
/// `RegionEntity.id`), which used to orphan the whole directory and detach a
/// completed download. `path` is the same key the backend names its own
/// archive directories after (`RegionArchivePath` in
/// `db/services/regions`), so the two layouts now agree.
String regionStorageDir(String root, String path) {
  return p.join(root, 'regions', assertValidRegionPath(path));
}

/// Build the on-disk path for a region's vector archive:
/// `<root>/regions/<path>/vector.pmtiles`. Throws [ArgumentError] if [path]
/// is invalid.
String regionVectorPath(String root, String path) {
  return p.join(regionStorageDir(root, path), 'vector.pmtiles');
}

/// Build the on-disk path for a region's DEM archive:
/// `<root>/regions/<path>/dem.pmtiles`. Throws [ArgumentError] if [path] is
/// invalid.
String regionDemPath(String root, String path) {
  return p.join(regionStorageDir(root, path), 'dem.pmtiles');
}
