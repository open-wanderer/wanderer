import 'dart:async';
import 'dart:io';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:receive_sharing_intent/receive_sharing_intent.dart';
import 'package:wanderer/components/toast_overlay.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/util/active_navigation_store.dart' as active_nav;
import 'package:wanderer/util/navigation_launch_util.dart';

import 'i18n/app_localizations.dart';
import 'objectbox.g.dart';
import 'provider/router_provider.dart';
import 'provider/local_settings_provider.dart';
import 'theme/theme.dart';
import 'util/trail_import_util.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final appDocDir = await getApplicationDocumentsDirectory();

  final dbPath = p.join(appDocDir.path, "objectbox");
  final store = await openStore(directory: dbPath);

  // store.box<TrailEntity>().removeAll();

  final cookiePath = p.join(appDocDir.path, ".cookies");
  final cookieDir = Directory(cookiePath);
  if (!await cookieDir.exists()) {
    await cookieDir.create(recursive: true);
  }

  final jar = PersistCookieJar(
    storage: FileStorage(cookiePath),
    ignoreExpires: false,
  );

  runApp(
    ProviderScope(
      overrides: [
        objectBoxProvider.overrideWithValue(store),
        cookieJarProvider.overrideWithValue(jar),
      ],
      child: MainApp(),
    ),
  );
}

class MainApp extends ConsumerStatefulWidget {
  const MainApp({super.key});

  @override
  ConsumerState<MainApp> createState() => _MainAppState();
}

class _MainAppState extends ConsumerState<MainApp> {
  bool _resumeHandled = false;
  ProviderSubscription? _authSub;

  // Files handed to the app via the OS share sheet, buffered until auth settles
  // with a signed-in user (a share can arrive on a cold, signed-out start).
  List<SharedMediaFile>? _pendingShare;
  StreamSubscription<List<SharedMediaFile>>? _shareSub;

  @override
  void initState() {
    super.initState();
    // One-shot resume check that waits for auth to settle so the GoRouter
    // redirect (which bounces unauthenticated users to /welcome) does not
    // race the resume-dialog push. Also replays a buffered share once a
    // signed-in user is available.
    _authSub = ref.listenManual<AsyncValue<UserEntity?>>(authProvider, (
      AsyncValue<UserEntity?>? prev,
      AsyncValue<UserEntity?> next,
    ) {
      if (next.isLoading) return;
      if (next.value != null) _maybeHandleShare();

      if (_resumeHandled) return;
      _resumeHandled = true;
      final user = next.value;
      if (user == null) return;
      WidgetsBinding.instance.addPostFrameCallback((_) => _maybeResume());
    }, fireImmediately: true);

    // Inbound share intents: while running (stream) and cold-start (initial).
    _shareSub = ReceiveSharingIntent.instance.getMediaStream().listen((files) {
      if (files.isEmpty) return;
      _pendingShare = files;
      _maybeHandleShare();
    }, onError: (_) {});

    ReceiveSharingIntent.instance.getInitialMedia().then((files) {
      if (files.isEmpty) return;
      _pendingShare = files;
      _maybeHandleShare();
      // Tell the plugin we consumed it so it isn't redelivered on the next
      // getInitialMedia() call.
      ReceiveSharingIntent.instance.reset();
    });
  }

  @override
  void dispose() {
    _authSub?.close();
    _shareSub?.cancel();
    super.dispose();
  }

  /// Runs the trail import for a buffered shared file once we know who's
  /// signed in and the navigator is ready. Keeps the file buffered otherwise
  /// so a share received while signed out replays on login.
  ///
  /// Routes optimistically off a cached ObjectBox session rather than
  /// waiting for authProvider to finish its network validation
  /// (auth_provider.dart's 3s-timeout-gated build()) — a share often arrives
  /// as a fresh cold start (Android reclaimed the backgrounded process; see
  /// singleTask in AndroidManifest.xml), and waiting for that gate here means
  /// a home-screen spinner flash before the import screen shows. The global
  /// gate stays intact for every other route; this is a narrow, accepted
  /// exception scoped to the share entry point. In the rare case the cached
  /// session turns out to be invalid, authProvider's own auth-error handling
  /// logs the user out and the router redirect bounces them to /welcome —
  /// same outcome as today, just after a visible import screen instead of
  /// before one.
  void _maybeHandleShare() {
    final pending = _pendingShare;
    if (pending == null || pending.isEmpty) return;

    final auth = ref.read(authProvider);
    final hasCachedUser = ref
        .read(objectBoxProvider)
        .box<UserEntity>()
        .getAll()
        .isNotEmpty;
    if (!hasCachedUser && (auth.isLoading || auth.value == null)) return;

    // Single-trail import: take the first shared file, ignore any extras.
    final file = pending.first;
    _pendingShare = null;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      final ctx = navigatorKey.currentContext;
      if (ctx == null) return;
      final l10n = AppLocalizations.of(ctx);
      if (l10n == null) return;
      importTrailFile(
        ref: ref,
        path: file.path,
        name: p.basename(file.path),
        navContext: ctx,
        l10n: l10n,
      );
    });
  }

  /// Checks for a persisted active-navigation row and, if resolvable, shows a
  /// resume dialog naming the trail. Accepting reopens navigation seeded from
  /// the row; declining (or an unresolvable/non-nav row) clears it silently.
  void _maybeResume() {
    final store = ref.read(objectBoxProvider);
    final row = active_nav.read(store);
    if (row == null) return;

    // Only nav-type rows are resumable today — this also future-proofs
    // against a later rec row landing here before that mode has its own
    // resume handling.
    if (row.sessionType != ActiveSessionType.nav || row.trailId == null) {
      active_nav.clear(store);
      return;
    }

    final response = readCachedNav(store, row.trailId!);
    if (response == null ||
        response.maneuvers.isEmpty ||
        response.shape.isEmpty) {
      // Trail not downloaded / corrupt cache — silently drop, no dialog.
      active_nav.clear(store);
      return;
    }

    final ctx = navigatorKey.currentContext;
    if (ctx == null) return;

    final query = store
        .box<TrailEntity>()
        .query(TrailEntity_.id.equals(row.trailId!))
        .build();
    final trailEntity = query.findFirst();
    query.close();
    final trailName = trailEntity?.name ?? row.trailId!;

    showDialog<bool>(
      context: ctx,
      builder: (dialogCtx) => AlertDialog(
        content: Text(
          AppLocalizations.of(dialogCtx)!.resume_navigation_prompt(trailName),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx).pop(false),
            child: Text(AppLocalizations.of(dialogCtx)!.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogCtx).pop(true),
            child: Text(AppLocalizations.of(dialogCtx)!.resume),
          ),
        ],
      ),
    ).then((accepted) {
      if (accepted == true) {
        navigatorKey.currentContext?.push(
          '/trail/${row.trailId}/navigate',
          extra: (response, row.isOffline ?? false, row),
        );
      } else {
        active_nav.clear(store);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final goRouter = ref.watch(routerProvider);

    return MaterialApp.router(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      debugShowCheckedModeBanner: false,
      theme: AppTheme.createTheme(Brightness.light),
      darkTheme: AppTheme.createTheme(Brightness.dark),
      themeMode: ref.watch(themeModeProvider),
      locale: ref.watch(localeProvider),

      routerConfig: goRouter,

      builder: (context, child) {
        return ToastOverlay(child: child!);
      },
    );
  }
}
