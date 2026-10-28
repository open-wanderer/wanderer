/// URL handling for the one address the whole app hangs off: the Wanderer
/// instance a user signs in to.
library;

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
