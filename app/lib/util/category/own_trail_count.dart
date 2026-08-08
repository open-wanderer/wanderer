import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';

/// Lazily counts the current user's own trails that use a given category or
/// subcategory. There is no dedicated count endpoint, so this
/// reuses the author-scoped `POST /profile/{handle}/trails` search (author is
/// enforced server-side) with a Meilisearch filter on the single
/// id, reading the total from the raw `SearchResponse`.
///
/// Fetched only on an OFF-toggle attempt; never preloaded. Returns 0
/// for an anonymous user.
Future<int> ownTrailCount(
  WidgetRef ref, {
  required bool isSubcategory,
  required String id,
}) async {
  final user = ref.read(authProvider).value;
  if (user == null) return 0;

  final handle = '@${user.preferredUsername}';
  // Same field names the vetted TrailFilter.toFilterText interpolates (A2);
  // `id` is a server-fetched 15-char PB id, never free-text.
  final field = isSubcategory ? 'subcategory_id' : 'category_id';
  final filter = "$field IN ['$id']";

  final response = await ref
      .read(apiProvider)
      .post(
        '/profile/$handle/trails',
        data: {
          'q': '',
          'options': {'filter': filter, 'hitsPerPage': 1, 'page': 1},
        },
      );

  final data = response.data;
  if (data is! Map<String, dynamic>) return 0;

  // Meilisearch returns totalHits (finite) or estimatedTotalHits depending on
  // config; prefer totalHits, then fall back to the returned hits length (A1).
  // Coerce defensively rather than blind-casting — these fields are
  // dynamic JSON values and may arrive as a double depending on the JSON
  // serializer / proxy layer between the Go backend and the client.
  final raw =
      data['totalHits'] ??
      data['estimatedTotalHits'] ??
      (data['hits'] as List?)?.length ??
      0;
  return raw is int ? raw : (raw as num).toInt();
}
