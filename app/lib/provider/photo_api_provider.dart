import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:dio/dio.dart';
import 'package:wanderer/provider/server_capabilities_provider.dart';

enum PhotoApiMode { legacy, assets, unknown }

typedef PhotoApiModeResolver = Future<PhotoApiMode> Function();

/// Resolved when sending, so queued photos survive server upgrades.
final photoApiModeProvider = Provider<PhotoApiModeResolver>((ref) {
  final capabilities = ref.watch(serverCapabilitiesProvider);
  return () async => switch (await capabilities.discover()) {
    true => PhotoApiMode.assets,
    false => PhotoApiMode.legacy,
    null => PhotoApiMode.unknown,
  };
});

/// An explicit legacy-field rejection is conclusive evidence of asset support.
bool isLegacyPhotoFieldRemoved(Object error) {
  if (error is! DioException || error.response?.statusCode != 400) return false;
  final body = error.response?.data;
  return body is Map &&
      body['message'] is String &&
      (body['message'] as String).endsWith('_photos_form_field_removed');
}
