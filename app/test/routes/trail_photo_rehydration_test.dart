// ignore_for_file: depend_on_referenced_packages
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:dio/dio.dart' hide ProgressCallback;
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';
import 'package:wanderer/components/base/wanderer_photo_picker.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/routes/trail_create_screen.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

const _actor = 'actor0000000001';
const _target = PhotoTarget.trail('trail0000000001');

class _Paths extends PathProviderPlatform {
  _Paths(this.path);
  final String path;
  @override
  Future<String> getApplicationDocumentsPath() async => path;
  @override
  Future<String> getTemporaryPath() async => path;
}

class _Api extends Api {
  _Api(this.client);
  final Dio client;
  @override
  Dio build() => client;
}

class _Auth extends Auth {
  @override
  UserEntity? build() => UserEntity(
    id: 'user00000000001',
    collectionId: 'users',
    collectionName: 'users',
    actorId: _actor,
    username: 'hiker',
    preferredUsername: 'hiker',
    email: 'hiker@a.example',
    iri: 'https://a.example/hiker',
    serverUrl: 'https://a.example',
    created: DateTime(2026),
    updated: DateTime(2026),
  );
}

// The photo flow uses the real form and router. Supply the rich text engine's
// ready/command protocol through a headless WebView, keeping its form pristine.
class _HeadlessWebView extends WebViewPlatform {
  @override
  PlatformWebViewController createPlatformWebViewController(
    PlatformWebViewControllerCreationParams params,
  ) => _Controller(params);
  @override
  PlatformNavigationDelegate createPlatformNavigationDelegate(
    PlatformNavigationDelegateCreationParams params,
  ) => _Navigation(params);
  @override
  PlatformWebViewWidget createPlatformWebViewWidget(
    PlatformWebViewWidgetCreationParams params,
  ) => _WebView(params);
}

class _Navigation extends PlatformNavigationDelegate {
  _Navigation(super.params) : super.implementation();
  @override
  Future<void> setOnNavigationRequest(
    NavigationRequestCallback callback,
  ) async {}
  @override
  Future<void> setOnPageStarted(PageEventCallback callback) async {}
  @override
  Future<void> setOnPageFinished(PageEventCallback callback) async {}
  @override
  Future<void> setOnProgress(ProgressCallback callback) async {}
  @override
  Future<void> setOnWebResourceError(WebResourceErrorCallback callback) async {}
}

class _WebView extends PlatformWebViewWidget {
  _WebView(super.params) : super.implementation();
  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}

class _Controller extends PlatformWebViewController {
  _Controller(super.params) : super.implementation();
  late JavaScriptChannelParams channel;
  void send(Map<String, Object?> message) => channel.onMessageReceived(
    JavaScriptMessage(message: jsonEncode(message)),
  );
  @override
  Future<void> setJavaScriptMode(JavaScriptMode javaScriptMode) async {}
  @override
  Future<void> addJavaScriptChannel(JavaScriptChannelParams params) async {
    channel = params;
  }

  @override
  Future<void> setPlatformNavigationDelegate(
    PlatformNavigationDelegate handler,
  ) async {}
  @override
  Future<void> loadFile(String path) async {
    send({'type': 'engineGlobalReady'});
  }

  @override
  Future<void> runJavaScript(String script) async {
    final id = RegExp(r'"id":"([^"]+)"').firstMatch(script)?.group(1);
    if (id == null) throw StateError('Unexpected rich text command: $script');
    send({
      'type': 'response',
      'id': id,
      'success': true,
      'payload': {'content': ''},
    });
    if (script.contains('"name":"init"')) {
      send({
        'type': 'event',
        'name': 'schemaReady',
        'payload': {'nodes': [], 'marks': [], 'commands': []},
      });
      send({'type': 'event', 'name': 'ready', 'payload': {}});
      send({
        'type': 'event',
        'name': 'stateChanged',
        'payload': {
          'doc': {'type': 'doc', 'pos': 0, 'end': 0, 'content': []},
        },
      });
    }
  }
}

class _DeferredJournal extends PhotoMutationService {
  _DeferredJournal(
    this.paths, {
    required super.api,
    required super.resolveMode,
    required super.documentsDirectory,
  });
  final Future<List<String>> paths;
  @override
  Future<List<String>> pendingPaths(
    PhotoTarget target,
    String actorId, {
    String? serverBaseUrl,
    List<String> excludeSources = const [],
  }) => paths;
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
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform oldPaths;
  late WebViewPlatform? oldWebView;
  late Store store;
  late Dio api;
  late PhotoMutationService journal;

