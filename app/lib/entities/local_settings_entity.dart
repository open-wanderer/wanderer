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

  LocalSettingsEntity({
    this.themeMode = 'system',
    this.backgroundLocationAsked = false,
  });
}
