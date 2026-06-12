import 'package:objectbox/objectbox.dart';

@Entity()
class SettingsEntity {
  @Id()
  int obxId = 0;

  String themeMode;

  SettingsEntity({this.themeMode = 'system'});
}
