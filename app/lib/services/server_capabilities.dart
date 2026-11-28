import 'dart:async';

import 'package:dio/dio.dart';

/// Session-local discovery; inconclusive probes never select legacy uploads.
class ServerCapabilities {
  ServerCapabilities(
    this.api, {
    this.retryDelay = const Duration(seconds: 30),
    this.longBackground = const Duration(minutes: 15),
    this.legacyLifetime = const Duration(minutes: 5),
    bool initiallyActive = true,
    DateTime Function()? now,
  }) : _active = initiallyActive,
       _now = now ?? DateTime.now;

  final Dio api;
  final Duration retryDelay;
  final Duration longBackground;
  final Duration legacyLifetime;
  final DateTime Function() _now;
  void Function()? onResolved;
  String? _server;
  bool? _assetApi;
  Future<bool?>? _pending;
  Timer? _retry;
  DateTime? _backgroundAt;
  DateTime? _checkedAt;
  DateTime? _retryAt;
  int _generation = 0;
  int _attempts = 0;
  bool _active;
  bool _disposed = false;
  bool _inBackground = false;

  bool get _configured {
    final host = Uri.tryParse(api.options.baseUrl)?.host;
    return host != null && host.isNotEmpty && host != 'unknown-server.local';
  }

  bool? get assetApi => _server == api.options.baseUrl ? _assetApi : null;

  void reset() {
    _generation++;
    _server = api.options.baseUrl;
    _assetApi = null;
    _pending = null;
    _checkedAt = null;
    _retryAt = null;
    _attempts = 0;
    _retry?.cancel();
    _retry = null;
  }

  void newSession() {
    _active = true;
    reset();
    unawaited(discover());
  }

  void endSession() {
    _active = false;
    reset();
  }

  void markAssetApi() {
    reset();
    _assetApi = true;
    _checkedAt = _now();
    onResolved?.call();
  }

  Future<bool?> discover({bool notify = false}) {
    if (_disposed || !_active || !_configured) return Future.value(null);
    if (_server != api.options.baseUrl) reset();
    final checked = _checkedAt;
    if (_assetApi == false &&
        checked != null &&
        _now().difference(checked) >= legacyLifetime)
      reset();
    if (_assetApi != null) return Future.value(_assetApi);
    if (_pending != null) return _pending!;
    final retryAt = _retryAt;
    if (retryAt != null && _now().isBefore(retryAt)) return Future.value(null);
    return _pending = _probe(_generation, api.options.baseUrl, notify);
  }

  Future<bool?> _probe(int generation, String server, bool notify) async {
    bool? answer;
    try {
      final response = await api.get<dynamic>(
        '/capabilities',
        options: Options(
          followRedirects: false,
          receiveTimeout: const Duration(seconds: 8),
          sendTimeout: const Duration(seconds: 8),
          validateStatus: (status) => status != null,
        ),
      );
      if (response.statusCode == 404) {
        answer = false;
      } else if (response.statusCode == 200 && response.data is Map) {
        final body = response.data as Map;
        if (body['assetApi'] == true) {
          answer = true;
        } else if (!body.containsKey('assetApi')) {
          answer = false;
        }
      }
    } catch (_) {
      // Connectivity, authorization and proxy failures leave discovery open.
    }
    if (_disposed ||
        generation != _generation ||
        server != api.options.baseUrl) {
      return null;
    }
    _assetApi = answer;
    _pending = null;
    _checkedAt = _now();
    _retry?.cancel();
    _retry = null;
    if (answer == null) {
      final exponent = _attempts < 4 ? _attempts++ : 4;
      final delay = retryDelay * (1 << exponent);
      _retryAt = _now().add(delay);
      if (_active && !_inBackground) {
        _retry = Timer(delay, () => unawaited(discover(notify: true)));
      }
    } else {
      _retryAt = null;
      _attempts = 0;
      if (notify) onResolved?.call();
    }
    return answer;
  }

  void background() {
    _inBackground = true;
    _backgroundAt ??= _now();
    _retry?.cancel();
    _retry = null;
  }

  Future<bool?> resume() {
    _inBackground = false;
    final since = _backgroundAt;
    _backgroundAt = null;
    if (since != null && _now().difference(since) >= longBackground) reset();
    final result = discover();
    final retryAt = _retryAt;
    if (_active &&
        _configured &&
        _retry == null &&
        retryAt != null &&
        _now().isBefore(retryAt)) {
      _retry = Timer(
        retryAt.difference(_now()),
        () => unawaited(discover(notify: true)),
      );
    }
    return result;
  }

  void dispose() {
    _disposed = true;
    _generation++;
    _retry?.cancel();
    onResolved = null;
  }
}
