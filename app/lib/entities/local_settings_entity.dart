import 'package:objectbox/objectbox.dart';

@Entity()
class LocalSettingsEntity {
  @Id()
  int obxId = 0;

  String themeMode;

  /// Whether the background-location disclosure has been shown once.
  ///
  /// Android 11+ cannot grant ACCESS_BACKGROUND_LOCATION from a runtime
  /// prompt — "Allow all the time" exists only in system settings — so the
  /// disclosure can never be answered in place. Without this flag it would
  /// reappear on every recording for anyone who has not gone to settings.
  bool backgroundLocationAsked;

  /// Loopback tile proxy port; `0` means not yet minted.
  ///
  /// MapLibre's ambient cache keys on the request URL, so a port that
  /// changes every launch orphans the whole tile cache on each cold start.
  /// Random per install, but persisted so it stays stable.
  int tileProxyPort;

  /// Loopback tile proxy secret path segment; empty means not yet minted.
  /// Minted once and persisted — a per-launch secret would churn the cache
  /// key exactly as an unstable port does.
  String tileProxySecret;

  /// `jsonEncode` of the last successful `/map/style-sources` response;
  /// empty means never fetched. Lets the proxy build an upstream redirect
  /// target on an offline cold start.
  String mapStyleSourcesJson;

  /// Resolved hillshade DEM XYZ template; empty means not yet resolved.
  /// Persisted for the same offline-cold-start reason as
  /// [mapStyleSourcesJson].
  String demTileTemplate;

  /// `jsonEncode` of up to `kMaxRecentServers` (see `recent_servers.dart`)
  /// instances the user has successfully signed in to, most recent first;
  /// empty means none.
  ///
  /// Lives on this device-level row on purpose: its whole job is to survive
  /// logout, and `account_data_purge.dart` never touches
  /// [LocalSettingsEntity]. Stores only public instance metadata (url, name,
  /// image), never credentials.
  String recentServersJson;

  LocalSettingsEntity({
    this.themeMode = 'system',
    this.backgroundLocationAsked = false,
    this.tileProxyPort = 0,
    this.tileProxySecret = '',
    this.mapStyleSourcesJson = '',
    this.demTileTemplate = '',
    this.recentServersJson = '',
  });
}
