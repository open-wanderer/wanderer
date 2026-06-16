// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_follows_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfileFollowsNotifier)
final profileFollowsProvider = ProfileFollowsNotifierFamily._();

final class ProfileFollowsNotifierProvider
    extends
        $AsyncNotifierProvider<ProfileFollowsNotifier, ProfileFollowsState> {
  ProfileFollowsNotifierProvider._({
    required ProfileFollowsNotifierFamily super.from,
    required (String, String) super.argument,
  }) : super(
         retry: null,
         name: r'profileFollowsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileFollowsNotifierHash();

  @override
  String toString() {
    return r'profileFollowsProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  ProfileFollowsNotifier create() => ProfileFollowsNotifier();

  @override
  bool operator ==(Object other) {
    return other is ProfileFollowsNotifierProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileFollowsNotifierHash() =>
    r'52a1cdf0dff1248c9942c41a7055dd6237bdae3a';

final class ProfileFollowsNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileFollowsNotifier,
          AsyncValue<ProfileFollowsState>,
          ProfileFollowsState,
          FutureOr<ProfileFollowsState>,
          (String, String)
        > {
  ProfileFollowsNotifierFamily._()
    : super(
        retry: null,
        name: r'profileFollowsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileFollowsNotifierProvider call(String handle, String type) =>
      ProfileFollowsNotifierProvider._(argument: (handle, type), from: this);

  @override
  String toString() => r'profileFollowsProvider';
}

abstract class _$ProfileFollowsNotifier
    extends $AsyncNotifier<ProfileFollowsState> {
  late final _$args = ref.$arg as (String, String);
  String get handle => _$args.$1;
  String get type => _$args.$2;

  FutureOr<ProfileFollowsState> build(String handle, String type);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<ProfileFollowsState>, ProfileFollowsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ProfileFollowsState>, ProfileFollowsState>,
              AsyncValue<ProfileFollowsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args.$1, _$args.$2));
  }
}
