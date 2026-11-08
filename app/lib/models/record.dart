abstract class IRecord {
  String get id;
  String get collectionId;
  String get collectionName;
  DateTime get created;
  DateTime get updated;
}

mixin RecordFunctions {
  String get id;
  String get collectionId;

  String? getFileUrl(String baseUrl, String? filename, {String? thumb}) {
    if (filename == null || filename.isEmpty) {
      return null;
    }

    if (Uri.tryParse(filename)?.hasAbsolutePath ?? false) {
      return filename;
    }

    final thumbParam = (thumb != null && thumb.isNotEmpty)
        ? '?thumb=$thumb'
        : '';

    return '$baseUrl/api/v1/files/$collectionId/$id/$filename$thumbParam';
  }
}
