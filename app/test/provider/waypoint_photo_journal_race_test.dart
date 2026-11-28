import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

const _actor = 'actor0000000001';
const _target = PhotoTarget.waypoint('point0000000001');

class _Api extends Api {
  _Api(this.client);
  final Dio client;
  @override
  Dio build() => client;
}

void main() {
  test(
    'saving journal picks during a running legacy upload sends each photo once',
    () async {
      final directory = Directory.systemTemp.createTempSync(
        'waypoint-journal-race-',
      );
      addTearDown(() => directory.delete(recursive: true));
      final first = File('${directory.path}/first.jpg')..writeAsBytesSync([1]);
      final second = File('${directory.path}/second.jpg')
        ..writeAsBytesSync([2]);
      final entered = Completer<void>();
      final release = Completer<void>();
      final sent = <String>[];
      final serverPhotos = <String>[];
      final waypoint = Waypoint(
        id: _target.id,
        lat: 46,
        lon: 7,
        author: _actor,
        created: DateTime(2026),
        updated: DateTime(2026),
      );
      final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
      api.interceptors.add(
        InterceptorsWrapper(
          onRequest: (request, handler) async {
            if (request.uri.path.endsWith('/file')) {
              final batch = (request.data as FormData).files
                  .map((entry) => entry.value.filename!)
                  .toList();
              sent.addAll(batch);
              if (!entered.isCompleted) {
                entered.complete();
                await release.future;
              }
              serverPhotos.addAll(batch);
            }
            handler.resolve(
              Response(
                requestOptions: request,
                statusCode: 200,
                data: jsonDecode(
                  jsonEncode(
                    waypoint.copyWith(photos: List.of(serverPhotos)).toJson(),
                  ),
                ),
              ),
            );
          },
        ),
      );
      final journal = PhotoMutationService(
        api: api,
        resolveMode: () async => PhotoApiMode.legacy,
        documentsDirectory: () async => directory,
      );
      final container = ProviderContainer(
        overrides: [
          apiProvider.overrideWith(() => _Api(api)),
          photoApiModeProvider.overrideWithValue(
            () async => PhotoApiMode.legacy,
          ),
          photoMutationServiceProvider.overrideWithValue(journal),
        ],
      );
      addTearDown(container.dispose);
      final uploading = journal.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [first.path],
      );
      await entered.future;
      final jobFile = await directory
          .list(recursive: true)
          .where((entry) => entry is File && entry.path.endsWith('.json'))
          .cast<File>()
          .single;
      final job = jsonDecode(await jobFile.readAsString()) as Map;
      final stored = ((job['files'] as List).single as Map)['path'] as String;
      final pendingCopy = '${directory.path}/$stored';
      final editing = container
          .read(waypointSaveProvider.notifier)
          .updateWaypoint(
            waypoint,
            waypoint.copyWith(
              name: 'Renamed point',
              localPhotos: [pendingCopy, second.path],
            ),
            authorId: _actor,
          );
      release.complete();
      expect((await uploading).pending, isFalse);
      final saved = await editing;
      expect(sent, ['first.jpg', 'second.jpg']);
      expect(saved.photos, ['first.jpg', 'second.jpg']);
      expect(saved.localPhotos, isEmpty);
    },
  );
}
