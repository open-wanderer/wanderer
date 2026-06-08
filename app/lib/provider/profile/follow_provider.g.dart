// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'follow_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(FollowNotifier)
final followProvider = FollowNotifierFamily._();

final class FollowNotifierProvider
    extends $AsyncNotifierProvider<FollowNotifier, FollowState> {
  FollowNotifierProvider._({
    required FollowNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'followProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$followNotifierHash();

  @override
  String toString() {
    return r'followProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  FollowNotifier create() => FollowNotifier();

  @override
  bool operator ==(Object other) {
    return other is FollowNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$followNotifierHash() => r'5abed6569f0ad25635ee50889583614f8e4c2303';

final class FollowNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          FollowNotifier,
          AsyncValue<FollowState>,
          FollowState,
          FutureOr<FollowState>,
          String
        > {
  FollowNotifierFamily._()
    : super(
        retry: null,
        name: r'followProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  FollowNotifierProvider call(String profileActorId) =>
      FollowNotifierProvider._(argument: profileActorId, from: this);

  @override
  String toString() => r'followProvider';
}

abstract class _$FollowNotifier extends $AsyncNotifier<FollowState> {
  late final _$args = ref.$arg as String;
  String get profileActorId => _$args;

  FutureOr<FollowState> build(String profileActorId);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<FollowState>, FollowState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<FollowState>, FollowState>,
              AsyncValue<FollowState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
