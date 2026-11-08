import 'package:flutter/material.dart';

/// A [RangeSlider] that keeps the in-progress selection in local state and only
/// reports it once the user lets go of the thumb.
///
/// Dragging a plain [RangeSlider] fires `onChanged` on every frame, so wiring a
/// filter update straight into it triggers a request per frame. Here the drag
/// only repaints; [onChangeEnd] is the single commit point.
class WandererRangeSlider extends StatefulWidget {
  const WandererRangeSlider({
    super.key,
    required this.values,
    required this.min,
    required this.max,
    required this.onChangeEnd,
    this.labelsBuilder,
    this.divisions,
  });

  final RangeValues values;
  final double min;
  final double max;

  /// Called once, with the final selection, when the drag ends.
  final ValueChanged<RangeValues> onChangeEnd;

  /// Builds the labels for the currently displayed values (drag values while
  /// the user is dragging).
  final RangeLabels Function(RangeValues values)? labelsBuilder;

  final int? divisions;

  @override
  State<WandererRangeSlider> createState() => _WandererRangeSliderState();
}

class _WandererRangeSliderState extends State<WandererRangeSlider> {
  /// Non-null while the local selection has not been reflected back by the
  /// owner yet — during the drag, and between the commit and the rebuild it
  /// causes.
  RangeValues? _localValues;

  @override
  void didUpdateWidget(covariant WandererRangeSlider oldWidget) {
    super.didUpdateWidget(oldWidget);
    // The owner moved the values (committed our drag, or reset the filter),
    // so it takes over again.
    if (widget.values != oldWidget.values) {
      _localValues = null;
    }
  }

  RangeValues get _displayedValues {
    final values = _localValues ?? widget.values;
    final start = values.start.clamp(widget.min, widget.max);
    final end = values.end.clamp(start, widget.max);
    return RangeValues(start, end);
  }

  @override
  Widget build(BuildContext context) {
    final values = _displayedValues;
    return RangeSlider(
      values: values,
      min: widget.min,
      max: widget.max,
      divisions: widget.divisions,
      labels: widget.labelsBuilder?.call(values),
      onChanged: (values) => setState(() => _localValues = values),
      onChangeEnd: (values) {
        setState(() => _localValues = values);
        widget.onChangeEnd(values);
      },
    );
  }
}
