import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';

/// A backend that always accepts `PUT /waypoint` and answers
/// `POST /waypoint/{id}/file` according to [photoUploadSucceeds].
///
/// The split matters: the photo upload is the largest and most timeout-prone
/// request in the create sequence, so "record created, photos failed" is the
/// common partial failure, not an exotic one.
class _FakeApi extends Api {
  _FakeApi({required this.photoUploadSucceeds});

  final bool photoUploadSucceeds;

  final List<String> requestedPaths = [];

  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requestedPaths.add(options.path);

          if (options.path == '/waypoint') {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'id': 'wpserverid1234',
                  'lat': 47.0,
                  'lon': 11.0,
                  'author': '000000000000000',
                  'created': '2026-01-01T00:00:00Z',
                  'updated': '2026-01-01T00:00:00Z',
                  'photos': <String>[],
                },
              ),
            );
            return;
          }

          if (!photoUploadSucceeds) {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.connectionTimeout,
              ),
            );
            return;
          }

          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: {
                'id': 'wpserverid1234',
                'lat': 47.0,
                'lon': 11.0,
                'author': '000000000000000',
                'created': '2026-01-01T00:00:00Z',
                'updated': '2026-01-01T00:00:00Z',
                'photos': ['uploaded.jpg'],
              },
            ),
          );
        },
      ),
    );
    return dio;
  }
}

void main() {
  late Directory tempDir;
  late String photoPath;

  setUp(() {
    tempDir = Directory.systemTemp.createTempSync('wp_create_test');
    photoPath = '${tempDir.path}/photo.jpg';
    File(photoPath).writeAsBytesSync([1, 2, 3]);
  });

  tearDown(() {
    if (tempDir.existsSync()) tempDir.deleteSync(recursive: true);
  });

  ({ProviderContainer container, _FakeApi api}) harness({
    required bool photoUploadSucceeds,
  }) {
    final api = _FakeApi(photoUploadSucceeds: photoUploadSucceeds);
    final container = ProviderContainer(
      overrides: [apiProvider.overrideWith(() => api)],
    );
    addTearDown(container.dispose);
    return (container: container, api: api);
  }

  Waypoint buildWaypoint({List<String> localPhotos = const []}) {
    return Waypoint(
      id: '',
      lat: 47.0,
      lon: 11.0,
      author: '000000000000000',
      created: DateTime(2026),
      updated: DateTime(2026),
      localKey: 'local-1-1',
      localPhotos: localPhotos,
    );
  }

  group('WaypointSave.create photo-upload partial failure (CR-02)', () {
    test('a waypoint with no photos creates and returns the record', () async {
      final pumped = harness(photoUploadSucceeds: true);

      final created = await pumped.container
          .read(waypointSaveProvider.notifier)
          .create(buildWaypoint(), authorId: 'actor1', trailId: 'trail1');

      expect(created.id, 'wpserverid1234');
      expect(pumped.api.requestedPaths, ['/waypoint']);
    });

    test('a successful photo upload returns the record with its photos', () async {
      final pumped = harness(photoUploadSucceeds: true);

      final created = await pumped.container
          .read(waypointSaveProvider.notifier)
          .create(
            buildWaypoint(localPhotos: [photoPath]),
            authorId: 'actor1',
            trailId: 'trail1',
          );

      expect(created.photos, ['uploaded.jpg']);
      expect(pumped.api.requestedPaths, [
        '/waypoint',
        '/waypoint/wpserverid1234/file',
      ]);
    });

    test(
      'a failed photo upload throws WaypointPhotoUploadException CARRYING the '
      'created record, so the caller can persist its server id',
      () async {
        final pumped = harness(photoUploadSucceeds: false);

        // Before this, the failure surfaced as a bare DioException and the
        // created record's id was lost. The drain then re-entered step 3 with
        // the waypoint still holding a `local-…` sentinel id and ran
        // `PUT /waypoint` again -- a duplicate waypoint on the server, with no
        // idempotency key to reconcile it (RESEARCH.md Pitfall 3).
        await expectLater(
          pumped.container.read(waypointSaveProvider.notifier).create(
            buildWaypoint(localPhotos: [photoPath]),
            authorId: 'actor1',
            trailId: 'trail1',
          ),
          throwsA(
            isA<WaypointPhotoUploadException>().having(
              (e) => e.created.id,
              'created.id',
              'wpserverid1234',
            ),
          ),
        );

        // The record really was created -- exactly once.
        expect(
          pumped.api.requestedPaths.where((p) => p == '/waypoint').length,
          1,
        );
      },
    );

    test(
      'the underlying transport failure is retained as `cause`, so a photo '
      'failure stays diagnosable rather than being flattened',
      () async {
        final pumped = harness(photoUploadSucceeds: false);

        Object? thrown;
        try {
          await pumped.container.read(waypointSaveProvider.notifier).create(
            buildWaypoint(localPhotos: [photoPath]),
            authorId: 'actor1',
            trailId: 'trail1',
          );
        } catch (e) {
          thrown = e;
        }

        expect(thrown, isA<WaypointPhotoUploadException>());
        expect((thrown as WaypointPhotoUploadException).cause, isA<DioException>());
      },
    );
  });
}
