import 'dart:async';

import 'package:flutter/material.dart';
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/navigate_response.dart';

part 'navigation_stats_provider.freezed.dart';
part 'navigation_stats_provider.g.dart';

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

/// Immutable snapshot of live navigation statistics for a single navigation
/// session.
///
/// Session-only in-memory state (mirrors [NavigationState]): discarded
/// when the family entry is disposed on screen exit. All values are derived
/// from the GPS [Position] stream fed via [NavigationStatsNotifier.onPosition].
///
/// Implemented as a [freezed] class so `copyWith`/equality are generated.
@freezed
abstract class NavigationStats with _$NavigationStats {
  const factory NavigationStats({
    /// Wall-clock elapsed time since the first GPS fix, excluding paused time.
    @Default(Duration.zero) Duration elapsed,

    /// Cumulative Haversine distance travelled, in metres.
    @Default(0) double distanceMeters,

    /// Cumulative positive altitude delta (ascent), in metres.
    @Default(0) double elevationGainMeters,

    /// Cumulative negative altitude delta (descent), stored as a positive
    /// metre value.
    @Default(0) double elevationLossMeters,

    /// Instantaneous GPS speed in km/h (guarded against NaN/negative).
    @Default(0) double currentSpeedKmh,

    /// Average speed in km/h (distance / elapsed).
    @Default(0) double averageSpeedKmh,

    /// Whether stat accumulation is currently frozen by the user.
    @Default(false) bool isPaused,
  }) = _NavigationStats;
}

/// Resume-seed snapshot used to rehydrate [NavigationStats] from a persisted
/// [ActiveNavigationEntity] row after a manual app termination.
///
/// Session-scoped, value-equal via the existing freezed part — no `fromJson`
/// needed since it's constructed directly from the persisted entity's fields,
/// never (de)serialized itself.
@freezed
abstract class NavigationStatsSeed with _$NavigationStatsSeed {
  const factory NavigationStatsSeed({
    required double distanceMeters,
    required double elevationGainMeters,
    required double elevationLossMeters,
    required Duration elapsed,
    required Duration pausedAccum,
    required bool isPaused,
  }) = _NavigationStatsSeed;
}

// ---------------------------------------------------------------------------
// Notifier
// ---------------------------------------------------------------------------

/// Riverpod notifier that computes live navigation statistics for a single
/// [NavigateResponse] session.
///
/// Family-keyed on [response] so each navigation session is isolated and
/// testable via a plain [ProviderContainer], exactly like
/// `navigationProvider`.
///
/// This notifier never opens its own GPS stream. It is fed purely via
/// [onPosition], called from the single broadcast stream owned by
/// `_NavigationScreenState`.
@riverpod
class NavigationStatsNotifier extends _$NavigationStatsNotifier {
  /// Altitude noise-floor (metres). Only altitude deltas at or above this
  /// magnitude are accumulated as gain/loss, so per-fix GPS jitter does not
  /// inflate elevation totals. Tunable in one place.
  static const _kAltitudeNoiseFloorMeters = 2.0;

  /// 1-second clock driving [NavigationStats.elapsed]. Independent of GPS
  /// cadence so the clock keeps ticking while the user stands still.
  Timer? _ticker;

  /// Wall-clock instant of the first GPS fix (start of the stopwatch).
  DateTime? _start;

  /// Total time spent paused, subtracted from wall-clock elapsed.
  Duration _pausedAccum = Duration.zero;

  /// Instant the current pause began; null when not paused.
  DateTime? _pauseStart;

  /// Last accumulated altitude reference (metres); null until first fix.
  double? _lastAltitude;

  /// Last position used for distance accumulation; null until first fix or
  /// after a resume re-anchor.
  Geographic? _lastPoint;

