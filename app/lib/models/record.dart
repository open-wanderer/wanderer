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

    final base = Uri.parse('${baseUrl.replaceFirst(RegExp(r'/+$'), '')}/');
    final source = Uri.parse(filename);
    final Uri resolved;
    if (source.hasScheme || source.hasAuthority || source.path.contains('/')) {
      resolved = base.resolveUri(source);
    } else {
      resolved = base
          .resolve('api/v1/files/$collectionId/$id/')
          .resolveUri(source);
    }

    if (thumb == null || thumb.isEmpty) return resolved.toString();
    return resolved
        .replace(
          queryParameters: {
            ...resolved.queryParametersAll,
            'thumb': [thumb],
          },
        )
        .toString();
  }
}
