import 'dart:convert';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show Override;
import 'package:flutter_test/flutter_test.dart';
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/oauth_provider.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/welcome/server_compatibility_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';
import 'package:wanderer/util/oauth_app_state.dart';
import 'package:wanderer/util/server_compatibility.dart';

// ---------------------------------------------------------------------------
// Drives the real Auth.loginWithOAuth. Plain `flutter test` has no ObjectBox
// native library, so the store is a fake, and the browser is the
// flutter_web_auth_2 method channel answered by the test.
// ---------------------------------------------------------------------------

const _issuedState = 'wanderer-app.payload.signature';
const _pocketbaseState = 'pb-state';
const _legacyState = '$_pocketbaseState.wanderer-app';

const _provider = OAuthProvider(
  name: 'oidc',
  displayName: 'Fake IdP',
  state: _pocketbaseState,
  codeVerifier: 'verifier',
  url:
      'https://idp.example/authorize?client_id=wanderer&state=$_pocketbaseState'
      '&redirect_uri=https%3A%2F%2Fserver.example%2Flogin%2Fredirect',
);

const _user = {
  'id': 'user1',
  'collectionId': 'users',
  'collectionName': 'users',
  'created': '2026-01-01T00:00:00.000Z',
  'updated': '2026-01-01T00:00:00.000Z',
  'username': 'hiker',
  'email': 'hiker@example.com',
  'emailVisibility': false,
  'verified': true,
  'expand': {
    'activitypub_actors_via_user': {
      'id': 'actor1',
      'collectionId': 'actors',
      'collectionName': 'activitypub_actors',
      'created': '2026-01-01T00:00:00.000Z',
      'updated': '2026-01-01T00:00:00.000Z',
      'username': 'hiker',
      'preferred_username': 'hiker',
      'iri': 'https://server.example/api/v1/activitypub/user/hiker',
      'inbox': 'https://server.example/api/v1/activitypub/user/hiker/inbox',
      'public_key': 'key',
      'last_fetched': '2026-01-01T00:00:00.000Z',
      'user': 'user1',
    },
  },
};

class _MemoryCookieStorage implements Storage {
  final _data = <String, String>{};

  @override
  Future<void> init(bool persistSession, bool ignoreExpires) async {}

  @override
  Future<String?> read(String key) async => _data[key];

  @override
  Future<void> write(String key, String value) async => _data[key] = value;

  @override
  Future<void> delete(String key) async => _data.remove(key);

  @override
  Future<void> deleteAll(List<String> keys) async => keys.forEach(_data.remove);
}

class _FakeBox<T> extends Fake implements Box<T> {
  final _rows = <T>[];

  @override
  List<T> getAll() => List.of(_rows);

  @override
  int put(T object, {PutMode mode = PutMode.put}) {
    _rows.add(object);
    return _rows.length;
  }

  @override
  int removeAll() {
    final count = _rows.length;
    _rows.clear();
    return count;
  }

  @override
  QueryBuilder<T> query([Condition<T>? qc]) => _EmptyQueryBuilder<T>();
}

class _EmptyQueryBuilder<T> extends Fake implements QueryBuilder<T> {
  @override
  Query<T> build() => _EmptyQuery<T>();
}

class _EmptyQuery<T> extends Fake implements Query<T> {
  @override
  T? findFirst() => null;

  @override
  void close() {}
}

class _FakeStore extends Fake implements Store {
  final _boxes = <Type, Object>{};

  @override
  Box<T> box<T>() => _boxes.putIfAbsent(T, _FakeBox<T>.new) as Box<T>;
}

class _NoServerSelected extends ServerSelectionNotifier {
  @override
  Future<ServerState> build() async => ServerState(const [], null);
}

class _ServerSelected extends ServerSelectionNotifier {
  @override
  Future<ServerState> build() async => ServerState(const [], _server);
}

const _server = ServerInstance(name: 'W', url: 'https://server.example');

