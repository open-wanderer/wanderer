import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/provider/toast_provider.dart';

class ToastOverlay extends ConsumerWidget {
  final Widget child;
  const ToastOverlay({super.key, required this.child});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.listen(toastProvider, (previous, next) {
      if (next.length > (previous?.length ?? 0)) {
        final toast = next.last;
        _displayToast(context, toast);
      }
    });

    return child;
  }

  void _displayToast(BuildContext context, ToastMessage toast) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        behavior: SnackBarBehavior.floating,
        content: Row(
          children: [
            FaIcon(toast.icon, color: _getColor(toast.type), size: 16),
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
        return Colors.redAccent;
      case ToastType.success:
        return Colors.greenAccent;
      case ToastType.warning:
        return Colors.orangeAccent;
      case ToastType.info:
        return Colors.blueAccent;
    }
  }
}
