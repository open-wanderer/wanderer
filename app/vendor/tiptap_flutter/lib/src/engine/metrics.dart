// Port-side performance instrumentation for the Tiptap bridge.
//
// These types collect timing measurements about the bridge and editor so the
// performance overlay can display them. They are NOT part of the engine
// protocol — they describe how the Flutter port observes its own behavior,
// not anything exchanged with the engine. Four things are measured:
//
//   - Command round-trips: how long a command takes from the moment it is
//     sent to the engine until its correlated response arrives.
//   - Engine load phases: how long each lifecycle transition takes during
//     cold start (loading -> pageLoaded -> engineGlobalReady -> schemaReady
//     -> ready).
//   - Typing latency: end-to-end time from a keystroke entering the editor's
//     input handler until the resulting document repaint completes. This is
//     measured by an in-order approximation (see [TypingSample]).
//   - Engine internal phases: how the engine spent the JavaScript-side slice
//     of a command's round-trip, reported by the engine as per-phase
//     durations (see [EngineTimings]). Unlike the three above — which the
//     port measures itself — these durations originate in the engine. The
//     port records them so the overlay can show where the round-trip's time
//     actually goes (transport vs engine compute), decomposing the round-trip
//     number the port already measures.
//
// [BridgeMetrics] is a plain mutable holder. The bridge owns one instance,
// records into it, and exposes it (plus a change stream) so the overlay can
// render live values.

import 'protocol_constants.dart';

/// A single command round-trip measurement.
///
/// Captures how long one command took from send to correlated response,
/// keyed by the command's name so the overlay can aggregate per command type.
class RoundTripSample {
  /// The command name (e.g., "insertText", "exec", "setTextSelection").
  final String commandName;

  /// The round-trip duration in milliseconds.
  final double milliseconds;

  /// When the sample was recorded (response arrival time).
  final DateTime timestamp;

  const RoundTripSample({
    required this.commandName,
    required this.milliseconds,
    required this.timestamp,
  });

  @override
  String toString() =>
      'RoundTripSample($commandName: ${milliseconds.toStringAsFixed(1)}ms)';
}

/// A single engine load-phase measurement.
///
/// Each phase is the time spent between two consecutive lifecycle state
/// transitions during initialization. The set of phases together describes
/// where cold-start time goes.
class LoadPhase {
  /// A human-readable phase name (e.g., "loading -> pageLoaded").
  final String name;

  /// The phase duration in milliseconds.
  final double milliseconds;

  const LoadPhase({required this.name, required this.milliseconds});

  @override
  String toString() => 'LoadPhase($name: ${milliseconds.toStringAsFixed(1)}ms)';
}

/// A single typing-latency measurement.
///
/// Measures end-to-end time from a keystroke entering the editor until the
/// resulting repaint completes. Because the engine does not (yet) echo a
/// correlation token on the stateChanged event that a keystroke produces,
/// the port pairs keystrokes to repaints by in-order approximation: the
/// oldest pending keystroke is matched to the next repaint. [exact] records
/// whether this sample came from an exact correlation (a future engine
/// token) or the approximation, so the overlay can be honest about precision.
class TypingSample {
  /// The kind of input that produced this sample ("insert", "delete",
  /// "newline").
  final String operation;

  /// The end-to-end duration in milliseconds.
  final double milliseconds;

  /// Whether this measurement came from an exact correlation token (true)
  /// or the in-order approximation (false). Always false until the engine
  /// emits a causedBy token on stateChanged.
  final bool exact;

  /// When the sample was recorded (repaint completion time).
  final DateTime timestamp;

  const TypingSample({
    required this.operation,
    required this.milliseconds,
    required this.exact,
    required this.timestamp,
  });

  @override
  String toString() =>
      'TypingSample($operation: ${milliseconds.toStringAsFixed(1)}ms, '
      'exact: $exact)';
}

/// A parsed engine-side timing breakdown for a single message.
///
/// The engine optionally attaches a `timings` object to responses and to
/// stateChanged events, mapping phase name to elapsed milliseconds. This
/// class reads that sparse map through [TimingPhase] constants into typed,
/// nullable fields — a field is null when the engine did not measure that
/// phase for the message (a response carries [handle]; a stateChanged carries
/// the build breakdown).
///
/// These are durations the engine reports, not measurements the port takes.
/// They are kept distinct from [RoundTripSample] for that reason: the
/// round-trip is the port's end-to-end number, and these durations explain
/// how much of it the engine spent inside JavaScript.
class EngineTimings {
  /// Total engine handler time for the command (entry to response send).
  final double? handle;

