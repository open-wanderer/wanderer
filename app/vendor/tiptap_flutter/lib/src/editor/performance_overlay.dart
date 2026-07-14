// Performance overlay that shows live timing metrics for the editor as a
// draggable bottom sheet.
//
// This replaces the former debug overlay. Document, selection, and raw-state
// inspection happen through terminal logs (the bridge still logs everything
// to the console); this overlay is dedicated to the information the terminal
// cannot easily show: paired timings and rolling statistics. It reads from
// the bridge's [BridgeMetrics] and rebuilds whenever a new sample lands.
//
// Four panels of information are shown:
//   - Engine load: per-phase cold-start breakdown and total.
//   - Command round-trips: per-command count, last, mean, min, and max.
//   - Engine phases: how the engine spent the JavaScript-side slice of a
//     command's round-trip, broken down by internal phase. The handle phase
//     is the engine's total in-JS time; comparing its mean against a
//     command's round-trip mean shows how much of the round-trip is engine
//     compute versus transport. The commandStates phase is the prime suspect
//     for per-keystroke cost.
//   - Typing latency: end-to-end keystroke-to-repaint statistics, labeled as
//     an in-order approximation, with a dropped-sample count.

import 'dart:async';

import 'package:flutter/material.dart';

import '../engine/metrics.dart';
import '../engine/protocol_constants.dart';
import 'editor_controller.dart';

/// A draggable bottom sheet overlay showing live performance metrics for the
/// editor: engine load phases, command round-trips, engine internal phases,
/// and typing latency.
///
/// Named with the Tiptap prefix to avoid colliding with Flutter's own
/// [PerformanceOverlay] widget, which is in scope wherever material.dart is
/// imported.
class TiptapPerformanceOverlay extends StatefulWidget {
  /// The controller whose bridge metrics are displayed.
  final EditorController controller;

  /// Called when the user closes the overlay.
  final VoidCallback onClose;

  const TiptapPerformanceOverlay({
    super.key,
    required this.controller,
    required this.onClose,
  });

  @override
  State<TiptapPerformanceOverlay> createState() =>
      _TiptapPerformanceOverlayState();
}

class _TiptapPerformanceOverlayState extends State<TiptapPerformanceOverlay> {
  /// Subscription to the metrics change stream, used to rebuild on new samples.
  StreamSubscription? _metricsSubscription;

