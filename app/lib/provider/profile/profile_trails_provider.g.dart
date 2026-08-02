// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_trails_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfileTrailsNotifier)
final profileTrailsProvider = ProfileTrailsNotifierFamily._();

final class ProfileTrailsNotifierProvider
    extends $AsyncNotifierProvider<ProfileTrailsNotifier, ProfileTrailsState> {
  ProfileTrailsNotifierProvider._({
    required ProfileTrailsNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'profileTrailsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileTrailsNotifierHash();

  @override
  String toString() {
    return r'profileTrailsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProfileTrailsNotifier create() => ProfileTrailsNotifier();

  @override
  bool operator ==(Object other) {
    return other is ProfileTrailsNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileTrailsNotifierHash() =>
    r'947100cd8e3cada16cbe4a7b3c7b0f3afe948bd4';

final class ProfileTrailsNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileTrailsNotifier,
          AsyncValue<ProfileTrailsState>,
          ProfileTrailsState,
          FutureOr<ProfileTrailsState>,
          String
        > {
  ProfileTrailsNotifierFamily._()
    : super(
        retry: null,
        name: r'profileTrailsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileTrailsNotifierProvider call(String handle) =>
      ProfileTrailsNotifierProvider._(argument: handle, from: this);

  @override
  String toString() => r'profileTrailsProvider';
}

abstract class _$ProfileTrailsNotifier
    extends $AsyncNotifier<ProfileTrailsState> {
  late final _$args = ref.$arg as String;
  String get handle => _$args;

  FutureOr<ProfileTrailsState> build(String handle);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<ProfileTrailsState>, ProfileTrailsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ProfileTrailsState>, ProfileTrailsState>,
              AsyncValue<ProfileTrailsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
