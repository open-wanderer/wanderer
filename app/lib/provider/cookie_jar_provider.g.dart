// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'cookie_jar_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(cookieJar)
final cookieJarProvider = CookieJarProvider._();

final class CookieJarProvider
    extends
        $FunctionalProvider<
          AsyncValue<PersistCookieJar>,
          PersistCookieJar,
          FutureOr<PersistCookieJar>
        >
    with $FutureModifier<PersistCookieJar>, $FutureProvider<PersistCookieJar> {
  CookieJarProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'cookieJarProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$cookieJarHash();

  @$internal
  @override
  $FutureProviderElement<PersistCookieJar> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<PersistCookieJar> create(Ref ref) {
    return cookieJar(ref);
  }
}

String _$cookieJarHash() => r'1085aa4085863dd9976ff32b11e389fb83a13c7c';
