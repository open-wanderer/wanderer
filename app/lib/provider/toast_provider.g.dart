// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'toast_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(Toast)
final toastProvider = ToastProvider._();

final class ToastProvider extends $NotifierProvider<Toast, List<ToastMessage>> {
  ToastProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'toastProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$toastHash();

  @$internal
  @override
  Toast create() => Toast();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<ToastMessage> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<ToastMessage>>(value),
    );
  }
}

String _$toastHash() => r'e04371162987c3be8537db14c22d7f32c28cf4a7';

abstract class _$Toast extends $Notifier<List<ToastMessage>> {
  List<ToastMessage> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<ToastMessage>, List<ToastMessage>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<ToastMessage>, List<ToastMessage>>,
              List<ToastMessage>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
