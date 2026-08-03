import 'dart:io';

import 'package:url_launcher/url_launcher.dart';

/// A third-party maps application we can hand a destination to.
///
/// Only used on iOS. Android resolves map apps through a single `geo:` intent
/// and shows the system chooser, so it needs no table.
class MapApp {
  /// Stable identifier, also used as the sheet's list key.
  final String id;

  /// Display name — a proper noun, deliberately untranslated.
  final String name;

  /// Scheme URL used with `canLaunchUrl` to detect the app. `null` means the
  /// app is always present (Apple Maps).
  ///
  /// Every scheme here must also appear in `LSApplicationQueriesSchemes` in
  /// `ios/Runner/Info.plist`, or iOS returns `false` regardless of what is
  /// installed.
  final String? probeUrl;

  /// Builds the directions URL. Some apps are detected via their custom
  /// scheme but launched through a universal link, so this is separate from
  /// [probeUrl].
  final String Function(double lat, double lon) directionsUrl;

  const MapApp({
    required this.id,
    required this.name,
    required this.probeUrl,
    required this.directionsUrl,
  });
}

/// Map apps we offer on iOS, in the order they appear in the chooser.
///
/// URL formats follow the `map_launcher` package's field-tested table
/// (https://github.com/mattermoran/map_launcher). Walking is requested where
/// the app supports a travel mode, since Wanderer destinations are trailheads.
/// Apple Maps is last because it is the only entry that is always installed.
const List<MapApp> iosMapApps = [
  MapApp(
    id: 'google',
    name: 'Google Maps',
    probeUrl: 'comgooglemaps://',
    directionsUrl: _googleDirections,
  ),
  MapApp(
    id: 'osmand',
    name: 'OsmAnd',
    probeUrl: 'osmandmaps://',
    directionsUrl: _osmandDirections,
  ),
  MapApp(
    id: 'organic',
    name: 'Organic Maps / MAPS.ME',
    probeUrl: 'mapsme://',
    directionsUrl: _organicDirections,
  ),
  MapApp(
    id: 'magicEarth',
    name: 'Magic Earth',
    probeUrl: 'magicearth://',
    directionsUrl: _magicEarthDirections,
  ),
  MapApp(
    id: 'mapyCz',
    name: 'Mapy.cz',
    probeUrl: 'szn-mapy://',
    directionsUrl: _mapyCzDirections,
  ),
  MapApp(
    id: 'here',
    name: 'HERE WeGo',
    probeUrl: 'here-location://',
    directionsUrl: _hereDirections,
  ),
  MapApp(
    id: 'waze',
    name: 'Waze',
    probeUrl: 'waze://',
    directionsUrl: _wazeDirections,
  ),
  MapApp(
    id: 'apple',
    name: 'Apple Maps',
    probeUrl: null,
    directionsUrl: _appleDirections,
  ),
];

String _googleDirections(double lat, double lon) =>
    'comgooglemaps://?daddr=$lat,$lon&directionsmode=walking';

String _osmandDirections(double lat, double lon) =>
    'osmandmaps://navigate?lat=$lat&lon=$lon';

String _organicDirections(double lat, double lon) =>
    'mapsme://route?v=1&ll=$lat,$lon';

String _magicEarthDirections(double lat, double lon) =>
    'magicearth://?walk_to&lat=$lat&lon=$lon';

String _mapyCzDirections(double lat, double lon) =>
    'https://mapy.cz/zakladni?y=$lat&x=$lon&z=16';

String _hereDirections(double lat, double lon) =>
    'https://share.here.com/r/$lat,$lon?m=w';

String _wazeDirections(double lat, double lon) =>
    'waze://?ll=$lat,$lon&navigate=yes';

/// Apple Maps universal link. Opens the app directly when installed, so it
/// needs no entry in `LSApplicationQueriesSchemes`.
String _appleDirections(double lat, double lon) =>
    'https://maps.apple.com/?daddr=$lat,$lon&dirflg=w';

/// The subset of [iosMapApps] actually installed, preserving table order.
///
/// Returns an empty list off iOS — callers should use the `geo:` intent there.
Future<List<MapApp>> installedMapApps() async {
  if (!Platform.isIOS) return const [];

  final installed = <MapApp>[];
  for (final app in iosMapApps) {
    final probe = app.probeUrl;
    if (probe == null || await canLaunchUrl(Uri.parse(probe))) {
      installed.add(app);
    }
  }
  return installed;
}