  @override
  NavigationStats build(NavigateResponse response, {NavigationStatsSeed? resume}) {
    // Cancel the timer when the family entry is disposed.
    ref.onDispose(() => _ticker?.cancel());

    if (resume == null) {
      return const NavigationStats();
    }

    _pausedAccum = resume.pausedAccum;
    // _tick's `DateTime.now().difference(_start!) - _pausedAccum` must yield
    // `resume.elapsed` at the first tick and advance by wall time thereafter.
    // Subtracting only `elapsed` while separately restoring `_pausedAccum`
    // would double-count paused time — this formula avoids that.
    _start = DateTime.now().subtract(resume.elapsed + resume.pausedAccum);
    _ticker ??= Timer.periodic(const Duration(seconds: 1), (_) => _tick());
    if (resume.isPaused) {
      _pauseStart = DateTime.now();
    }

    return NavigationStats(
      distanceMeters: resume.distanceMeters,
      elevationGainMeters: resume.elevationGainMeters,
      elevationLossMeters: resume.elevationLossMeters,
      isPaused: resume.isPaused,
      elapsed: resume.elapsed,
    );
  }

  /// The total time spent paused so far — not part of [NavigationStats] state
  /// but needed by the screen to persist the accumulator.
  Duration get pausedAccum => _pausedAccum;

  /// Feeds a GPS [Position] into the accumulator.
  ///
  /// While paused, accumulation is frozen and current speed is forced to 0.
  /// The first fix only sets the distance/altitude references (no distance is
  /// added). Invalid speed values (NaN/negative) are clamped to 0.
  void onPosition(geo.Position pos) {
    // While paused, do not accumulate distance/elevation; force speed to 0.
    if (state.isPaused) {
      if (state.currentSpeedKmh != 0) {
        state = state.copyWith(currentSpeedKmh: 0);
      }
      return;
    }

    // Start the stopwatch + ticker on the first fix.
    _start ??= DateTime.now();
    _ticker ??= Timer.periodic(const Duration(seconds: 1), (_) => _tick());

    final here = Geographic(lat: pos.latitude, lon: pos.longitude);

    // Distance accumulation (Haversine). First fix only anchors the reference.
    var dist = state.distanceMeters;
    if (_lastPoint != null) {
      final calculator = SphericalGreatCircle(_lastPoint!);
      dist += calculator.distanceTo(here);
    }
    _lastPoint = here;

    // Elevation gain/loss with a noise-floor threshold. The
    // reference altitude is only updated when the threshold is crossed, so
    // small drifts never accumulate.
    var gain = state.elevationGainMeters;
    var loss = state.elevationLossMeters;
    if (_lastAltitude != null) {
      final delta = pos.altitude - _lastAltitude!;
      if (delta.abs() >= _kAltitudeNoiseFloorMeters) {
        if (delta > 0) {
          gain += delta;
        } else {
          loss += -delta;
        }
        _lastAltitude = pos.altitude;
      }
    } else {
      _lastAltitude = pos.altitude;
    }

    // Current speed: m/s → km/h, guarding NaN/negative.
    final rawSpeed = pos.speed;
    final speedKmh = (rawSpeed.isNaN || rawSpeed < 0) ? 0.0 : rawSpeed * 3.6;

    state = state.copyWith(
      distanceMeters: dist,
      elevationGainMeters: gain,
      elevationLossMeters: loss,
      currentSpeedKmh: speedKmh,
    );
  }

  /// 1-second tick: updates [NavigationStats.elapsed] and the derived average
  /// speed. No-op while paused or before the first fix.
  void _tick() {
    if (state.isPaused || _start == null) return;
    final elapsed = DateTime.now().difference(_start!) - _pausedAccum;
    final avg = elapsed.inSeconds > 0
        ? state.distanceMeters / elapsed.inSeconds * 3.6
        : 0.0;
    state = state.copyWith(elapsed: elapsed, averageSpeedKmh: avg);
  }

  /// Toggles the paused state.
  ///
  /// On pause: records the pause start and forces current speed to 0.
  /// On resume: adds the paused interval to [_pausedAccum] and resets the
  /// distance/altitude references so the next fix re-anchors without producing
  /// a jump.
  void togglePause() {
    if (state.isPaused) {
      // Resume.
      if (_pauseStart != null) {
        _pausedAccum += DateTime.now().difference(_pauseStart!);
        _pauseStart = null;
      }
      // Re-anchor so the paused interval contributes no distance/elevation.
      _lastPoint = null;
      _lastAltitude = null;
      state = state.copyWith(isPaused: false);
    } else {
      // Pause.
      _pauseStart = DateTime.now();
      state = state.copyWith(isPaused: true, currentSpeedKmh: 0);
    }
  }
}
