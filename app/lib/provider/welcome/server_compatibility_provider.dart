import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';
import 'package:wanderer/util/server_compatibility.dart';

part 'server_compatibility_provider.g.dart';

/// Whether the server at [serverUrl] is compatible with the app.
///
/// Keyed by URL so it runs as soon as a server is selected, before anyone
/// types credentials, and again for every server switched to. Takes the URL
/// explicitly rather than reading [apiProvider]'s base URL, so it never races
/// the base-URL update that accompanies a selection.
@riverpod
Future<ServerCompatibility> serverCompatibility(Ref ref, String serverUrl) {
  return checkServerCompatibility(ref.read(apiProvider), serverUrl);
}

/// Whether the currently selected server is known to be incompatible — what
/// the welcome, login and register screens block on.
///
/// False while the check is still running or could not tell: blocking on
/// anything less than a definite answer would lock people out of a working
/// server over a moment offline.
@riverpod
bool selectedServerIncompatible(Ref ref) {
  final url = ref.watch(serverSelectionProvider).value?.selectedServer?.url;
  if (url == null) return false;
  return ref.watch(serverCompatibilityProvider(url)).value?.isIncompatible ??
      false;
}