typedef _Answer = ResponseBody Function(RequestOptions options);

ResponseBody _json(
  Object body,
  int status, [
  Map<String, List<String>> headers = const {},
]) => ResponseBody.fromString(
  jsonEncode(body),
  status,
  headers: {
    Headers.contentTypeHeader: [Headers.jsonContentType],
    ...headers,
  },
);

/// Records requests and answers them by path. Anything unexpected is a 500, so
/// a stray request fails the test instead of hanging it.
class _Server implements HttpClientAdapter {
  _Server(this.appState);

  /// Answers `POST /auth/oauth/app-state`.
  _Answer appState;

  final requests = <RequestOptions>[];

  List<RequestOptions> to(String suffix) =>
      requests.where((r) => r.path.endsWith(suffix)).toList();

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    if (options.path.endsWith('/auth/oauth/app-state')) {
      return appState(options);
    }
    if (options.path.endsWith('/auth/oauth')) {
      return _json({'token': 'token', 'record': _user}, 200);
    }
    if (options.path.endsWith('/user/user1')) {
      return _json(_user, 200);
    }
    return _json({'message': 'unexpected request'}, 500);
  }

  @override
  void close({bool force = false}) {}
}

/// The `flutter_web_auth_2` channel: records each URL opened and answers with
/// the callback [respond] builds from the `state` the provider was given.
class _Browser {
  _Browser(this.respond);

  String Function(String sentState) respond;

  final opened = <Uri>[];

  static const _channel = MethodChannel('flutter_web_auth_2');

  void install() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_channel, (call) async {
          if (call.method != 'authenticate') return null;
          final url = Uri.parse((call.arguments as Map)['url'] as String);
          opened.add(url);
          return respond(url.queryParameters['state']!);
        });
    addTearDown(
      () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(_channel, null),
    );
  }
}

String _success(String state) {
  final query = Uri(queryParameters: {'code': 'code1', 'state': state}).query;
  return 'wanderer://oauth-callback?$query';
}

ResponseBody _issue(RequestOptions _) =>
    _json({'state': _issuedState, 'expiresIn': 600}, 200);

class _Harness {
  _Harness(this.server, this.browser, this.container);

  final _Server server;
  final _Browser browser;
  final ProviderContainer container;

  Future<void> login() =>
      container.read(authProvider.notifier).loginWithOAuth(_provider);

  AsyncValue<UserEntity?> get state => container.read(authProvider);

  Object? get error => state.error;
}

Future<_Harness> _harness({
  _Answer appState = _issue,
  String Function(String sentState) browser = _success,
  List<Override> overrides = const [],
  bool selectServer = false,
}) async {
  final server = _Server(appState);
  final fakeBrowser = _Browser(browser)..install();
  final container = ProviderContainer(
    retry: (_, _) => null,
    overrides: [
      objectBoxProvider.overrideWith(_TestObjectBox.new),
      cookieJarProvider.overrideWithValue(
        PersistCookieJar(storage: _MemoryCookieStorage()),
      ),
      serverSelectionProvider.overrideWith(
        selectServer ? _ServerSelected.new : _NoServerSelected.new,
      ),
      ...overrides,
    ],
  );
  addTearDown(container.dispose);
  container.listen(authProvider, (_, _) {});
  container.listen(serverSelectionProvider, (_, _) {});
  await container.read(serverSelectionProvider.future);
  await container.read(authProvider.future);

  container.read(apiProvider.notifier).updateBaseUrl(_server.url);
  container.read(apiProvider).httpClientAdapter = server;
  return _Harness(server, fakeBrowser, container);
}

