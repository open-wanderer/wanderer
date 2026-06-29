import 'dart:convert';

import 'package:objectbox/objectbox.dart';
import 'package:wanderer/models/settings.dart';

@Entity()
class SettingsEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;

  String? unit;
  String? languageCode;
  String? bio;
  String? locationJson;
  String? user;
  String? privacyJson;
  String? notificationsJson;

  SettingsEntity({
    required this.id,
    this.unit,
    this.languageCode,
    this.bio,
    this.locationJson,
    this.user,
    this.privacyJson,
    this.notificationsJson,
  });

  factory SettingsEntity.fromModel(Settings settings) {
    return SettingsEntity(
      id: settings.id ?? '',
      unit: settings.unit,
      languageCode: settings.language?.name,
      bio: settings.bio,
      locationJson: settings.location != null
          ? jsonEncode(settings.location!.toJson())
          : null,
      user: settings.user,
      privacyJson: settings.privacy != null
          ? jsonEncode(settings.privacy!.toJson())
          : null,
      notificationsJson: settings.notifications != null
          ? jsonEncode(
              settings.notifications!.map((k, v) => MapEntry(k, v.toJson())),
            )
          : null,
    );
  }
}

extension SettingsEntityMapping on SettingsEntity {
  Settings toModel() {
    Language? lang;
    if (languageCode != null) {
      lang = Language.values.firstWhere(
        (l) => l.name == languageCode,
        orElse: () => Language.en,
      );
    }

    SettingsLocation? loc;
    if (locationJson != null) {
      loc = SettingsLocation.fromJson(
        jsonDecode(locationJson!) as Map<String, dynamic>,
      );
    }

    SettingsPrivacy? priv;
    if (privacyJson != null) {
      priv = SettingsPrivacy.fromJson(
        jsonDecode(privacyJson!) as Map<String, dynamic>,
      );
    }

    Map<String, NotificationPreference>? notifs;
    if (notificationsJson != null) {
      final decoded = jsonDecode(notificationsJson!) as Map<String, dynamic>;
      notifs = decoded.map(
        (k, v) => MapEntry(
          k,
          NotificationPreference.fromJson(v as Map<String, dynamic>),
        ),
      );
    }

    return Settings(
      id: id,
      unit: unit,
      language: lang,
      bio: bio,
      location: loc,
      user: user,
      privacy: priv,
      notifications: notifs,
    );
  }
}
