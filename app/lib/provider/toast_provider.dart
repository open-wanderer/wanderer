import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toast_provider.g.dart';

enum ToastType { info, success, warning, error }

// Monotonic counter, not a timestamp: two toasts queued within the same
// millisecond (e.g. two toasts from a single save-result callback) must never
// collide, or `remove()` below deletes both when the first one's timer fires.
int _nextToastId = 0;

class ToastMessage {
  final String id;
  final ToastType type;
  final FaIconData icon;
  final String text;

  ToastMessage({
    String? id,
    required this.type,
    required this.icon,
    required this.text,
  }) : id = id ?? (_nextToastId++).toString();
}

@Riverpod(keepAlive: true)
class Toast extends _$Toast {
  @override
  List<ToastMessage> build() => [];

  void add(ToastMessage toast) {
    state = [...state, toast];

    Future.delayed(const Duration(seconds: 4), () {
      remove(toast.id);
    });
  }

  void remove(String id) {
    state = [
      for (final t in state)
        if (t.id != id) t,
    ];
  }
}
