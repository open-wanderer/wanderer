// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'local_settings_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(LocalSettingsNotifier)
final localSettingsProvider = LocalSettingsNotifierProvider._();

final class LocalSettingsNotifierProvider
    extends $NotifierProvider<LocalSettingsNotifier, LocalSettingsEntity> {
  LocalSettingsNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'localSettingsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$localSettingsNotifierHash();

  @$internal
  @override
  LocalSettingsNotifier create() => LocalSettingsNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(LocalSettingsEntity value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<LocalSettingsEntity>(value),
    );
  }
}

String _$localSettingsNotifierHash() =>
    r'546e3c8fd14eb9d01acfc3a8d90ba8ce193afcad';

abstract class _$LocalSettingsNotifier extends $Notifier<LocalSettingsEntity> {
  LocalSettingsEntity build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<LocalSettingsEntity, LocalSettingsEntity>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<LocalSettingsEntity, LocalSettingsEntity>,
              LocalSettingsEntity,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}

@ProviderFor(themeMode)
final themeModeProvider = ThemeModeProvider._();

final class ThemeModeProvider
    extends $FunctionalProvider<ThemeMode, ThemeMode, ThemeMode>
    with $Provider<ThemeMode> {
  ThemeModeProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'themeModeProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$themeModeHash();

  @$internal
  @override
  $ProviderElement<ThemeMode> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ThemeMode create(Ref ref) {
    return themeMode(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ThemeMode value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ThemeMode>(value),
    );
  }
}

String _$themeModeHash() => r'688b55e2c54c3ee17b0d204c96143c1ca617264f';
