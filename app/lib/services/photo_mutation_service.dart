import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/deferred_photo_session_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/server_capabilities_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/store/app_photo_paths.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/local/id.dart';

final photoMutationServiceProvider = Provider<PhotoMutationService>((ref) {
  return PhotoMutationService(
    api: ref.watch(apiProvider),
    resolveMode: () => ref.read(photoApiModeProvider)(),
    markAssetApi: () => ref.read(serverCapabilitiesProvider).markAssetApi(),
    sessionEpoch: () =>
        ref.mounted ? ref.read(apiProvider.notifier).sessionEpoch : -1,
    currentAccountId: () =>
        ref.mounted ? currentAccountId(ref.read(objectBoxProvider)) : null,
    onSessionInvalid: (error) async {
      if (ref.mounted)
        await ref.read(authProvider.notifier).signOutIfSessionInvalid(error);
    },
    canSend: (base, actor) {
      if (!ref.mounted || ref.read(apiProvider).options.baseUrl != base)
        return false;
      if (ref.read(deferredPhotoSessionProvider).blocks(base, actor))
        return false;
      final store = ref.read(objectBoxProvider);
      final accountId = currentAccountId(store);
      if (accountId == null) return false;
      final query = store
          .box<UserEntity>()
          .query(UserEntity_.id.equals(accountId))
          .build();
      final user = query.findFirst();
      query.close();
      return user?.actorId == actor;
    },
  );
});

class PhotoMutationResult {
  final List<String> photos;
  final bool pending;
  final List<String> pendingPaths;
  final List<String> completedPaths;
  final String? coverPhoto;
  final bool madeProgress;
  final int rejectedCount;
  final bool dropped;
  final bool confirmedDeleted;
  final DateTime? nextAttemptAt;
  const PhotoMutationResult(
    this.photos, {
    required this.pending,
    this.pendingPaths = const [],
    this.completedPaths = const [],
    this.coverPhoto,
    this.madeProgress = false,
    this.rejectedCount = 0,
    this.dropped = false,
    this.confirmedDeleted = false,
    this.nextAttemptAt,
  });
}

/// Durable photo intents, with a separate lock and retry schedule per target.
class PhotoMutationService {
  final Dio api;
  final PhotoApiModeResolver resolveMode;
  final Future<Directory> Function() documentsDirectory;
  final bool Function(String baseUrl, String actorId)? canSend;
  final int Function()? sessionEpoch;
  final String? Function()? currentAccountId;
  final Future<void> Function(Object error)? onSessionInvalid;
  final void Function()? markAssetApi;
  final DateTime Function() now;
  final void Function(PhotoTarget, String, String, List<String>)?
  onCancelledSources;
  // Provider invalidation must not give another instance concurrent file access.
  static final _locks = <String, Future<void>>{};
  static final _scopeEpoch = <String, int>{};
  static final _closedScopes = <String>{};
  final _assetServers = <String>{};

  PhotoMutationService({
    required this.api,
    required this.resolveMode,
    this.documentsDirectory = getApplicationDocumentsDirectory,
    this.canSend,
    this.sessionEpoch,
    this.currentAccountId,
    this.onSessionInvalid,
    this.markAssetApi,
    this.onCancelledSources,
    this.now = DateTime.now,
  });

  Future<T> _locked<T>(
    File file,
    Future<T> Function() work, {
    bool allowClosed = false,
  }) {
    final scope = file.parent.path, epoch = _scopeEpoch[file.parent.path] ?? 0;
    final closed = _closedScopes.contains(scope);
    final result = (_locks[file.path] ?? Future.value()).then((_) {
      if (!allowClosed &&
          (closed ||
              _closedScopes.contains(scope) ||
              (_scopeEpoch[scope] ?? 0) != epoch))
        throw StateError('Photo scope removed');
      return work();
    });
    final tail = result.then<void>(
      (_) {},
      onError: (Object _, StackTrace _) {},
    );
    _locks[file.path] = tail;
    tail.then((_) {
      if (identical(_locks[file.path], tail)) _locks.remove(file.path);
    });
    return result;
  }

  String _scope(String base, String actor) =>
      base64Url.encode(utf8.encode('$base\n$actor')).replaceAll('=', '');
  String _hash(String value) {
    var hash = 2166136261, other = 0x9e3779b9;
    for (final unit in utf8.encode(value)) {
      hash = ((hash ^ unit) * 16777619) & 0xffffffff;
      other = ((other ^ unit) * 2246822519) & 0xffffffff;
    }
    return '${hash.toRadixString(16).padLeft(8, '0')}${other.toRadixString(16).padLeft(8, '0')}';
  }

