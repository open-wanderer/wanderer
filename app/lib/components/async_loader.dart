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
      _ => mockData,
    };

    return Skeletonizer(
      enabled: !asyncValue.hasValue,
      enableSwitchAnimation: true,
      child: builder(data),
    );
  }
}
