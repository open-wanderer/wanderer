import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/provider/navigation_provider.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// [NavigationState.breadcrumb] entries are [Wpt] (carrying ele/time), not
/// [Geographic] — extracts just lat/lon for equality against a plain
/// [Geographic] fix.
Geographic _asGeo(Wpt wpt) => Geographic(lat: wpt.lat!, lon: wpt.lon!);

/// Build a [NavigateResponse] with three maneuvers and a known shape so that
/// the 30 m advancement threshold is deterministic in tests.

NavigateResponse _buildSampleResponse() {
  return NavigateResponse(
    shape: [
      [47.000, 9.000],
      [47.001, 9.000],
      [47.002, 9.000],
      [47.003, 9.000], // index 3 — maneuver 1 target
      [47.004, 9.000],
      [47.005, 9.000],
      [47.006, 9.000], // index 6 — maneuver 2 target
    ],
    maneuvers: const [
      NavigateManeuver(
        instruction: 'Start',
        length: 300.0,
        beginShapeIndex: 0,
        bearing: 0.0,
      ),
      NavigateManeuver(
        instruction: 'Turn right',
        length: 300.0,
        beginShapeIndex: 3,
        bearing: 90.0,
      ),
      NavigateManeuver(
        instruction: 'Arrive',
        length: 0.0,
        beginShapeIndex: 6,
        bearing: 0.0,
      ),
    ],
  );
}

/// A position very close to shape[3] (maneuver 1 target) — within 30 m.
///
/// shape[3] = Geographic(47.003, 9.000). Offset by ~0.0001° lat ≈ 11 m.
const _nearManeuver1 = Geographic(lat: 47.00301, lon: 9.000);

/// A position clearly off-route near the start — large cross-track distance so
/// its along-track projection stays at ~0 m (well short of maneuver 1).
///
/// The route runs north along lon 9.000; this point sits ~760 m east at the
/// start latitude (47.000, 9.010). Under along-track projection it projects
/// onto shape[0], so [currentManeuverIndex] must remain at 0.
const _farFromManeuver1 = Geographic(lat: 47.000, lon: 9.010);

/// A position very close to shape[6] (maneuver 2 begin) — within 30 m.
///
/// shape[6] = Geographic(47.006, 9.000). Offset by ~0.00001° lat ≈ 1 m. Its
/// projected along-track distance is ~667 m, past maneuver 1's ~333 m.
const _nearManeuver2 = Geographic(lat: 47.00601, lon: 9.000);

/// A position projecting to roughly the route midpoint between maneuver 0
/// (~0 m) and maneuver 1 (~333 m) — about ~178 m along-track. It lies on the
/// route between shape[1] and shape[2] and is > 30 m short of maneuver 1.
const _midpointBeforeManeuver1 = Geographic(lat: 47.0016, lon: 9.000);

/// A position exactly on shape[6] (the maneuver 2 vertex). Its along-track
/// projection must equal maneuver 2's cumulative distance (~667 m).
const _exactlyManeuver2 = Geographic(lat: 47.006, lon: 9.000);

/// Builds a tight hairpin route: an "entry" leg climbing north (shape[0..50])
/// then an "exit" leg descending back south only ~15 m away — offset 0.0002°
/// longitude at this latitude — from shape[51..101]. Every entry-leg point
/// has a spatially-close counterpart on the exit leg that sits hundreds of
/// metres away *along the route*, the exact ambiguity a nearest-point
/// projection gets wrong.
NavigateResponse _buildHairpinResponse() {
  const startLat = 47.0000;
  const stepLat = 0.0001; // ~11.1 m
  const entryLon = 9.0000;
  const exitLon = 9.0002; // ~15.2 m east at this latitude

  final shape = <List<double>>[];
  for (var i = 0; i <= 50; i++) {
    shape.add([startLat + stepLat * i, entryLon]);
  }
  for (var i = 50; i >= 0; i--) {
    shape.add([startLat + stepLat * i, exitLon]);
  }

  return NavigateResponse(
    shape: shape,
    maneuvers: const [
      NavigateManeuver(instruction: 'Start', length: 0, beginShapeIndex: 0),
      NavigateManeuver(
        instruction: 'Continue up entry leg',
        length: 0,
        beginShapeIndex: 5,
      ),
      NavigateManeuver(
        instruction: 'Switchback turn',
        length: 0,
        beginShapeIndex: 51,
      ),
      NavigateManeuver(
        instruction: 'Near end of exit leg',
        length: 0,
        beginShapeIndex: 95,
      ),
      NavigateManeuver(instruction: 'Arrive', length: 0, beginShapeIndex: 101),
    ],
  );
}