  List<Map<String, dynamic>> _files(Map<String, dynamic> job) =>
      (job['files'] as List).cast<Map<String, dynamic>>();
  bool _waiting(Map<String, dynamic> entry) =>
      entry['photo'] == null &&
      entry['rejected'] != true &&
      entry['cancelled'] != true;
  bool _pending(Map<String, dynamic> job) =>
      _files(job).any(_waiting) ||
      (job['removed'] as List).isNotEmpty ||
      job['coverPending'] == true;
  String _basename(String photo) => Uri.parse(photo).pathSegments.last;
  void _cancelCover(Map<String, dynamic> job) {
    job['coverPending'] = false;
    job.remove('cover');
    job.remove('resolvedCover');
  }

  Future<File> _file(PhotoTarget target, String actor, String base) async {
    final docs = (await documentsDirectory()).path;
    return File(
      p.join(
        docs,
        'photo-mutations',
        _scope(base, actor),
        '${target.kind}-${recordIdDirSegment(target.id)}.json',
      ),
    );
  }

  Future<List<File>> _jobs(String actor, String base) async {
    final dir = (await _file(
      const PhotoTarget.trail('scope'),
      actor,
      base,
    )).parent;
    if (!await dir.exists()) return [];
    return [
      await for (final entry in dir.list())
        if (entry is File && entry.path.endsWith('.json')) entry,
    ];
  }

  Future<void> _write(File file, Map<String, dynamic> job) async {
    if (_closedScopes.contains(file.parent.path))
      throw StateError('Photo scope removed');
    await file.parent.create(recursive: true);
    final temporary = File('${file.path}.tmp');
    await temporary.writeAsString(jsonEncode(job), flush: true);
    await temporary.rename(file.path);
  }

  Future<void> _delete(String? path, String docs) async {
    if (path == null) return;
    final file = File(resolvePhotoPath(docs, path));
    try {
      if (await file.exists()) await file.delete();
    } on FileSystemException {
      // A receipt or removed job already makes this copy reclaimable.
    }
  }

  Future<Map<String, dynamic>?> _read(File file, String docs) async {
    if (_closedScopes.contains(file.parent.path))
      throw StateError('Photo scope removed');
    if (!await file.exists()) return null;
    final job = Map<String, dynamic>.from(
      jsonDecode(await file.readAsString()) as Map,
    );
    for (final entry in _files(job)) {
      final oldSource = entry['source'] as String?;
      final oldPath = entry['path'] as String?;
      entry['name'] ??= p.basename(oldSource ?? oldPath ?? 'photo');
      if (oldPath != null) entry['path'] = storePhotoPath(docs, oldPath);
      if (oldSource != null) {
        entry['sourceKey'] ??= _hash(oldSource);
        try {
          entry['source'] = storePhotoPath(docs, oldSource);
        } on ArgumentError {
          entry.remove('source');
        }
      }
      final key = entry['key'] as String;
      if (key.contains('\n')) {
        final parts = key.split('\n');
        entry['key'] = _hash(
          '${entry['source'] ?? parts.first}\n${parts.skip(1).join('\n')}',
        );
      }
      if (job['cover'] != null &&
          (job['cover'] == oldSource || job['cover'] == oldPath))
        job['cover'] = 'pending:${entry['key']}';
      if (entry['copying'] == true) {
        final copy = File(resolvePhotoPath(docs, entry['path'] as String));
        if (!await copy.exists() && entry['source'] is String) {
          final source = File(
            resolvePhotoPath(docs, entry['source'] as String),
          );
          if (await source.exists()) await _copy(source.path, copy.path);
        }
        entry['copying'] = false;
        if (!await copy.exists()) {
          entry['rejected'] = true;
          job['rejectedCount'] = (job['rejectedCount'] as int? ?? 0) + 1;
        }
      }
      if ((entry['rejected'] == true || entry['cancelled'] == true) &&
          job['cover'] == 'pending:${entry['key']}')
        _cancelCover(job);
      if (!_waiting(entry)) await _delete(entry['path'] as String?, docs);
    }
    await _write(file, job);
    return job;
  }

  Future<void> _copy(String source, String destination) async {
    final copy = File(destination);
    await _locked(File(p.join(copy.parent.path, '.copies')), () async {
      if (_closedScopes.contains(copy.parent.path))
        throw StateError('Photo scope removed');
      final temporary = File('$destination.part');
      await File(source).copy(temporary.path);
      await temporary.rename(destination);
    });
  }

