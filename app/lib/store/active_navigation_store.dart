import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/objectbox.g.dart';

/// Best-effort persistence helpers for the single "active session" row.
///
/// Generic over [ActiveSessionType] — none of these functions take a
/// `trailId` parameter; they all operate on "the single active row" by
/// construction (callers always clear before starting a fresh session).
/// Every function swallows errors, mirroring `_recacheNav` in
/// `launch_navigation.dart` — a persistence fault must never crash or
/// interrupt an in-progress navigation/recording session.

/// Saves [entity] and returns the assigned/kept `obxId`.
///
/// On any error, returns `entity.obxId` unchanged (a failed best-effort write
/// must not corrupt the caller's tracked row id) rather than rethrowing.
int save(Store store, ActiveNavigationEntity entity) {
  try {
    return store.box<ActiveNavigationEntity>().put(entity);
  } catch (_) {
    return entity.obxId;
  }
}

/// Reads the single active session row, or null if none exists.
///
/// There should be at most one row by construction — callers always clear
/// before creating a fresh one.
ActiveNavigationEntity? read(Store store) {
  try {
    final query = store.box<ActiveNavigationEntity>().query().build();
    try {
      return query.findFirst();
    } finally {
      query.close();
    }
  } catch (_) {
    return null;
  }
}

/// Clears the active session row (if any).
void clear(Store store) {
  try {
    store.box<ActiveNavigationEntity>().removeAll();
  } catch (_) {}
}
