import 'dart:convert';

import 'package:objectbox/objectbox.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/objectbox.g.dart';

@Entity()
class CategoryEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;
  String name;
  String? icon;
  String? shortName;
  String? translationsJson;
  String? settingsJson;

  CategoryEntity({
    required this.id,
    required this.name,
    this.icon,
    this.shortName,
    this.translationsJson,
    this.settingsJson,
  });

  factory CategoryEntity.fromModel(Category c) {
    return CategoryEntity(
      id: c.id,
      name: c.name,
      icon: c.icon,
      shortName: c.shortName,
      translationsJson: c.translations != null
          ? jsonEncode(c.translations!.map((k, v) => MapEntry(k, v.toJson())))
          : null,
      settingsJson: c.settings != null ? jsonEncode(c.settings) : null,
    );
  }
}

/// Builds the [CategoryEntity] for [category] carrying the ObjectBox id of the
/// row that already holds it, so a put UPDATES that row instead of replacing
/// it.
///
/// The [CategoryEntity.id] counterpart of [actorEntityForUpsert], for the same
/// reason: `id` is `@Unique(onConflict: replace)`, so putting a plain
/// `CategoryEntity.fromModel(...)` (`obxId == 0`) deletes the existing row and
/// inserts a new one under a new ObjectBox id — and every
/// `ToOne<CategoryEntity>` pointing at the old id (notably
/// [TrailEntity.category]) silently resolves to null from then on. Reusing the
/// id keeps those relations intact across every trail write.
CategoryEntity categoryEntityForUpsert(Store store, Category category) {
  final entity = CategoryEntity.fromModel(category);
  final query = store
      .box<CategoryEntity>()
      .query(CategoryEntity_.id.equals(category.id))
      .build();
  final existing = query.findFirst();
  query.close();
  if (existing != null) entity.obxId = existing.obxId;
  return entity;
}

extension CategoryEntityMapping on CategoryEntity {
  Category toModel() {
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

    return Category(
      id: id,
      name: name,
      icon: icon,
      shortName: shortName,
      translations: translations,
      settings: settings,
    );
  }
}