  @override
  void initState() {
    super.initState();

    /// Rebuild whenever a new metric sample is recorded so the displayed
    /// values stay live.
    _metricsSubscription = widget.controller.metricsStream.listen((_) {
      if (mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    _metricsSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final metrics = widget.controller.metrics;

    return DraggableScrollableSheet(
      initialChildSize: 0.45,
      minChildSize: 0.1,
      maxChildSize: 0.9,
      builder: (context, scrollController) {
        return Container(
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surface,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withAlpha(51),
                blurRadius: 10,
                offset: const Offset(0, -2),
              ),
            ],
          ),
          child: Column(
            children: [
              /// Drag handle and close button.
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Row(
                  children: [
                    const SizedBox(width: 48),
                    const Spacer(),
                    Container(
                      width: 32,
                      height: 4,
                      decoration: BoxDecoration(
                        color: Theme.of(context).colorScheme.outline,
                        borderRadius: BorderRadius.circular(2),
                      ),
                    ),
                    const Spacer(),
                    IconButton(
                      icon: const Icon(Icons.close, size: 20),
                      onPressed: widget.onClose,
                    ),
                  ],
                ),
              ),

              /// Title.
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Align(
                  alignment: Alignment.centerLeft,
                  child: Text(
                    'Performance',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
                  ),
                ),
              ),

              /// Metrics content.
              Expanded(
                child: ListView(
                  controller: scrollController,
                  padding: const EdgeInsets.all(16),
                  children: [
                    _buildLoadSection(metrics),
                    const SizedBox(height: 24),
                    _buildRoundTripSection(metrics),
                    const SizedBox(height: 24),
                    _buildEnginePhaseSection(metrics),
                    const SizedBox(height: 24),
                    _buildTypingSection(metrics),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  // ---------------------------------------------------------------------------
  // Engine load section
  // ---------------------------------------------------------------------------

  /// Build the engine cold-start breakdown: each lifecycle phase and the total.
  Widget _buildLoadSection(BridgeMetrics metrics) {
    final phases = metrics.loadPhases;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _sectionHeader('Engine load'),
        if (phases.isEmpty)
          _emptyHint('No load data yet.')
        else ...[
          for (final phase in phases)
            _metricRow(phase.name, _ms(phase.milliseconds)),
          const Divider(height: 16),
          _metricRow(
            'Total cold start',
            metrics.totalLoadMs != null ? _ms(metrics.totalLoadMs!) : '—',
            emphasize: true,
          ),
        ],
      ],
    );
  }

  // ---------------------------------------------------------------------------
  // Command round-trip section
  // ---------------------------------------------------------------------------

  /// Build the per-command round-trip table: count, last, mean, min, max.
  Widget _buildRoundTripSection(BridgeMetrics metrics) {
    final stats = metrics.commandStats;

    /// Sort command names for stable ordering in the table.
    final names = stats.keys.toList()..sort();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _sectionHeader('Command round-trips'),
        if (names.isEmpty)
          _emptyHint('No commands sent yet.')
        else ...[
          /// Column headers.
          _statsHeaderRow('command'),
          const SizedBox(height: 4),
          for (final name in names) _statsRow(name, stats[name]!),
        ],
      ],
    );
  }

  // ---------------------------------------------------------------------------
  // Engine phase section
  // ---------------------------------------------------------------------------

  /// Build the engine internal-phase table: how the engine spent the
  /// JavaScript-side slice of the round-trip, per phase.
  ///
  /// The phases are shown in a fixed, meaningful order rather than sorted
  /// alphabetically: handle first (the engine's total in-JS time, the figure
  /// to compare against round-trip mean), then the build sub-phases in the
  /// order the engine runs them, with commandStates — the prime suspect —
  /// among them. The note above the table makes the transport-vs-compute
  /// reading explicit so the panel interprets itself.
  Widget _buildEnginePhaseSection(BridgeMetrics metrics) {
    final stats = metrics.enginePhaseStats;

    /// Fixed display order. Only phases that actually have samples render, so
    /// a response-only session shows just handle, and a typing session shows
    /// the full build breakdown. The two commandStates sub-phases sit directly
    /// under commandStates so the table reads as a breakdown: the can/active
    /// split shows which half of the sweep dominates.
    const order = <String>[
      TimingPhase.handle,
      TimingPhase.serializeDoc,
      TimingPhase.commandStates,
      TimingPhase.commandStatesCan,
      TimingPhase.commandStatesActive,
      TimingPhase.active,
      TimingPhase.docDiff,
      TimingPhase.total,
    ];

    final present = [
      for (final p in order)
        if (stats.containsKey(p)) p,
    ];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _sectionHeader('Engine phases'),

        /// Reading note: the panel interprets itself so the numbers aren't
        /// ambiguous. handle is the engine's whole in-JS cost; the gap between
        /// it and a command's round-trip mean is transport overhead.
        _emptyHint(
          'How the engine spends its slice of the round-trip (ms). '
          '"handle" is total engine time; round-trip minus handle is '
          'transport. "commandStates" is the per-keystroke suspect.',
        ),
        const SizedBox(height: 8),
        if (present.isEmpty)
          _emptyHint(
            'No engine timings yet. They arrive once a command round-trip '
            'has completed — type or run a command to populate this.',
          )
        else ...[
          _statsHeaderRow('phase'),
          const SizedBox(height: 4),
          for (final phase in present)
            _statsRow(
              _phaseLabel(phase),
              stats[phase]!,
              emphasizeName: phase == TimingPhase.commandStates,
            ),
        ],
      ],
    );
  }

  /// Map a phase key to its display label. The two commandStates sub-phases
  /// are shown indented and named by the call they measure, so the table
  /// reads as a breakdown of the commandStates row above them rather than as
  /// three unrelated rows. All other phases display their raw key.
  String _phaseLabel(String phase) {
    switch (phase) {
      case TimingPhase.commandStatesCan:
        return '  ↳ can()';
      case TimingPhase.commandStatesActive:
        return '  ↳ isActive()';
      default:
        return phase;
    }
  }

  // ---------------------------------------------------------------------------
  // Typing latency section
  // ---------------------------------------------------------------------------

  /// Build the typing-latency panel with rolling stats and the approximation
  /// caveat.
  Widget _buildTypingSection(BridgeMetrics metrics) {
    final stats = metrics.typingStats;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _sectionHeader('Typing latency'),

        /// Honesty note: this is an in-order approximation, not an exact
        /// measurement, until the engine echoes a correlation token.
        _emptyHint(
          'End-to-end keystroke to repaint. Approximate (in-order pairing); '
          'ambiguous samples are dropped, not guessed.',
        ),
        const SizedBox(height: 8),
        if (stats.count == 0)
          _emptyHint('No keystrokes measured yet.')
        else ...[
          _metricRow('Samples', '${stats.count}'),
          _metricRow('Last', _ms(stats.last)),
          _metricRow('Mean', _ms(stats.mean), emphasize: true),
          _metricRow('Min', _ms(stats.displayMin)),
          _metricRow('Max', _ms(stats.max)),
        ],
        _metricRow('Dropped (ambiguous)', '${metrics.droppedTypingSamples}'),
      ],
    );
  }

  // ---------------------------------------------------------------------------
  // Shared building blocks
  // ---------------------------------------------------------------------------

  /// Header row for a stats table whose first column is named [firstColumn]
  /// (e.g., "command" or "phase"), followed by the n/last/mean/min/max columns.
  /// Shared by the round-trip and engine-phase tables so their columns align.
  Widget _statsHeaderRow(String firstColumn) {
    const headerStyle = TextStyle(
      fontSize: 11,
      fontWeight: FontWeight.w600,
      color: Colors.grey,
    );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          Expanded(flex: 4, child: Text(firstColumn, style: headerStyle)),
          const Expanded(
            flex: 2,
            child: Text('n', style: headerStyle, textAlign: TextAlign.right),
          ),
          const Expanded(
            flex: 3,
            child: Text('last', style: headerStyle, textAlign: TextAlign.right),
          ),
          const Expanded(
            flex: 3,
            child: Text('mean', style: headerStyle, textAlign: TextAlign.right),
          ),
          const Expanded(
            flex: 3,
            child: Text('min', style: headerStyle, textAlign: TextAlign.right),
          ),
          const Expanded(
            flex: 3,
            child: Text('max', style: headerStyle, textAlign: TextAlign.right),
          ),
        ],
      ),
    );
  }

  /// A single stats row (count, last, mean, min, max) labeled by [name].
  /// Shared by the round-trip and engine-phase tables. [emphasizeName] bolds
  /// the row label to flag a phase of interest (e.g., commandStates).
  Widget _statsRow(
    String name,
    RollingStats stats, {
    bool emphasizeName = false,
  }) {
    const cellStyle = TextStyle(fontFamily: 'monospace', fontSize: 12);
    Widget cell(String text, int flex, {bool isName = false}) => Expanded(
      flex: flex,
      child: Text(
        text,
        style: isName && emphasizeName
            ? cellStyle.copyWith(fontWeight: FontWeight.w700)
            : cellStyle,
        textAlign: isName ? TextAlign.left : TextAlign.right,
        overflow: TextOverflow.ellipsis,
      ),
    );

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          cell(name, 4, isName: true),
          cell('${stats.count}', 2),
          cell(_ms(stats.last), 3),
          cell(_ms(stats.mean), 3),
          cell(_ms(stats.displayMin), 3),
          cell(_ms(stats.max), 3),
        ],
      ),
    );
  }

  /// A section header label.
  Widget _sectionHeader(String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(
        text,
        style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
      ),
    );
  }

  /// A labeled metric row with the value right-aligned.
  Widget _metricRow(String label, String value, {bool emphasize = false}) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          Expanded(
            child: Text(
              label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: emphasize ? FontWeight.w600 : FontWeight.normal,
              ),
            ),
          ),
          Text(
            value,
            style: TextStyle(
              fontFamily: 'monospace',
              fontSize: 13,
              fontWeight: emphasize ? FontWeight.w700 : FontWeight.normal,
            ),
          ),
        ],
      ),
    );
  }

  /// A muted hint line for empty states and notes.
  Widget _emptyHint(String text) {
    return Text(
      text,
      style: const TextStyle(fontSize: 12, color: Colors.grey, height: 1.4),
    );
  }

  /// Format a millisecond value for display.
  String _ms(double ms) => '${ms.toStringAsFixed(1)} ms';
}
