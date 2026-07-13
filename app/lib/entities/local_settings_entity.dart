import 'package:objectbox/objectbox.dart';

@Entity()
class LocalSettingsEntity {
  @Id()
  int obxId = 0;

  String themeMode;

  int consecutiveAuthValidationFailures;

  LocalSettingsEntity({
    this.themeMode = 'system',
    this.consecutiveAuthValidationFailures = 0,
  });
}
