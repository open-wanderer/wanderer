import 'dart:math' as math;

import 'package:maplibre/maplibre.dart';

/// Result of matching a single GPS fix against the route polyline.
class MapMatchResult {
  const MapMatchResult({
    required this.alongTrackMeters,
    required this.shapeIndex,
  });

  /// Committed along-track distance (metres) from the route start to the
  /// matched position.
  final double alongTrackMeters;

  /// Index of the shape segment (`shape[shapeIndex]` → `shape[shapeIndex+1]`)
  /// the matched position falls on.
  final int shapeIndex;
}

/// A single hypothesis in the Viterbi trellis: a candidate position along the
/// route, together with its cumulative log-probability score.
class _Hypothesis {
  const _Hypothesis({
    required this.segmentIndex,
    required this.alongTrackMeters,
    required this.score,
  });

  final int segmentIndex;
  final double alongTrackMeters;
  final double score;
}

/// Online (streaming) HMM map-matcher for a single, known route polyline.
///
/// Scores each candidate position by how well it fits the GPS fix (emission:
/// cross-track distance, optional heading agreement) and by how well the
/// along-route distance to it matches the ground distance actually travelled
/// since the last fix (transition), then keeps the best-scoring chain of
/// candidates (Viterbi) rather than deciding each fix in isolation. This is
/// what lets a self-intersecting route (hairpin, switchback, out-and-back)
/// resist snapping onto a spatially-close point that's actually far away
/// along the route: that jump has good emission but bad transition, so it
/// loses to the coherent nearby chain — unless the fixes stay off the
/// current leg for a while, in which case [offRouteCrossTrackThresholdMeters]
/// triggers a wider recovery search (see there for why).
///
/// The *reported* along-track distance is a separately gated "commit" on top
/// of that internal beam (see [commitConfirmFixes]): a route can contain a
/// self-similar spur narrower than the recovery window (two legs occupying
/// near-identical coordinates but only a couple hundred metres apart
/// along-track — the same ambiguity a hairpin has, just too wide for a
/// single fix's cross-track/heading signal to resolve on its own, especially
/// under a bike's wider cornering). Letting the top hypothesis flip legs
/// silently would report a route as finished mid-ride. Instead, a forward
/// jump larger than plausible ground movement must keep winning for several
/// consecutive fixes before it's trusted — exactly the same evidence a
/// genuine cross-country shortcut also produces, just confirmed over time
/// instead of on the first fix.
class RouteMapMatcher {
  RouteMapMatcher({
    required List<Geographic> shape,
    required List<double> shapeCumulativeMeters,
    double? initialAlongTrackMeters,
    this.accuracyFloorMeters = 8.0,
    this.transitionSigmaMeters = 12.0,
    this.headingSigmaDegrees = 35.0,
    this.topK = 3,
    this.coldStartLookAheadMeters = 1000.0,
    this.warmWindowFactor = 6.0,
    this.warmWindowFloorMeters = 20.0,
    this.warmWindowCapMeters = 150.0,
    this.warmWindowCapSpeedCoeff = 20.0,
    this.backMarginMeters = 15.0,
    this.movingSpeedThresholdMetersPerSecond = 0.3,
    this.offRouteCrossTrackThresholdMeters = 40.0,
    this.crossTrackSpeedCoeff = 3.0,
    this.recoveryTriggerFixes = 3,
    this.recoveryLookAheadMeters = 400.0,
    this.recoveryTransitionSigmaMultiplier = 20.0,
    this.commitConfirmFixes = 3,
  }) : _shape = shape,
       _cumulative = shapeCumulativeMeters {
    if (initialAlongTrackMeters != null && _shape.length >= 2) {
      final seededIndex = _shapeIndexForAlongTrack(initialAlongTrackMeters);
      _hypotheses = [
        _Hypothesis(
          segmentIndex: seededIndex,
          alongTrackMeters: initialAlongTrackMeters,
          score: 0,
        ),
      ];
      _committedAtd = initialAlongTrackMeters;
      _committedSegmentIndex = seededIndex;
    }
  }

  final List<Geographic> _shape;
  final List<double> _cumulative;

  /// Floor applied to GPS `accuracy` when deriving the emission sigma.
  final double accuracyFloorMeters;

  /// Sigma (metres) for the transition term's Gaussian on
  /// `|alongTrackDelta − measuredGroundDistance|`.
  final double transitionSigmaMeters;

  /// Sigma (degrees) for the optional heading-agreement emission term.
  final double headingSigmaDegrees;

