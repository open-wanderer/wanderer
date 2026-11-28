import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/services/asset_photo_client.dart';

const actor = 'actor0000000001';
Trail trail() => Trail.empty().copyWith(id: 'trail0000000001', author: actor);
Waypoint waypoint() => Waypoint(
  id: 'waypoint0000001',
  lat: 1,
  lon: 1,
  author: actor,
  created: DateTime(2026),
  updated: DateTime(2026),
);

class _Api extends Api {
  _Api(this.api);
  final Dio api;
  @override
  Dio build() => api;
}

class _Auth extends Auth {
  @override
  UserEntity? build() => null;
}

void main() {
  late Directory directory;
  late File photo;
  late Dio api;
  late PhotoMutationService service;
  late ProviderContainer container;
  late PhotoApiMode mode;
  late List<RequestOptions> requests;
  late bool rejectDelete;
  void Function(RequestOptions)? observe;

  Future<List<File>> jobs() async => [
    await for (final entry in directory.list(recursive: true))
      if (entry is File && entry.path.endsWith('.json')) entry,
  ];
  setUp(() {
    directory = Directory.systemTemp.createTempSync('photo-job-ack');
    photo = File('${directory.path}/camera.jpg')..writeAsBytesSync([1]);
    mode = PhotoApiMode.assets;
    requests = [];
    rejectDelete = false;
    observe = null;
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          observe?.call(options);
          if (options.method == 'DELETE' && rejectDelete) {
            handler.reject(
              DioException(
                requestOptions: options,
                response: Response(requestOptions: options, statusCode: 503),
              ),
            );
            return;
          }
          final dynamic data;
          if (options.uri.path.endsWith('/assets')) {
            data = [
              for (final entry in (options.data as FormData).files)
                {
                  'id': 'asset0000000001',
                  'collectionId': 'assets',
                  'file': entry.value.filename,
                },
            ];
          } else if (options.method == 'DELETE') {
            data = {'acknowledged': true};
          } else {
            data = jsonDecode(
              jsonEncode(
                options.uri.path.contains('/waypoint')
                    ? waypoint().toJson()
                    : trail().toJson(),
              ),
            );
          }
          handler.resolve(
            Response(requestOptions: options, statusCode: 200, data: data),
          );
        },
      ),
    );
    service = PhotoMutationService(
      api: api,
      resolveMode: () async => mode,
      documentsDirectory: () async => directory,
    );
    container = ProviderContainer(
      overrides: [
        apiProvider.overrideWith(() => _Api(api)),
        authProvider.overrideWith(() => _Auth()),
        photoApiModeProvider.overrideWithValue(() async => mode),
        photoMutationServiceProvider.overrideWithValue(service),
      ],
    );
  });
  tearDown(() async {
    container.dispose();
    await directory.delete(recursive: true);
  });

  for (final target in ['trail', 'waypoint']) {
    test(
      'successful interactive $target save acknowledges its completed job',
      () async {
        if (target == 'trail') {
          await container
              .read(trailSaveProvider.notifier)
              .createTrail(
                trail().copyWith(id: ''),
                authorId: actor,
                newPhotos: [photo],
              );
        } else {
          await container
              .read(waypointSaveProvider.notifier)
              .create(
                waypoint().copyWith(id: '', localPhotos: [photo.path]),
                authorId: actor,
                trailId: trail().id,
              );
        }
        expect(await jobs(), isEmpty);
        expect(
          requests.where((request) => request.uri.path.endsWith('/assets')),
          hasLength(1),
        );
      },
    );

    test('failed $target deletion preserves pending photo work', () async {
      mode = PhotoApiMode.unknown;
      if (target == 'trail') {
        await container
            .read(trailSaveProvider.notifier)
            .updateTrail(
              trail(),
              trail(),
              authorId: actor,
              newPhotos: [photo],
              removedPhotoFilenames: const [],
            );
      } else {
        await container
            .read(waypointSaveProvider.notifier)
            .updateWaypoint(
              waypoint(),
              waypoint().copyWith(localPhotos: [photo.path]),
              authorId: actor,
            );
      }
      rejectDelete = true;
      final Future<void> failed = target == 'trail'
          ? container.read(trailSaveProvider.notifier).deleteTrail(trail())
          : container.read(waypointSaveProvider.notifier).delete(waypoint());
      await expectLater(failed, throwsA(isA<DioException>()));
      expect(await service.pendingCount(actor), 1);
      rejectDelete = false;
      if (target == 'trail') {
        await container.read(trailSaveProvider.notifier).deleteTrail(trail());
      } else {
        await container.read(waypointSaveProvider.notifier).delete(waypoint());
      }
      expect(await jobs(), isEmpty);
    });
  }

  test(
    'a server switch between child saves prevents the next child write',
    () async {
      observe = (options) {
        if (options.uri.path == '/api/v1/waypoint') {
          api.options.baseUrl = 'https://b.example/api/v1';
        }
      };
      final input = trail().copyWith(
        id: '',
        expand: TrailExpand(
          waypointsViaTrail: [
            waypoint().copyWith(id: ''),
            waypoint().copyWith(id: ''),
          ],
        ),
      );
      await expectLater(
        container
            .read(trailSaveProvider.notifier)
            .createTrail(input, authorId: actor, newPhotos: const []),
        throwsStateError,
      );
      expect(
        requests.where((request) => request.uri.path == '/api/v1/waypoint'),
        hasLength(1),
      );
      expect(
        requests.every((request) => request.uri.host == 'a.example'),
        isTrue,
      );
    },
  );

  test(
    'deleting an unexpanded trail also removes its waypoint photo jobs',
    () async {
      mode = PhotoApiMode.unknown;
      await service.apply(
        target: PhotoTarget.trail(trail().id),
        actorId: actor,
        newPhotos: [photo.path],
      );
      await service.apply(
        target: PhotoTarget.waypoint(waypoint().id),
        actorId: actor,
        newPhotos: [photo.path],
      );
      api.interceptors.insert(
        0,
        InterceptorsWrapper(
          onRequest: (options, handler) {
            if (options.method == 'GET' &&
                options.uri.path.endsWith('/trail/${trail().id}')) {
              expect(options.queryParameters['expand'], 'waypoints_via_trail');
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: 200,
                  data: {
                    ...trail().toJson(),
                    'expand': {
                      'waypoints_via_trail': [waypoint().toJson()],
                    },
                  },
                ),
              );
            } else {
              handler.next(options);
            }
          },
        ),
      );
      await container.read(trailSaveProvider.notifier).deleteTrail(trail());
      expect(await jobs(), isEmpty);
    },
  );
}
