import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toast_provider.g.dart';

enum ToastType { info, success, warning, error }

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
  }) : id = id ?? DateTime.now().millisecondsSinceEpoch.toString();
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
