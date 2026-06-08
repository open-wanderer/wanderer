// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_lists_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfileListsNotifier)
final profileListsProvider = ProfileListsNotifierFamily._();

final class ProfileListsNotifierProvider
    extends $AsyncNotifierProvider<ProfileListsNotifier, ProfileListsState> {
  ProfileListsNotifierProvider._({
    required ProfileListsNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'profileListsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileListsNotifierHash();

  @override
  String toString() {
    return r'profileListsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProfileListsNotifier create() => ProfileListsNotifier();

  @override
  bool operator ==(Object other) {
    return other is ProfileListsNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileListsNotifierHash() =>
    r'4549b14d9d18b234bb1b957b1cef7c31326d29ef';

final class ProfileListsNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileListsNotifier,
          AsyncValue<ProfileListsState>,
          ProfileListsState,
          FutureOr<ProfileListsState>,
          String
        > {
  ProfileListsNotifierFamily._()
    : super(
        retry: null,
        name: r'profileListsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileListsNotifierProvider call(String handle) =>
      ProfileListsNotifierProvider._(argument: handle, from: this);

  @override
  String toString() => r'profileListsProvider';
}

abstract class _$ProfileListsNotifier
    extends $AsyncNotifier<ProfileListsState> {
  late final _$args = ref.$arg as String;
  String get handle => _$args;

  FutureOr<ProfileListsState> build(String handle);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<ProfileListsState>, ProfileListsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ProfileListsState>, ProfileListsState>,
              AsyncValue<ProfileListsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