  Future<PhotoMutationResult> apply({
    required PhotoTarget target,
    required String actorId,
    List<String> newPhotos = const [],
    List<String> removedPhotos = const [],
    List<String> removedPendingPaths = const [],
    List<String> existingPhotos = const [],
    String? coverPhoto,
    bool setCover = false,
    String? serverBaseUrl,
    bool replacePendingPhotos = false,
    bool existingPhotosAreAuthoritative = true,
    bool retryImmediately = true,
    PhotoApiMode? mode,
  }) async {
    final base = serverBaseUrl ?? api.options.baseUrl;
    final epoch = sessionEpoch?.call();
    final file = await _file(target, actorId, base);
    return _locked(file, () async {
      if (sessionEpoch?.call() != epoch)
        throw StateError('Photo session changed');
      final docs = (await documentsDirectory()).path;
      final job =
          await _read(file, docs) ??
          <String, dynamic>{
            'kind': target.kind,
            'id': target.id,
            'base': base,
            'actor': actorId,
            'files': <dynamic>[],
            'removed': <dynamic>[],
            'photos': <dynamic>[],
          };
      final previousRejections = job['rejectedCount'] as int? ?? 0;
      if (!_pending(job) && existingPhotosAreAuthoritative) {
        job['photos'] = List<String>.of(existingPhotos);
        job['removed'] = <String>[];
        job['deleted'] = <String>[];
        job.remove('resolvedCover');
      }
      final files = _files(job);
      final cancelled = removedPendingPaths
          .map((path) => resolvePhotoPath(docs, path))
          .toSet();
      final discarded = files
          .where(
            (entry) => [entry['path'], entry['source']].whereType<String>().any(
              (path) => cancelled.contains(resolvePhotoPath(docs, path)),
            ),
          )
          .toList();
      for (final entry in discarded) {
        final photo = entry['photo'];
        if (photo is String && !(job['removed'] as List).contains(photo))
          (job['removed'] as List).add(photo);
      }
      if (discarded.any((entry) => job['cover'] == 'pending:${entry['key']}'))
        _cancelCover(job);
      files.removeWhere(discarded.contains);
      job['files'] = files;
      await _write(file, job);
      if (discarded.isNotEmpty)
        onCancelledSources?.call(target, actorId, base, [
          for (final entry in discarded)
            if (entry['source'] is String)
              resolvePhotoPath(docs, entry['source'] as String),
        ]);
      for (final entry in discarded)
        await _delete(entry['path'] as String?, docs);
      final photos = (job['photos'] as List).cast<String>();
      final excluded = {
        ...(job['removed'] as List),
        ...(job['deleted'] as List? ?? const []),
      };
      photos.addAll(
        existingPhotos.where(
          (photo) => !photos.contains(photo) && !excluded.contains(photo),
        ),
      );
      var nextFile = job['nextFile'] as int? ?? files.length;
      for (final picked in newPhotos) {
        final source = resolvePhotoPath(docs, picked);
        if (files.any(
          (entry) =>
              entry['cancelled'] == true &&
              entry['source'] is String &&
              resolvePhotoPath(docs, entry['source'] as String) == source,
        ))
          continue;
        if (files.any(
          (entry) =>
              entry['path'] != null &&
              resolvePhotoPath(docs, entry['path'] as String) == source,
        ))
          continue;
        final stat = await File(source).stat();
        String? storedSource;
        try {
          storedSource = storePhotoPath(docs, source);
        } on ArgumentError {
          /* Picker cache. */
        }
        final key = _hash(
          '${storedSource ?? source}\n${stat.size}\n${stat.modified.microsecondsSinceEpoch}',
        );
        if (files.any((entry) => entry['key'] == key)) continue;
        final path = storePhotoPath(
          docs,
          p.join(
            file.parent.path,
            '${target.kind}-${target.id}-${nextFile++}-${p.basename(source)}',
          ),
        );
        final entry = <String, dynamic>{
          'key': key,
          'sourceKey': _hash(source),
          'path': path,
          'name': p.basename(source),
          'copying': true,
          if (storedSource != null) 'source': storedSource,
        };
        files.add(entry);
        job['files'] = files;
        job['nextFile'] = nextFile;
        if (stat.size > AssetPhotoClient.maxFileBytes) {
          entry['copying'] = false;
          entry['rejected'] = true;
          job['rejectedCount'] = (job['rejectedCount'] as int? ?? 0) + 1;
          await _write(file, job);
          continue;
        }
        await _write(
          file,
          job,
        ); // Reserve the copy before any file can be orphaned.
        await _copy(source, resolvePhotoPath(docs, path));
        entry['copying'] = false;
        await _write(file, job);
      }
      final removed = (job['removed'] as List).cast<String>();
      for (final photo in removedPhotos) {
        if (!removed.contains(photo)) removed.add(photo);
        photos.remove(photo);
      }
      if (setCover) {
        var selection = coverPhoto;
        for (final entry in files) {
          if (selection != null &&
              (entry['sourceKey'] == _hash(selection) ||
                  entry['path'] == selection ||
                  (entry['path'] is String &&
                      resolvePhotoPath(docs, entry['path'] as String) ==
                          selection))) {
            selection = 'pending:${entry['key']}';
            break;
          }
        }
        job['cover'] = selection;
        job['coverPending'] = true;
        if (files.any(
          (entry) =>
              (entry['rejected'] == true || entry['cancelled'] == true) &&
              selection == 'pending:${entry['key']}',
        ))
          _cancelCover(job);
      }
      job['photos'] = photos;
      job['removed'] = removed;
      await _write(file, job);
      if (retryImmediately) job.remove('nextAttempt');
      final result = await _flush(
        file,
        job,
        docs,
        mode: mode,
        previousRejections: previousRejections,
        initialProgress:
            discarded.isNotEmpty ||
            (job['rejectedCount'] as int? ?? 0) > previousRejections,
      );
      return result;
    });
  }

