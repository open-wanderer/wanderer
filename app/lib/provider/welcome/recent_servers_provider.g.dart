// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'recent_servers_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The persisted "Last used" instance list shown in the server picker.
///
/// `keepAlive` because [ref.read]-only writers reach this from the keepAlive
/// `Auth` notifier on login success (see `auth_provider.dart`), mirroring
/// `LocalSettingsNotifier`, which is keepAlive for the same reason.

@ProviderFor(RecentServersNotifier)
final recentServersProvider = RecentServersNotifierProvider._();

/// The persisted "Last used" instance list shown in the server picker.
///
/// `keepAlive` because [ref.read]-only writers reach this from the keepAlive
/// `Auth` notifier on login success (see `auth_provider.dart`), mirroring
/// `LocalSettingsNotifier`, which is keepAlive for the same reason.
final class RecentServersNotifierProvider
    extends $NotifierProvider<RecentServersNotifier, List<ServerInstance>> {
  /// The persisted "Last used" instance list shown in the server picker.
  ///
  /// `keepAlive` because [ref.read]-only writers reach this from the keepAlive
  /// `Auth` notifier on login success (see `auth_provider.dart`), mirroring
  /// `LocalSettingsNotifier`, which is keepAlive for the same reason.
  RecentServersNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'recentServersProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$recentServersNotifierHash();

  @$internal
  @override
  RecentServersNotifier create() => RecentServersNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<ServerInstance> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<ServerInstance>>(value),
    );
  }
}

String _$recentServersNotifierHash() =>
    r'1c775f96c8487bf5130d3991488922ed457670db';

/// The persisted "Last used" instance list shown in the server picker.
///
/// `keepAlive` because [ref.read]-only writers reach this from the keepAlive
/// `Auth` notifier on login success (see `auth_provider.dart`), mirroring
/// `LocalSettingsNotifier`, which is keepAlive for the same reason.

abstract class _$RecentServersNotifier extends $Notifier<List<ServerInstance>> {
  List<ServerInstance> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<ServerInstance>, List<ServerInstance>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<ServerInstance>, List<ServerInstance>>,
              List<ServerInstance>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