  /// The document-tree serialization walk.
  final double? serializeDoc;

  /// The command-state sweep (canExec + isActive for every command).
  final double? commandStates;

  /// The canExec half of the command-state sweep alone (a sub-phase of
  /// [commandStates]). Null unless the engine is reporting the split.
  final double? commandStatesCan;

  /// The isActive half of the command-state sweep alone (a sub-phase of
  /// [commandStates]). Null unless the engine is reporting the split.
  final double? commandStatesActive;

  /// The combined active marks/nodes/stored-marks extraction.
  final double? active;

  /// The change-detection JSON.stringify of the document.
  final double? docDiff;

  /// Total onTransaction time (build + diff + sends).
  final double? total;

  const EngineTimings({
    this.handle,
    this.serializeDoc,
    this.commandStates,
    this.commandStatesCan,
    this.commandStatesActive,
    this.active,
    this.docDiff,
    this.total,
  });

  /// Parse a raw `timings` map from a response or stateChanged event. Reads
  /// each phase through its [TimingPhase] key, coercing the JSON number
  /// (which may arrive as int or double) to double. Missing keys stay null.
  factory EngineTimings.fromJson(Map<String, dynamic> json) {
    double? read(String key) {
      final value = json[key];
      if (value is num) return value.toDouble();
      return null;
    }

    return EngineTimings(
      handle: read(TimingPhase.handle),
      serializeDoc: read(TimingPhase.serializeDoc),
      commandStates: read(TimingPhase.commandStates),
      commandStatesCan: read(TimingPhase.commandStatesCan),
      commandStatesActive: read(TimingPhase.commandStatesActive),
      active: read(TimingPhase.active),
      docDiff: read(TimingPhase.docDiff),
      total: read(TimingPhase.total),
    );
  }

  /// The phase durations as a name→ms map, including only phases that were
  /// actually present. Used by [BridgeMetrics] to fold each phase into its
  /// rolling stats without enumerating fields at the call site.
  Map<String, double> get presentPhases {
    final result = <String, double>{};
    if (handle != null) result[TimingPhase.handle] = handle!;
    if (serializeDoc != null) result[TimingPhase.serializeDoc] = serializeDoc!;
    if (commandStates != null) {
      result[TimingPhase.commandStates] = commandStates!;
    }
    if (commandStatesCan != null) {
      result[TimingPhase.commandStatesCan] = commandStatesCan!;
    }
    if (commandStatesActive != null) {
      result[TimingPhase.commandStatesActive] = commandStatesActive!;
    }
    if (active != null) result[TimingPhase.active] = active!;
    if (docDiff != null) result[TimingPhase.docDiff] = docDiff!;
    if (total != null) result[TimingPhase.total] = total!;
    return result;
  }

  /// Whether any phase was present in the parsed map.
  bool get isEmpty =>
      handle == null &&
      serializeDoc == null &&
      commandStates == null &&
      commandStatesCan == null &&
      commandStatesActive == null &&
      active == null &&
      docDiff == null &&
      total == null;

  @override
  String toString() => 'EngineTimings($presentPhases)';
}

/// Rolling aggregate statistics over a series of millisecond measurements.
///
/// Tracks count, last, min, max, and a running mean without retaining every
/// sample. Used for per-command round-trip stats, typing latency, and
/// per-phase engine timings.
class RollingStats {
  /// Number of samples recorded.
  int count = 0;

  /// The most recent sample value in milliseconds.
  double last = 0;

  /// The smallest sample value seen, in milliseconds.
  double min = double.infinity;

  /// The largest sample value seen, in milliseconds.
  double max = 0;

  /// Running sum, used to compute the mean. Kept private to callers via [mean].
  double _sum = 0;

  /// The arithmetic mean of all samples in milliseconds, or 0 if none.
  double get mean => count == 0 ? 0 : _sum / count;

  /// Record a new measurement, updating all aggregates.
  void add(double ms) {
    count++;
    last = ms;
    _sum += ms;
    if (ms < min) min = ms;
    if (ms > max) max = ms;
  }

  /// The minimum value, or 0 if no samples have been recorded (avoids
  /// surfacing the sentinel infinity to the UI).
  double get displayMin => count == 0 ? 0 : min;
}

