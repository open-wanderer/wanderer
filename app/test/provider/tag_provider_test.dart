import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';

/// How the fake backend answers the one `GET /tag?filter=…` this provider
/// issues.
enum _Outcome {
  /// 200 with a well-formed PocketBase list payload.
  ok,

  /// The device is offline — `isConnectionFailure` returns true for this.
  connectionError,

  /// The backend answered, badly. `isConnectionFailure` returns false, so
  /// this must NOT be treated as "offline".
  serverError,
}

class _FakeApi extends Api {
  _FakeApi(this.outcome);

  final _Outcome outcome;

  int requestCount = 0;
  RequestOptions? lastRequest;

  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requestCount++;
          lastRequest = options;
          switch (outcome) {
            case _Outcome.ok:
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: 200,
                  data: {
                    'items': [
                      {'id': 'tag1', 'name': 'alpine'},
                      {'id': 'tag2', 'name': 'alpaca'},
                    ],
                    'page': 1,
                    'perPage': 30,
                    'totalPages': 1,
                    'totalItems': 2,
                  },
                ),
              );
            case _Outcome.connectionError:
              handler.reject(
                DioException(
                  requestOptions: options,
                  type: DioExceptionType.connectionError,
                ),
              );
            case _Outcome.serverError:
              handler.reject(
                DioException(
                  requestOptions: options,
                  type: DioExceptionType.badResponse,
                  response: Response(requestOptions: options, statusCode: 500),
                ),
              );
          }
        },
      ),
    );
    return dio;
  }
}

({ProviderContainer container, _FakeApi api}) _containerWithApi(
  _Outcome outcome,
) {
  final api = _FakeApi(outcome);
  final container = ProviderContainer(
    overrides: [apiProvider.overrideWith(() => api)],
  );
  addTearDown(container.dispose);
  return (container: container, api: api);
}

ProviderContainer _container(_Outcome outcome) =>
    _containerWithApi(outcome).container;

void main() {
  // OFFUI-02. The observable contract is the same in every failure mode —
  // the autocomplete shows nothing, because there is deliberately no tag
  // cache and a hiker offline types a free-form tag instead. What differs is
  // what the provider records: a connection failure is an ordinary empty
  // result, while a real fault stays visible as an error rather than being
  // silently indistinguishable from airplane mode.
  group('TagNotifier.searchByName', () {
    test(
      'an empty query short-circuits without touching the network',
      () async {
        final container = _container(_Outcome.ok);
        final notifier = container.read(tagProvider.notifier);

        final result = await notifier.searchByName('');

        expect(result, isEmpty);
        expect(container.read(tagProvider).value, isEmpty);
      },
    );

    test('a successful search returns the parsed tags', () async {
      final container = _container(_Outcome.ok);
      final notifier = container.read(tagProvider.notifier);

      final result = await notifier.searchByName('alp');

      expect(result.map((t) => t.name), ['alpine', 'alpaca']);
      expect(container.read(tagProvider).value?.length, 2);
    });

    test(
      'offline: returns no suggestions and publishes empty DATA, not an '
      'error — being offline is an expected outcome here, not a fault',
      () async {
        final container = _container(_Outcome.connectionError);
        final notifier = container.read(tagProvider.notifier);

        final result = await notifier.searchByName('alp');

        expect(result, isEmpty);
        final state = container.read(tagProvider);
        expect(state.hasError, isFalse);
        expect(state.value, isEmpty);
      },
    );

    test('a non-connection failure also yields no suggestions, but is retained '
        'as an error so it stays distinguishable from being offline', () async {
      final container = _container(_Outcome.serverError);
      final notifier = container.read(tagProvider.notifier);

      final result = await notifier.searchByName('alp');

      // Identical from the widget's point of view...
      expect(result, isEmpty);
      // ...but not from a diagnostic one. A blanket catch made a 500 and
      // airplane mode the same silent empty list.
      expect(container.read(tagProvider).hasError, isTrue);
    });

    test(
      'a quote in the query is escaped so it cannot close the filter literal '
      'and widen the query',
      () async {
        final pumped = _containerWithApi(_Outcome.ok);
        final notifier = pumped.container.read(tagProvider.notifier);

        // Unescaped this becomes `name~'x' || id!=''` — a different, far
        // broader query than the caller wrote.
        await notifier.searchByName("x' || id!='");

        final filter = pumped.api.lastRequest!.queryParameters['filter'];
        expect(filter, r"name~'x\' || id!=\''");
      },
    );

    test(
      'the filter travels as a query parameter, not concatenated into the path',
      () async {
        final pumped = _containerWithApi(_Outcome.ok);
        final notifier = pumped.container.read(tagProvider.notifier);

        await notifier.searchByName('alp');

        // Concatenating into the path is what made escaping everyone's
        // problem and no-one's job.
        expect(pumped.api.lastRequest!.path, '/tag');
        expect(pumped.api.lastRequest!.queryParameters['filter'], "name~'alp'");
      },
    );
  });
}