  /// Number of hypotheses retained after each update (Viterbi beam width).
  final int topK;

  /// Search window (metres, from the route start) used only for the first
  /// fix of a session, when there's no prior position to measure movement
  /// from. Also guards against a loop trail's near-trailhead endpoint
  /// segment winning the very first fix.
  final double coldStartLookAheadMeters;

  /// Warm-update forward search window = `max(warmWindowFloorMeters,
  /// groundDistance * warmWindowFactor)`, capped at the speed-adjusted
  /// [warmWindowCapMeters] (see [warmWindowCapSpeedCoeff]). Bounding
  /// candidate generation by plausible travel distance is what keeps a
  /// hairpin's far leg out of consideration in the first place.
  final double warmWindowFactor;
  final double warmWindowFloorMeters;
  final double warmWindowCapMeters;

  /// Added to [warmWindowCapMeters] per m/s of GPS `speed`. At walking pace
  /// this is negligible; at biking pace it keeps normal-window tracking
  /// (which along-track-gates candidates) covering enough ground that
  /// off-route recovery — and the wrong-leg ambiguity it risks — is rarely
  /// needed at all.
  final double warmWindowCapSpeedCoeff;

  /// How far behind a hypothesis a candidate may fall and still be
  /// considered, tolerating GPS jitter before the transition term rejects it.
  final double backMarginMeters;

  /// GPS speed below which heading is ignored for emission (compass drift
  /// dominates at rest).
  final double movingSpeedThresholdMetersPerSecond;

  /// Cross-track distance beyond which the normal window is considered to
  /// have no plausible match, opening the wider [recoveryLookAheadMeters]
  /// search. A genuine shortcut has small ground distance but a large
  /// along-route saving, so it would fail the transition check same as a
  /// noisy fix would — the distinguishing signal is that it's *sustained*
  /// (cross-track to the current leg keeps growing) rather than a single
  /// blip, which is what [recoveryTriggerFixes] detects.
  final double offRouteCrossTrackThresholdMeters;

  /// Added to [offRouteCrossTrackThresholdMeters] per m/s of GPS `speed`. A
  /// bike's wider cornering radius and larger in-motion GPS error push
  /// cross-track past a walking-tuned threshold on ordinary curves, not just
  /// genuine excursions — without this, bikes would enter recovery mode (and
  /// its wrong-leg risk) far more readily than the geometry warrants.
  final double crossTrackSpeedCoeff;

  /// Consecutive off-route fixes required before recovery mode actually
  /// widens the search window. A single wide-cornered or noisy fix no longer
  /// opens the window that a self-similar spur could be ambiguous within.
  final int recoveryTriggerFixes;

  /// Forward search window used only once off-route — wide enough to
  /// re-acquire a leg reached by a genuine cross-country shortcut, but still
  /// bounded so a single off-route blip can't leap arbitrarily far.
  final double recoveryLookAheadMeters;

  /// Multiplies [transitionSigmaMeters] while off-route: a shortcut's ground
  /// distance has no reason to match the route distance it saves, so
  /// recovery scoring is emission-led.
  final double recoveryTransitionSigmaMultiplier;

  /// Consecutive fixes a forward jump larger than plausible ground movement
  /// must keep winning before [update] reports it — see the class doc. `1`
  /// reproduces committing on the first fix (no gating).
  final int commitConfirmFixes;

  List<_Hypothesis> _hypotheses = const [];
  Geographic? _lastPos;
  int _consecutiveOffRoute = 0;

  /// The along-track distance actually reported by [update]. May lag behind
  /// the beam's current best hypothesis while a large forward jump awaits
  /// confirmation — see [commitConfirmFixes].
  double _committedAtd = 0.0;
  int _committedSegmentIndex = 0;

  /// The not-yet-confirmed jump candidate and how many consecutive fixes
  /// it has kept winning. Reset whenever the leading hypothesis changes or
  /// a jump is confirmed.
  double? _pendingJumpAtd;
  int _pendingJumpFixes = 0;

  double get _totalLength => _cumulative.isEmpty ? 0.0 : _cumulative.last;