/// Mutable holder for all port-side performance metrics.
///
/// The bridge owns one instance and records into it as commands complete,
/// lifecycle transitions occur, the editor reports typing samples, and the
/// engine reports phase timings. The overlay reads from it to render live
/// values. A change callback ([onChange]) lets the bridge notify listeners
/// (via its own stream) whenever a new sample lands.
class BridgeMetrics {
  /// Maximum number of recent round-trip samples retained in the ring buffer.
  static const int _maxRoundTrips = 100;

  /// Maximum number of recent typing samples retained in the ring buffer.
  static const int _maxTypingSamples = 100;

  /// Recent round-trip samples, newest last. Bounded to [_maxRoundTrips].
  final List<RoundTripSample> _roundTrips = [];
  List<RoundTripSample> get roundTrips => List.unmodifiable(_roundTrips);

  /// Per-command rolling round-trip statistics, keyed by command name.
  final Map<String, RollingStats> _commandStats = {};
  Map<String, RollingStats> get commandStats => Map.unmodifiable(_commandStats);

  /// Engine load phases, in the order they occurred.
  final List<LoadPhase> _loadPhases = [];
  List<LoadPhase> get loadPhases => List.unmodifiable(_loadPhases);

  /// Total engine cold-start time in milliseconds, set once the engine
  /// reaches the ready state. Null until then.
  double? totalLoadMs;

  /// Recent typing samples, newest last. Bounded to [_maxTypingSamples].
  final List<TypingSample> _typingSamples = [];
  List<TypingSample> get typingSamples => List.unmodifiable(_typingSamples);

  /// Rolling statistics over all typing samples.
  final RollingStats typingStats = RollingStats();

  /// Number of keystrokes whose latency could not be measured because the
  /// in-order approximation was ambiguous at the time. A high count relative
  /// to typing volume signals that the approximation is blind in this session
  /// and an exact correlation token would be worth adding on the engine side.
  int droppedTypingSamples = 0;

  /// Per-phase rolling statistics over engine-reported timings, keyed by
  /// phase name ([TimingPhase] values). Populated as responses and
  /// stateChanged events carrying a `timings` object arrive. This is the
  /// decomposition of the round-trip: comparing the [TimingPhase.handle]
  /// stats here against the per-command round-trip stats above shows how much
  /// of the round-trip is engine compute versus transport.
  final Map<String, RollingStats> _enginePhaseStats = {};
  Map<String, RollingStats> get enginePhaseStats =>
      Map.unmodifiable(_enginePhaseStats);

  /// Optional callback invoked after any sample is recorded, so the owner
  /// (the bridge) can emit a change notification on its metrics stream.
  void Function()? onChange;

  /// Record a command round-trip and update the per-command stats.
  void recordRoundTrip(String commandName, double ms) {
    final sample = RoundTripSample(
      commandName: commandName,
      milliseconds: ms,
      timestamp: DateTime.now(),
    );
    _roundTrips.add(sample);
    if (_roundTrips.length > _maxRoundTrips) {
      _roundTrips.removeRange(0, _roundTrips.length - _maxRoundTrips);
    }
    (_commandStats[commandName] ??= RollingStats()).add(ms);
    onChange?.call();
  }

  /// Record a completed load phase.
  void recordLoadPhase(String name, double ms) {
    _loadPhases.add(LoadPhase(name: name, milliseconds: ms));
    onChange?.call();
  }

  /// Record a typing-latency sample.
  void recordTypingSample(String operation, double ms, {required bool exact}) {
    final sample = TypingSample(
      operation: operation,
      milliseconds: ms,
      exact: exact,
      timestamp: DateTime.now(),
    );
    _typingSamples.add(sample);
    if (_typingSamples.length > _maxTypingSamples) {
      _typingSamples.removeRange(0, _typingSamples.length - _maxTypingSamples);
    }
    typingStats.add(ms);
    onChange?.call();
  }

  /// Record that a keystroke's latency could not be measured.
  void recordDroppedTypingSample() {
    droppedTypingSamples++;
    onChange?.call();
  }

  /// Fold an engine-reported timing breakdown into the per-phase rolling
  /// stats. Each present phase updates its own [RollingStats]. Called by the
  /// bridge whenever a response or stateChanged carries a `timings` object;
  /// an empty breakdown is ignored so disabled engine metrics record nothing.
  void recordEngineTimings(EngineTimings timings) {
    if (timings.isEmpty) return;
    for (final entry in timings.presentPhases.entries) {
      (_enginePhaseStats[entry.key] ??= RollingStats()).add(entry.value);
    }
    onChange?.call();
  }
}
