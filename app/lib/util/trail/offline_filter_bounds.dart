/// Derives offline `TrailFilter` slider bounds from the trails actually on
/// this device, instead of a fixed compile-time constant.
///
/// Offline, the only trails that can possibly match a search are the ones on
/// this device -- so bounds taken from those rows fit the real search space
/// exactly, and a hiker is never handed a 500 km slider when nothing on the
/// phone exceeds 40 km. [kOfflineTrailFilterValues] survives as the
/// empty-store floor: a device holding nothing for an axis still needs a
/// usable filter sheet.
library;

import 'dart:math' as math;

import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';

/// Three independent, null-dropped value lists, one per axis -- NOT
/// row-aligned. `PropertyQuery<double>.find()` excludes null values unless
/// `replaceNullWith` is supplied (objectbox 5.3.1), and `distance`,
/// `elevationGain` and `elevationLoss` are all nullable on `TrailEntity`. So
/// the three lists returned by `readLocalTrailMetrics` can have different
/// lengths, and there is no positional correspondence between them or to the
/// underlying row set. **Do not zip these lists** -- doing so would pair one
/// trail's distance with another trail's elevation. This shape is safe today
/// only because [computeOfflineTrailFilterValues] takes an independent
/// per-axis maximum from each list and never correlates entries across axes.
typedef LocalTrailMetrics = ({
  List<double> distances,
  List<double> elevationGains,
  List<double> elevationLosses,
});

/// The empty-store floor.
///
/// **When it applies.** Only when the device holds no value at all for that
/// axis: fresh install, signed out, or every matching row having a null
/// column. It is a per-axis fallback, not a whole-object one. Its numbers are
/// deliberately generous because "we know nothing" is the honest description
/// of that state -- narrowing the sliders would imply a small search space
/// that has not been observed.
///
/// **Filter text.** `buildDefaultTrailFilter` sets max == limit on every
/// axis, and `toFilterText()` emits an upper-bound clause only when max <
/// limit. So the fallback emits NO upper bound at all and can never exclude a
/// trail, whatever these numbers are. Task 2's filter-text equality test
/// pins that, with computed values.
///
/// **Slider range.** `trail_filter_screen.dart` binds each slider's `max:` to
/// the corresponding `*Limit`, and `trailFilterProvider` is one family shared
/// by `'map'`, `'library'` and `'profile_trail_<handle>'`, so on an empty
/// device these are the full-scale ranges of every filter sheet in the app
/// while offline.
const TrailFilterValues kOfflineTrailFilterValues = TrailFilterValues(
  minDistance: 0,
  maxDistance: 100000,
  minElevationGain: 0,
  maxElevationGain: 5000,
  minElevationLoss: 0,
  maxElevationLoss: 5000,
  minDuration: 0,
  maxDuration: 86400,
);

/// The rounding step for the distance axis.
///
/// **The step IS the floor.** Because [nextBoundAbove] always returns
/// strictly more than the observed max and always lands on a multiple of the
/// step, the smallest bound it can ever produce is exactly one step. So
/// there is no separate per-axis floor constant and none is needed -- a
/// device holding one 800 m walk gets a 5 km slider, not an 800 m one,
/// purely as a consequence of the rounding.
///
/// A floor ABOVE one step was considered and rejected: a 10 km floor applied
/// to an 800 m collection would put that trail at 8% of slider travel, which
/// is worse aiming than the 16% a 5 km scale gives it. A floor makes aiming
/// worse in precisely the case it was proposed to help. The steps are
/// instead chosen so that ONE step is itself a usable full scale.
const double kOfflineDistanceStepMeters = 5000;

/// The rounding step for both elevation axes (gain and loss). See
/// [kOfflineDistanceStepMeters] for the rationale, which applies identically
/// here.
const double kOfflineElevationStepMeters = 250;

/// Rounds [observedMax] strictly up to the next multiple of [step].
///
/// Guards first: a non-finite or non-positive [observedMax] is treated as 0.
///
/// The `+ step` is unconditional, and that is the point: a plain `ceil`
/// would leave an observed max that is an exact multiple of the step sitting
/// on the slider's extreme, where the only way to select it as an upper
/// bound is to leave the handle at maximum -- which is also the "no upper
/// bound" position. The unconditional step guarantees headroom for every
/// input.
double nextBoundAbove(double observedMax, double step) {
  final safeMax = (observedMax.isFinite && observedMax > 0) ? observedMax : 0;
  return (safeMax / step).floorToDouble() * step + step;
}

