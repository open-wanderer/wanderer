import 'package:collection/collection.dart';

/// The linked asset information kept alongside a record's display URLs.
class PhotoAsset {
  const PhotoAsset({
    required this.id,
    this.collectionId = 'assets',
    this.file = '',
    this.metadata = const {},
    this.externalProvider,
    this.externalId,
    this.takenAt,
  });

  final String id;
  final String collectionId;
  final String file;
  final Map<String, dynamic> metadata;
  final String? externalProvider;
  final String? externalId;
  final String? takenAt;

  factory PhotoAsset.fromJson(Map<String, dynamic> json) => PhotoAsset(
    id: json['id'] as String,
    collectionId: json['collectionId'] as String? ?? 'assets',
    file: json['file'] as String? ?? '',
    metadata: (json['metadata'] as Map?)?.cast<String, dynamic>() ?? const {},
    externalProvider: json['external_provider'] as String?,
    externalId: json['external_id'] as String?,
    takenAt: json['taken_at'] as String?,
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'collectionId': collectionId,
    'file': file,
    'metadata': metadata,
    if (externalProvider != null) 'external_provider': externalProvider,
    if (externalId != null) 'external_id': externalId,
    if (takenAt != null) 'taken_at': takenAt,
  };

  @override
  bool operator ==(Object other) =>
      other is PhotoAsset &&
      other.id == id &&
      other.collectionId == collectionId &&
      other.file == file &&
      other.externalProvider == externalProvider &&
      other.externalId == externalId &&
      other.takenAt == takenAt &&
      const DeepCollectionEquality().equals(other.metadata, metadata);

  @override
  int get hashCode => Object.hash(
    id,
    collectionId,
    file,
    externalProvider,
    externalId,
    takenAt,
    const DeepCollectionEquality().hash(metadata),
  );

  String get url => file.isNotEmpty
      ? '/api/v1/files/$collectionId/$id/$file'
      : '/api/v1/assets/$id/file';

  /// Mirrors the server's marker and the editor's reserved preview names.
  bool get isGeneratedRoutePreview {
    if (_metadataString(['generated', 'kind']) == 'route-preview') return true;
    final filename = [
      _metadataString(['remote', 'filename']),
      _metadataString(['original_filename']),
      file,
      externalId ?? '',
      id,
    ].firstWhere((value) => value.isNotEmpty).toLowerCase();
    return (externalProvider == null || externalProvider!.isEmpty) &&
        (takenAt == null || takenAt!.isEmpty) &&
        RegExp(
          r'^(?:wanderer-route-preview\.(?:webp|png)|wanderer_route_preview_[a-z0-9]{10}(?:_[a-z0-9]{10})*\.(?:webp|png))$',
        ).hasMatch(filename);
  }

  String _metadataString(List<String> path) {
    Object? value = metadata;
    for (final key in path) {
      if (value is! Map) return '';
      value = value[key];
    }
    return value is String ? value.trim() : '';
  }
}

class PhotoAssetLink {
  const PhotoAssetLink({
    required this.asset,
    this.isThumbnail = false,
    this.photoAsset,
  });

  final String asset;
  final bool isThumbnail;
  final PhotoAsset? photoAsset;

  factory PhotoAssetLink.fromJson(Map<String, dynamic> json) {
    final expanded = (json['expand'] as Map?)?['asset'];
    return PhotoAssetLink(
      asset: json['asset'] as String,
      isThumbnail: json['is_thumbnail'] as bool? ?? false,
      photoAsset: expanded is Map
          ? PhotoAsset.fromJson(expanded.cast<String, dynamic>())
          : null,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is PhotoAssetLink &&
      other.asset == asset &&
      other.isThumbnail == isThumbnail &&
      other.photoAsset == photoAsset;

  @override
  int get hashCode => Object.hash(asset, isThumbnail, photoAsset);

  Map<String, dynamic> toJson() => {
    'asset': asset,
    'is_thumbnail': isThumbnail,
    if (photoAsset != null) 'expand': {'asset': photoAsset!.toJson()},
  };
}
