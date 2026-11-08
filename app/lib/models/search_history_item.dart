import 'package:wanderer/models/global_search_models.dart';

/// A search result the hiker tapped, remembered as a snapshot of what the
/// result looked like at the time -- the history renders it with the same
/// tile as the live result, without refetching.
sealed class SearchHistoryItem {
  const SearchHistoryItem();

  /// Identity within one account's history: tapping the same result again
  /// moves its entry to the top instead of adding a second one.
  String get key;
}

final class TrailHistoryItem extends SearchHistoryItem {
  final TrailSearchResult trail;
  const TrailHistoryItem(this.trail);

  @override
  String get key => 'trail:${trail.id}';
}

final class ListHistoryItem extends SearchHistoryItem {
  final ListSearchResult list;
  const ListHistoryItem(this.list);

  @override
  String get key => 'list:${list.id}';
}

final class LocationHistoryItem extends SearchHistoryItem {
  final LocationSearchResult location;
  const LocationHistoryItem(this.location);

  // Geocoding results carry no id; name plus coordinates is what tells two
  // places apart.
  @override
  String get key =>
      'location:${location.name}@${location.lat.toStringAsFixed(5)},'
      '${location.lon.toStringAsFixed(5)}';
}

final class ActorHistoryItem extends SearchHistoryItem {
  final ActorSearchResult actor;
  const ActorHistoryItem(this.actor);

  @override
  String get key => 'actor:${actor.iri.isNotEmpty ? actor.iri : actor.id}';
}
