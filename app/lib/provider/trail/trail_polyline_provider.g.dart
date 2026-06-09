// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_polyline_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(trailPolyline)
final trailPolylineProvider = TrailPolylineFamily._();

final class TrailPolylineProvider
    extends
        $FunctionalProvider<
          AsyncValue<Polyline<Object>?>,
          Polyline<Object>?,
          FutureOr<Polyline<Object>?>
        >
    with
        $FutureModifier<Polyline<Object>?>,
        $FutureProvider<Polyline<Object>?> {
  TrailPolylineProvider._({
    required TrailPolylineFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'trailPolylineProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$trailPolylineHash();

  @override
  String toString() {
    return r'trailPolylineProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<Polyline<Object>?> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<Polyline<Object>?> create(Ref ref) {
    final argument = this.argument as String;
    return trailPolyline(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is TrailPolylineProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$trailPolylineHash() => r'7c9acd31645638350890036d7803d0f4cc53fb5b';

final class TrailPolylineFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<Polyline<Object>?>, String> {
  TrailPolylineFamily._()
    : super(
        retry: null,
        name: r'trailPolylineProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  TrailPolylineProvider call(String trailId) =>
      TrailPolylineProvider._(argument: trailId, from: this);

  @override
  String toString() => r'trailPolylineProvider';
}
