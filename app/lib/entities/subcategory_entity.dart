import 'dart:convert';

import 'package:objectbox/objectbox.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';

@Entity()
class SubcategoryEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;

  @Index()
  String category;

  String name;
  String? shortName;
  String? icon;
  String? badgeIcon;
  String? translationsJson;
  String? settingsJson;

  SubcategoryEntity({
    required this.id,
    required this.category,
    required this.name,
    this.shortName,
    this.icon,
    this.badgeIcon,
    this.translationsJson,
    this.settingsJson,
  });

  factory SubcategoryEntity.fromModel(Subcategory s) {
    return SubcategoryEntity(
      id: s.id,
      category: s.category,
      name: s.name,
      shortName: s.shortName,
      icon: s.icon,
      badgeIcon: s.badgeIcon,
      translationsJson: s.translations != null
          ? jsonEncode(s.translations!.map((k, v) => MapEntry(k, v.toJson())))
          : null,
      settingsJson: s.settings != null ? jsonEncode(s.settings) : null,
    );
  }
}

extension SubcategoryEntityMapping on SubcategoryEntity {
  Subcategory toModel() {
    Map<String, CategoryTranslation>? translations;
    if (translationsJson != null) {
      final decoded = jsonDecode(translationsJson!) as Map<String, dynamic>;
      translations = decoded.map(
        (k, v) => MapEntry(
          k,
          CategoryTranslation.fromJson(v as Map<String, dynamic>),
        ),
      );
    }

    Map<String, dynamic>? settings;
    if (settingsJson != null) {
      settings = jsonDecode(settingsJson!) as Map<String, dynamic>;
    }

    return Subcategory(
      id: id,
      category: category,
      name: name,
      shortName: shortName,
      icon: icon,
      badgeIcon: badgeIcon,
      translations: translations,
      settings: settings,
    );
  }
}
