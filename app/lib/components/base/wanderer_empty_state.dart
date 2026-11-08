import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_button.dart';

/// Placeholder shown where a list would be, with an optional next step.
///
/// Always scrollable, even when the content fits: these sit inside a
/// `RefreshIndicator`, which needs a scrollable descendant to pull on.
class WandererEmptyState extends StatelessWidget {
  final FaIconData icon;
  final String title;
  final String body;
  final String? actionLabel;
  final VoidCallback? onAction;

  const WandererEmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.body,
    this.actionLabel,
    this.onAction,
  }) : assert((actionLabel == null) == (onAction == null));

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = theme.colorScheme.onSurfaceVariant;

    return LayoutBuilder(
      builder: (context, constraints) => SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        child: ConstrainedBox(
          constraints: BoxConstraints(
            minWidth: constraints.maxWidth,
            minHeight: math.max(
              0,
              constraints.maxHeight - MediaQuery.paddingOf(context).bottom,
            ),
          ),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24.0),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                FaIcon(
                  icon,
                  size: 64,
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
                ),
                const SizedBox(height: 16),
                Text(
                  title,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.titleMedium!.copyWith(color: color),
                ),
                const SizedBox(height: 8),
                ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 360),
                  child: Text(
                    body,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodyMedium!.copyWith(color: color),
                  ),
                ),
                if (actionLabel != null) ...[
                  const SizedBox(height: 24),
                  // WandererButton fills its parent; this is an optional next
                  // step, so it stays the size of its label.
                  IntrinsicWidth(
                    child: WandererButton(
                      onPressed: onAction,
                      child: Text(actionLabel!),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}
