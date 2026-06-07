/// Shared pagination constants for profile-related providers.
///
/// Using named constants instead of magic numbers makes it easy to tune
/// page sizes in one place and ensures the feed and search providers
/// remain consistent with each other.

/// Number of feed items (trails/lists) to fetch per page in
/// [ProfileFeedNotifier].
const int kProfileFeedPerPage = 10;

/// Number of Meilisearch hits to fetch per page in
/// [ProfileTrailsNotifier] and [ProfileListsNotifier].
const int kProfileSearchPerPage = 20;