  setUp(() async {
    directory = Directory.systemTemp.createTempSync('trail-editor-rehydrate-');
    oldPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(directory.path);
    oldWebView = WebViewPlatform.instance;
    WebViewPlatform.instance = _HeadlessWebView();
    store = await openStore(directory: '${directory.path}/database');
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) {
          final preferences = request.uri.path.endsWith('-preference');
          handler.resolve(
            Response(
              requestOptions: request,
              statusCode: 200,
              data: preferences
                  ? []
                  : {
                      'items': [],
                      'page': 1,
                      'perPage': 50,
                      'totalItems': 0,
                      'totalPages': 0,
                    },
            ),
          );
        },
      ),
    );
    journal = PhotoMutationService(
      api: api,
      resolveMode: () async => PhotoApiMode.unknown,
      documentsDirectory: () async => directory,
    );
  });
  tearDown(() async {
    store.close();
    PathProviderPlatform.instance = oldPaths;
    if (oldWebView != null) WebViewPlatform.instance = oldWebView!;
    await directory.delete(recursive: true);
  });

  void cacheThumbnail(String path) {
    final key = FileImage(File(path));
    PaintingBinding.instance.imageCache.putIfAbsent(key, _Thumbnail.new);
    addTearDown(() => PaintingBinding.instance.imageCache.evict(key));
  }

  Future<GoRouter> openEditor(
    WidgetTester tester,
    PhotoMutationService service,
  ) async {
    final trail = Trail.empty().copyWith(
      id: _target.id,
      name: 'Stored hike',
      author: _actor,
    );
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => const Scaffold(body: Text('Home')),
        ),
        GoRoute(
          path: '/edit',
          builder: (_, _) => TrailCreateScreen(trail: trail),
        ),
      ],
    );
    addTearDown(router.dispose);
    final container = ProviderContainer(
      overrides: [
        apiProvider.overrideWith(() => _Api(api)),
        authProvider.overrideWith(_Auth.new),
        objectBoxProvider.overrideWithValue(store),
        photoMutationServiceProvider.overrideWithValue(service),
        mapStyleJsonProvider.overrideWith((ref) => Completer<String>().future),
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
    unawaited(router.push<void>('/edit'));
    await tester.runAsync(() async {
      await tester.pump();
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pump(const Duration(milliseconds: 500));
    return router;
  }

  for (final hasPending in [false, true]) {
    testWidgets(
      'opening a server trail with ${hasPending ? 'pending photos' : 'an empty journal'} leaves no discard prompt',
      (tester) async {
        final source = File('${directory.path}/camera.jpg')
          ..writeAsBytesSync([1]);
        final pending = hasPending
            ? (await tester.runAsync(() async {
                await journal.apply(
                  target: _target,
                  actorId: _actor,
                  newPhotos: [source.path],
                );
                return journal.pendingPaths(_target, _actor);
              }))!
            : <String>[];
        pending.forEach(cacheThumbnail);
        await openEditor(tester, journal);
        final form = tester.state<FormBuilderState>(find.byType(FormBuilder));
        expect(form.fields['photos']!.value, pending);
        expect(form.fields['photos']!.isDirty, isFalse);
        expect(form.isDirty, isFalse);
        final picker = tester.widget<WandererPhotoPicker>(
          find.byType(WandererPhotoPicker),
        );
        expect(picker.initialLocalPhotos, pending);
        await tester.runAsync(() async {
          await tester.tap(find.byType(BackButtonIcon));
          await tester.pumpAndSettle();
        });
        await tester.pumpAndSettle();
        expect(find.byType(AlertDialog), findsNothing);
        expect(find.byType(TrailCreateScreen), findsNothing);
        expect(find.text('Home'), findsOneWidget);
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );
  }
  testWidgets('late pending photos preserve picks the user already changed', (
    tester,
  ) async {
    final pending = File('${directory.path}/pending.jpg')
      ..writeAsBytesSync([1]);
    final picked = File('${directory.path}/picked.jpg')..writeAsBytesSync([2]);
    cacheThumbnail(pending.path);
    cacheThumbnail(picked.path);
    final ready = Completer<List<String>>();
    final delayed = _DeferredJournal(
      ready.future,
      api: api,
      resolveMode: () async => PhotoApiMode.unknown,
      documentsDirectory: () async => directory,
    );
    await openEditor(tester, delayed);
    final form = tester.state<FormBuilderState>(find.byType(FormBuilder));
    form.fields['photos']!.didChange([picked.path]);
    ready.complete([pending.path]);
    await tester.pump();
    expect(form.fields['photos']!.value, [picked.path, pending.path]);
    expect(form.fields['photos']!.isDirty, isTrue);
    final picker = tester.widget<WandererPhotoPicker>(
      find.byType(WandererPhotoPicker),
    );
    expect(picker.initialLocalPhotos, [picked.path, pending.path]);
    await tester.tap(find.byType(BackButtonIcon));
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.byType(AlertDialog), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pump(const Duration(milliseconds: 500));
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
