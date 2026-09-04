import 'package:objectbox/objectbox.dart';

/// Discriminates the kind of persisted "active session" row.
///
/// `nav` is trail-based turn-by-turn navigation (this plan). `rec` is
/// reserved for a future trail-less GPS-recording mode that reuses this same
/// table without a schema migration.
enum ActiveSessionType { nav, rec }

/// ObjectBox entity persisting the single active navigation/recording
/// session. There is at most one row at a time (see `active_navigation_store`
/// helpers) — a fresh session always clears any stale prior row.
@Entity()
class ActiveNavigationEntity {
  @Id()
  int obxId = 0;

  @Transient()
  ActiveSessionType sessionType = ActiveSessionType.nav;

  int get dbSessionType => sessionType.index;

  set dbSessionType(int value) {
    sessionType = (value >= 0 && value < ActiveSessionType.values.length)
        ? ActiveSessionType.values[value]
        : ActiveSessionType.nav;
  }

  /// Nullable — present only when [sessionType] == [ActiveSessionType.nav].
  /// Not `@Unique`: nullable fields cannot carry it, and uniqueness isn't
  /// needed since read()/clear() operate on "the single row" by construction.
  @Index()
  String? trailId;

  /// Nullable, trail-specific.
  bool? isOffline;

  /// Nullable, `rec`-specific — the Valhalla costing string (`'pedestrian'`
  /// or `'bicycle'`) chosen via `showTravelProfileSheet` at record start
  /// (`trail_source_select_screen.dart`'s `_openRecorder`). Persisted so a
  /// resumed-after-kill recording keeps the user's chosen profile for
  /// "Follow roads" instead of silently falling back to pedestrian. Null for
  /// `nav` sessions (costing there is derived from the trail's category
  /// instead) and for a `rec` row persisted before this field existed.
  String? recordingCosting;

  /// Nullable, trail-specific (no maneuvers exist for a future `rec` session).
  int? currentManeuverIndex;

  /// The traveled points (`NavigationState.breadcrumb`), encoded via
  /// `PolylineUtil.encode`/`.decode`. Shared/session-agnostic field — for a
  /// future `rec` session this would be the primary payload.
  String? breadcrumbPolyline;

  List<double>? elevations;
  List<int>? timestampsUtc;

  /// The session's own copy of its route, as `NavigateResponse` JSON.
  ///
  /// Nullable, `nav`-specific. Resuming used to re-read the route from
  /// `TrailEntity.navCacheJson`, which only exists for a trail the account has
  /// downloaded — so navigating an undownloaded trail left nothing to resume
  /// from, and the session was dropped on relaunch. The route is session
  /// state, not library state, so it is carried here instead and a resume no
  /// longer depends on what happens to be in the library.
  ///
  /// Null for `rec` sessions (no route) and for a `nav` row persisted before
  /// this field existed, which still falls back to the trail cache.
  String? navResponseJson;

  double distanceMeters;
  double elevationGainMeters;
  double elevationLossMeters;
  int currentElapsedSeconds;
  int pausedAccumSeconds;
  bool isPaused;

  @Property(type: PropertyType.dateUtc)
  DateTime updatedAtUtc;

  ActiveNavigationEntity({
    this.obxId = 0,
    this.sessionType = ActiveSessionType.nav,
    this.trailId,
    this.isOffline,
    this.recordingCosting,
    this.currentManeuverIndex,
    this.breadcrumbPolyline,
    this.elevations,
    this.timestampsUtc,
    this.navResponseJson,
    this.distanceMeters = 0,
    this.elevationGainMeters = 0,
    this.elevationLossMeters = 0,
    this.currentElapsedSeconds = 0,
    this.pausedAccumSeconds = 0,
    this.isPaused = false,
    required this.updatedAtUtc,
  });
}
