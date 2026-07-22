import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_catalog_entry.dart';
import 'package:wanderer/models/region_status.dart';

void main() {
  group('RegionCatalogEntry.fromJson', () {
    test('full "ready" fixture yields non-null optionals', () {
      final json = {
        'id': 'de-nrw',
        'name': 'North Rhine-Westphalia',
        'bbox': [5.9, 50.3, 9.5, 52.5],
        'status': 'ready',
        'version': '2026-07-01',
        'vector_url': '/api/v1/regions/de-nrw/download',
        'vector_size': 123456,
        'dem_status': 'ready',
        'dem_url': '/api/v1/regions/de-nrw/download-dem',
        'dem_size': 654321,
      };

      final entry = RegionCatalogEntry.fromJson(json);

      expect(entry.id, 'de-nrw');
      expect(entry.name, 'North Rhine-Westphalia');
      expect(entry.status, CatalogStatus.ready);
      expect(entry.version, '2026-07-01');
      expect(entry.vectorUrl, '/api/v1/regions/de-nrw/download');
      expect(entry.vectorSize, 123456);
      expect(entry.demStatus, CatalogStatus.ready);
      expect(entry.demUrl, '/api/v1/regions/de-nrw/download-dem');
      expect(entry.demSize, 654321);
      expect(entry.error, isNull);
    });

    test('minimal "building" fixture yields null optionals', () {
      final json = {
        'id': 'de-bay',
        'name': 'Bavaria',
        'bbox': [8.9, 47.2, 13.9, 50.6],
        'status': 'building',
      };

      final entry = RegionCatalogEntry.fromJson(json);

      expect(entry.status, CatalogStatus.building);
      expect(entry.version, isNull);
      expect(entry.vectorUrl, isNull);
      expect(entry.vectorSize, isNull);
      expect(entry.demStatus, isNull);
      expect(entry.demUrl, isNull);
      expect(entry.demSize, isNull);
      expect(entry.error, isNull);
    });

    test('bbox decodes to a List<double> of length 4 in expected order', () {
      final json = {
        'id': 'de-nrw',
        'name': 'North Rhine-Westphalia',
        'bbox': [5.9, 50.3, 9.5, 52.5],
        'status': 'building',
      };

      final entry = RegionCatalogEntry.fromJson(json);

      expect(entry.bbox.length, 4);
      expect(entry.bbox[0], 5.9);
      expect(entry.bbox[1], 50.3);
      expect(entry.bbox[2], 9.5);
      expect(entry.bbox[3], 52.5);
    });

    test('error status fixture carries the error message', () {
      final json = {
        'id': 'de-nrw',
        'name': 'North Rhine-Westphalia',
        'bbox': [5.9, 50.3, 9.5, 52.5],
        'status': 'error',
        'error': 'pmtiles extract failed',
      };

      final entry = RegionCatalogEntry.fromJson(json);

      expect(entry.status, CatalogStatus.error);
      expect(entry.error, 'pmtiles extract failed');
    });
  });
}
