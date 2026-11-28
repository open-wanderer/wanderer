import 'dart:convert';
import 'dart:io';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/user.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

const _account = 'account00000001';
const _actor = 'actor0000000001';
const _otherAccount = 'account00000002';
const _otherActor = 'actor0000000002';
const _target = PhotoTarget.trail('trail0000000001');
const _stamp = '2026-01-01T00:00:00Z';

Map<String, dynamic> _body(String id) {
  final actor = id == _account ? _actor : _otherActor;
  return {
    'id': id,
    'collectionId': 'users',
    'collectionName': 'users',
    'created': _stamp,
    'updated': _stamp,
    'username': id,
    'email': 'hiker@a.example',
    'emailVisibility': false,
    'verified': true,
    'expand': {
      'activitypub_actors_via_user': {
        'id': actor,
        'collectionId': 'actors',
        'collectionName': 'activitypub_actors',
        'created': _stamp,
        'updated': _stamp,
        'username': id,
        'preferred_username': id,
        'iri': 'https://a.example/$actor',
        'inbox': 'https://a.example/$actor/inbox',
        'public_key': 'key',
        'last_fetched': _stamp,
        'user': id,
      },
    },
  };
}

class _Documents extends PathProviderPlatform {
  _Documents(this.path);
  final String path;
  @override
  Future<String?> getApplicationDocumentsPath() async => path;
}

class _Api extends Api {
  _Api(this.client);
  final Dio client;
  @override
  Dio build() => client;
}

class _Selection extends ServerSelectionNotifier {
  @override
  Future<ServerState> build() async => ServerState(const [], null);
}

