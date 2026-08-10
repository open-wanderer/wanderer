import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:skeletonizer/skeletonizer.dart';
import 'package:wanderer/components/base/wanderer_error.dart';

class AsyncLoader<T> extends StatelessWidget {
  const AsyncLoader({
    required this.asyncValue,
    required this.mockData,
    required this.builder,
    this.enableSwitchAnimation = true,
    super.key,
  });

  final AsyncValue<T> asyncValue;
  final T mockData;
  final Widget Function(T data) builder;
  Widget errorBuilder(Object err) => WandererError(err: err);

  final bool enableSwitchAnimation;

  @override
  Widget build(BuildContext context) {
    if (asyncValue.hasError) {
      return errorBuilder(asyncValue.error!);
    }

    final data = switch (asyncValue) {
      AsyncData<T>(value: final v) => v,
      _ => asyncValue.value ?? mockData,
    };

    return Skeletonizer(
      enabled: asyncValue.isLoading,
      enableSwitchAnimation: true,
      switchAnimationConfig: const SwitchAnimationConfig(
        layoutBuilder: _switchLayoutBuilder,
      ),
      child: builder(data),
    );
  }
}

/// Custom [AnimatedSwitcher] layout builder for [Skeletonizer]'s
/// skeleton-to-content cross-fade.
///
/// The package default (`AnimatedSwitcher.defaultLayoutBuilder`) overlays the
/// switched children in a `Stack(alignment: Alignment.center)` under the
/// default `StackFit.loose`. That Stack is present for the lifetime of the
/// widget -- not just while a transition is animating -- so it permanently
/// loosens whatever constraints this `Skeletonizer` receives, even tight
/// ones from a fixed-height parent (e.g. `trail_detail_screen.dart`'s
/// `Positioned.fill > Padding`). A `SingleChildScrollView` responds to those
/// loosened constraints by shrink-wrapping to its actual content height
/// instead of filling the available space, and the Stack then centers that
/// shrunk box inside itself -- producing an empty band above (and below)
/// whenever content is shorter than the panel (e.g. `TrailPanel`'s Summit
/// Book / Comments tabs vs. the taller About tab).
///
/// `StackFit.passthrough` hands the Stack's own incoming constraints straight
/// to its child unmodified: tight constraints stay tight (forcing full-height
/// fill, eliminating the shrink-wrap/centering artifact), while call sites
/// that already sit inside a scrollable (already-loose or unbounded
/// constraints) are unaffected, since `passthrough`'s allowed range is always
/// a subset of `loose`'s.
Widget _switchLayoutBuilder(
  Widget? currentChild,
  List<Widget> previousChildren,
) {
  return Stack(
    fit: StackFit.passthrough,
    children: [...previousChildren, ?currentChild],
  );
}
