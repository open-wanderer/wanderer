import 'package:dio/dio.dart';

/// A resolved reverse-geocoded location: a country-less [label] (used when
/// every visible anchor shares one country) and a country-inclusive
/// [fullLabel] (used otherwise). Mirrors web's `search_store.ts`.
class ReverseLocationResult {
  final String label;
  final String fullLabel;
  final String country;

  const ReverseLocationResult({
    required this.label,
    required this.fullLabel,
    required this.country,
  });

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is ReverseLocationResult &&
          other.label == label &&
          other.fullLabel == fullLabel &&
          other.country == country);

  @override
  int get hashCode => Object.hash(label, fullLabel, country);
}

/// Builds a comma-joined location description from a Nominatim `address`
/// object: `road` (only when [includeRoad]), the first present of
/// city/town/hamlet/village, `state`, then `country` (only when
/// [includeCountry]). Missing keys are skipped, so no empty commas.
String getLocationDescription(
  Map<String, dynamic> address, {
  bool includeRoad = false,
  bool includeCountry = true,
}) {
  final parts = <String>[];

  final road = address['road'];
  if (includeRoad && road is String && road.isNotEmpty) {
    parts.add(road);
  }

  final city =
      address['city'] as String? ??
      address['town'] as String? ??
      address['hamlet'] as String? ??
      address['village'] as String?;
  if (city != null && city.isNotEmpty) {
    parts.add(city);
  }

  final state = address['state'];
  if (state is String && state.isNotEmpty) {
    parts.add(state);
  }

  final country = address['country'];
  if (includeCountry && country is String && country.isNotEmpty) {
    parts.add(country);
  }

  return parts.join(', ');
}

/// Builds a [ReverseLocationResult] from a Nominatim `address` object.
/// [label] falls back to [fullLabel] when the country-less description is
/// empty.
ReverseLocationResult getReverseLocationResult(
  Map<String, dynamic> address, {
  bool includeRoad = false,
}) {
  final country = address['country'] as String? ?? '';
  final label = getLocationDescription(
    address,
    includeRoad: includeRoad,
    includeCountry: false,
  );
  final fullLabel = getLocationDescription(address, includeRoad: includeRoad);

  return ReverseLocationResult(
    label: label.isNotEmpty ? label : fullLabel,
    fullLabel: fullLabel,
    country: country,
  );
}

/// Reverse-geocodes a coordinate via the server's `/geocoding/reverse` proxy,
/// returning a structured [ReverseLocationResult], or `null` when Nominatim
/// has no address for the point. Lets [DioException] propagate — this is a
/// pure fetch; the caller decides how to handle failures.
Future<ReverseLocationResult?> searchLocationReverseStructured(
  Dio api,
  double lat,
  double lon, {
  bool includeRoad = true,
  CancelToken? cancelToken,
}) async {
  final response = await api.get(
    '/geocoding/reverse',
    queryParameters: {'lat': lat, 'lon': lon},
    cancelToken: cancelToken,
  );

  final features = response.data['features'] as List<dynamic>? ?? [];
  if (features.isEmpty) {
    return null;
  }

  final properties =
      (features[0] as Map<String, dynamic>)['properties']
          as Map<String, dynamic>?;
  final address = properties?['address'] as Map<String, dynamic>?;
  if (address == null) {
    return null;
  }

  return getReverseLocationResult(address, includeRoad: includeRoad);
}