  Future<List<String>> pendingPaths(
    PhotoTarget target,
    String actorId, {
    String? serverBaseUrl,
    List<String> excludeSources = const [],
  }) async {
    final file = await _file(
      target,
      actorId,
      serverBaseUrl ?? api.options.baseUrl,
    );
    return _locked(file, () async {
      final docs = (await documentsDirectory()).path;
      final job = await _read(file, docs);
      return job == null
          ? []
          : _files(job)
                .where(_waiting)
                .where(
                  (entry) => !excludeSources.any(
                    (source) =>
                        entry['source'] is String &&
                        resolvePhotoPath(docs, source) ==
                            resolvePhotoPath(docs, entry['source'] as String),
                  ),
                )
                .map((entry) => resolvePhotoPath(docs, entry['path'] as String))
                .toList();
    });
  }

  Future<bool> hasJob(
    PhotoTarget target,
    String actorId, {
    String? serverBaseUrl,
  }) async {
    final file = await _file(
      target,
      actorId,
      serverBaseUrl ?? api.options.baseUrl,
    );
    return _locked(file, file.exists);
  }

  Future<int> pendingCount(String actorId, {String? serverBaseUrl}) async {
    var count = 0;
    for (final file in await _jobs(
      actorId,
      serverBaseUrl ?? api.options.baseUrl,
    )) {
      try {
        final job =
            jsonDecode(await file.readAsString()) as Map<String, dynamic>;
        if (_pending(job)) count++;
      } on FileSystemException {
        /* A completed job may have been pruned. */
      }
    }
    return count;
  }

  Future<void> _dropFile(
    File file,
    Map<String, dynamic>? job,
    String docs,
  ) async {
    if (await file.exists()) await file.delete();
    if (job != null)
      for (final entry in _files(job))
        await _delete(entry['path'] as String?, docs);
  }

  Future<void> drop(
    PhotoTarget target,
    String actorId, {
    String? serverBaseUrl,
  }) async {
    final file = await _file(
      target,
      actorId,
      serverBaseUrl ?? api.options.baseUrl,
    );
    await _locked(file, () async {
      final docs = (await documentsDirectory()).path;
      await _dropFile(file, await _read(file, docs), docs);
    });
  }

  Future<void> acknowledge(
    PhotoTarget target,
    String actorId, {
    String? serverBaseUrl,
  }) async {
    final file = await _file(
      target,
      actorId,
      serverBaseUrl ?? api.options.baseUrl,
    );
    await _locked(file, () async {
      final docs = (await documentsDirectory()).path;
      final job = await _read(file, docs);
      if (job != null && !_pending(job)) await _dropFile(file, job, docs);
    });
  }

  Future<void> pruneCompleted(
    PhotoTarget target,
    String actorId, {
    String? serverBaseUrl,
  }) => acknowledge(target, actorId, serverBaseUrl: serverBaseUrl);
  Future<void> dropScope(String actorId, {String? serverBaseUrl}) async {
    final base = serverBaseUrl ?? api.options.baseUrl;
    final dir = (await _file(
      const PhotoTarget.trail('scope'),
      actorId,
      base,
    )).parent;
    final purge = File(p.join(dir.path, '.scope'));
    await _locked(purge, () async {
      _closedScopes.add(dir.path);
      _scopeEpoch[dir.path] = (_scopeEpoch[dir.path] ?? 0) + 1;
      try {
        // Finish admitted work before enumerating: it may still reserve a job
        // or persist the receipt of an upload already accepted by the server.
        await Future.wait([
          for (final entry in _locks.entries)
            if (p.dirname(entry.key) == dir.path && entry.key != purge.path)
              entry.value,
        ]);
        final docs = (await documentsDirectory()).path;
        for (final file in await _jobs(actorId, base)) {
          final job = Map<String, dynamic>.from(
            jsonDecode(await file.readAsString()) as Map,
          );

          await _dropFile(file, job, docs);
        }
        if (await dir.exists()) await dir.delete(recursive: true);
      } finally {
        _closedScopes.remove(dir.path);
      }
    }, allowClosed: true);
  }

