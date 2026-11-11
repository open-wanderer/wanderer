import 'dart:convert';
import 'dart:io';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/components/toast_overlay.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/user.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/store/active_navigation_store.dart' as active_nav;
import 'package:wanderer/store/app_photo_paths.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/util/geo/polyline.dart';

const _account = 'account00000001';
const _actor = 'actor0000000001';
const _target = PhotoTarget.trail('trail0000000001');
const _stamp = '2026-01-01T00:00:00Z';
const _navRoute = '/trail/trail0000000001/navigate';

Map<String, dynamic> _user() => {
  'id': _account,
  'collectionId': 'users',
  'collectionName': 'users',
  'created': _stamp,
  'updated': _stamp,
  'username': 'hiker',
  'email': 'hiker@a.example',
  'emailVisibility': false,
  'verified': true,
  'expand': {
    'activitypub_actors_via_user': {
      'id': _actor,
      'collectionId': 'actors',
      'collectionName': 'activitypub_actors',
      'created': _stamp,
      'updated': _stamp,
      'username': 'hiker',
      'preferred_username': 'hiker',
      'iri': 'https://a.example/$_actor',
      'inbox': 'https://a.example/$_actor/inbox',
      'public_key': 'key',
      'last_fetched': _stamp,
      'user': _account,
    },
  },
};

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
    for (final key in keys) _values.remove(key);
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
  late bool guest;
  late int uploads;
  late int userRequests;
  late int photoRequests;
  late int captures;
  late List<String> copies;
  late String breadcrumb;
  GoRouter? uiRouter;
  ProviderSubscription<bool>? connectivity;

  Future<void> openContainer() async {
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
    container.listen(authProvider, (_, _) {});
    await container.read(authProvider.future);
  }

  setUp(() async {
    directory = Directory.systemTemp.createTempSync('photo-active-session-');
    previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Documents(directory.path);
    configurePhotoDocumentsDirectory(directory.path);
    store = await openStore(directory: '${directory.path}/objectbox');
    store.box<UserEntity>().put(
      User.fromJson(_user()).toEntity(serverUrl: 'https://a.example'),
    );
    jar = PersistCookieJar(storage: _CookieStorage());
    await jar.saveFromResponse(Uri.parse('https://a.example'), [
      Cookie('pb_auth', 'session')
        ..expires = DateTime.now().add(const Duration(days: 1)),
    ]);
    guest = false;
    uploads = 0;
    userRequests = 0;
    photoRequests = 0;
    captures = 0;
    breadcrumb = PolylineUtil.encode(const [
      Geographic(lat: 46.0, lon: 7.0),
      Geographic(lat: 46.0001, lon: 7.0001),
      Geographic(lat: 46.0002, lon: 7.0003),
    ]);
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) {
          final path = request.uri.path;
          var status = 200;
          dynamic data;
          if (path.endsWith('/auth/login')) {
            data = {'token': 'session', 'record': _user()};
          } else if (path.contains('/user/')) {
            userRequests++;
            status = guest ? 404 : 200;
            data = guest
                ? {'message': 'The requested resource was not found.'}
                : _user();
          } else if (path.endsWith('/capabilities')) {
            data = {'assetApi': true};
          } else if (path.endsWith('/health')) {
            data = {'status': 'ok'};
          } else if (path.endsWith('/assets')) {
            photoRequests++;
            status = guest ? 503 : 200;
            if (guest) {
              data = {'message': 'Try again'};
            } else {
              uploads += (request.data as FormData).files.length;
              data = [
                {
                  'id': 'asset0000000001',
                  'collectionId': 'assets',
                  'file': 'picked.jpg',
                },
              ];
            }
          } else if (path.endsWith('/trail/${_target.id}')) {
            photoRequests++;
            status = guest ? 404 : 200;
            data = guest
                ? {'message': 'The requested resource was not found.'}
                : {
                    'id': _target.id,
                    'photos': ['/api/v1/files/assets/asset0000000002/old.jpg'],
                  };
          } else if (path.contains('/assets/') && request.method == 'DELETE') {
            photoRequests++;
            data = <String, dynamic>{};
          } else if (path.endsWith('/trail/form') && request.method == 'PUT') {
            captures++;
            data = {
              'id': 'trail0000000002',
              'name': 'Recorded hike',
              'author': _actor,
              'created': _stamp,
              'updated': _stamp,
              'photos': <String>[],
            };
          } else {
            throw StateError('Unexpected ${request.method} ${request.uri}');
          }
          final response = Response<Object>(
            requestOptions: request,
            statusCode: status,
            data: data,
            headers: Headers.fromMap({
              Headers.contentTypeHeader: [Headers.jsonContentType],
            }),
          );
          if (status >= 400 && request.validateStatus(status) != true) {
            handler.reject(
              DioException(requestOptions: request, response: response),
            );
          } else {
            handler.resolve(response);
          }
        },
      ),
    );
    await openContainer();
    for (var i = 0; i < 100 && userRequests == 0; i++) {
      await Future<void>.delayed(const Duration(milliseconds: 2));
    }
    expect(userRequests, greaterThan(0));
    await Future<void>.delayed(const Duration(milliseconds: 5));
  });

  tearDown(() async {
    connectivity?.close();
    connectivity = null;
    uiRouter?.dispose();
    uiRouter = null;
    container.dispose();
    store.close();
    PathProviderPlatform.instance = previousPaths;
    await directory.delete(recursive: true);
  });

  Future<void> begin(ActiveSessionType type) async {
    active_nav.save(
      store,
      ActiveNavigationEntity(
        sessionType: type,
        trailId: type == ActiveSessionType.nav ? _target.id : null,
        breadcrumbPolyline: breadcrumb,
        elevations: [500, 501, 502],
        timestampsUtc: [1767225600000, 1767225601000, 1767225602000],
        currentElapsedSeconds: 2,
        updatedAtUtc: DateTime.utc(2026),
      ),
    );
    final picked = File('${directory.path}/picked.jpg')
      ..writeAsBytesSync([1, 2, 3]);
    final journal = container.read(photoMutationServiceProvider);
    await journal.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [picked.path],
      existingPhotos: ['old.jpg'],
      removedPhotos: ['old.jpg'],
      mode: PhotoApiMode.unknown,
    );
    copies = await journal.pendingPaths(_target, _actor);
    expect(copies, hasLength(1));
    await picked.delete();
  }

  Future<void> connect() async {
    Future<void>? drain;
    connectivity?.close();
    connectivity = container.listen<bool>(onlineStatusProvider, (
      previous,
      next,
    ) {
      if (previous == false && next) {
        drain = container.read(trailSyncProvider.notifier).drainIfOnline();
      }
    });
    container.read(onlineStatusProvider.notifier).markOffline();
    container.read(onlineStatusProvider.notifier).markOnline();
    expect(drain, isNotNull);
    await drain;
  }

  Future<void> mount(WidgetTester tester, String route) async {
    // Reuse the production auth redirect and its refresh provider. Map/tracelet
    // screens need a device, so only the leaf page is replaced by a label.
    container.listen(routerProvider, (_, _) {});
    final production = container.read(routerProvider);
    uiRouter = GoRouter(
      initialLocation: route,
      refreshListenable: container.read(routerListenableProvider),
      redirect: production.configuration.topRedirect,
      routes: [
        for (final path in [
          '/record',
          _navRoute,
          '/welcome',
          '/map',
          '/trail/create/edit',
        ])
          GoRoute(
            path: path,
            builder: (context, state) => Scaffold(body: Text(state.uri.path)),
          ),
      ],
    );
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: uiRouter,
          builder: (context, child) => ToastOverlay(child: child!),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(uiRouter!.state.uri.path, route);
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
    });
  }

  Future<void> makeRetryDue() async {
    await for (final entry in directory.list(recursive: true)) {
      if (entry is! File ||
          !entry.path.endsWith('.json') ||
          !entry.path.contains('photo-mutations'))
        continue;
      final job =
          jsonDecode(await entry.readAsString()) as Map<String, dynamic>;
      job['nextAttempt'] = DateTime.utc(2000).toIso8601String();
      await entry.writeAsString(jsonEncode(job));
    }
  }

  Future<void> waitForSignOut() async {
    for (var i = 0; i < 400; i++) {
      final auth = container.read(authProvider);
      if (store.box<UserEntity>().getAll().isEmpty &&
          !auth.isLoading &&
          auth.value == null)
        break;
      await Future<void>.delayed(const Duration(milliseconds: 5));
    }
    expect(store.box<UserEntity>().getAll(), isEmpty);
    expect(container.read(authProvider).isLoading, isFalse);
    expect(container.read(authProvider).value, isNull);
  }

  Future<void> restart(WidgetTester tester, {bool discard = false}) async {
    await tester.pumpWidget(const SizedBox.shrink());
    connectivity?.close();
    connectivity = null;
    uiRouter?.dispose();
    uiRouter = null;
    container.dispose();
    if (discard) active_nav.clear(store);
    await tester.runAsync(() async {
      await jar.deleteAll();
      await openContainer();
    });
  }

  for (final type in ActiveSessionType.values) {
    testWidgets(
      'a revoked session keeps the ${type.name} breadcrumb and screen when connectivity returns',
      (tester) async {
        await tester.runAsync(() => begin(type));
        final route = type == ActiveSessionType.rec ? '/record' : _navRoute;
        await mount(tester, route);
        final validationBefore = userRequests;
        guest = true;
        await tester.runAsync(connect);
        await tester.pumpAndSettle();
        final active = active_nav.read(store);
        debugPrint(
          'After connectivity: active=${active != null}, breadcrumb=${active?.breadcrumbPolyline}, route=${uiRouter!.state.uri.path}',
        );
        expect(userRequests, greaterThan(validationBefore));
        expect(active?.breadcrumbPolyline, breadcrumb);
        expect(active?.timestampsUtc, [
          1767225600000,
          1767225601000,
          1767225602000,
        ]);
        expect(uiRouter!.state.uri.path, route);
        expect(find.text(route), findsOneWidget);
        expect(find.text('/welcome'), findsNothing);
        expect(store.box<UserEntity>().getAll().single.id, _account);
        expect(copies.every((path) => File(path).existsSync()), isTrue);
        expect(uploads, 0);
        final hint = lookupAppLocalizations(const Locale('en'))
            .photos_session_expired;
        expect(container.read(toastProvider).single.text, hint);
        expect(find.text(hint), findsOneWidget);
        expect(find.byType(SnackBar), findsOneWidget);
        final sent = photoRequests;
        await tester.runAsync(connect);
        expect(photoRequests, sent);
      },
    );
  }

  testWidgets(
    'a finished recording is saved before deferred sign-out and both uploads resume after the same account returns',
    (tester) async {
      await tester.runAsync(() => begin(ActiveSessionType.rec));
      await mount(tester, '/record');
      guest = true;
      await tester.runAsync(connect);
      await tester.pumpAndSettle();
      expect(active_nav.read(store)?.breadcrumbPolyline, breadcrumb);
      const localId = 'local-1-1';
      const recordedXml =
          '<gpx version="1.1"><trk><trkseg>'
          '<trkpt lat="46" lon="7"><ele>500</ele></trkpt>'
          '<trkpt lat="46.0001" lon="7.0001"><ele>501</ele></trkpt>'
          '<trkpt lat="46.0002" lon="7.0003"><ele>502</ele></trkpt>'
          '</trkseg></trk></gpx>';
      final draft = Trail.empty().copyWith(
        name: 'Recorded hike',
        expand: const TrailExpand(gpxData: recordedXml),
        originalGpxData: recordedXml,
      );
      await tester.runAsync(() async {
        persistDraftThenClearSession(
          writeDraft: () => saveNewLocalTrail(
            store,
            trail: draft,
            ownerAccountId: _account,
            authorActorId: _actor,
            localId: localId,
            trailLocalPhotos: const [],
            waypointLocalPhotosByKey: const {},
            asDraft: true,
          ),
          clearSession: () => active_nav.clear(store),
        );
        await waitForSignOut();
      });
      await tester.pumpAndSettle();
      expect(uiRouter!.state.uri.path, '/welcome');
      final saved = store.box<TrailEntity>().getAll().single;
      expect(saved.localId, localId);
      expect(saved.owner, _account);
      expect(saved.gpxData, recordedXml);
      expect(saved.originalGpxData, recordedXml);
      expect(saved.syncState, TrailSyncState.draft);
      expect(copies.every((path) => File(path).existsSync()), isTrue);
      expect(uploads, 0);
      await tester.runAsync(() async {
        guest = false;
        await container.read(authProvider.notifier).login('hiker', 'password');
        expect(
          promoteDraftToPending(store, localId, accountId: _account),
          isTrue,
        );
        await makeRetryDue();
        await connect();
        await connect();
      });
      await tester.pumpAndSettle();
      expect(uploads, 1);
      expect(captures, 1);
      expect(
        await tester.runAsync(
          () => container
              .read(photoMutationServiceProvider)
              .pendingPaths(_target, _actor),
        ),
        isEmpty,
      );
      final rows = store.box<TrailEntity>().getAll();
      expect(
        rows.every(
          (row) =>
              row.id == 'trail0000000002' &&
              row.syncState == TrailSyncState.synced,
        ),
        isTrue,
      );
    },
  );

  testWidgets(
    'an invalid session survives restart with an active recording and signs out after discard',
    (tester) async {
      await tester.runAsync(() => begin(ActiveSessionType.rec));
      await mount(tester, '/record');
      guest = true;
      await tester.runAsync(connect);
      await tester.pumpAndSettle();
      expect(active_nav.read(store)?.breadcrumbPolyline, breadcrumb);
      final sent = photoRequests;
      await restart(tester);
      await mount(tester, '/record');
      expect(active_nav.read(store)?.breadcrumbPolyline, breadcrumb);
      expect(store.box<UserEntity>().getAll().single.id, _account);
      await tester.runAsync(connect);
      expect(photoRequests, sent);
      await tester.runAsync(() async {
        active_nav.clear(store);
        await waitForSignOut();
      });
      await tester.pumpAndSettle();
      expect(uiRouter!.state.uri.path, '/welcome');
      expect(copies.every((path) => File(path).existsSync()), isTrue);
      expect(uploads, 0);
    },
  );

  testWidgets(
    'an invalid session signs out on restart when no active session remains',
    (tester) async {
      await tester.runAsync(() => begin(ActiveSessionType.rec));
      await mount(tester, '/record');
      guest = true;
      await tester.runAsync(connect);
      await tester.pumpAndSettle();
      expect(active_nav.read(store)?.breadcrumbPolyline, breadcrumb);
      await restart(tester, discard: true);
      await tester.runAsync(waitForSignOut);
      await mount(tester, '/welcome');
      expect(copies.every((path) => File(path).existsSync()), isTrue);
      expect(uploads, 0);
    },
  );

  testWidgets(
    'an interactive invalid-session response cannot purge a recording',
    (tester) async {
      await tester.runAsync(() => begin(ActiveSessionType.rec));
      await mount(tester, '/record');
      final request = RequestOptions(path: '/user/$_account');
      final invalid = DioException(
        requestOptions: request,
        response: Response(
          requestOptions: request,
          statusCode: 404,
          data: {'message': 'The requested resource was not found.'},
        ),
      );
      await tester.runAsync(
        () => container
            .read(authProvider.notifier)
            .signOutIfSessionInvalid(invalid),
      );
      await tester.pumpAndSettle();
      expect(active_nav.read(store)?.breadcrumbPolyline, breadcrumb);
      expect(uiRouter!.state.uri.path, '/record');
      expect(copies.every((path) => File(path).existsSync()), isTrue);
    },
  );
}
