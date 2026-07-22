/// Path-safety helpers for the app-wide region tile repository.
///
/// A catalog region `id` (e.g. `"de-nrw"`) flows from a fetched `GET
/// /api/v1/regions` manifest into an on-disk directory name. Even though the
/// catalog is admin-controlled and already validated server-side against the
/// same allow-list (see `web/src/routes/api/v1/regions/[id]/download/+server.ts`'s
/// `RegionIdSchema`), this file re-validates every id on the client before it
/// ever reaches `package:path` — defense in depth against a corrupted local
/// catalog row or a future non-catalog caller (RESEARCH Pitfall 4 / T-23-01).
///
/// The id is NEVER string-concatenated into a path; every path builder routes
/// it through [assertValidRegionId] first, then joins via `package:path`.
library;

import 'package:path/path.dart' as p;

/// Byte-for-byte the same shape as the backend's `RegionIdSchema` regex —
/// an id the server accepts is the same id the app accepts. Lowercase
/// alphanumerics, `_`, and `-` only; must start with an alphanumeric.
final RegExp regionIdPattern = RegExp(r'^[a-z0-9][a-z0-9_-]*$');

/// Whether [id] matches the region-id allow-list.
bool isValidRegionId(String id) => regionIdPattern.hasMatch(id);

/// Returns [id] unchanged if valid; otherwise throws [ArgumentError] — no
/// path may be built from an id that fails this check.
String assertValidRegionId(String id) {
  if (!isValidRegionId(id)) {
    throw ArgumentError.value(
      id,
      'regionId',
      r'must match ^[a-z0-9][a-z0-9_-]*$',
    );
  }
  return id;
}

/// Build the on-disk storage directory for a region under [root]:
/// `<root>/regions/<id>/`. Throws [ArgumentError] if [id] is invalid.
String regionStorageDir(String root, String id) {
  return p.join(root, 'regions', assertValidRegionId(id));
}

/// Build the on-disk path for a region's vector archive:
/// `<root>/regions/<id>/vector.pmtiles`. Throws [ArgumentError] if [id] is
/// invalid.
String regionVectorPath(String root, String id) {
  return p.join(regionStorageDir(root, id), 'vector.pmtiles');
}

/// Build the on-disk path for a region's DEM archive:
/// `<root>/regions/<id>/dem.pmtiles`. Throws [ArgumentError] if [id] is
/// invalid.
String regionDemPath(String root, String id) {
  return p.join(regionStorageDir(root, id), 'dem.pmtiles');
}