  Future<PhotoMutationResult> drain(
    String actorId, {
    PhotoApiMode? mode,
  }) async {
    final base = api.options.baseUrl;
    final jobs = await _jobs(actorId, base);
    if (jobs.isEmpty) return const PhotoMutationResult([], pending: false);
    final selected = mode ?? await resolveMode();
    var pending = false, progress = false;
    var rejected = 0;
    final completed = <String>[];
    for (final file in jobs) {
      final result = await _locked(file, () async {
        final docs = (await documentsDirectory()).path;
        final job = await _read(file, docs);
        return job == null
            ? const PhotoMutationResult([], pending: false)
            : await _flush(file, job, docs, mode: selected);
      });
      pending |= result.pending;
      progress |= result.madeProgress;
      rejected += result.rejectedCount;
      completed.addAll(result.completedPaths);
    }
    await sweepOrphanCopies(actorId, serverBaseUrl: base);
    return PhotoMutationResult(
      [],
      pending: pending,
      madeProgress: progress,
      rejectedCount: rejected,
      completedPaths: completed,
    );
  }

  /// Reservations and receipts make every copy either live or reclaimable.
  Future<void> sweepOrphanCopies(
    String actorId, {
    String? serverBaseUrl,
  }) async {
    final scope = (await _file(
      const PhotoTarget.trail('scope'),
      actorId,
      serverBaseUrl ?? api.options.baseUrl,
    )).parent;
    await _locked(File(p.join(scope.path, '.copies')), () async {
      if (_closedScopes.contains(scope.path)) return;
      final jobs = await _jobs(actorId, serverBaseUrl ?? api.options.baseUrl);
      if (!await scope.exists()) return;
      final docs = (await documentsDirectory()).path;
      final reserved = <String>{};
      for (final file in jobs) {
        Map<String, dynamic> job;
        try {
          job = jsonDecode(await file.readAsString()) as Map<String, dynamic>;
        } on FileSystemException {
          continue;
        }
        for (final entry in _files(job)) {
          if (_waiting(entry) && entry['path'] is String)
            reserved.add(resolvePhotoPath(docs, entry['path'] as String));
        }
      }
      await for (final entry in scope.list()) {
        if (entry is! File ||
            entry.path.endsWith('.json') ||
            entry.path.endsWith('.tmp'))
          continue;
        if (!reserved.contains(entry.path) &&
            !reserved.contains(entry.path.replaceFirst(RegExp(r'\.part$'), '')))
          await entry.delete();
      }
    });
  }