class _TestObjectBox extends ObjectBox {
  @override
  Store build() => _FakeStore();
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('on a server with the endpoint', () {
    test('sends the issued state unchanged and completes the login', () async {
      final h = await _harness();

      await h.login();

      expect(h.error, isNull);
      expect(h.state.value?.id, 'user1');

      final request = h.server.to('/auth/oauth/app-state').single;
      expect(request.method, 'POST');
      expect(request.data, {'provider': 'oidc'});

      final opened = h.browser.opened.single;
      expect(opened.queryParameters['state'], _issuedState);
      expect(opened.query, isNot(contains(_pocketbaseState)));
      expect(opened.queryParameters['client_id'], 'wanderer');
      expect(
        opened.queryParameters['redirect_uri'],
        'https://server.example/login/redirect',
      );

      expect(h.server.to('/auth/oauth').single.data, {
        'name': 'oidc',
        'code': 'code1',
        'codeVerifier': 'verifier',
      });
    });

    test('asks for a new state on every attempt', () async {
      var issued = 0;
      final h = await _harness(
        appState: (_) => _json({'state': 'state-${++issued}'}, 200),
      );

      await h.login();
      await h.login();

      expect(h.server.to('/auth/oauth/app-state'), hasLength(2));
      expect(h.browser.opened.map((u) => u.queryParameters['state']), [
        'state-1',
        'state-2',
      ]);
    });

    test('asks only after the compatibility check passed', () async {
      final h = await _harness(
        selectServer: true,
        overrides: [
          serverCompatibilityProvider(_server.url).overrideWith(
            (ref) async => const ServerCompatibility(false, '0.20.0'),
          ),
        ],
      );

      await h.login();

      expect(h.error, isA<IncompatibleServerException>());
      expect(h.server.requests, isEmpty);
      expect(h.browser.opened, isEmpty);
    });
  });

  group('rejects a callback that does not match', () {
    for (final (name, callback) in [
      ('another state', 'wanderer://oauth-callback?code=code1&state=other'),
      (
        'the issued state with a suffix',
        'wanderer://oauth-callback?code=code1&state=$_issuedState.wanderer-app',
      ),
      (
        'the unsigned PocketBase state',
        'wanderer://oauth-callback?code=code1&state=$_pocketbaseState',
      ),
      (
        'the marker state',
        'wanderer://oauth-callback?code=code1&state=$_legacyState',
      ),
      ('no state', 'wanderer://oauth-callback?code=code1'),
    ]) {
      test(name, () async {
        final h = await _harness(browser: (_) => callback);

        await h.login();

        expect(h.error.toString(), contains('did not match'));
        expect(h.server.to('/auth/oauth'), isEmpty);
      });
    }

    test('without showing the error text it carries', () async {
      final h = await _harness(
        browser: (_) =>
            'wanderer://oauth-callback?error=x&error_description=Call%20me'
            '&state=other',
      );

      await h.login();

      expect(h.error.toString(), contains('did not match'));
      expect(h.error.toString(), isNot(contains('Call me')));
    });

    test('a matching state without a code', () async {
      final h = await _harness(
        browser: (state) => 'wanderer://oauth-callback?state=$state',
      );

      await h.login();

      expect(h.error.toString(), contains('did not match'));
      expect(h.server.to('/auth/oauth'), isEmpty);
    });
  });

  test('reports the error of a matching callback', () async {
    final h = await _harness(
      browser: (state) =>
          'wanderer://oauth-callback?error=access_denied'
          '&error_description=Denied%20at%20the%20provider&state=$state',
    );

    await h.login();

    expect(h.error.toString(), contains('Denied at the provider'));
    expect(h.server.to('/auth/oauth'), isEmpty);
  });

