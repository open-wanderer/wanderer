class ObjectArrayDiff<T> {
  final List<T> added;
  final List<T> updated;
  final List<T> deleted;

  const ObjectArrayDiff({
    required this.added,
    required this.updated,
    required this.deleted,
  });
}

/// Diffs [oldArray] against [newArray] by id (via [idOf]), classifying each
/// entry in [newArray] as added (no matching old id) or updated (matching id,
/// unequal value), and each entry in [oldArray] with no match in [newArray]
/// as deleted.
ObjectArrayDiff<T> compareObjectArrays<T>(
  List<T> oldArray,
  List<T> newArray, {
  required String Function(T) idOf,
}) {
  final oldById = {for (final item in oldArray) idOf(item): item};
  final newIds = newArray.map(idOf).toSet();

  final added = <T>[];
  final updated = <T>[];

  for (final item in newArray) {
    final old = oldById[idOf(item)];
    if (old == null) {
      added.add(item);
    } else if (old != item) {
      updated.add(item);
    }
  }

  final deleted = oldArray
      .where((item) => !newIds.contains(idOf(item)))
      .toList();

  return ObjectArrayDiff(added: added, updated: updated, deleted: deleted);
}
