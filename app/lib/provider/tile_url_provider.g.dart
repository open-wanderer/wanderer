// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tile_url_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(tileUrl)
final tileUrlProvider = TileUrlProvider._();

final class TileUrlProvider
    extends $FunctionalProvider<AsyncValue<String>, String, FutureOr<String>>
    with $FutureModifier<String>, $FutureProvider<String> {
  TileUrlProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tileUrlProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tileUrlHash();

  @$internal
  @override
  $FutureProviderElement<String> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<String> create(Ref ref) {
    return tileUrl(ref);
  }
}

String _$tileUrlHash() => r'd551b810e12dccf518878c47df66a4d8043f1c49';