  /// Feeds one GPS fix into the matcher and returns the newly committed
  /// match. [heading]/[headingAccuracy]/[speed]/[accuracy] are all optional;
  /// omitting them degrades to emission-by-cross-track-only, and disables
  /// the speed-adaptive widening described on [warmWindowCapSpeedCoeff] and
  /// [crossTrackSpeedCoeff].
  MapMatchResult update({
    required Geographic pos,
    double? heading,
    double? headingAccuracy,
    double? speed,
    double? accuracy,
  }) {
    if (_shape.length < 2 || _cumulative.isEmpty) {
      return const MapMatchResult(alongTrackMeters: 0, shapeIndex: 0);
    }

    final isFirstFix = _lastPos == null;
    final groundDistance = isFirstFix
        ? 0.0
        : SphericalGreatCircle(_lastPos!).distanceTo(pos);
    _lastPos = pos;

    final sources = _hypotheses.isEmpty
        ? const [_Hypothesis(segmentIndex: 0, alongTrackMeters: 0, score: 0)]
        : _hypotheses;

    final baseAtd = sources.map((h) => h.alongTrackMeters).reduce(math.min);
    final maxSourceAtd = sources
        .map((h) => h.alongTrackMeters)
        .reduce(math.max);

    final effSpeed = math.max(0.0, speed ?? 0.0);
    final effectiveWarmWindowCap =
        warmWindowCapMeters + warmWindowCapSpeedCoeff * effSpeed;
    final effectiveOffRouteThreshold =
        offRouteCrossTrackThresholdMeters + crossTrackSpeedCoeff * effSpeed;
    final plausibleStep = math.max(
      warmWindowFloorMeters,
      groundDistance * warmWindowFactor,
    );

    final Map<int, _Candidate> candidatesBysegment;
    var effectiveTransitionSigma = transitionSigmaMeters;

    if (isFirstFix) {
      _consecutiveOffRoute = 0;
      final candidates = _candidatesInWindow(
        pos,
        math.max(0.0, baseAtd),
        math.min(_totalLength, baseAtd + coldStartLookAheadMeters),
      );
      candidatesBysegment = {for (final c in candidates) c.segmentIndex: c};
    } else {
      final normalWindow = (groundDistance * warmWindowFactor).clamp(
        warmWindowFloorMeters,
        effectiveWarmWindowCap,
      );
      final normalMin = math.max(0.0, baseAtd - backMarginMeters);
      final normalMax = math.min(_totalLength, maxSourceAtd + normalWindow);
      final normalCandidates = _candidatesInWindow(pos, normalMin, normalMax);

      final bestNormalCrossTrack = normalCandidates.isEmpty
          ? double.infinity
          : normalCandidates.map((c) => c.crossTrackMeters).reduce(math.min);

      candidatesBysegment = {
        for (final c in normalCandidates) c.segmentIndex: c,
      };

      final isOffRoute = bestNormalCrossTrack > effectiveOffRouteThreshold;
      _consecutiveOffRoute = isOffRoute ? _consecutiveOffRoute + 1 : 0;

      if (isOffRoute && _consecutiveOffRoute >= recoveryTriggerFixes) {
        final recoveryMax = math.min(
          _totalLength,
          maxSourceAtd + recoveryLookAheadMeters,
        );
        final recoveryCandidates = _candidatesInWindow(
          pos,
          normalMin,
          recoveryMax,
        );
        for (final c in recoveryCandidates) {
          candidatesBysegment[c.segmentIndex] = c;
        }
        effectiveTransitionSigma =
            transitionSigmaMeters * recoveryTransitionSigmaMultiplier;
      }
    }

    final candidates = candidatesBysegment.values;

    final sigmaZ = math.max(
      accuracyFloorMeters,
      accuracy ?? accuracyFloorMeters,
    );
    final useHeading =
        heading != null &&
        (headingAccuracy == null || headingAccuracy <= 60) &&
        (speed == null || speed >= movingSpeedThresholdMetersPerSecond);

    var scored = <_Hypothesis>[];
    for (final c in candidates) {
      final emission =
          _logGaussian(c.crossTrackMeters, sigmaZ) +
          (useHeading
              ? _logGaussian(
                  _angularDiff(c.bearingDegrees, heading),
                  headingSigmaDegrees,
                )
              : 0.0);

      double best = double.negativeInfinity;
      if (isFirstFix) {
        // No transition term exists yet to break emission ties, and a
        // self-overlapping route (e.g. starting near an out-and-back's own
        // endpoint) can tie *every* candidate in the cold-start window at
        // zero cross-track. This nudge is far too small to affect any real
        // emission gap — it only makes that tie resolve deterministically
        // toward the candidate nearest the caller-supplied start, instead of
        // an implementation-defined sort outcome that could land anywhere
        // in the window.
        best = -1e-6 * (c.alongTrackMeters - baseAtd).abs();
      } else {
        for (final source in sources) {
          final delta = c.alongTrackMeters - source.alongTrackMeters;
          if (delta < -backMarginMeters) continue;
          final mismatch = (delta - groundDistance).abs();
          final transition = _logGaussian(mismatch, effectiveTransitionSigma);
          final candidateScore = source.score + transition;
          if (candidateScore > best) best = candidateScore;
        }
      }
      if (best == double.negativeInfinity) continue;

      scored.add(
        _Hypothesis(
          segmentIndex: c.segmentIndex,
          alongTrackMeters: c.alongTrackMeters,
          score: best + emission,
        ),
      );
    }

    if (scored.isEmpty) {
      // No candidate fell in any window (e.g. a wild GPS outlier, or the
      // route ended) — hold the previous commit rather than lose state.
      return MapMatchResult(
        alongTrackMeters: _committedAtd,
        shapeIndex: _committedSegmentIndex,
      );
    }

    scored.sort((a, b) => b.score.compareTo(a.score));
    final globalBest = scored.first;

    // Keep whichever hypothesis best explains "I stayed on the leg I was
    // already committed to" alive in the beam, even if topK pruning would
    // otherwise drop it in favour of several higher-scoring candidates on a
    // spatially-close-but-far-along-route leg — that continuation is what
    // [_gateCommit] needs available next fix to eventually reject a false
    // jump once real evidence (fixes staying on the true leg) arrives.
    final continuation = isFirstFix
        ? null
        : _findContinuation(
            scored,
            _committedAtd,
            groundDistance,
            plausibleStep,
          );

    var kept = scored.take(topK).toList(growable: false);
    if (continuation != null &&
        !kept.any(
          (h) =>
              h.segmentIndex == continuation.segmentIndex &&
              h.alongTrackMeters == continuation.alongTrackMeters,
        )) {
      kept = [...kept.take(math.max(0, topK - 1)), continuation];
    }

    final maxScore = kept.map((h) => h.score).reduce(math.max);
    // Rebase against the running best so scores don't drift to large
    // negative magnitudes over a long session; ranking is unaffected.
    _hypotheses = kept
        .map(
          (h) => _Hypothesis(
            segmentIndex: h.segmentIndex,
            alongTrackMeters: h.alongTrackMeters,
            score: h.score - maxScore,
          ),
        )
        .toList(growable: false);

    if (isFirstFix) {
      _committedAtd = globalBest.alongTrackMeters;
      _committedSegmentIndex = globalBest.segmentIndex;
      _pendingJumpAtd = null;
      _pendingJumpFixes = 0;
    } else {
      _gateCommit(globalBest, plausibleStep, groundDistance);
    }

    return MapMatchResult(
      alongTrackMeters: _committedAtd,
      shapeIndex: _committedSegmentIndex,
    );
  }

