import 'package:path/path.dart' as p;

const _photoRoots = {'library', 'unsynced', 'photo-mutations'};

/// Stores app-owned photos independently of the current mobile container path.
String storePhotoPath(String documents, String path) {
  var relative = p.normalize(path);
  if (p.isAbsolute(relative)) {
    if (p.isWithin(documents, relative)) {
      relative = p.relative(relative, from: documents);
    } else {
      final segments = p.split(relative);
      final marker = segments.lastIndexWhere(
        (part) => part == 'Documents' || part == 'app_flutter',
      );
      if (marker < 0) throw ArgumentError('Photo is outside app storage');
      relative = p.joinAll(segments.skip(marker + 1));
    }
  }
  final segments = p.split(relative);
  if (segments.isEmpty ||
      !_photoRoots.contains(segments.first) ||
      segments.contains('..') ||
      p.isAbsolute(relative)) {
    throw ArgumentError('Invalid stored photo path');
  }
  return relative;
}

/// Accepts earlier absolute app paths while keeping picker paths usable.
String resolvePhotoPath(String documents, String stored) {
  try {
    return p.join(documents, storePhotoPath(documents, stored));
  } on ArgumentError {
    if (p.isAbsolute(stored)) return stored;
    rethrow;
  }
}

String? _currentDocuments;

void configurePhotoDocumentsDirectory(String documents) {
  _currentDocuments = documents;
}

String resolveStoredPhotoPath(String stored) {
  if (_currentDocuments == null) return stored;
  try {
    return resolvePhotoPath(_currentDocuments!, stored);
  } on ArgumentError {
    return stored;
  }
}

String storeCurrentPhotoPath(String path) {
  if (_currentDocuments == null) return path;
  try {
    return storePhotoPath(_currentDocuments!, path);
  } on ArgumentError {
    return path;
  }
}

/// Server photo references are distinct from app-owned pending or cached files.
bool isServerPhotoReference(String photo) {
  if (photo.isEmpty) return false;
  final uri = Uri.tryParse(photo);
  final path = uri?.path ?? photo;
  return uri?.hasAuthority == true ||
      photo.startsWith('/api/') ||
      (!path.contains('/') && !path.contains('\\'));
}
