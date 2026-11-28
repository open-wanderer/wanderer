import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/models/photo_asset.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';

class _Api extends Api {
  final requests = <RequestOptions>[];
  bool acceptPendingUpload = false;
  @override
  Dio build() {
    final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          if (acceptPendingUpload) {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'photos': ['uploaded.jpg'],
                },
              ),
            );
            return;
          }
          handler.reject(DioException(requestOptions: options));
        },
      ),
    );
    return api;
  }
}

void main() {
  test(
    'committed response fallback keeps IDs with nested expanded models',
    () async {
      final request = RequestOptions(path: '/trail/form');
      Future<Map<String, dynamic>> committed(Map<String, dynamic> fallback) =>
          photoMetadataWrite(
            Future<Response<dynamic>>.error(
              DioException(
                requestOptions: request,
                response: Response(
                  requestOptions: request,
                  statusCode: 500,
                  data: {
                    'detail': {
                      'recordCommitted': true,
                      'savedRecord': {'id': 'saved0000000001'},
                    },
                  },
                ),
              ),
            ),
            fallback,
          );
      final trail = Trail.empty().copyWith(expand: const TrailExpand(tags: []));
      expect(
        Trail.fromJson(await committed(trail.toJson())).id,
        'saved0000000001',
      );
      final waypoint = Waypoint(
        id: '',
        lat: 1,
        lon: 1,
        created: DateTime(2026),
        updated: DateTime(2026),
        expand: const WaypointExpand(
          photoAssets: [
            PhotoAssetLink(
              asset: 'asset0000000001',
              photoAsset: PhotoAsset(id: 'asset0000000001'),
            ),
          ],
        ),
      );
      final saved = Waypoint.fromJson(await committed(waypoint.toJson()));
      expect(saved.id, 'saved0000000001');
      expect(saved.expand?.photoAssets?.single.asset, 'asset0000000001');
    },
  );

  test('the queue resumes photos for an already-created waypoint', () async {
    final directory = Directory.systemTemp.createTempSync('waypoint-resume');
    addTearDown(() => directory.deleteSync(recursive: true));
    final source = File('${directory.path}/photo.jpg')..writeAsBytesSync([1]);
    final api = _Api()..acceptPendingUpload = true;
    final container = ProviderContainer(
      overrides: [
        apiProvider.overrideWith(() => api),
        photoMutationServiceProvider.overrideWith(
          (ref) => PhotoMutationService(
            api: ref.read(apiProvider),
            resolveMode: () async => PhotoApiMode.legacy,
            documentsDirectory: () async => directory,
          ),
        ),
      ],
    );
    addTearDown(container.dispose);
    final saved = await container
        .read(waypointSaveProvider.notifier)
        .saveQueuedWaypoint(
          Waypoint(
            id: 'waypoint0000001',
            lat: 1,
            lon: 1,
            created: DateTime(2026),
            updated: DateTime(2026),
            localPhotos: [source.path],
          ),
          authorId: 'actor0000000001',
          trailId: 'trail0000000001',
        );
    expect(saved.photos, ['uploaded.jpg']);
    expect(saved.localPhotos, isEmpty);
    expect(api.requests, hasLength(1));
    expect(api.requests.single.method, 'POST');
    expect(
      api.requests.single.uri.path,
      '/api/v1/waypoint/waypoint0000001/file',
    );
  });

  for (final target in ['trail', 'waypoint']) {
    for (final switchServer in [false, true]) {
      test(
        '$target stops when ${switchServer ? 'server' : 'session'} changes during discovery',
        () async {
          final mode = Completer<PhotoApiMode>();
          final api = _Api();
          final container = ProviderContainer(
            overrides: [
              apiProvider.overrideWith(() => api),
              photoApiModeProvider.overrideWithValue(() => mode.future),
            ],
          );
          addTearDown(container.dispose);
          final Future<Object> operation;
          if (target == 'trail') {
            operation = container
                .read(trailSaveProvider.notifier)
                .createTrail(
                  Trail.empty(),
                  authorId: 'actor0000000001',
                  newPhotos: const [],
                );
          } else {
            operation = container
                .read(waypointSaveProvider.notifier)
                .create(
                  Waypoint(
                    id: '',
                    lat: 1,
                    lon: 1,
                    created: DateTime(2026),
                    updated: DateTime(2026),
                  ),
                  authorId: 'actor0000000001',
                  trailId: 'trail0000000001',
                );
          }
          await Future<void>.delayed(Duration.zero);
          if (switchServer) {
            api.updateBaseUrl('https://b.example');
          } else {
            api.renewSession();
          }
          mode.complete(PhotoApiMode.legacy);
          await expectLater(operation, throwsStateError);
          expect(api.requests, isEmpty);
        },
      );
    }
  }
}
