import 'package:objectbox/objectbox.dart';

@Entity()
class LocalSettingsEntity {
  @Id()
  int obxId = 0;

  String themeMode;

  LocalSettingsEntity({this.themeMode = 'system'});
}