  group('on a server without the endpoint', () {
    // SvelteKit's error page on servers older than the JSON API errors.
    ResponseBody notFound(RequestOptions _) => ResponseBody.fromString(
      '<!doctype html><h1>Not Found</h1>',
      404,
      headers: {
        Headers.contentTypeHeader: ['text/html'],
      },
    );

    test('falls back to the marker flow on a 404', () async {
      final h = await _harness(appState: notFound);

      await h.login();

      expect(h.error, isNull);
      expect(h.state.value?.id, 'user1');
      expect(h.browser.opened.single.queryParameters['state'], _legacyState);
      expect(h.server.to('/auth/oauth').single.data, {
        'name': 'oidc',
        'code': 'code1',
        'codeVerifier': 'verifier',
      });
    });

    test('still rejects a mismatched callback', () async {
      final h = await _harness(
        appState: notFound,
        browser: (_) => _success(_pocketbaseState),
      );

      await h.login();

      expect(h.error.toString(), contains('did not match'));
      expect(h.server.to('/auth/oauth'), isEmpty);
    });

    test('remembers nothing: an updated server is used right away', () async {
      final h = await _harness(appState: notFound);

      await h.login();
      h.server.appState = _issue;
      await h.login();

      expect(h.browser.opened.map((u) => u.queryParameters['state']), [
        _legacyState,
        _issuedState,
      ]);
    });
  });

  group('fails the login before the browser opens', () {
    Future<void> expectFailure(_Answer answer, Matcher matcher) async {
      final h = await _harness(appState: answer);

      await h.login();

      expect(h.error, matcher);
      expect(h.browser.opened, isEmpty);
      expect(h.server.to('/auth/oauth'), isEmpty);
    }

    Matcher failure(OAuthAppStateFailure failure, {int? retryAfter}) =>
        isA<OAuthAppStateException>()
            .having((e) => e.failure, 'failure', failure)
            .having((e) => e.retryAfterSeconds, 'retryAfter', retryAfter);

    test('400 invalid_provider', () async {
      await expectFailure(
        (_) => _json({'message': 'invalid_provider'}, 400),
        failure(OAuthAppStateFailure.invalidProvider),
      );
    });

    test('400 invalid_params', () async {
      await expectFailure(
        (_) => _json({'message': 'invalid_params', 'detail': []}, 400),
        failure(OAuthAppStateFailure.rejected),
      );
    });

    test('429 with Retry-After', () async {
      await expectFailure(
        (_) => _json(
          {'message': 'too_many_requests'},
          429,
          {
            'retry-after': ['42'],
          },
        ),
        failure(OAuthAppStateFailure.rateLimited, retryAfter: 42),
      );
    });

    test('429 with a date in Retry-After, from a proxy', () async {
      await expectFailure(
        (_) => _json({}, 429, {
          'retry-after': ['Wed, 21 Oct 2026 07:28:00 GMT'],
        }),
        failure(OAuthAppStateFailure.rateLimited),
      );
    });

    test('503 backend_unavailable', () async {
      await expectFailure(
        (_) => _json({'message': 'backend_unavailable'}, 503),
        failure(OAuthAppStateFailure.serverUnavailable),
      );
    });

    test('502 with an HTML body, from a proxy', () async {
      await expectFailure(
        (_) => ResponseBody.fromString('<h1>Bad Gateway</h1>', 502),
        failure(OAuthAppStateFailure.serverUnavailable),
      );
    });

    test('405, which is not a server without the endpoint', () async {
      await expectFailure(
        (_) => _json({'message': 'method_not_allowed'}, 405),
        failure(OAuthAppStateFailure.rejected),
      );
    });

    for (final (name, body) in <(String, Object)>[
      ('no state', {'expiresIn': 600}),
      ('an empty state', {'state': ''}),
      ('a state of the wrong type', {'state': 1}),
    ]) {
      test('200 with $name', () async {
        await expectFailure(
          (_) => _json(body, 200),
          failure(OAuthAppStateFailure.rejected),
        );
      });
    }

    test('200 with an HTML body', () async {
      await expectFailure(
        (_) => ResponseBody.fromString('<html></html>', 200),
        failure(OAuthAppStateFailure.rejected),
      );
    });

    test('a connection failure, without falling back', () async {
      await expectFailure(
        (options) => throw DioException.connectionError(
          requestOptions: options,
          reason: 'no route to host',
        ),
        isA<DioException>(),
      );
    });
  });
}