/// Builds a ~330 m out-and-back spur route: the return leg retraces the
/// outbound leg's exact coordinates (shape[j] == shape[60 - j] for j <= 29),
/// so points before the apex have a spatially-identical twin a couple
/// hundred metres away *along the route* — inside the map-matcher's
/// off-route recovery window. This is the geometry that produced a reported
/// false "route finished" while biking: a burst of wide-cornering fixes
/// (bikes corner wider than the foot-pace recording) pushes cross-track past
/// the off-route threshold right in the danger zone.
NavigateResponse _buildNarrowSpurResponse() {
  const startLat = 47.0000;
  const stepLat = 0.0001; // ~11.1 m
  const lon = 9.0000;

  final outbound = List.generate(31, (i) => [startLat + stepLat * i, lon]);
  final inbound = outbound.reversed.skip(1).toList();
  final shape = [...outbound, ...inbound];

  return NavigateResponse(
    shape: shape,
    maneuvers: const [
      NavigateManeuver(instruction: 'Start', length: 0, beginShapeIndex: 0),
      NavigateManeuver(
        instruction: 'Approach turnaround',
        length: 0,
        beginShapeIndex: 25,
      ),
      NavigateManeuver(
        instruction: 'On the return leg (false-jump target)',
        length: 0,
        beginShapeIndex: 40,
      ),
      NavigateManeuver(instruction: 'Arrive', length: 0, beginShapeIndex: 60),
    ],
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('NavigationNotifier', () {
    late ProviderContainer container;
    late NavigateResponse response;

    setUp(() {
      response = _buildSampleResponse();
      container = ProviderContainer();
    });

    tearDown(() {
      container.dispose();
    });

    test(
      'initial state: currentManeuverIndex is 0 and breadcrumb is empty',
      () {
        final state = container.read(navigationProvider(response));
        expect(state.currentManeuverIndex, 0);
        expect(state.breadcrumb, isEmpty);
      },
    );

    test(
      'onPosition far from maneuver 1 target: index stays at 0, one breadcrumb appended',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        notifier.onPosition(_farFromManeuver1);

        final state = container.read(navigationProvider(response));
        expect(state.currentManeuverIndex, 0);
        expect(state.breadcrumb.length, 1);
        expect(_asGeo(state.breadcrumb[0]), _farFromManeuver1);
      },
    );

    test(
      'onPosition within 30 m of shape[maneuver1.beginShapeIndex] advances index to 1',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        notifier.onPosition(_nearManeuver1);

        final state = container.read(navigationProvider(response));
        expect(state.currentManeuverIndex, 1);
      },
    );

    test(
      'breadcrumb keeps a stable list identity across appends; '
      'breadcrumbLength is the change signal',
      () {
        // The per-fix O(1) append contract: the state exposes ONE view object
        // for the whole session (listeners must key on breadcrumbLength, the
        // screen's GeoJSON updater relies on this), and every append bumps
        // breadcrumbLength in the emitted state.
        final notifier = container.read(navigationProvider(response).notifier);
        final before = container.read(navigationProvider(response));
        expect(before.breadcrumbLength, 0);

        notifier.onPosition(_farFromManeuver1);
        final after1 = container.read(navigationProvider(response));
        notifier.onPosition(_farFromManeuver1);
        final after2 = container.read(navigationProvider(response));

        expect(identical(before.breadcrumb, after1.breadcrumb), isTrue);
        expect(identical(after1.breadcrumb, after2.breadcrumb), isTrue);
        expect(after1.breadcrumbLength, 1);
        expect(after2.breadcrumbLength, 2);
        expect(after2.breadcrumb.length, 2);
      },
    );

    test(
      'onPosition returns true exactly when the maneuver index advances',
      () {
        // The screen persists on advance using this return value instead of a
        // before/after pair of provider reads — it must track state exactly.
        final notifier = container.read(navigationProvider(response).notifier);

        expect(notifier.onPosition(_farFromManeuver1), isFalse);
        expect(notifier.onPosition(_nearManeuver1), isTrue);
        expect(notifier.onPosition(_nearManeuver1), isFalse);
      },
    );

    test(
      'onPosition at last maneuver does not throw and does not advance further',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        // Advance to last maneuver (index 2) by calling onPosition twice near
        // each successive target.
        const nearManeuver1 = Geographic(lat: 47.00301, lon: 9.000);
        const nearManeuver2 = Geographic(lat: 47.00601, lon: 9.000);
        notifier.onPosition(nearManeuver1); // advances to 1
        notifier.onPosition(nearManeuver2); // advances to 2 (last)

        expect(
          container.read(navigationProvider(response)).currentManeuverIndex,
          2,
        );

        // Calling onPosition again at the last maneuver must not throw and must
        // not change the index beyond the last.
        expect(
          () => notifier.onPosition(const Geographic(lat: 47.009, lon: 9.000)),
          returnsNormally,
        );
        expect(
          container.read(navigationProvider(response)).currentManeuverIndex,
          2,
        );
      },
    );

    test(
      'advancement is forward-only: near earlier maneuver never decrements',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        // Advance to index 1 first.
        notifier.onPosition(_nearManeuver1);
        expect(
          container.read(navigationProvider(response)).currentManeuverIndex,
          1,
        );

        // A position near shape[0] (maneuver 0 target) must NOT decrement index.
        notifier.onPosition(const Geographic(lat: 47.000, lon: 9.000));
        expect(
          container.read(navigationProvider(response)).currentManeuverIndex,
          1,
        );
      },
    );

    test(
      'onPosition near maneuver 2 from index 0 skips maneuver 1 and advances directly to index 2',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        // Single fix near shape[6] (maneuver 2 begin) while still at index 0.
        // Along-track projection must advance past the skipped maneuver 1.
        notifier.onPosition(_nearManeuver2);

        expect(
          container.read(navigationProvider(response)).currentManeuverIndex,
          2,
        );
      },
    );

    test(
      'onPosition projecting to route midpoint before maneuver 1 does not advance',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        // ~178 m along-track — well under maneuver 1's ~333 m minus the 30 m
        // along-track buffer.
        notifier.onPosition(_midpointBeforeManeuver1);

        expect(
          container.read(navigationProvider(response)).currentManeuverIndex,
          0,
        );
      },
    );

    test(
      'onPosition exactly on maneuver 2 vertex projects to its cumulative distance and advances to index 2',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        // A position exactly on shape[6] projects to ~667 m along-track (within
        // a few meters of maneuver 2's cumulative distance), reaching maneuver 2.
        notifier.onPosition(_exactlyManeuver2);

        expect(
          container.read(navigationProvider(response)).currentManeuverIndex,
          2,
        );
      },
    );

    test(
      'onPosition with recordBreadcrumb: false does not append to breadcrumb '
      '(simulates a fix received while paused/stationary)',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        notifier.onPosition(_farFromManeuver1, recordBreadcrumb: false);

        final state = container.read(navigationProvider(response));
        expect(state.breadcrumb, isEmpty);
      },
    );

    test(
      'onPosition with recordBreadcrumb: false still advances maneuver index '
      '(pausing must not stop turn-by-turn progress tracking)',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        notifier.onPosition(_nearManeuver1, recordBreadcrumb: false);

        final state = container.read(navigationProvider(response));
        expect(state.currentManeuverIndex, 1);
        expect(state.breadcrumb, isEmpty);
      },
    );

    test(
      'onPosition with recordBreadcrumb defaulting to true appends to '
      'breadcrumb (simulates a fix received while unpaused/moving)',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        notifier.onPosition(_farFromManeuver1);

        final state = container.read(navigationProvider(response));
        expect(state.breadcrumb.length, 1);
        expect(_asGeo(state.breadcrumb[0]), _farFromManeuver1);
      },
    );

    test(
      'each onPosition call appends exactly one Geographic to breadcrumb',
      () {
        final notifier = container.read(navigationProvider(response).notifier);

        const p1 = Geographic(lat: 47.001, lon: 9.001);
        const p2 = Geographic(lat: 47.002, lon: 9.002);
        const p3 = Geographic(lat: 47.003, lon: 9.003);

        notifier.onPosition(p1);
        notifier.onPosition(p2);
        notifier.onPosition(p3);

        final breadcrumb = container
            .read(navigationProvider(response))
            .breadcrumb;
        expect(breadcrumb.length, 3);
        expect(_asGeo(breadcrumb[0]), p1);
        expect(_asGeo(breadcrumb[1]), p2);
        expect(_asGeo(breadcrumb[2]), p3);
      },
    );
  });

  group('NavigationNotifier recording mode (empty response)', () {
    test(
      'empty NavigateResponse resolves to an empty state without throwing, '
      'then a fed position still appends to the breadcrumb',
      () {
        const emptyResponse = NavigateResponse(maneuvers: [], shape: []);
        final container = ProviderContainer();
        addTearDown(container.dispose);

        expect(
          () => container.read(navigationProvider(emptyResponse)),
          returnsNormally,
        );

        final state = container.read(navigationProvider(emptyResponse));
        expect(state.currentManeuverIndex, 0);
        expect(state.breadcrumb, isEmpty);

        final notifier = container.read(
          navigationProvider(emptyResponse).notifier,
        );
        expect(
          () => notifier.onPosition(const Geographic(lat: 47.0, lon: 9.0)),
          returnsNormally,
        );

        final afterState = container.read(navigationProvider(emptyResponse));
        expect(afterState.breadcrumb.length, 1);
        expect(afterState.currentManeuverIndex, 0);
      },
    );
  });

  group('NavigationNotifier hairpin route handling', () {
    test(
      'a single noisy fix near a hairpin apex does not skip ahead to the '
      'exit leg maneuvers',
      () {
        final hairpinResponse = _buildHairpinResponse();
        final container = ProviderContainer();
        addTearDown(container.dispose);

        final notifier = container.read(
          navigationProvider(hairpinResponse).notifier,
        );
        final shape = hairpinResponse.shapeAsGeographic;

        // Walk the first 10 points of the entry leg for real — advances past
        // maneuver 1 ("Continue up entry leg", beginShapeIndex 5).
        for (var i = 0; i <= 9; i++) {
          notifier.onPosition(shape[i]);
        }
        expect(
          container
              .read(navigationProvider(hairpinResponse))
              .currentManeuverIndex,
          1,
        );

        // One noisy fix near shape[10], offset toward the exit leg by ~11 m
        // — spatially closer to the exit leg's same-latitude point than to
        // the entry leg it's actually on. The old nearest-point projection
        // would snap onto the exit leg and skip straight to maneuver 3 or 4.
        final apex = shape[10];
        notifier.onPosition(Geographic(lat: apex.lat, lon: apex.lon + 0.00015));

        expect(
          container
              .read(navigationProvider(hairpinResponse))
              .currentManeuverIndex,
          1,
          reason:
              'a single noisy fix near the apex must not skip past the '
              'switchback turn maneuvers onto the exit leg',
        );
      },
    );
  });

  group('NavigationNotifier narrow out-and-back spur handling', () {
    test(
      'sustained wide-cornering fixes at bike speed through the spur apex '
      'do not skip ahead to the return-leg maneuvers',
      () {
        final spurResponse = _buildNarrowSpurResponse();
        final container = ProviderContainer();
        addTearDown(container.dispose);

        final notifier = container.read(
          navigationProvider(spurResponse).notifier,
        );
        final shape = spurResponse.shapeAsGeographic;

        // Walk the outbound leg for real, at bike speed — advances past
        // "Approach turnaround" (beginShapeIndex 25... reached later, so
        // just past "Start" for now).
        for (var i = 0; i <= 18; i++) {
          notifier.onPosition(shape[i], speed: 6.0);
        }

        // A burst of wide-cornering fixes through the danger zone — offset
        // enough to exceed even the speed-adjusted recovery trigger for
        // several consecutive fixes, as a bike's wider turning radius
        // plausibly would on a spur originally recorded on foot. The old
        // behavior would let this snap onto the return leg's identical
        // coordinates and report the route as finished.
        for (var i = 19; i <= 22; i++) {
          final wide = Geographic(
            lat: shape[i].lat,
            lon: shape[i].lon + 0.0009, // ~68 m east
          );
          notifier.onPosition(wide, speed: 6.0);
        }

        // Resume tracking correctly along the real outbound leg, up to (but
        // not through) the turnaround.
        for (var i = 23; i <= 29; i++) {
          notifier.onPosition(shape[i], speed: 6.0);
        }

        final currentIndex = container
            .read(navigationProvider(spurResponse))
            .currentManeuverIndex;

        expect(
          currentIndex,
          lessThan(2),
          reason:
              'must still be on/before "Approach turnaround" — a false '
              'snap onto the return leg would have skipped straight to the '
              '"On the return leg" or "Arrive" maneuvers',
        );
      },
    );
  });
}
