import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';

class WandererButton extends StatelessWidget {
  final FaIconData? icon;
  final String? tooltip;
  final VoidCallback? onPressed;
  final Widget? child;
  final bool primary;
  final bool secondary;
  final bool loading;
  final bool disabled;
  final bool large;

  const WandererButton({
    super.key,
    this.icon,
    this.tooltip,
    this.onPressed,
    this.child,
    this.primary = false,
    this.secondary = false,
    this.loading = false,
    this.disabled = false,
    this.large = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    ButtonStyle buttonStyle;

    if (primary) {
      buttonStyle =
          ElevatedButton.styleFrom(
            backgroundColor: theme.colorScheme.primary,
            foregroundColor: theme.colorScheme.onPrimary,
            minimumSize: Size(0, large ? 56 : 40),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
            ),
            elevation: 0,
          ).copyWith(
            foregroundColor: WidgetStateProperty.resolveWith((states) {
              if (states.contains(WidgetState.disabled)) {
                return theme.colorScheme.onInverseSurface.withValues(
                  alpha: 0.48,
                );
              }
              return theme.colorScheme.onPrimary;
            }),
            backgroundColor: WidgetStateProperty.resolveWith((states) {
              if (states.contains(WidgetState.hovered)) {
                return theme.colorScheme.primaryContainer;
              }
              return theme.colorScheme.primary;
            }),
          );
    } else if (secondary) {
      buttonStyle =
          OutlinedButton.styleFrom(
            foregroundColor: theme.colorScheme.onSurface,
            side: BorderSide(color: Theme.of(context).primaryColor),
            minimumSize: Size(0, large ? 56 : 40),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
            ),
          ).copyWith(
            backgroundColor: WidgetStateProperty.resolveWith((states) {
              if (states.contains(WidgetState.hovered)) {
                return theme.colorScheme.secondaryContainer;
              }
              return Colors.transparent;
            }),
          );
    } else {
      buttonStyle = TextButton.styleFrom();
    }

    Widget button = _buildButton(context, buttonStyle);

    if (tooltip != null && tooltip!.isNotEmpty) {
      return Tooltip(message: tooltip!, child: button);
    }

    return button;
  }

  Widget _buildButton(BuildContext context, ButtonStyle style) {
    final bool isEnabled = !disabled && !loading && onPressed != null;

    return Stack(
      alignment: Alignment.center,
      children: [
        SizedBox(
          width: double.infinity,
          child: Opacity(
            opacity: loading ? 0.0 : 1.0,
            child: primary
                ? ElevatedButton(
                    onPressed: isEnabled ? onPressed : null,
                    style: style,
                    child: _buildContent(),
                  )
                : OutlinedButton(
                    onPressed: isEnabled ? onPressed : null,
                    style: style,
                    child: _buildContent(),
                  ),
          ),
        ),

        if (loading)
          Center(
            child: SizedBox(
              width: 20,
              height: 20,
              child: CircularProgressIndicator(
                strokeWidth: 3,
                valueColor: AlwaysStoppedAnimation<Color>(
                  Theme.of(context).colorScheme.onSurface,
                ),
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildContent() {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (icon != null) ...[FaIcon(icon, size: 16), const SizedBox(width: 8)],
        ?child,
      ],
    );
  }
}
