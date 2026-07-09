// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'navigation_stats_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Riverpod notifier that computes live navigation statistics for a single
/// [NavigateResponse] session.
///
/// Family-keyed on [response] so each navigation session is isolated and
/// testable via a plain [ProviderContainer] (D-17), exactly like
/// `navigationProvider`.
///
/// D-13: this notifier NEVER opens its own GPS stream. It is fed purely via
/// [onPosition], called from the single broadcast stream owned by
/// `_NavigationScreenState`.

@ProviderFor(NavigationStatsNotifier)
final navigationStatsProvider = NavigationStatsNotifierFamily._();

/// Riverpod notifier that computes live navigation statistics for a single
/// [NavigateResponse] session.
///
/// Family-keyed on [response] so each navigation session is isolated and
/// testable via a plain [ProviderContainer] (D-17), exactly like
/// `navigationProvider`.
///
/// D-13: this notifier NEVER opens its own GPS stream. It is fed purely via
/// [onPosition], called from the single broadcast stream owned by
/// `_NavigationScreenState`.
final class NavigationStatsNotifierProvider
    extends $NotifierProvider<NavigationStatsNotifier, NavigationStats> {
  /// Riverpod notifier that computes live navigation statistics for a single
  /// [NavigateResponse] session.
  ///
  /// Family-keyed on [response] so each navigation session is isolated and
  /// testable via a plain [ProviderContainer] (D-17), exactly like
  /// `navigationProvider`.
  ///
  /// D-13: this notifier NEVER opens its own GPS stream. It is fed purely via
  /// [onPosition], called from the single broadcast stream owned by
  /// `_NavigationScreenState`.
  NavigationStatsNotifierProvider._({
    required NavigationStatsNotifierFamily super.from,
    required NavigateResponse super.argument,
  }) : super(
         retry: null,
         name: r'navigationStatsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$navigationStatsNotifierHash();

  @override
  String toString() {
    return r'navigationStatsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  NavigationStatsNotifier create() => NavigationStatsNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NavigationStats value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NavigationStats>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is NavigationStatsNotifierProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$navigationStatsNotifierHash() =>
    r'acd8e8a7ec22dd83e9e20e68bc18f5a4ee5e190a';

/// Riverpod notifier that computes live navigation statistics for a single
/// [NavigateResponse] session.
///
/// Family-keyed on [response] so each navigation session is isolated and
/// testable via a plain [ProviderContainer] (D-17), exactly like
/// `navigationProvider`.
///
/// D-13: this notifier NEVER opens its own GPS stream. It is fed purely via
/// [onPosition], called from the single broadcast stream owned by
/// `_NavigationScreenState`.

final class NavigationStatsNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          NavigationStatsNotifier,
          NavigationStats,
          NavigationStats,
          NavigationStats,
          NavigateResponse
        > {
  NavigationStatsNotifierFamily._()
    : super(
        retry: null,
        name: r'navigationStatsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Riverpod notifier that computes live navigation statistics for a single
  /// [NavigateResponse] session.
  ///
  /// Family-keyed on [response] so each navigation session is isolated and
  /// testable via a plain [ProviderContainer] (D-17), exactly like
  /// `navigationProvider`.
  ///
  /// D-13: this notifier NEVER opens its own GPS stream. It is fed purely via
  /// [onPosition], called from the single broadcast stream owned by
  /// `_NavigationScreenState`.

  NavigationStatsNotifierProvider call(NavigateResponse response) =>
      NavigationStatsNotifierProvider._(argument: response, from: this);

  @override
  String toString() => r'navigationStatsProvider';
}

/// Riverpod notifier that computes live navigation statistics for a single
/// [NavigateResponse] session.
///
/// Family-keyed on [response] so each navigation session is isolated and
/// testable via a plain [ProviderContainer] (D-17), exactly like
/// `navigationProvider`.
///
/// D-13: this notifier NEVER opens its own GPS stream. It is fed purely via
/// [onPosition], called from the single broadcast stream owned by
/// `_NavigationScreenState`.

abstract class _$NavigationStatsNotifier extends $Notifier<NavigationStats> {
  late final _$args = ref.$arg as NavigateResponse;
  NavigateResponse get response => _$args;

  NavigationStats build(NavigateResponse response);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<NavigationStats, NavigationStats>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<NavigationStats, NavigationStats>,
              NavigationStats,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
