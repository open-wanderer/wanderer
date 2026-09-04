import 'dart:async';
import 'dart:io';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:receive_sharing_intent/receive_sharing_intent.dart';
import 'package:wanderer/components/toast_overlay.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/region/tile_proxy_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/services/tile_proxy_server.dart';
import 'package:wanderer/provider/account_scope_invalidation.dart';
import 'package:wanderer/store/active_navigation_store.dart' as active_nav;
import 'package:wanderer/store/local_photo_store.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/actions/launch_navigation.dart';
import 'package:wanderer/services/tracelet_position_source.dart';

import 'i18n/app_localizations.dart';
import 'objectbox.g.dart';
import 'provider/router_provider.dart';
import 'provider/local_settings_provider.dart';
import 'theme/theme.dart';
import 'actions/import_trail_file.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final appDocDir = await getApplicationDocumentsDirectory();

  final dbPath = p.join(appDocDir.path, "objectbox");
  final store = await openStore(directory: dbPath);
  final proxyServer = await TileProxyServer.start(store);

  final cookiePath = p.join(appDocDir.path, ".cookies");
  final cookieDir = Directory(cookiePath);
  if (!await cookieDir.exists()) {
    await cookieDir.create(recursive: true);
  }

  final jar = PersistCookieJar(
    storage: FileStorage(cookiePath),
    ignoreExpires: false,
  );

  // Fire-and-forget: the setting persists in MapLibre's own database, and a
  // map surfacing before it lands only misses caching for the first tiles.
  unawaited(_raiseAmbientTileCacheSize());

  runApp(
    ProviderScope(
      retry: _providerRetry,
      overrides: [
        objectBoxProvider.overrideWithValue(store),
        tileProxyBaseUrlProvider.overrideWithValue(proxyServer.baseUrl),
        cookieJarProvider.overrideWithValue(jar),
      ],
      child: MainApp(),
    ),
  );
}

/// Global provider retry policy, replacing Riverpod's default of 10 attempts
/// over ~45 s. Every keepAlive provider inherited that default, so a single
/// offline window — routine on a trail — fired a burst of up to 10 radio
/// wakeups per failing provider. At most 2 retries (400 ms, then 800 ms),
/// matching the tuning `trailFilterRetry` already established; individual
/// providers can still override via `@Riverpod(retry: ...)`.
Duration? _providerRetry(int retryCount, Object error) {
  if (retryCount >= 2) return null;
  return Duration(milliseconds: 400 * (1 << retryCount));
}

/// Raises MapLibre's ambient tile cache above its 50 MB default.
///
/// Measured panning pulls roughly 80 MB of tiles per 90 seconds, so at the
/// default the cache evicts faster than it fills and panning back over ground
/// covered a minute ago refetches all of it. Every byte the app pulls is
/// camera-driven tile traffic (an idle map measured exactly zero), so this is
/// the main lever on data use.
///
/// Nearly all of that volume is the hillshade raster-DEM source (~384 KB per
/// 512 px tile, vs tens of KB per vector tile), so the cache is sized for DEM:
/// 512 MB holds ~1300 DEM tiles — enough that browsing back over a
/// region-sized area at z8–12 serves from cache instead of refetching.
///
/// Best-effort: unsupported platforms and native failures are logged and
/// ignored, since a wrong cache size must never stop the app from starting.
Future<void> _raiseAmbientTileCacheSize() async {
  if (!ml.OfflineManager.isSupported) return;
  try {
    final manager = await ml.OfflineManager.createInstance();
    try {
      await manager.setMaximumAmbientCacheSize(bytes: 512 * 1024 * 1024);
    } finally {
      // Releases this JNI handle only — the native manager is a singleton
      // and the cache setting persists in its database.
      manager.dispose();
    }
  } catch (e) {
    debugPrint('main: ambient tile cache size not applied — $e');
  }
}

class MainApp extends ConsumerStatefulWidget {
  const MainApp({super.key});

  @override
  ConsumerState<MainApp> createState() => _MainAppState();
}

class _MainAppState extends ConsumerState<MainApp> with WidgetsBindingObserver {
  bool _resumeHandled = false;
  ProviderSubscription? _authSub;

  // Account-switch cache invalidation: tracks the last-seen auth
  // user id so a change is detected exactly once per switch, without acting
  // on the listener's first (baseline) emission.
  String? _lastAuthUserId;
  bool _authSeen = false;