  /// Updates [_committedAtd]/[_committedSegmentIndex] from this fix's
  /// leading hypothesis, gating any forward jump larger than [plausibleStep]
  /// behind [commitConfirmFixes] consecutive fixes agreeing on (approximately)
  /// the same jumped-to position — see the class doc for why.
  void _gateCommit(
    _Hypothesis globalBest,
    double plausibleStep,
    double groundDistance,
  ) {
    final advance = globalBest.alongTrackMeters - _committedAtd;

    if (advance <= plausibleStep) {
      _committedAtd = globalBest.alongTrackMeters;
      _committedSegmentIndex = globalBest.segmentIndex;
      _pendingJumpAtd = null;
      _pendingJumpFixes = 0;
      return;
    }

    final sameJump =
        _pendingJumpAtd != null &&
        (globalBest.alongTrackMeters - _pendingJumpAtd!).abs() <= plausibleStep;
    _pendingJumpAtd = globalBest.alongTrackMeters;
    _pendingJumpFixes = sameJump ? _pendingJumpFixes + 1 : 1;

    if (_pendingJumpFixes >= commitConfirmFixes) {
      _committedAtd = globalBest.alongTrackMeters;
      _committedSegmentIndex = globalBest.segmentIndex;
      _pendingJumpAtd = null;
      _pendingJumpFixes = 0;
    } else {
      // Not yet confirmed — coast forward on the current leg by measured
      // ground distance rather than freezing (a real, non-ambiguous advance
      // shouldn't stall) or jumping (an unconfirmed one shouldn't commit).
      _committedAtd = math.min(
        globalBest.alongTrackMeters,
        _committedAtd + groundDistance,
      );
      _committedSegmentIndex = _shapeIndexForAlongTrack(_committedAtd);
    }
  }

