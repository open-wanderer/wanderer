import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/actions/confirm_signout.dart';
import 'package:wanderer/components/base/wanderer_photo_picker.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/routes/waypoint_create_screen.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

const actor = 'actor0000000001';
const account = 'user00000000001';
const target = PhotoTarget.waypoint('point0000000001');

// The editor test exercises journal rehydration and removal intent. Supply
// decoded pixels under the FileImage key so thumbnail IO cannot outlive a route.
class _ThumbnailCompleter extends ImageStreamCompleter {
  _ThumbnailCompleter() {
    final recorder = ui.PictureRecorder();
    ui.Canvas(recorder).drawColor(const ui.Color(0xFFFFFFFF), ui.BlendMode.src);
    final picture = recorder.endRecording();
    setImage(ImageInfo(image: picture.toImageSync(1, 1)));
    picture.dispose();
  }
}

class _Paths extends PathProviderPlatform {
  _Paths(this.path);
  final String path;
  @override
  Future<String> getApplicationDocumentsPath() async => path;
}

class _Api extends Api {
  _Api(this.api);
  final Dio api;
  @override
  Dio build() => api;
}

class _Auth extends Auth {
  _Auth(this.user);
  final UserEntity user;
  @override
  UserEntity? build() => user;
}

UserEntity _user() => UserEntity(
  id: account,
  collectionId: 'users',
  collectionName: 'users',
  actorId: actor,
  username: 'hiker',
  preferredUsername: 'hiker',
  email: 'hiker@a.example',
  iri: 'https://a.example/hiker',
  serverUrl: 'https://a.example',
  created: DateTime(2026),
  updated: DateTime(2026),
);

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform oldPaths;
  late PhotoMutationService journal;
  late Dio api;
  late File first;
  late File second;

  setUp(() async {
    directory = Directory.systemTemp.createTempSync('pending-editor-');
    oldPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(directory.path);
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    journal = PhotoMutationService(
      api: api,
      resolveMode: () async => PhotoApiMode.unknown,
      documentsDirectory: () async => directory,
    );
    final png = base64Decode(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=',
    );
    first = File('${directory.path}/first.png')..writeAsBytesSync(png);
    second = File('${directory.path}/second.png')..writeAsBytesSync(png);
  });
  tearDown(() async {
    PathProviderPlatform.instance = oldPaths;
    await directory.delete(recursive: true);
  });

  testWidgets(
    'reopened waypoint editor displays pending photos and returns an explicit removal intent',
    (tester) async {
      await tester.runAsync(
        () => journal.apply(
          target: target,
          actorId: actor,
          newPhotos: [first.path],
        ),
      );
      final pending = (await tester.runAsync(
        () => journal.pendingPaths(target, actor),
      ))!.single;
      final imageKey = FileImage(File(pending));
      final imageCache = PaintingBinding.instance.imageCache;
      imageCache.putIfAbsent(imageKey, _ThumbnailCompleter.new);
      addTearDown(() => imageCache.evict(imageKey));
      expect(imageCache.containsKey(imageKey), isTrue);
      final waypoint = Waypoint(
        id: target.id,
        lat: 46,
        lon: 7,
        created: DateTime(2026),
        updated: DateTime(2026),
      );
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (_, _) => const Scaffold(body: Text('Home')),
          ),
          GoRoute(
            path: '/edit',
            builder: (_, _) => WaypointCreateScreen(waypoint: waypoint),
          ),
        ],
      );
      addTearDown(router.dispose);
      final container = ProviderContainer(
        overrides: [
          apiProvider.overrideWith(() => _Api(api)),
          authProvider.overrideWith(() => _Auth(_user())),
          photoMutationServiceProvider.overrideWithValue(journal),
        ],
      );
      addTearDown(container.dispose);
      await container.read(authProvider.future);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: const [Locale('en')],
          ),
        ),
      );
      final result = router.push<Waypoint>('/edit');
      await tester.runAsync(() async {
        await tester.pumpAndSettle();
        await Future<void>.delayed(const Duration(milliseconds: 100));
      });
      await tester.pumpAndSettle();
      final picker = tester.widget<WandererPhotoPicker>(
        find.byType(WandererPhotoPicker),
      );
      expect(picker.initialLocalPhotos, [pending]);
      await tester.ensureVisible(find.byType(WandererPhotoPicker));
      await tester.tap(
        find.descendant(
          of: find.byType(WandererPhotoPicker),
          matching: find.byWidgetPredicate(
            (widget) =>
                widget is FaIcon && widget.icon == FontAwesomeIcons.xmark.data,
          ),
        ),
      );
      await tester.pump();
      picker.onChanged([second.path]);
      await tester.tap(
        find
            .descendant(
              of: find.byType(AppBar),
              matching: find.byType(IconButton),
            )
            .last,
      );
      await tester.pumpAndSettle();
      final edited = (await result)!;
      expect(edited.removedPendingPhotos, [pending]);
      expect(edited.localPhotos, [second.path]);
      await tester.pumpWidget(const SizedBox.shrink());
      // Returning the editor result records intent; the owning trail save applies
      // it to the journal. Cancellation itself is covered by journal lifecycle tests.
      expect(await tester.runAsync(() => journal.pendingPaths(target, actor)), [
        pending,
      ]);
      expect(File(pending).existsSync(), isTrue);
    },
  );

  testWidgets(
    'sign-out warns for a pending photo job without an unsynced trail',
    (tester) async {
      final store = (await tester.runAsync(
        () => openStore(directory: '${directory.path}/database'),
      ))!;
      addTearDown(store.close);
      store.box<UserEntity>().put(_user());
      await tester.runAsync(
        () => journal.apply(
          target: target,
          actorId: actor,
          newPhotos: [first.path],
        ),
      );
      bool? confirmed;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            objectBoxProvider.overrideWithValue(store),
            photoMutationServiceProvider.overrideWithValue(journal),
          ],
          child: MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: const [Locale('en')],
            home: Consumer(
              builder: (context, ref, _) => Scaffold(
                body: TextButton(
                  onPressed: () async {
                    confirmed = await confirmSignOutWithUnsyncedTrails(
                      context,
                      ref,
                    );
                  },
                  child: const Text('Sign out'),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.runAsync(() async {
        await tester.tap(find.text('Sign out'));
        await Future<void>.delayed(const Duration(milliseconds: 100));
      });
      await tester.pumpAndSettle();
      expect(find.textContaining('One upload is pending'), findsOneWidget);
      expect(
        find.textContaining('removes pending photo changes'),
        findsOneWidget,
      );
      await tester.runAsync(() async {
        await tester.tap(find.text('Cancel'));
        await tester.pumpAndSettle();
      });
      await tester.pumpAndSettle();
      expect(confirmed, isFalse);
      expect(await tester.runAsync(() => journal.pendingCount(actor)), 1);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
