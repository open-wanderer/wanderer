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

/// Validates a SERVER-issued record id ([Trail.id], [Waypoint.id], ...)
/// against a path-segment whitelist and returns it unchanged.
///
/// The counterpart to [localIdDirSegment] for ids this device did not mint.
/// Same discipline, different threat: a local id is ours and can be held to
/// its exact minted shape, while a record id arrives over the network from a
/// server that may be federated or compromised. An id containing `..` or a
/// separator, interpolated straight into a path, writes outside the directory
/// the caller intended.
///
/// Deliberately looser than PocketBase's own `^[a-z0-9]{15}$`: the security
/// property needed here is only "cannot escape the parent directory", and
/// pinning the exact PocketBase shape would turn any future id-format change
/// -- or any federated peer with a different convention -- into silently
/// broken downloads. `[A-Za-z0-9_-]` admits no `.`, no `/` and no `\`, so
/// `..` and every traversal spelling are rejected regardless.
String recordIdDirSegment(String id) {
  final pattern = RegExp(r'^[A-Za-z0-9_-]{1,64}$');
  if (!pattern.hasMatch(id)) {
    throw ArgumentError.value(
      id,
      'id',
      r'must match ^[A-Za-z0-9_-]{1,64}$ to be used as a path segment',
    );
  }
  return id;
}

/// Validates a filename destined to become the last segment of a path.
///
/// Unlike [recordIdDirSegment] this must admit dots, because real filenames
/// carry extensions -- so it rejects by enumeration instead: the empty
/// string, `.`, `..`, and anything still carrying a path separator. Intended
/// for names derived from a network-supplied URL, where `p.basename` alone is
/// not enough (`p.basename('..')` is `'..'`, and `Uri.path` percent-decodes,
/// so `%2e%2e` arrives as `..`).
String fileNameSegment(String name) {
  final isUnsafe =
      name.isEmpty ||
      name == '.' ||
      name == '..' ||
      name.contains('/') ||
      name.contains(r'\');
  if (isUnsafe) {
    throw ArgumentError.value(
      name,
      'name',
      'must be a plain filename, not empty, "." , ".." or separator-bearing',
    );
  }
  return name;
}
