/// Store-backed read/write of the operator's upstream tile/glyph/sprite
/// templates and the resolved DEM template.
///
/// The proxy builds redirect targets from these, and an offline cold start
/// must know them with no network call available. Single home for both the
/// Riverpod side and the proxy, which holds a raw [Store].
library;

import 'dart:convert';

import 'package:wanderer/entities/local_settings_entity.dart';
import 'package:wanderer/models/map_style_sources.dart';
import 'package:wanderer/objectbox.g.dart';

/// Reads the persisted `/map/style-sources` response from [store].
///
/// Returns `null` when nothing has ever been persisted, or when the
/// persisted value is malformed in any way — persisted data is never
/// trusted to be well-formed.
MapStyleSources? readPersistedMapStyleSources(Store store) {
  final entity = store.box<LocalSettingsEntity>().getAll().firstOrNull;
  final raw = entity?.mapStyleSourcesJson;
  if (raw == null || raw.isEmpty) return null;
  try {
    return MapStyleSources.fromJson(
      jsonDecode(raw) as Map<String, dynamic>,
    );
  } catch (_) {
    return null;
  }
}

/// Persists [sources] as the latest known `/map/style-sources` response.
void writePersistedMapStyleSources(Store store, MapStyleSources sources) {
  final box = store.box<LocalSettingsEntity>();
  final entity = box.getAll().firstOrNull ?? LocalSettingsEntity();
  entity.mapStyleSourcesJson = jsonEncode(sources.toJson());
  box.put(entity);
}

/// Reads the persisted resolved hillshade DEM XYZ template from [store].
///
/// Returns `null` when nothing has ever been persisted.
String? readPersistedDemTileTemplate(Store store) {
  final entity = store.box<LocalSettingsEntity>().getAll().firstOrNull;
  final raw = entity?.demTileTemplate;
  if (raw == null || raw.isEmpty) return null;
  return raw;
}

/// Persists [template] as the resolved hillshade DEM XYZ template.
void writePersistedDemTileTemplate(Store store, String template) {
  final box = store.box<LocalSettingsEntity>();
  final entity = box.getAll().firstOrNull ?? LocalSettingsEntity();
  entity.demTileTemplate = template;
  box.put(entity);
}
