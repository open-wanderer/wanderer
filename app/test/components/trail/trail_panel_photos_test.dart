import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/base/wanderer_photo.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/store/app_photo_paths.dart';
import 'package:wanderer/util/image_cache.dart';

class _Auth extends Auth {
  @override
  UserEntity? build() => UserEntity(
    id: 'account00000001',
    collectionId: 'users',
    collectionName: 'users',
    actorId: 'actor0000000001',
    username: 'hiker',
    preferredUsername: 'hiker',
    email: 'hiker@a.example',
    iri: 'https://a.example/hiker',
    serverUrl: 'https://a.example',
    created: DateTime(2026),
    updated: DateTime(2026),
  );
}

class _Online extends OnlineStatus {
  @override
  bool build() => true;
}

class _Sync extends TrailSync {
  @override
  Set<String> build() => {};
}

class _Thumbnail extends ImageStreamCompleter {
  _Thumbnail() {
    final recorder = ui.PictureRecorder();
    ui.Canvas(recorder).drawColor(const ui.Color(0xFFFFFFFF), ui.BlendMode.src);
    final picture = recorder.endRecording();
    setImage(ImageInfo(image: picture.toImageSync(1, 1)));
    picture.dispose();
  }
}

void main() {
  setUp(() => configurePhotoDocumentsDirectory('/test/Documents'));

  Future<void> render(WidgetTester tester, Trail trail) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    // Supply decoded thumbnails: rendering still uses the real PhotoCollage
    // and WandererPhoto widgets, without disk codecs or an image HTTP server.
    for (final key in <Object>[
      for (final path in trail.localPhotos) FileImage(File(path)),
      for (final photo in trail.photos)
        wandererImageProvider(
          trail.getFileUrl('https://a.example', photo, thumb: '1200x0')!,
        ),
    ]) {
      PaintingBinding.instance.imageCache.putIfAbsent(key, _Thumbnail.new);
      addTearDown(() => PaintingBinding.instance.imageCache.evict(key));
    }
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authProvider.overrideWith(_Auth.new),
          unitProvider.overrideWithValue('metric'),
          onlineStatusProvider.overrideWith(_Online.new),
          trailSyncProvider.overrideWith(_Sync.new),
        ],
        child: MaterialApp(
          localizationsDelegates: const [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('en'),
          home: Scaffold(
            body: TrailPanel(trail: trail, scrollController: controller),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  TrailEntity row() =>
      TrailEntity.fromModel(
          Trail.empty().copyWith(
            id: 'trail0000000001',
            name: 'Partly uploaded hike',
          ),
        )
        ..localId = 'local-1-1'
        ..syncState = TrailSyncState.pending
        ..photos = ['uploaded.jpg'];

  testWidgets('a partly uploaded capture renders each retained photo once', (
    tester,
  ) async {
    final entity = row()
      ..localPhotos = ['unsynced/account00000001/local-1-1/picked.jpg'];
    await render(tester, entity.toModel());
    final rendered = tester
        .widgetList<WandererPhoto>(find.byType(WandererPhoto))
        .toList();
    expect(rendered.where((photo) => photo.path != null), hasLength(1));
    expect(rendered.where((photo) => photo.url != null), hasLength(0));
  });

  testWidgets(
    'a capture with cleared receipts still renders its server photo',
    (tester) async {
      await render(tester, row().toModel());
      final rendered = tester
          .widgetList<WandererPhoto>(find.byType(WandererPhoto))
          .toList();
      expect(rendered, hasLength(1));
      expect(rendered.single.url, contains('uploaded.jpg'));
      expect(rendered.single.path, isNull);
    },
  );
}
