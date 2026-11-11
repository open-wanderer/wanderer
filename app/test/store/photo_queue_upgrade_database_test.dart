import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('new cache property preserves all released ObjectBox identifiers', () {
    final oldModel = jsonDecode(
      File('test/fixtures/beta25_objectbox-model.json').readAsStringSync(),
    ) as Map;
    final newModel =
        jsonDecode(File('lib/objectbox-model.json').readAsStringSync()) as Map;
    final entities = {
      for (final entity in newModel['entities'] as List)
        (entity as Map)['name']: entity,
    };
    final oldUids = <String>{};
    for (final entity in oldModel['entities'] as List) {
      final old = entity as Map;
      final fresh = entities[old['name']] as Map;
      expect(fresh['id'], old['id']);
      oldUids.add((old['id'] as String).split(':').last);
      final properties = {
        for (final property in fresh['properties'] as List)
          (property as Map)['name']: property,
      };
      for (final property in old['properties'] as List) {
        expect(properties[(property as Map)['name']], property);
        oldUids.add((property['id'] as String).split(':').last);
      }
    }
    for (final key in [
      'retiredEntityUids',
      'retiredPropertyUids',
      'retiredIndexUids',
      'retiredRelationUids',
    ]) {
      oldUids.addAll((oldModel[key] as List).map((uid) => uid.toString()));
      expect(newModel[key], containsAll(oldModel[key] as List));
    }
    final waypoint = entities['WaypointEntity'] as Map;
    final added = (waypoint['properties'] as List).cast<Map>().singleWhere(
      (property) => property['name'] == 'downloadedPhotos',
    );
    final previous = (oldModel['entities'] as List).cast<Map>().singleWhere(
      (entity) => entity['name'] == 'WaypointEntity',
    );
    expect(
      int.parse((added['id'] as String).split(':').first),
      greaterThan(
        int.parse((previous['lastPropertyId'] as String).split(':').first),
      ),
    );
    expect(oldUids, isNot(contains((added['id'] as String).split(':').last)));
  });
}
