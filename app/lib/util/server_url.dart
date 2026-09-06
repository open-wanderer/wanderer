/// URL handling for the one address the whole app hangs off: the Wanderer
/// instance a user signs in to.
///
/// Two shapes of the same value are normalised here — what a user types into
/// the instance picker, and what a signed-in user's actor IRI implies — so both
/// arrive at `Api.updateBaseUrl` as a valid absolute origin with its port (and
/// any subpath prefix) intact.
library;

/// The path every LOCAL actor IRI carries, from the backend's own construction:
/// `fmt.Sprintf("%s/api/v1/activitypub/user/%s", origin, username)`
/// (db/util/activitypub.go). Everything before it is the operator's `ORIGIN`,
/// which is what the app needs as its API base — including a subpath prefix,
/// which a scheme+host reconstruction would silently drop.
const String _localActorPathMarker = '/api/v1/activitypub/user/';

/// Normalises a user-entered instance address into something usable as an API
/// base URL, or null when it cannot be made into one.
///
/// - A missing scheme is filled in with `https://` — the picker's own hint
///   ("e.g. wanderer.to") invites a bare host, and Dio's `baseUrl` setter
///   THROWS on a hostless value, which is what used to leave the picker sitting
///   there doing nothing instead of popping.
/// - Trailing slashes are stripped so `"$serverUrl/api/v1"` never doubles up.
/// - A port and a path prefix are preserved: both are part of the address of a
///   self-hosted instance.
///
/// Returns null for empty input and for anything that still has no host after
/// normalisation, so callers can reject it before it reaches the api client.
String? normalizeServerUrl(String raw) {
  var value = raw.trim();
  if (value.isEmpty) return null;

  if (!RegExp(r'^https?://', caseSensitive: false).hasMatch(value)) {
    value = 'https://$value';
  }

  while (value.endsWith('/')) {
    value = value.substring(0, value.length - 1);
  }

  final uri = Uri.tryParse(value);
  if (uri == null || uri.host.isEmpty) return null;
  return value;
}

/// Derives the instance's base URL from a signed-in user's actor [iri].
///
/// Keeps the PORT and any subpath prefix. Rebuilding this as
/// `Uri(scheme: ..., host: ...)` dropped both, so an instance on a non-default
/// port persisted a base URL pointing at the wrong place — after which the app
/// really was unreachable on the next cold start, and every avatar/photo URL
/// built from it 404'd.
///
/// Returns null when [iri] is not an absolute http(s) URL, leaving the caller
/// to decide what to fall back to.
String? serverUrlFromActorIri(String iri) {
  final uri = Uri.tryParse(iri.trim());
  if (uri == null || uri.host.isEmpty) return null;
  if (uri.scheme != 'http' && uri.scheme != 'https') return null;

  // Dart drops a scheme's default port during parsing, so `hasPort` is true
  // only for a port that actually needs carrying (8080, 8443, ...).
  final authority = uri.hasPort ? '${uri.host}:${uri.port}' : uri.host;

  // A remote actor's IRI (or a future route change) need not contain the
  // marker; the origin alone is still the best answer available.
  final markerIndex = uri.path.indexOf(_localActorPathMarker);
  final prefix = markerIndex > 0 ? uri.path.substring(0, markerIndex) : '';

  return '${uri.scheme}://$authority$prefix';
}
