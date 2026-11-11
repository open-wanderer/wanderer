import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/user_entity.dart';

final deferredPhotoSessionProvider = Provider((ref) => DeferredPhotoSession());

/// Keeps an expired account fenced until sign-in, including across restarts.
class DeferredPhotoSession {
  Map<String, dynamic>? _scope;
  Future<void>? _restored;
  Future<void> _tail = Future.value();
  bool _changed = false;
  String? _account;
  void selectAccount(String account) => _account = account;

  String _base(UserEntity user) =>
      '${user.serverUrl.replaceFirst(RegExp(r'/+$'), '')}/api/v1';
  bool matches(UserEntity user) => blocksAccount(_base(user), user.id);
  bool blocksBase(String base) =>
      _scope?['base'] == base && _scope?['account'] == _account;
  bool blocksAccount(String base, String account) =>
      _scope?['base'] == base && _scope?['account'] == account;
  bool blocks(String base, String actor) =>
      _scope?['base'] == base && _scope?['actor'] == actor;

  Future<void> _serial(Future<void> Function(File file) work) {
    final next = _tail.then((_) async {
      try {
        final root = await getApplicationDocumentsDirectory();
        await work(File('${root.path}/deferred-photo-session.json'));
      } on Exception catch (error) {
        debugPrint('Deferred photo session persistence failed: $error');
      }
    });
    _tail = next.then<void>((_) {}, onError: (Object _, StackTrace _) {});
    return next;
  }

  Future<void> restore() => _restored ??= _serial((file) async {
    if (!await file.exists()) return;
    try {
      final saved = jsonDecode(await file.readAsString());
      if (!_changed && saved is Map<String, dynamic>) _scope = saved;
    } on FormatException {
      // An incomplete marker cannot replace a cached account or its work.
    }
  });

  Future<void> mark(UserEntity user) {
    selectAccount(user.id);
    _changed = true;
    final saved = {
      'base': _base(user),
      'account': user.id,
      'actor': user.actorId,
    };
    _scope = saved;
    return _serial((file) async {
      await file.parent.create(recursive: true);
      final temporary = File('${file.path}.tmp');
      await temporary.writeAsString(jsonEncode(saved), flush: true);
      await temporary.rename(file.path);
    });
  }

  Future<void> clear() {
    _changed = true;
    _scope = null;
    return _serial((file) async {
      if (await file.exists()) await file.delete();
    });
  }
}
