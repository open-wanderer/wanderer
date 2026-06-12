// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_style_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(mapStyle)
final mapStyleProvider = MapStyleProvider._();

final class MapStyleProvider
    extends $FunctionalProvider<AsyncValue<Style>, Style, FutureOr<Style>>
    with $FutureModifier<Style>, $FutureProvider<Style> {
  MapStyleProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapStyleProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapStyleHash();

  @$internal
  @override
  $FutureProviderElement<Style> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<Style> create(Ref ref) {
    return mapStyle(ref);
  }
}

String _$mapStyleHash() => r'eb53a3e00c72aa4e18a51be28051fe1b03efa294';