class _CookieStorage implements Storage {
  final _values = <String, String>{};
  @override
  Future<void> init(bool persistSession, bool ignoreExpires) async {}
  @override
  Future<String?> read(String key) async => _values[key];
  @override
  Future<void> write(String key, String value) async => _values[key] = value;
  @override
  Future<void> delete(String key) async => _values.remove(key);
  @override
  Future<void> deleteAll(List<String> keys) async {
    for (final key in keys) {
      _values.remove(key);
    }
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform previousPaths;
  late Store store;
  late Dio api;
  late ProviderContainer container;
  late PersistCookieJar jar;
  late File picked;
  late int validationStatus;
  late bool proxyValidationPage;
  late int uploads;
  late int userRequests;
  late String loginAccount;
  late bool guest;
  late bool assetServer;
  late int guestUploadStatus;
  late int targetLookups;
  late List<String> serverPhotos;

  setUp(() async {
    directory = Directory.systemTemp.createTempSync('photo-auth-resume-');
    previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Documents(directory.path);
    store = await openStore(directory: '${directory.path}/objectbox');
    store.box<UserEntity>().put(
      User.fromJson(_body(_account)).toEntity(serverUrl: 'https://a.example'),
    );
    jar = PersistCookieJar(storage: _CookieStorage());
    await jar.saveFromResponse(Uri.parse('https://a.example'), [
      Cookie('pb_auth', 'session')
        ..expires = DateTime.now().add(const Duration(days: 1)),
    ]);
    picked = File('${directory.path}/picked.jpg')..writeAsBytesSync([1, 2, 3]);
    validationStatus = 200;
    proxyValidationPage = false;
    uploads = 0;
    userRequests = 0;
    loginAccount = _account;
    guest = false;
    assetServer = true;
    guestUploadStatus = 401;
    targetLookups = 0;
    serverPhotos = ['/api/v1/files/assets/asset0000000002/old.jpg'];
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          int status = 200;
          dynamic data;
          if (options.uri.path.endsWith('/auth/login')) {
            data = {'token': 'session', 'record': _body(loginAccount)};
          } else if (options.uri.path.contains('/user/')) {
            userRequests++;
            status = validationStatus;
            data = status == 200
                ? _body(options.uri.pathSegments.last)
                : proxyValidationPage
                ? '<html><h1>404 Not Found</h1></html>'
                : {
                    'code': status,
                    'message': "The requested resource wasn't found.",
                    'data': <String, dynamic>{},
                  };
          } else if (options.uri.path.endsWith('/capabilities')) {
            status = assetServer ? 200 : 404;
            data = assetServer ? {'assetApi': true} : {'message': 'Not found'};
          } else if (options.uri.path.endsWith('/assets') ||
              options.uri.path.endsWith('/trail/form/${_target.id}')) {
            if (guest) {
              status = guestUploadStatus;
              data = {'message': 'Session required'};
            } else {
              uploads++;
              serverPhotos.add(
                assetServer
                    ? '/api/v1/files/assets/asset0000000001/picked.jpg'
                    : 'picked.jpg',
              );
              data = assetServer
                  ? [
                      {
                        'id': 'asset0000000001',
                        'collectionId': 'assets',
                        'file': 'picked.jpg',
                      },
                    ]
                  : {'photos': List.of(serverPhotos)};
            }
          } else if (options.uri.path.endsWith('/trail/${_target.id}')) {
            if (options.method == 'GET') targetLookups++;
            status = guest ? 404 : 200;
            data = guest
                ? {'message': 'The requested resource was not found.'}
                : {'id': _target.id, 'photos': List.of(serverPhotos)};
          } else if (options.uri.path.endsWith(
                '/trail/${_target.id}/thumbnail',
              ) ||
              options.method == 'DELETE') {
            data = <String, dynamic>{};
          } else {
            throw StateError(
              'Unexpected request ${options.method} ${options.uri}',
            );
          }
          if (status >= 400 && options.validateStatus(status) != true) {
            handler.reject(
              DioException(
                requestOptions: options,
                response: Response(
                  requestOptions: options,
                  statusCode: status,
                  data: data,
                  headers: Headers.fromMap({
                    Headers.contentTypeHeader: [
                      data is Map ? Headers.jsonContentType : 'text/html',
                    ],
                  }),
                ),
              ),
            );
          } else {
            handler.resolve(
              Response(requestOptions: options, statusCode: status, data: data),
            );
          }
        },
      ),
    );
    container = ProviderContainer(
      retry: (_, _) => null,
      overrides: [
        apiProvider.overrideWith(() => _Api(api)),
        objectBoxProvider.overrideWithValue(store),
        cookieJarProvider.overrideWithValue(jar),
        serverSelectionProvider.overrideWith(_Selection.new),
      ],
    );
    container.listen(serverSelectionProvider, (_, _) {});
    await container.read(serverSelectionProvider.future);
  });

  tearDown(() async {
    container.dispose();
    store.close();
    PathProviderPlatform.instance = previousPaths;
    await directory.delete(recursive: true);
  });

  Future<void> addPending() async {
    await container
        .read(photoMutationServiceProvider)
        .apply(
          target: _target,
          actorId: _actor,
          newPhotos: [picked.path],
          mode: PhotoApiMode.unknown,
        );
  }

  Future<void> restoreAuth() async {
    container.listen(authProvider, (_, _) {});
    await container.read(authProvider.future);
    for (var i = 0; i < 50 && userRequests == 0; i++) {
      await Future<void>.delayed(const Duration(milliseconds: 2));
    }
    expect(userRequests, greaterThan(0));
  }

  test('background API JSON 404 ends the session and preserves pending photos until the same account signs in again', () async {
    await addPending();
    validationStatus = 404;
    await restoreAuth();
    for (
      var i = 0;
      i < 100 && store.box<UserEntity>().getAll().isNotEmpty;
      i++
    ) {
      await Future<void>.delayed(const Duration(milliseconds: 2));
    }
    expect(store.box<UserEntity>().getAll(), isEmpty);
    expect(
      await container
          .read(photoMutationServiceProvider)
          .pendingPaths(_target, _actor),
      hasLength(1),
    );
    await container
        .read(photoMutationServiceProvider)
        .drain(_actor, mode: PhotoApiMode.assets);
    expect(uploads, 0);
    validationStatus = 200;
    await container.read(authProvider.notifier).login('hiker', 'password');
    expect(container.read(authProvider).value?.id, _account);
    final result = await container
        .read(photoMutationServiceProvider)
        .drain(_actor, mode: PhotoApiMode.assets);
    expect(result.pending, isFalse);
    expect(uploads, 1);
  });

  test(
    'a proxy 404 validation page keeps the cached account and its pending work',
    () async {
      await addPending();
      validationStatus = 404;
      proxyValidationPage = true;
      await restoreAuth();
      await Future<void>.delayed(const Duration(milliseconds: 20));
      expect(store.box<UserEntity>().getAll().single.id, _account);
      expect(container.read(authProvider).value?.id, _account);
      expect(
        await container
            .read(photoMutationServiceProvider)
            .pendingPaths(_target, _actor),
        hasLength(1),
      );
    },
  );

  test('switching accounts keeps old pending photos scoped until their owner returns', () async {
    await addPending();
    await restoreAuth();
    loginAccount = _otherAccount;
    await container.read(authProvider.notifier).login('other', 'password');
    expect(container.read(authProvider).value?.id, _otherAccount);
    final journal = container.read(photoMutationServiceProvider);
    expect(await journal.pendingPaths(_target, _otherActor), isEmpty);
    await journal.drain(_otherActor, mode: PhotoApiMode.assets);
    await journal.drain(_actor, mode: PhotoApiMode.assets);
    expect(uploads, 0);
    expect(await journal.pendingPaths(_target, _actor), hasLength(1));
    loginAccount = _account;
    await container.read(authProvider.notifier).login('hiker', 'password');
    await container
        .read(photoMutationServiceProvider)
        .drain(_actor, mode: PhotoApiMode.assets);
    expect(uploads, 1);
  });

  for (final scenario in [
    (
      assets: true,
      uploadStatus: 503,
      label: 'asset lookup after a temporary upload failure',
    ),
    (
      assets: true,
      uploadStatus: 401,
      label: 'asset API 401 before a legacy-name removal',
    ),
    (
      assets: false,
      uploadStatus: 401,
      label: 'v0.21.2 upload 401 before a cover change',
    ),
  ]) {
    test(
      'guest ${scenario.label} keeps the job and only photo copy until the same account returns',
      () async {
        assetServer = scenario.assets;
        await restoreAuth();
        final journal = container.read(photoMutationServiceProvider);
        await journal.apply(
          target: _target,
          actorId: _actor,
          newPhotos: [picked.path],
          existingPhotos: ['old.jpg'],
          removedPhotos: scenario.assets ? ['old.jpg'] : [],
          setCover: !scenario.assets,
          coverPhoto: 'old.jpg',
          mode: PhotoApiMode.unknown,
        );
        final copies = await journal.pendingPaths(_target, _actor);
        expect(copies, hasLength(1));
        await picked.delete();
        guest = true;
        guestUploadStatus = scenario.uploadStatus;
        validationStatus = 404;
        await container.read(trailSyncProvider.notifier).drainIfOnline();
        expect(store.box<UserEntity>().getAll(), isEmpty);
        expect(container.read(toastProvider), isEmpty);
        expect(copies.every((path) => File(path).existsSync()), isTrue);
        expect(
          await container
              .read(photoMutationServiceProvider)
              .pendingPaths(_target, _actor),
          copies,
        );
        expect(targetLookups, scenario.uploadStatus == 401 ? 0 : 2);
        expect(uploads, 0);

        // Advance the persisted retry deadline without waiting thirty seconds.
        final job =
            (await directory
                    .list(recursive: true)
                    .where(
                      (entry) =>
                          entry is File &&
                          entry.path.endsWith('.json') &&
                          entry.path.contains('photo-mutations'),
                    )
                    .toList())
                .cast<File>()
                .single;
        final saved =
            jsonDecode(await job.readAsString()) as Map<String, dynamic>;
        saved['nextAttempt'] = DateTime.utc(2000).toIso8601String();
        await job.writeAsString(jsonEncode(saved));
        guest = false;
        validationStatus = 200;
        await container.read(authProvider.notifier).login('hiker', 'password');
        await container.read(trailSyncProvider.notifier).drainIfOnline();
        await container.read(trailSyncProvider.notifier).drainIfOnline();
        expect(uploads, 1);
        expect(
          await container
              .read(photoMutationServiceProvider)
              .pendingPaths(_target, _actor),
          isEmpty,
        );
        expect(container.read(toastProvider), isEmpty);
      },
    );
  }

  test('confirmed sign-out removes standalone pending photos', () async {
    await addPending();
    await restoreAuth();
    await container.read(authProvider.notifier).logout(purgePhotoJobs: true);
    expect(
      await container
          .read(photoMutationServiceProvider)
          .pendingPaths(_target, _actor),
      isEmpty,
    );
  });
}
