import 'dart:async';

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
  /// Auto-dismiss timers keyed by toast id — cancellable [Timer]s, not
  /// fire-and-forget `Future.delayed`s: a manually-dismissed toast's delayed
  /// callback used to outlive the toast (and, in principle, the provider)
  /// as an uncancellable pending timer per toast shown.
  final Map<String, Timer> _dismissTimers = {};

  @override
  List<ToastMessage> build() {
    ref.onDispose(() {
      for (final timer in _dismissTimers.values) {
        timer.cancel();
      }
      _dismissTimers.clear();
    });
    return [];
  }

  void add(ToastMessage toast) {
    state = [...state, toast];

    _dismissTimers[toast.id]?.cancel();
    _dismissTimers[toast.id] = Timer(const Duration(seconds: 4), () {
      remove(toast.id);
    });
  }

  void remove(String id) {
    _dismissTimers.remove(id)?.cancel();
    state = [
      for (final t in state)
        if (t.id != id) t,
    ];
  }
}