  // The deferred-upload drain's three triggers. Cold start needs its
  // own one-shot kick because `AppLifecycleState.resumed` never fires on a
  // fresh launch; `_syncDrainColdStartKicked` keeps a later auth
  // re-emission from re-firing it (the drain's own re-entrancy guard would
  // make a duplicate harmless anyway, but the one-shot keeps intent legible).
  // The connectivity half lives in its own `listenManual` subscription,
  // closed alongside `_authSub` in [dispose].
  bool _syncDrainColdStartKicked = false;
  ProviderSubscription? _onlineStatusSub;

  // Files handed to the app via the OS share sheet, buffered until auth settles
  // with a signed-in user (a share can arrive on a cold, signed-out start).
  List<SharedMediaFile>? _pendingShare;
  StreamSubscription<List<SharedMediaFile>>? _shareSub;

  // Listener waiting for the router to leave the splash route before the share
  // import opens its bottom sheet. Held so dispose() can detach it.
  VoidCallback? _shareRouteWaiter;
  VoidCallback? _resumeRouteWaiter;
  GoRouter? _shareRouteWaiterRouter;
  GoRouter? _resumeRouteWaiterRouter;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);

    // Seed the app-wide online status as early as possible; unawaited since
    // startup must not block on a network probe.
    unawaited(ref.read(onlineStatusProvider.notifier).refresh());

    // The startup orphan sweep: a photo directory left behind by a crash
    // between "server accepted the create" and "local files deleted" is
    // reclaimed here. `unsyncedLocalIds` includes rows still `uploading`
    // after a crash, so a resume-in-progress upload's photos are never swept
    // out from under it. Unawaited so app start never blocks on filesystem
    // work; independent of the drain kicks below.
    final store = ref.read(objectBoxProvider);
    unawaited(
      sweepOrphanedUnsyncedPhotos(keepLocalIds: unsyncedLocalIds(store)),
    );

    // The connectivity-regained trigger: fires only on a false-to-true
    // transition. `listenManual` with no `fireImmediately` never calls this
    // for the provider's baseline value, so there is no separate guard
    // needed for that case. Closed alongside `_authSub` in [dispose].
    _onlineStatusSub = ref.listenManual<bool>(onlineStatusProvider, (
      bool? prev,
      bool next,
    ) {
      if (prev == false && next == true) {
        unawaited(ref.read(trailSyncProvider.notifier).drainIfOnline());
      }
    });

    // One-shot resume check that waits for auth to settle so the GoRouter
    // redirect (which bounces unauthenticated users to /welcome) does not
    // race the resume-dialog push. Also replays a buffered share once a
    // signed-in user is available.
    _authSub = ref.listenManual<AsyncValue<UserEntity?>>(authProvider, (
      AsyncValue<UserEntity?>? prev,
      AsyncValue<UserEntity?> next,
    ) {
      if (next.isLoading) return;

      // Drop every keepAlive cache holding account-scoped state on any auth
      // user-id change. `_authSeen` gates this: `fireImmediately:
      // true` makes the first emission a baseline, not a change, so it must
      // not trigger an invalidation.
      final userId = next.value?.id;
      if (_authSeen && userId != _lastAuthUserId) {
        invalidateAccountScopedProviders(ref);
      }
      _authSeen = true;
      _lastAuthUserId = userId;

      if (next.value != null) {
        _maybeHandleShare();

        // The cold-start trigger: `AppLifecycleState.resumed` never
        // fires on a fresh launch, so the drain needs its own kick once a
        // signed-in user has settled. One-shot — a later auth re-emission
        // (e.g. a token refresh) must not re-fire it.
        if (!_syncDrainColdStartKicked) {
          _syncDrainColdStartKicked = true;
          unawaited(ref.read(trailSyncProvider.notifier).drainIfOnline());
        }
      }

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

  // The foreground trigger.
  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      unawaited(ref.read(trailSyncProvider.notifier).drainIfOnline());
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _authSub?.close();
    _onlineStatusSub?.close();
    _shareSub?.cancel();
    _detachShareRouteWaiter();
    _detachResumeRouteWaiter();
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
  ///
  /// The actual sheet is not opened here: on a cold share start the app is
  /// still on the `/` splash route, which the router redirect is guaranteed to
  /// leave for `/map` as soon as auth settles. That redirect rebuilds the route
  /// stack and tears down any modal route on top of it — which silently
  /// cancelled the import's track-save-options sheet. See
  /// [_runImportWhenRouterSettled].
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

    // Release the splash's trail-reveal hold for the same reason this method
    // skips the auth gate: a share is usually a cold start, and making the
    // import wait out a ~700ms animation is precisely the splash flash the
    // optimistic routing above exists to avoid. The reveal is decoration on a
    // wait; here there is a real destination to get to.
    ref.read(splashRevealProvider.notifier).complete();

    _runImportWhenRouterSettled(file);
  }

  /// Runs [importTrailFile] for [file] once the router has settled on a real
  /// route, so the import's bottom sheet is never opened over the `/` splash
  /// (a route the redirect always leaves, taking the sheet with it).
  ///
  /// If the router settles on an auth route instead, the optimistic cached
  /// session turned out to be invalid — the file goes back into
  /// [_pendingShare] so the auth listener replays it after login.
  void _runImportWhenRouterSettled(SharedMediaFile file) {
    // A newer share supersedes one still waiting — single-trail import.
    _detachShareRouteWaiter();

    const authRoutes = {'/login', '/register', '/welcome', '/select-server'};
    final router = ref.read(routerProvider);

    void attempt() {
      final location = router.routerDelegate.currentConfiguration.uri.path;
      if (location == '/') return; // Still on the splash — keep waiting.

      _detachShareRouteWaiter();

      if (authRoutes.contains(location)) {
        _pendingShare = [file];
        return;
      }

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

    _shareRouteWaiter = attempt;
    _shareRouteWaiterRouter = router;
    router.routerDelegate.addListener(attempt);
    attempt();
  }

  void _detachShareRouteWaiter() {
    final waiter = _shareRouteWaiter;
    if (waiter == null) return;
    _shareRouteWaiterRouter?.routerDelegate.removeListener(waiter);
    _shareRouteWaiter = null;
    _shareRouteWaiterRouter = null;
  }

  /// Checks for a persisted active-navigation row and reopens the session it
  /// describes.
  ///
  /// A session whose native tracking is still running goes straight back to
  /// its screen: the foreground notification is up, the user can see it is
  /// live, and asking whether to resume something visibly in progress reads as
  /// a bug. Only a row whose tracking has already stopped — a stale session
  /// from an earlier launch — is worth a dialog, where resuming really is a
  /// question. Declining (or an unresolvable/non-nav row) clears it silently.
  void _maybeResume() {
    final store = ref.read(objectBoxProvider);
    final row = active_nav.read(store);
    if (row == null) {
      // No session to resume, but `stopOnTerminate: false` means a native
      // tracking service can outlive the process that started it (e.g. a
      // prior launch's declined resume) — reconcile so it never tracks
      // ownerless. No-op when nothing is running.
      unawaited(TraceletPositionSource.stopOrphanedTracking());
      return;
    }

    if (row.sessionType == ActiveSessionType.rec) {
      _resumeWhenRouterSettled(() => _maybeResumeRecording(store, row));
      return;
    }

    // Only nav-type rows are resumable below — an unresolvable/non-nav row
    // is dropped silently (and any surviving native tracking session with
    // it, see above).
    if (row.sessionType != ActiveSessionType.nav || row.trailId == null) {
      active_nav.clear(store);
      unawaited(TraceletPositionSource.stopOrphanedTracking());
      return;
    }

    final response = readCachedNav(store, row.trailId!);
    if (response == null ||
        response.maneuvers.isEmpty ||
        response.shape.isEmpty) {
      // Trail not downloaded / corrupt cache — silently drop, no dialog.
      active_nav.clear(store);
      unawaited(TraceletPositionSource.stopOrphanedTracking());
      return;
    }

    final query = store
        .box<TrailEntity>()
        .query(TrailEntity_.id.equals(row.trailId!))
        .build();
    final trailEntity = query.findFirst();
    query.close();
    final trailName = trailEntity?.name ?? row.trailId!;

    _resumeWhenRouterSettled(
      () => _maybeResumeNavigation(store, row, response, trailName),
    );
  }

  /// Reopens a `.nav` session, or asks first when its tracking has stopped.
  Future<void> _maybeResumeNavigation(
    Store store,
    ActiveNavigationEntity row,
    NavigateResponse response,
    String trailName,
  ) async {
    if (await TraceletPositionSource.isTracking()) {
      await _pushNavigationResume(row, response);
      return;
    }

    // Read after the await — the navigator context can change while the
    // tracking state is being fetched.
    final ctx = navigatorKey.currentContext;
    if (ctx == null || !ctx.mounted) return;

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
    ).then((accepted) async {
      if (accepted == true) {
        await _pushNavigationResume(row, response);
      } else {
        // Declined: drop the row AND the native tracking session that
        // survived termination for it (stopOnTerminate: false).
        active_nav.clear(store);
        unawaited(TraceletPositionSource.stopOrphanedTracking());
      }
    });
  }

  /// Runs [action] once the router has left the `/` splash for a real route.
  ///
  /// Anything pushed while the splash is still up does not survive: the
  /// redirect leaves `/` as soon as auth settles and the reveal hold releases,
  /// and that rebuild discards imperative routes — the same failure
  /// [_runImportWhenRouterSettled] exists to avoid, and it takes the resume
  /// dialog with it just as readily as the pushed screen.
  ///
  /// Offline this was reliable rather than rare: the connectivity probe in the
  /// resume path only resolves once its request has timed out, which lands the
  /// push inside the reveal's 3.5s failsafe window, so the recording screen
  /// appeared for an instant and was then replaced by `/map`.
  ///
  /// The hold is released outright — as the share-import path does, and for
  /// the same reason: the reveal is decoration on a wait, and here there is a
  /// real destination to get to. An auth route means the optimistic cached
  /// session turned out invalid, so the resume is dropped rather than opened
  /// behind a login screen; the row survives for the next launch.
  void _resumeWhenRouterSettled(Future<void> Function() action) {
    _detachResumeRouteWaiter();

    ref.read(splashRevealProvider.notifier).complete();

    const authRoutes = {'/login', '/register', '/welcome', '/select-server'};
    final router = ref.read(routerProvider);

    void attempt() {
      final location = router.routerDelegate.currentConfiguration.uri.path;
      if (location == '/') return; // Still on the splash — keep waiting.

      _detachResumeRouteWaiter();
      if (authRoutes.contains(location)) return;

      unawaited(action());
    }

    _resumeRouteWaiter = attempt;
    _resumeRouteWaiterRouter = router;
    router.routerDelegate.addListener(attempt);
    attempt();
  }

  void _detachResumeRouteWaiter() {
    final waiter = _resumeRouteWaiter;
    if (waiter == null) return;
    _resumeRouteWaiterRouter?.routerDelegate.removeListener(waiter);
    _resumeRouteWaiter = null;
    _resumeRouteWaiterRouter = null;
  }

  /// Reopens a navigation session, re-probing connectivity first.
  ///
  /// The persisted `isOffline` flag is deliberately not trusted: connectivity
  /// may have changed since the session was last saved (saved online,
  /// relaunched in airplane mode). A stale `isOffline=false` sends
  /// NavigationScreen down the online style path, whose `/map/style-sources`
  /// fetch hangs offline and freezes the map on its loading spinner. The
  /// cached response already makes navigation itself work offline; this flag
  /// only selects the map style path.
  Future<void> _pushNavigationResume(
    ActiveNavigationEntity row,
    NavigateResponse response,
  ) async {
    final isOffline = !await ref.read(onlineStatusProvider.notifier).refresh();
    navigatorKey.currentContext?.push(
      '/trail/${row.trailId}/navigate',
      // No fresh fix to seed on resume — same as a brand-new session pending
      // its first tracelet fix.
      extra: (response, isOffline, row, null),
    );
  }

  /// Reopens a recording session. Re-probes connectivity for the same reason
  /// as [_pushNavigationResume]; the router reads it back off
  /// `resume.isOffline`.
  Future<void> _pushRecordingResume(ActiveNavigationEntity row) async {
    row.isOffline = !await ref.read(onlineStatusProvider.notifier).refresh();
    navigatorKey.currentContext?.push('/record', extra: row);
  }

  /// Resumes an in-progress `ActiveSessionType.rec` row — mirrors
  /// [_maybeResume]'s `.nav` dialog structure but with no trail name (a
  /// recording session has no trail) and skips `readCachedNav` entirely
  /// (there is no trail to look up). Accepting re-pushes `/record` seeded
  /// with the row; `NavigationScreen.initState` already rehydrates
  /// breadcrumb + stats generically from `resumeSession`.
  Future<void> _maybeResumeRecording(
    Store store,
    ActiveNavigationEntity row,
  ) async {
    if (await TraceletPositionSource.isTracking()) {
      await _pushRecordingResume(row);
      return;
    }

    // Read after the await — see [_maybeResume].
    final ctx = navigatorKey.currentContext;
    if (ctx == null || !ctx.mounted) return;

    showDialog<bool>(
      context: ctx,
      builder: (dialogCtx) => AlertDialog(
        content: Text(AppLocalizations.of(dialogCtx)!.resume_recording_prompt),
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
    ).then((accepted) async {
      if (accepted == true) {
        await _pushRecordingResume(row);
      } else {
        // Declined: drop the row AND the native tracking session that
        // survived termination for it (stopOnTerminate: false).
        active_nav.clear(store);
        unawaited(TraceletPositionSource.stopOrphanedTracking());
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final goRouter = ref.watch(routerProvider);

    return MaterialApp.router(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        FormBuilderLocalizations.delegate,
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