/// Per-axis bound computation shared by [computeOfflineTrailFilterValues]'s
/// three axes, so they cannot drift apart from one another.
double _axisBound(List<double> values, double step, double emptyFallback) {
  final finiteValues = values.where((v) => v.isFinite).toList();
  if (finiteValues.isEmpty) return emptyFallback;
  return nextBoundAbove(finiteValues.reduce(math.max), step);
}

/// Computes offline filter slider bounds from [metrics], the on-device
/// trails visible to the signed-in account.
///
/// Per axis: non-finite values are dropped, and if nothing remains for that
/// axis the matching [kOfflineTrailFilterValues] field is used; otherwise
/// the axis's rounded-up maximum from [nextBoundAbove]. Minima are always 0
/// and `maxDuration` is always the constant --
/// `buildDefaultTrailFilter` (`trail_filter_provider.dart`) reads none of
/// them, so nobody spends time computing them.
TrailFilterValues computeOfflineTrailFilterValues(LocalTrailMetrics metrics) {
  return TrailFilterValues(
    minDistance: 0,
    maxDistance: _axisBound(
      metrics.distances,
      kOfflineDistanceStepMeters,
      kOfflineTrailFilterValues.maxDistance,
    ),
    minElevationGain: 0,
    maxElevationGain: _axisBound(
      metrics.elevationGains,
      kOfflineElevationStepMeters,
      kOfflineTrailFilterValues.maxElevationGain,
    ),
    minElevationLoss: 0,
    maxElevationLoss: _axisBound(
      metrics.elevationLosses,
      kOfflineElevationStepMeters,
      kOfflineTrailFilterValues.maxElevationLoss,
    ),
    minDuration: 0,
    maxDuration: kOfflineTrailFilterValues.maxDuration,
  );
}

/// Reads the raw distance/elevationGain/elevationLoss columns for every
/// `TrailEntity` visible to [accountId] -- the exact union of everything
/// this account can search offline.
///
/// The condition (`owner.equals(accountId) |
/// savedByUserIds.containsElement(accountId)`) is a deliberate choice, not a
/// copy:
///
/// - It is the exact union of everything this account can search offline.
///   The library sheet's own read is `savedByUserIds.containsElement(userId)`
///   (`trail_library_provider.dart:28`); the own-trails read is this same
///   broad net narrowed further in Dart (`local_trail_store.dart:366-371`);
///   the map has no offline search at all. A bound must never be SMALLER
///   than the searchable space, so the union -- a superset of the
///   own-trails set after its Dart-side refinement -- is the correct side
///   to err on.
/// - It is account-scoped, which is an obligation and not merely tidiness. An unfiltered read would let account B's slider top out at
///   340 km because account A downloaded a 340 km trail onto this phone --
///   a real, if small, disclosure: the slider's maximum reveals that SOME
///   trail on this device is that long. `owner` and `savedByUserIds` are the
///   two ways a row becomes visible to an account and both are filtered
///   here.
///
/// Uses `.find()`, not `.max()` -- `DoublePropertyQuery.max()` throws on an
/// aggregate over an empty result set (a required case here: fresh install,
/// or every matching row having a null column), while `.find()` has a
/// documented, deterministic empty behaviour (an empty list). It never
/// materialises a `TrailEntity`, never runs `toModel()` and never parses
/// GPX -- the same "no entity materialisation" property `.max()` was chosen
/// for, at the cost of one `List<double>` of at most a few hundred entries.
///
/// Deliberately uncovered by `flutter test`: this repo has no ObjectBox test
/// harness (`libobjectbox.dylib` fails to load -- see the header of
/// `test/util/local_trail_store_test.dart`). Do NOT add a source-grep test
/// that inspects this function's text and reports itself as coverage; that
/// pattern has hidden a real gap here before. The account scoping here is
/// instead carried by the
/// `key_links` wiring entry and by the account-switch device check in
/// `<verification>`.
LocalTrailMetrics readLocalTrailMetrics(
  Store store, {
  required String accountId,
}) {
  final query = store
      .box<TrailEntity>()
      .query(
        TrailEntity_.owner.equals(accountId) |
            TrailEntity_.savedByUserIds.containsElement(accountId),
      )
      .build();

  final distanceQuery = query.property(TrailEntity_.distance);
  final elevationGainQuery = query.property(TrailEntity_.elevationGain);
  final elevationLossQuery = query.property(TrailEntity_.elevationLoss);

  try {
    return (
      distances: distanceQuery.find(),
      elevationGains: elevationGainQuery.find(),
      elevationLosses: elevationLossQuery.find(),
    );
  } finally {
    distanceQuery.close();
    elevationGainQuery.close();
    elevationLossQuery.close();
    query.close();
  }
}
