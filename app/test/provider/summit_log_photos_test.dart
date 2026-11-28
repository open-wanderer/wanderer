import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/trail/summit_log_provider.dart';

class _PhotoApi extends Api {
  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) {
          expect(request.path, '/summit-log');
          final expand = (request.queryParameters['expand'] as String).split(
            ',',
          );
          final includesAssets = expand.contains(
            'summit_log_assets_via_summit_log.asset',
          );
          handler.resolve(
            Response(
              requestOptions: request,
              statusCode: 200,
              data: {
                'page': 1,
                'perPage': 30,
                'totalPages': 1,
                'totalItems': 1,
                'items': [
                  {
                    'id': 'log',
                    'date': '2026-01-01',
                    'collectionId': 'summit_logs',
                    'collectionName': 'summit_logs',
                    'created': '2026-01-01T00:00:00Z',
                    'updated': '2026-01-01T00:00:00Z',
                    if (includesAssets)
                      'photos': ['/api/v1/files/assets/photo/summit.jpg'],
                  },
                ],
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
  test(
    'asks for summit-log asset photos used by the trail panel and cards',
    () async {
      final container = ProviderContainer(
        overrides: [apiProvider.overrideWith(_PhotoApi.new)],
      );
      addTearDown(container.dispose);
      final logs = await container.read(summitLogListProvider('trail').future);
      expect(logs.single.photos, ['/api/v1/files/assets/photo/summit.jpg']);
      expect(
        logs.single.getFileUrl('https://a.example', logs.single.photos.single),
        'https://a.example/api/v1/files/assets/photo/summit.jpg',
      );
    },
  );
}
