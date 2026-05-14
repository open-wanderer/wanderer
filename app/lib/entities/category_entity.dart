import 'package:objectbox/objectbox.dart';
import 'package:wanderer/models/category.dart';

@Entity()
class CategoryEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;
  String name;

  CategoryEntity({required this.id, required this.name});

  factory CategoryEntity.fromModel(Category c) {
    return CategoryEntity(id: c.id, name: c.name);
  }
}

extension CategoryEntityMapping on CategoryEntity {
  Category toModel() {
    return Category(id: id, name: name);
  }
}
