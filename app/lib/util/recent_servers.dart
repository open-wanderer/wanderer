/// Pure recording logic for the "Last used" instance list shown in the
/// server picker.
///
/// The list exists so a user who has SIGNED IN to a server — a custom URL or
/// a remote-list entry — can reselect it with one tap instead of retyping it
/// after logging out. Recording happens on login success only (see
/// `auth_provider.dart`'s `_recordLastUsedServer`): a server that was merely
/// typed, selected, or unreachable is not memorable, and treating any of
/// those as "last used" would fill the list with typos and dead hosts.
///
/// This library is intentionally free of Flutter and ObjectBox imports — it
/// is pure input/output over [ServerInstance] values, unit-tested in
/// isolation, and reused by both the persistence provider and its tests.
library;

import 'dart:convert';

import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/util/server_url.dart';

/// The list holds at most this many entries, most recent first.
const int kMaxRecentServers = 2;

/// Computes the dedupe key for [url], or null when it does not normalize.
///
/// Hosts are case-insensitive, so the whole normalized URL is lowercased.
/// This deliberately also lowercases the path component — a small,
/// documented over-merge (a case-sensitive subpath could in principle
/// collide with a different one) that is harmless in practice for the
/// self-hosted-instance URLs this list stores.
String? _dedupeKey(String url) {
  final normalized = normalizeServerUrl(url);
  if (normalized == null) return null;
  return normalized.toLowerCase();
}

/// Returns a new list with [server] recorded as the most recently used
/// instance, most recent first, deduplicated by normalized URL and capped at
/// [max] entries.
///
/// Returns [current] unchanged when `server.url` does not normalize (empty
/// or hostless input). [current] is never mutated.
///
/// When the incoming [server] has no name and an existing entry shares its
/// dedupe key, the existing entry's metadata (name, image, ...) is kept —
/// this is what lets a user retype a known URL without degrading it to a
/// nameless custom entry. Otherwise (a named incoming server — remote-list
/// data) the incoming metadata replaces whatever was stored.
List<ServerInstance> pushRecentServer(
  List<ServerInstance> current,
  ServerInstance server, {
  int max = kMaxRecentServers,
}) {
  final normalized = normalizeServerUrl(server.url);
  if (normalized == null) return current;

  final key = normalized.toLowerCase();
  ServerInstance? existing;
  for (final entry in current) {
    if (_dedupeKey(entry.url) == key) {
      existing = entry;
      break;
    }
  }

  final head = (server.name == null && existing != null)
      ? existing.copyWith(url: normalized)
      : server.copyWith(url: normalized);

  final rest = current.where((entry) => _dedupeKey(entry.url) != key);

  final result = [head, ...rest];
  return result.length > max ? result.sublist(0, max) : result;
}

/// Returns a new list without the entry sharing [server]'s dedupe key.
///
/// Returns [current] unchanged when `server.url` does not normalize or no
/// entry matches. [current] is never mutated.
List<ServerInstance> removeRecentServer(
  List<ServerInstance> current,
  ServerInstance server,
) {
  final key = _dedupeKey(server.url);
  if (key == null) return current;
  if (!current.any((entry) => _dedupeKey(entry.url) == key)) return current;

  return current.where((entry) => _dedupeKey(entry.url) != key).toList();
}

/// Encodes [servers] for on-device persistence.
String encodeRecentServers(List<ServerInstance> servers) {
  return jsonEncode(servers.map((s) => s.toJson()).toList());
}

/// Decodes a persisted recent-servers list, or an empty list for anything
/// malformed.
///
/// Persisted data is never trusted to be well-formed (same stance as
/// `readPersistedMapStyleSources` in `map_source_persistence.dart`): empty
/// input, a `jsonDecode` failure, or a non-list all yield `[]`. Each element
/// is parsed in its own try/catch — a non-map element, a `fromJson` failure,
/// or an unnormalizable url drops just that entry, and valid siblings
/// survive. Only the first entry per dedupe key is kept, and the result is
/// capped at [max]. This function never throws.
List<ServerInstance> decodeRecentServers(
  String raw, {
  int max = kMaxRecentServers,
}) {
  if (raw.isEmpty) return const [];

  final dynamic decoded;
  try {
    decoded = jsonDecode(raw);
  } catch (_) {
    return const [];
  }
  if (decoded is! List) return const [];

  final result = <ServerInstance>[];
  final seenKeys = <String>{};

  for (final element in decoded) {
    try {
      if (element is! Map) continue;
      final server = ServerInstance.fromJson(
        Map<String, dynamic>.from(element),
      );
      final normalized = normalizeServerUrl(server.url);
      if (normalized == null) continue;
      final key = normalized.toLowerCase();
      if (!seenKeys.add(key)) continue;
      result.add(server.copyWith(url: normalized));
      if (result.length >= max) break;
    } catch (_) {
      continue;
    }
  }

  return result;
}
