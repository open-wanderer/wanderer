/// Collision-free local identity for a trail or waypoint that has not (yet)
/// been assigned a server id.
///
/// ObjectBox's `id` columns on [TrailEntity] and [WaypointEntity] are both
/// `@Unique(onConflict: ConflictStrategy.replace)`. A local capture cannot
/// use `''` for that column — two unsynced trails saved back to back would
/// silently replace one another. A minted local id gives every unsynced row
/// a stable, collision-free identity that survives until the drain assigns
/// it a real server id.

/// Prefix for every minted local id.
///
/// PocketBase mints record ids from `[a-z0-9]{15}` with no hyphen, so no
/// server id can ever start with this prefix — the sentinel is unambiguous
/// without needing to inspect id length or character class.
const String kLocalIdPrefix = 'local-';

/// Monotonically incrementing counter used to disambiguate two [mintLocalId]
/// calls that land in the same microsecond. Removes the same-microsecond
/// collision risk flagged in RESEARCH.md assumption A1, without pulling in a
/// `uuid` dependency for something this narrow.
int _seq = 0;

/// Mints a new, permanent local identity.
///
/// Format: `local-<microsecondsSinceEpoch>-<seq>`. Callers must not attempt
/// to parse meaning out of the numeric parts beyond uniqueness — they exist
/// only to make the value collision-free.
String mintLocalId() {
  final id = '$kLocalIdPrefix${DateTime.now().microsecondsSinceEpoch}-$_seq';
  _seq++;
  return id;
}

/// Whether [id] is a local sentinel id rather than a server-issued one.
bool isLocalId(String id) => id.startsWith(kLocalIdPrefix);

/// Validates [localId] against the exact shape [mintLocalId] produces and
/// returns it unchanged.
///
/// This is the ONLY sanctioned way a local id may become a filesystem path
/// segment (ASVS V5 / T-36-01-02) — mirrors `map_cache_path.dart`'s
/// whitelist-and-reject discipline. Throws [ArgumentError] for anything that
/// does not match `^local-\d+-\d+$`, which rejects path traversal attempts
/// (`../escape`), embedded separators (`local-/etc`), and the empty string.
String localIdDirSegment(String localId) {
  final pattern = RegExp(r'^local-\d+-\d+$');
  if (!pattern.hasMatch(localId)) {
    throw ArgumentError.value(
      localId,
      'localId',
      r'must match ^local-\d+-\d+$',
    );
  }
  return localId;
}