  /// Among [scored], the highest-scoring hypothesis that represents staying
  /// on the current leg — its along-track advance from [committedAtd] is at
  /// most a plausible step, so it's excluded from being "the jump" itself.
  _Hypothesis? _findContinuation(
    List<_Hypothesis> scored,
    double committedAtd,
    double groundDistance,
    double plausibleStep,
  ) {
    final target = committedAtd + groundDistance;
    _Hypothesis? bestMatch;
    var bestDistance = double.infinity;
    for (final h in scored) {
      if (h.alongTrackMeters - committedAtd > plausibleStep) {
        continue;
      }
      final d = (h.alongTrackMeters - target).abs();
      if (d < bestDistance) {
        bestDistance = d;
        bestMatch = h;
      }
    }
    return bestMatch;
  }

  int _shapeIndexForAlongTrack(double atd) {
    var idx = _lowerBound(_cumulative, atd);
    if (idx > 0) idx -= 1;
    return idx.clamp(0, _shape.length - 2).toInt();
  }

  /// Finds candidate projections onto every route segment whose along-track
  /// range intersects `[minAtd, maxAtd]`. One candidate per segment (its
  /// perpendicular foot), not a single global nearest — the caller's
  /// emission/transition scoring decides which one wins.
  List<_Candidate> _candidatesInWindow(
    Geographic pos,
    double minAtd,
    double maxAtd,
  ) {
    var start = _lowerBound(_cumulative, minAtd);
    if (start > 0) start -= 1;
    start = start.clamp(0, _shape.length - 2).toInt();

    const metersPerDegree = 111320.0;
    final candidates = <_Candidate>[];

    for (var i = start; i < _shape.length - 1; i++) {
      if (_cumulative[i] > maxAtd) break;
      if (_cumulative[i + 1] < minAtd) continue;

      final a = _shape[i];
      final b = _shape[i + 1];

      final midLatRad = ((a.lat + b.lat) / 2.0) * math.pi / 180.0;
      final cosMidLat = math.cos(midLatRad);

      final abX = (b.lon - a.lon) * metersPerDegree * cosMidLat;
      final abY = (b.lat - a.lat) * metersPerDegree;

      final apX = (pos.lon - a.lon) * metersPerDegree * cosMidLat;
      final apY = (pos.lat - a.lat) * metersPerDegree;

      final segLenSq = abX * abX + abY * abY;

      double t;
      if (segLenSq <= 0.0) {
        t = 0.0;
      } else {
        t = (apX * abX + apY * abY) / segLenSq;
        t = t.clamp(0.0, 1.0);
      }

      final footX = apX - t * abX;
      final footY = apY - t * abY;
      final crossTrack = math.sqrt(footX * footX + footY * footY);

      final segmentMeters = SphericalGreatCircle(a).distanceTo(b);
      final atd = _cumulative[i] + t * segmentMeters;
      final bearing = SphericalGreatCircle(a).initialBearingTo(b);

      candidates.add(
        _Candidate(
          segmentIndex: i,
          alongTrackMeters: atd,
          crossTrackMeters: crossTrack,
          bearingDegrees: bearing,
        ),
      );
    }

    return candidates;
  }

  double _logGaussian(double x, double sigma) {
    final s = sigma <= 0 ? 1e-6 : sigma;
    return -0.5 * (x * x) / (s * s);
  }

  /// Smallest absolute angular difference between two bearings, in
  /// `[0, 180]` degrees.
  double _angularDiff(double a, double b) {
    var d = (a - b) % 360;
    if (d < 0) d += 360;
    if (d > 180) d = 360 - d;
    return d;
  }

  /// Index of the first element in the (ascending, sorted) [list] that is
  /// `>= value`; `list.length` if none.
  int _lowerBound(List<double> list, double value) {
    var lo = 0;
    var hi = list.length;
    while (lo < hi) {
      final mid = (lo + hi) >> 1;
      if (list[mid] < value) {
        lo = mid + 1;
      } else {
        hi = mid;
      }
    }
    return lo;
  }
}

class _Candidate {
  const _Candidate({
    required this.segmentIndex,
    required this.alongTrackMeters,
    required this.crossTrackMeters,
    required this.bearingDegrees,
  });

  final int segmentIndex;
  final double alongTrackMeters;
  final double crossTrackMeters;
  final double bearingDegrees;
}
