import 'dart:async';

import 'package:flutter/material.dart';
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/services/tracelet_position_source.dart'
    show hasUsableAltitude;

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

    /// Whether stat accumulation is currently frozen by the user via the
    /// manual pause button.
    @Default(false) bool isPaused,

    /// Whether stat accumulation is currently frozen because tracelet's
    /// native speed-motion engine reports the user as stationary. Distinct
    /// from [isPaused] (manual) — the two compose (frozen = either true) via
    /// [NavigationStatsNotifier._applyFrozen].
    @Default(false) bool isStationary,
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

  /// Total time spent frozen (manually paused and/or stationary), subtracted
  /// from wall-clock elapsed.
  Duration _pausedAccum = Duration.zero;

  /// Instant the current frozen interval began; null when not frozen.
  /// Generalizes the old `_pauseStart` to cover both manual pause and
  /// stationary freezing via [_applyFrozen], so overlapping intervals are
  /// counted once, not twice.
  DateTime? _frozenSince;

  /// Backs [NavigationStats.isPaused] — true while manually paused via the
  /// pause button.
  bool _manualPaused = false;

  /// Backs [NavigationStats.isStationary] — true while tracelet's native
  /// speed-motion engine reports the user as stationary.
  bool _stationary = false;

  /// Last accumulated altitude reference (metres); null until first fix.
  double? _lastAltitude;

  /// Last position used for distance accumulation; null until first fix or
  /// after a resume re-anchor.
  Geographic? _lastPoint;

  @override
  NavigationStats build(
    NavigateResponse response, {
    NavigationStatsSeed? resume,
  }) {
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
    // A session resumed in the paused state starts with no ticker — the
    // frozen↔unfrozen transitions own the timer's lifecycle (see
    // [_applyFrozen]); a 1 Hz wakeup whose tick is a guaranteed no-op is
    // pure battery waste over an hours-long pause.
    if (!resume.isPaused) {
      _ticker ??= Timer.periodic(const Duration(seconds: 1), (_) => _tick());
    }
    // Stationary state is NOT persisted/resumed here — it is re-derived
    // within seconds of session resume via tracelet's engine re-emitting a
    // SpeedMotionEvent once tracking restarts. Any stationary interval that
    // was already in progress before termination is already folded into
    // `resume.pausedAccum`.
    _manualPaused = resume.isPaused;
    if (resume.isPaused) {
      _frozenSince = DateTime.now();
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
  /// While frozen (manually paused and/or stationary), accumulation is
  /// frozen and current speed is forced to 0. The first fix only sets the
  /// distance/altitude references (no distance is added). Invalid speed
  /// values (NaN/negative) are clamped to 0.
  void onPosition(geo.Position pos) {
    // While frozen, do not accumulate distance/elevation; force speed to 0.
    if (state.isPaused || state.isStationary) {
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
    //
    // A fix with no real altitude reading must never anchor the reference or
    // be diffed against it — see [hasUsableAltitude], which owns that
    // judgement (and explains why it is not simply an accuracy check).
    // Anchoring on the fabricated 0 that TraceletPositionSource's seed fix
    // carries would make the next genuine reading register as a single-step
    // "gain" of the device's full absolute altitude. Skip such fixes for
    // elevation purposes; the reference anchors on the first subsequent fix
    // that does carry a real reading — the same rule computeTrailMetrics
    // already applies to waypoints with no usable `ele`.
    var gain = state.elevationGainMeters;
    var loss = state.elevationLossMeters;
    if (hasUsableAltitude(pos)) {
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
  /// speed. No-op while frozen (paused or stationary) or before the first
  /// fix — freezing on stationary this way is what turns [elapsed] into
  /// time-in-motion automatically, with no separate "motion clock".
  void _tick() {
    if (state.isPaused || state.isStationary || _start == null) return;
    final elapsed = DateTime.now().difference(_start!) - _pausedAccum;
    final avg = elapsed.inSeconds > 0
        ? state.distanceMeters / elapsed.inSeconds * 3.6
        : 0.0;
    state = state.copyWith(elapsed: elapsed, averageSpeedKmh: avg);
  }

  /// Toggles the manual paused state (the pause button).
  ///
  /// Routes through [_applyFrozen] so overlapping manual pause and
  /// stationary intervals never double-count paused time.
  void togglePause() {
    _manualPaused = !_manualPaused;
    state = state.copyWith(isPaused: _manualPaused);
    _applyFrozen(_manualPaused || _stationary);
  }

  /// Called externally (from the screen) whenever tracelet's native
  /// speed-motion engine transitions between moving and stationary. This
  /// notifier does not compute motion itself — [stationary] is purely a
  /// reaction to [TraceletPositionSource.isMovingStream].
  ///
  /// Idempotent against duplicate stream events: a no-op if [stationary]
  /// matches the current state.
  void setStationary(bool stationary) {
    if (stationary == _stationary) return;
    _stationary = stationary;
    state = state.copyWith(isStationary: _stationary);
    _applyFrozen(_manualPaused || _stationary);
  }

  /// Single derived "frozen" transition handler shared by [togglePause] and
  /// [setStationary]. `frozen = _manualPaused || _stationary`, so an overlap
  /// between the two (e.g. manually paused while also stationary) is counted
  /// once, not twice: the accumulator only starts/stops on real frozen ↔
  /// unfrozen transitions, not on every individual toggle.
  ///
  /// On false→true: records [_frozenSince] and forces current speed to 0.
  /// On true→false: adds the elapsed frozen interval to [_pausedAccum] —
  /// but only once the stopwatch has actually started (`_start != null`).
  /// Tracelet's motion engine can flip stationary↔moving before the first
  /// GPS fix ever arrives (e.g. waiting for a lock while physically still),
  /// which set `_frozenSince` and later accumulated real wall-clock seconds
  /// into `_pausedAccum` for a session that hadn't started yet — inflating
  /// `_pausedAccum` beyond the elapsed time `_tick` computes once `_start`
  /// is finally set, making `elapsed` go negative. Time frozen before the
  /// session starts isn't part of any elapsed interval, so it's discarded
  /// rather than accumulated. Also clears [_frozenSince] and re-anchors the
  /// distance/altitude references so the frozen interval contributes zero
  /// distance/elevation — identical to the previous manual-pause-only
  /// resume behavior.
  void _applyFrozen(bool nowFrozen) {
    final wasFrozen = _frozenSince != null;
    if (nowFrozen == wasFrozen) return;

    if (nowFrozen) {
      _frozenSince = DateTime.now();
      state = state.copyWith(currentSpeedKmh: 0);
      // Frozen ticks are guaranteed no-ops (see [_tick]) — stop the 1 Hz
      // timer entirely instead of waking up to do nothing for the whole
      // paused/stationary interval.
      _ticker?.cancel();
      _ticker = null;
    } else {
      if (_start != null) {
        _pausedAccum += DateTime.now().difference(_frozenSince!);
      }
      _frozenSince = null;
      // Re-anchor so the frozen interval contributes no distance/elevation.
      _lastPoint = null;
      _lastAltitude = null;
      // Restart the clock only for a session that has actually started —
      // before the first fix, [onPosition] owns ticker creation.
      if (_start != null) {
        _ticker ??= Timer.periodic(const Duration(seconds: 1), (_) => _tick());
      }
    }
  }
}
