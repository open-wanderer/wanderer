import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/provider/region/region_repository.dart';

Map<String, dynamic> _fullReadyJson({String id = 'de-nrw'}) => {
  'id': id,
  'name': 'North Rhine-Westphalia',
  'bbox': [5.9, 50.3, 9.5, 52.5],
  'status': 'ready',
  'version': '2026-07-01',
  'vector_url': '/api/v1/regions/$id/download',
  'vector_size': 123456,
  'dem_status': 'ready',
  'dem_url': '/api/v1/regions/$id/download-dem',
  'dem_size': 654321,
};

Map<String, dynamic> _minimalBuildingJson({String id = 'de-bay'}) => {
  'id': id,
  'name': 'Bavaria',
  'bbox': [8.9, 47.2, 13.9, 50.6],
  'status': 'building',
};

void main() {
  group('parseRegionCatalog', () {
    test('a full-ready + minimal-building fixture yields 2 entries', () {
      final entries = parseRegionCatalog([
        _fullReadyJson(),
        _minimalBuildingJson(),
      ]);

      expect(entries, hasLength(2));
      expect(entries[0].id, 'de-nrw');
      expect(entries[0].status, CatalogStatus.ready);
      expect(entries[1].id, 'de-bay');
      expect(entries[1].status, CatalogStatus.building);
    });

    test('a valid + malformed element drops the malformed one', () {
      final entries = parseRegionCatalog([
        _fullReadyJson(),
        {'id': 'de-bad'}, // missing required name/bbox/status
      ]);

      expect(entries, hasLength(1));
      expect(entries.single.id, 'de-nrw');
    });

    test('an empty array yields an empty list', () {
      expect(parseRegionCatalog([]), isEmpty);
    });

    test('a non-List payload throws RegionCatalogException', () {
      expect(
        () => parseRegionCatalog({'items': []}),
        throwsA(isA<RegionCatalogException>()),
      );
    });
  });

  group('fetchRegionCatalog', () {
    Dio buildDio(Future<Response<dynamic>> Function(RequestOptions) handler) {
      final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, requestHandler) async {
            try {
              final response = await handler(options);
              requestHandler.resolve(response);
            } on DioException catch (e) {
              requestHandler.reject(e);
            }
          },
        ),
      );
      return dio;
    }

    test('a successful response resolves to the parsed list', () async {
      final dio = buildDio(
        (options) async => Response(
          requestOptions: options,
          statusCode: 200,
          data: [_fullReadyJson(), _minimalBuildingJson()],
        ),
      );

      final entries = await fetchRegionCatalog(dio);

      expect(entries, hasLength(2));
    });

    test(
      'a DioException (offline/500) throws RegionCatalogException, not '
      'DioException, and never swallows into an empty list',
      () async {
        final dio = buildDio(
          (options) async => throw DioException(
            requestOptions: options,
            type: DioExceptionType.connectionError,
          ),
        );

        await expectLater(
          fetchRegionCatalog(dio),
          throwsA(isA<RegionCatalogException>()),
        );
      },
    );
  });

  group('orphanedRegionIds', () {
    RegionEntity entity(String id) => RegionEntity(id: id, name: id);

    test('returns persisted ids absent from the fetched set', () {
      final result = orphanedRegionIds(
        {'a', 'b'},
        [entity('a'), entity('c')],
      );

      expect(result, {'c'});
    });

    test('returns an empty set when every persisted id is still fetched', () {
      final result = orphanedRegionIds(
        {'a', 'b', 'c'},
        [entity('a'), entity('b'), entity('c')],
      );

      expect(result, isEmpty);
    });

    test('returns an empty set when there are no persisted entities', () {
      expect(orphanedRegionIds({'a'}, []), isEmpty);
    });
  });
}
