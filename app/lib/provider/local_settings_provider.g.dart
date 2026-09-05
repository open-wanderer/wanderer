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
    r'906198edb6ddaa4d3cc80d79cd4c6f9501f446a5';

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

@ProviderFor(locale)
final localeProvider = LocaleProvider._();

final class LocaleProvider
    extends $FunctionalProvider<Locale?, Locale?, Locale?>
    with $Provider<Locale?> {
  LocaleProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'localeProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$localeHash();

  @$internal
  @override
  $ProviderElement<Locale?> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Locale? create(Ref ref) {
    return locale(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Locale? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Locale?>(value),
    );
  }
}

String _$localeHash() => r'91fb75995faecbb71c068dbc8c7cd89d4f22a974';

@ProviderFor(unit)
final unitProvider = UnitProvider._();

final class UnitProvider extends $FunctionalProvider<String, String, String>
    with $Provider<String> {
  UnitProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'unitProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$unitHash();

  @$internal
  @override
  $ProviderElement<String> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  String create(Ref ref) {
    return unit(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String>(value),
    );
  }
}

String _$unitHash() => r'61cb2b8e04879730d2520f184f8c89b7c5d086a9';