  Future<PhotoMutationResult> _flush(
    File file,
    Map<String, dynamic> job,
    String docs, {
    PhotoApiMode? mode,
    bool initialProgress = false,
    int? previousRejections,
  }) async {
    final rejectionBaseline =
        previousRejections ?? (job['rejectedCount'] as int? ?? 0);
    var batchLimit = _files(job).length;
    final photos = (job['photos'] as List).cast<String>();
    var progress = initialProgress, transient = false, confirmedDeleted = false;
    PhotoMutationResult result({bool dropped = false}) => PhotoMutationResult(
      photos,
      pending: !dropped && _pending(job),
      madeProgress: progress,
      dropped: dropped,
      confirmedDeleted: confirmedDeleted,
      rejectedCount:
          (job['rejectedCount'] as int? ?? 0) -
          rejectionBaseline +
          (dropped ? _files(job).where(_waiting).length : 0),
      completedPaths: _files(job)
          .where((entry) => entry['source'] is String && !_waiting(entry))
          .map((entry) => resolvePhotoPath(docs, entry['source'] as String))
          .toList(),
      coverPhoto: job['resolvedCover'] as String?,
      nextAttemptAt: job['nextAttempt'] is String
          ? DateTime.parse(job['nextAttempt'] as String)
          : null,
      pendingPaths: dropped
          ? []
          : _files(job)
                .where(_waiting)
                .map((entry) => resolvePhotoPath(docs, entry['path'] as String))
                .toList(),
    );
    if (!_pending(job)) return result();
    final due = job['nextAttempt'] as String?;
    if (due != null && now().isBefore(DateTime.parse(due))) return result();
    final base = job['base'] as String;
    final epoch = sessionEpoch?.call(),
        scopeEpoch = _scopeEpoch[file.parent.path] ?? 0;
    var authenticationLost = false;
    Object? sessionError;
    bool sessionCurrent() =>
        api.options.baseUrl == base &&
        sessionEpoch?.call() == epoch &&
        canSend?.call(base, job['actor'] as String) != false;
    bool allowed() =>
        !authenticationLost &&
        !_closedScopes.contains(file.parent.path) &&
        (_scopeEpoch[file.parent.path] ?? 0) == scopeEpoch &&
        sessionCurrent();
    if (!allowed()) return result();
    var selected = _assetServers.contains(base)
        ? PhotoApiMode.assets
        : mode ?? await resolveMode();
    if (selected == PhotoApiMode.unknown || !allowed()) return result();
    final target = PhotoTarget(job['kind'] as String, job['id'] as String);
    final client = AssetPhotoClient(api, baseUrl: base, canSend: allowed);
    Future<bool> failed(
      Object error, {
      Map<String, dynamic>? entry,
      String? removal,
      bool cover = false,
    }) async {
      if (_closedScopes.contains(file.parent.path)) {
        transient = true;
        return false;
      }
      bool apiStatus(Object error, List<int> statuses) =>
          error is DioException &&
          statuses.contains(error.response?.statusCode) &&
          error.response?.data is Map &&
          (error.response!.data as Map)['message'] is String;
      void sessionFailed(Object error) {
        authenticationLost = true;
        sessionError = error;
        transient = true;
      }

      // One API 401 fences every later request in this pass, even when a
      // subsequent private-target lookup would hide the record with a 404.
      if (apiStatus(error, [401])) sessionFailed(error);
      if (authenticationLost) return false;
      if (isApiMissingTarget(error) && allowed()) {
        final account = currentAccountId?.call();
        Object confirmation = error;
        try {
          await api.get(
            '$base/${target.kind == 'summit_log' ? 'summit-log' : target.kind}/${target.id}',
          );
          confirmation = StateError('Target still exists');
        } catch (lookupError) {
          confirmation = lookupError;
        }
        if (apiStatus(confirmation, [401])) sessionFailed(confirmation);
        if (!isApiMissingTarget(confirmation) ||
            !allowed() ||
            account == null) {
          transient = true;
          return false;
        }
        // A guest also gets a JSON 404 for an existing private target. Only
        // the same account's immediately adjacent lookup can prove deletion.
        try {
          final session = await api.get('$base/user/$account');
          if (!allowed() ||
              session.statusCode != 200 ||
              session.data is! Map ||
              (session.data as Map)['id'] != account) {
            transient = true;
            return false;
          }
        } catch (sessionFailure) {
          if (allowed() && apiStatus(sessionFailure, [401, 403, 404]))
            sessionFailed(sessionFailure);
          transient = true;
          return false;
        }
        confirmedDeleted =
            (confirmation as DioException).response?.statusCode == 404;
        progress = true;
        await _dropFile(file, job, docs);
        return true;
      }
      if (entry != null && isPhotoFileRejection(error)) {
        entry['rejected'] = true;
        if (removal != null) (job['removed'] as List).remove(removal);
        if (cover) job['coverPending'] = false;
        job['rejectedCount'] = (job['rejectedCount'] as int? ?? 0) + 1;
        progress = true;
      } else {
        transient = true;
      }
      await _write(file, job);
      if (entry != null && entry['rejected'] == true)
        await _delete(entry['path'] as String?, docs);
      return false;
    }

    bool splitBatch(Object error) {
      if (error is! DioException ||
          ![400, 413, 415].contains(error.response?.statusCode))
        return false;
      final body = error.response?.data;
      final detail = body is Map ? body['detail'] : null;
      if (detail is Map &&
          detail['completedFileCount'] is int &&
          isPhotoFileRejection(error))
        return false;
      final form = error.requestOptions.data;
      if (form is! FormData || form.files.length < 2) return false;
      batchLimit = (form.files.length + 1) ~/ 2;
      return true;
    }

    // A legacy rejection is conclusive even if the discovery answer was cached.
    while (selected == PhotoApiMode.legacy && _files(job).any(_waiting)) {
      try {
        await _legacyUploads(file, job, docs, allowed, batchLimit);
        progress = true;
      } catch (error) {
        if (isLegacyPhotoRejection(error)) {
          _assetServers.add(base);
          markAssetApi?.call();
          selected = PhotoApiMode.assets;
        } else if (splitBatch(error)) {
          continue;
        } else if (await failed(
          error,
          entry: _files(job).where(_waiting).firstOrNull,
        ))
          return result(dropped: true);
        if (transient) break;
      }
    }
    if (selected == PhotoApiMode.assets) {
      while (true) {
        final missing = _files(job).where(_waiting).take(batchLimit).toList();
        if (missing.isEmpty || transient) break;
        try {
          await client.upload(
            missing
                .map((entry) => resolvePhotoPath(docs, entry['path'] as String))
                .toList(),
            target,
            filenames: missing.map((entry) => entry['name'] as String).toList(),
            onProgress: (count, uploaded) async {
              for (var i = 0; i < count; i++) {
                missing[i]['photo'] = uploaded[i];
                if (!photos.contains(uploaded[i])) photos.add(uploaded[i]);
              }
              progress = true;
              job['photos'] = photos;
              await _write(file, job);
              for (final entry in missing.take(count))
                await _delete(entry['path'] as String?, docs);
            },
          );
        } catch (error) {
          if (splitBatch(error)) continue;
          final remaining = missing.where(_waiting).toList();
          if (await failed(error, entry: remaining.firstOrNull))
            return result(dropped: true);
          if (transient) break;
        }
      }
    }
    // Removal and cover failures do not depend on an upload having succeeded.
    for (final removal in List<String>.from(job['removed'] as List)) {
      for (var retry = 0; retry < 2; retry++) {
        try {
          String? resolved = removal;
          if (selected == PhotoApiMode.assets) {
            resolved = await _resolveLegacy(job, target, removal, allowed);
            if (resolved != null) await client.remove(resolved, target);
          } else {
            await _legacyForm(job, target, allowed, removed: removal);
          }
          (job['removed'] as List).remove(removal);
          photos.remove(removal);
          photos.remove(resolved);
          job['deleted'] = {
            ...(job['deleted'] as List? ?? const []),
            removal,
            if (resolved != null) resolved,
          }.toList();
          progress = true;
          await _write(file, job);
          break;
        } catch (error) {
          if (selected == PhotoApiMode.legacy &&
              isLegacyPhotoRejection(error)) {
            _assetServers.add(base);
            markAssetApi?.call();
            selected = PhotoApiMode.assets;
            continue;
          }
          if (await failed(error, removal: removal))
            return result(dropped: true);
          break;
        }
      }
    }
    final requestedCover = job['cover'] as String?;
    if (job['coverPending'] == true &&
        requestedCover != null &&
        requestedCover.startsWith('pending:')) {
      final selectedFile = _files(job)
          .where((entry) => 'pending:${entry['key']}' == requestedCover)
          .firstOrNull;
      if (selectedFile == null ||
          selectedFile['rejected'] == true ||
          selectedFile['cancelled'] == true) {
        _cancelCover(job);
        progress = true;
      }
    }
    if (job['coverPending'] == true) {
      try {
        var cover = job['cover'] as String?;
        if (cover != null && cover.startsWith('pending:')) {
          final entry = _files(job)
              .where((entry) => 'pending:${entry['key']}' == cover)
              .firstOrNull;
          if (entry != null && _waiting(entry))
            throw StateError('Cover is awaiting its photo');
          cover = entry?['photo'] as String?;
        }
        if (selected == PhotoApiMode.assets) {
          final resolved = cover == null
              ? null
              : await _resolveLegacy(job, target, cover, allowed);
          if (cover == null || resolved != null)
            await client.cover(target.id, resolved);
          cover = resolved;
        } else {
          final index = photos.indexOf(cover ?? '');
          if (!allowed()) throw StateError('Photo session changed');
          await api.post(
            '$base/trail/${target.id}',
            data: {'thumbnail': index < 0 ? 0 : index},
          );
        }
        job['resolvedCover'] = cover;
        job['coverPending'] = false;
        progress = true;
        await _write(file, job);
      } catch (error) {
        if (await failed(error, cover: true)) return result(dropped: true);
      }
    }
    if (transient && _pending(job)) {
      final attempt = progress ? 1 : (job['attempts'] as int? ?? 0) + 1;
      const delays = [30, 120, 600, 1800];
      job['attempts'] = attempt;
      job['nextAttempt'] = now()
          .add(
            Duration(
              seconds: delays[(attempt - 1).clamp(0, delays.length - 1)],
            ),
          )
          .toIso8601String();
    } else if (progress || !_pending(job)) {
      job['attempts'] = 0;
      job.remove('nextAttempt');
    }
    if (!_closedScopes.contains(file.parent.path)) await _write(file, job);
    // Persist the retry before ending the session and invalidating providers.
    if (sessionError != null && sessionCurrent())
      await onSessionInvalid?.call(sessionError!);
    return result();
  }

