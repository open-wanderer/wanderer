import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/provider/toast_provider.dart';

// toast_overlay.dart
class ToastOverlay extends ConsumerWidget {
  final Widget child;
  const ToastOverlay({super.key, required this.child});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Listen for new toasts in the state
    ref.listen(toastProvider, (previous, next) {
      // If the list grew, show the latest one
      if (next.length > (previous?.length ?? 0)) {
        final toast = next.last;
        _displayToast(context, toast);
      }
    });

    return child;
  }

  void _displayToast(BuildContext context, ToastMessage toast) {
    // Use the ScaffoldMessenger that MaterialApp provides
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        behavior: SnackBarBehavior.floating,
        backgroundColor: _getColor(toast.type),
        content: Row(
          children: [
            FaIcon(toast.icon, color: Colors.white, size: 16),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                toast.text,
                style: const TextStyle(color: Colors.white),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Color _getColor(ToastType type) {
    switch (type) {
      case ToastType.error:
        return Colors.red.shade600;
      case ToastType.success:
        return Colors.green.shade600;
      case ToastType.warning:
        return Colors.orange.shade700;
      case ToastType.info:
        return Colors.blue.shade600;
    }
  }
}
