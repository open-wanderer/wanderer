// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_feed_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfileFeedNotifier)
final profileFeedProvider = ProfileFeedNotifierFamily._();

final class ProfileFeedNotifierProvider
    extends $AsyncNotifierProvider<ProfileFeedNotifier, ProfileFeedState> {
  ProfileFeedNotifierProvider._({
    required ProfileFeedNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'profileFeedProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileFeedNotifierHash();

  @override
  String toString() {
    return r'profileFeedProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProfileFeedNotifier create() => ProfileFeedNotifier();

  @override
  bool operator ==(Object other) {
    return other is ProfileFeedNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileFeedNotifierHash() =>
    r'4ac5f55e0fa6774252be244813f1e1131ecf9a0a';

final class ProfileFeedNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileFeedNotifier,
          AsyncValue<ProfileFeedState>,
          ProfileFeedState,
          FutureOr<ProfileFeedState>,
          String
        > {
  ProfileFeedNotifierFamily._()
    : super(
        retry: null,
        name: r'profileFeedProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileFeedNotifierProvider call(String handle) =>
      ProfileFeedNotifierProvider._(argument: handle, from: this);

  @override
  String toString() => r'profileFeedProvider';
}

abstract class _$ProfileFeedNotifier extends $AsyncNotifier<ProfileFeedState> {
  late final _$args = ref.$arg as String;
  String get handle => _$args;

  FutureOr<ProfileFeedState> build(String handle);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<ProfileFeedState>, ProfileFeedState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ProfileFeedState>, ProfileFeedState>,
              AsyncValue<ProfileFeedState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