  Future<String?> _resolveLegacy(
    Map<String, dynamic> job,
    PhotoTarget target,
    String photo,
    bool Function() allowed,
  ) async {
    if (photoAssetId(photo) != null) return photo;
    if (!allowed()) throw StateError('Photo session changed');
    final relation = target.kind == 'trail'
        ? null
        : '${target.kind}_assets_via_${target.kind}.asset';
    late Response<dynamic> response;
    try {
      response = await api.get(
        '${job['base']}/${target.kind == 'summit_log' ? 'summit-log' : target.kind}/${target.id}',
        queryParameters: relation == null ? null : {'expand': relation},
      );
    } on DioException catch (error) {
      if ([401, 403, 404].contains(error.response?.statusCode)) rethrow;
      throw StateError('Photo lookup unavailable');
    }
    final current = ((response.data as Map)['photos'] as List).cast<String>();
    final match = current
        .where((url) => _basename(url) == _basename(photo))
        .firstOrNull;
    final photos = (job['photos'] as List).cast<String>()
      ..removeWhere((old) => photoAssetId(old) == null);
    final excluded = {
      ...(job['removed'] as List),
      ...(job['deleted'] as List? ?? const []),
    };
    photos.addAll(
      current.where(
        (url) =>
            !photos.contains(url) &&
            !excluded.any((old) => _basename(old as String) == _basename(url)),
      ),
    );
    job['photos'] = photos;
    if (match != null && photoAssetId(match) == null)
      throw StateError('Photo assets unavailable');
    return match;
  }

