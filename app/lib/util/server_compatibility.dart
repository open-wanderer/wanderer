import 'package:dio/dio.dart';

/// The oldest wanderer server version this app works with, as reported in
/// `software.version` of the server's NodeInfo 2.1 document.
///
/// NodeInfo itself is the first gate: it ships only with app-compatible
/// servers (the `-app` images, then every release from v0.22.0, which merges
/// the app branch). A server without it is too old regardless of version.
/// Raise this when the app starts depending on a newer server.
const List<int> kMinimumServerVersion = [0, 21, 0];

/// Thrown when signing in to a server whose wanderer version the app cannot
/// talk to — the backstop for a sign-in started before
/// [checkServerCompatibility] had answered for the selected server.
///
/// Without this check such a server fails somewhere inside the sign-in
/// round-trips — typically as a parse error on a response shape the app does
/// not know — which tells the user nothing about the actual cause.
class IncompatibleServerException implements Exception {
  /// The server's wanderer version, when it reports one. Servers predating
  /// NodeInfo do not.
  final String? serverVersion;

  const IncompatibleServerException([this.serverVersion]);

  @override
  String toString() =>
      'IncompatibleServerException: server version ${serverVersion ?? 'unknown'} '
      'is older than ${kMinimumServerVersion.join('.')}';
}

/// Parses the `major.minor.patch` prefix of [version], ignoring any
/// pre-release or build suffix. Returns null when there is none.
List<int>? _parseVersion(String version) {
  final match = RegExp(r'^v?(\d+)\.(\d+)\.(\d+)').firstMatch(version);
  if (match == null) return null;
  return [for (var i = 1; i <= 3; i++) int.parse(match.group(i)!)];
}

bool _isAtLeast(List<int> version, List<int> minimum) {
  for (var i = 0; i < 3; i++) {
    if (version[i] != minimum[i]) return version[i] > minimum[i];
  }
  return true;
}

/// Whether a NodeInfo 2.1 answer comes from a compatible server: `true` or
/// `false` when that can be told, `null` when it cannot.
///
/// - A wanderer document decides by its `software.version`. One that does not
///   parse (a development build) is given the benefit of the doubt: having
///   NodeInfo at all already rules out the servers this check exists for.
/// - Any other successful answer, or a 404, is incompatible: a wanderer
///   server from before NodeInfo, or not a wanderer server.
/// - Any other status is undecided: a server or proxy error says nothing
///   about the version behind it.
bool? isCompatibleNodeInfo(int? statusCode, Object? data) {
  final software = data is Map ? data['software'] : null;
  if (software is Map && software['name'] == 'wanderer') {
    final raw = software['version'];
    final version = raw is String ? _parseVersion(raw) : null;
    return version == null || _isAtLeast(version, kMinimumServerVersion);
  }
  if (statusCode == 404) return false;
  if (statusCode != null && statusCode >= 200 && statusCode < 300) {
    return false;
  }
  return null;
}

/// What the app could find out about a server's compatibility.
class ServerCompatibility {
  /// `true` or `false` when that could be told, `null` when it could not
  /// (the server was unreachable or answered with an error).
  final bool? isCompatible;

  /// The server's wanderer version, when it reports one. Servers predating
  /// NodeInfo do not.
  final String? serverVersion;

  const ServerCompatibility(this.isCompatible, [this.serverVersion]);

  bool get isIncompatible => isCompatible == false;
}

/// Checks whether the wanderer server at [serverUrl] is compatible with the
/// app, by its NodeInfo 2.1 document.
///
/// Reads the document straight from its fixed wanderer path instead of
/// following the `/.well-known/nodeinfo` discovery link: that link is built
/// from the server's `ORIGIN`, which need not be the address the app reaches
/// it under (a LAN IP, a second domain).
Future<ServerCompatibility> checkServerCompatibility(
  Dio api,
  String serverUrl,
) async {
  final nodeInfoUrl = Uri.parse(serverUrl).resolve('/.well-known/nodeinfo/2.1');

  final Response<dynamic> response;
  try {
    response = await api.getUri(
      nodeInfoUrl,
      options: Options(validateStatus: (_) => true),
    );
  } on DioException {
    return const ServerCompatibility(null);
  }

  final data = response.data;
  final software = data is Map ? data['software'] : null;
  final version = software is Map ? software['version'] : null;
  return ServerCompatibility(
    isCompatibleNodeInfo(response.statusCode, data),
    version is String ? version : null,
  );
}