  Future<void> _legacyUploads(
    File file,
    Map<String, dynamic> job,
    String docs,
    bool Function() allowed,
    int batchLimit,
  ) async {
    final missing = _files(job).where(_waiting).take(batchLimit).toList();
    if (missing.isEmpty) return;
    final target = PhotoTarget(job['kind'] as String, job['id'] as String);
    final returned = await _legacyForm(
      job,
      target,
      allowed,
      files: missing,
      docs: docs,
    );
    for (var i = 0; i < missing.length; i++) {
      missing[i]['photo'] = returned[returned.length - missing.length + i];
    }
    final photos = (job['photos'] as List).cast<String>()
      ..clear()
      ..addAll(returned);
    job['photos'] = photos;
    await _write(file, job);
    for (final entry in missing) await _delete(entry['path'] as String?, docs);
  }

  Future<List<String>> _legacyForm(
    Map<String, dynamic> job,
    PhotoTarget target,
    bool Function() allowed, {
    List<Map<String, dynamic>> files = const [],
    String? removed,
    String? docs,
  }) async {
    final form = FormData();
    if (target.kind == 'trail') form.fields.add(MapEntry('id', target.id));
    for (final entry in files)
      form.files.add(
        MapEntry(
          target.kind == 'trail' ? 'photos+' : 'photos',
          await MultipartFile.fromFile(
            resolvePhotoPath(docs!, entry['path'] as String),
            filename: entry['name'] as String,
          ),
        ),
      );
    if (removed != null)
      form.fields.add(MapEntry('photos-', _basename(removed)));
    if (!allowed()) throw StateError('Photo session changed');
    final response = await api.post(
      target.kind == 'trail'
          ? '${job['base']}/trail/form/${target.id}'
          : '${job['base']}/waypoint/${target.id}/file',
      data: form,
    );
    return ((response.data as Map)['photos'] as List).cast<String>();
  }
}

bool isApiMissingTarget(Object error) {
  if (error is! DioException ||
      ![403, 404].contains(error.response?.statusCode))
    return false;
  final response = error.response!;
  return response.data is Map &&
      (response.data as Map)['message'] is String &&
      response.headers
              .value(Headers.contentTypeHeader)
              ?.contains('application/json') ==
          true;
}

bool isPhotoFileRejection(Object error) {
  if (error is! DioException) return false;
  final status = error.response?.statusCode;
  if (status == 413 || status == 415) return true;
  if (status != 400 || error.response?.data is! Map) return false;
  final body = error.response!.data as Map;
  final detail = body['detail'] is Map ? body['detail'] as Map : body;
  final data = detail['data'];
  return data is Map &&
      ['file', 'photos'].any((field) {
        final validation = data[field];
        return validation is Map &&
            [
              'validation_file_size_limit',
              'validation_invalid_mime_type',
            ].contains(validation['code']);
      });
}

bool isLegacyPhotoRejection(Object error) {
  if (error is! DioException || error.response?.statusCode != 400) return false;
  final body = error.response?.data;
  return body is Map &&
      body['message'] is String &&
      (body['message'] as String).endsWith('_photos_form_field_removed');
}

/// A write can succeed before its response refresh fails. The server supplies
/// the committed ID so the caller can continue without creating a second row.
Future<Map<String, dynamic>> photoMetadataWrite(
  Future<Response<dynamic>> request,
  Map<String, dynamic> fallback,
) async {
  try {
    final response = await request;
    return Map<String, dynamic>.from(response.data as Map);
  } on DioException catch (error) {
    final body = error.response?.data;
    final detail = body is Map ? body['detail'] : null;
    final saved = detail is Map ? detail['savedRecord'] : null;
    if (detail is Map &&
        detail['recordCommitted'] == true &&
        saved is Map &&
        saved['id'] is String &&
        (saved['id'] as String).isNotEmpty) {
      return {
        ...Map<String, dynamic>.from(jsonDecode(jsonEncode(fallback)) as Map),
        ...Map<String, dynamic>.from(saved),
      };
    }
    rethrow;
  }
}

/// Pins metadata and its photo intent to the server/session selected at save.
class PhotoWriteSession {
  final Dio api;
  final String baseUrl;
  final int epoch;
  final int Function() currentEpoch;

  PhotoWriteSession(this.api, {required this.currentEpoch})
    : baseUrl = api.options.baseUrl,
      epoch = currentEpoch();

  void check() {
    if (api.options.baseUrl != baseUrl || currentEpoch() != epoch) {
      throw StateError('Server session changed during save');
    }
  }

  String route(String path) {
    check();
    return '$baseUrl$path';
  }
}
