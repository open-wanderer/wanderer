import 'dart:io';

import 'package:flutter_local_notifications/flutter_local_notifications.dart';

class DownloadNotificationService {
  static const _notificationId = 42;
  static const _channelId = 'trail_download';
  static const _channelName = 'Trail Downloads';

  /// Offline map region downloads get their OWN channel, not `trail_download`.
  /// A region archive is a long, user-initiated background transfer with a
  /// very different cadence from a trail save, and a separate channel lets the
  /// user mute one without losing the other in the Android system settings.
  static const _regionChannelId = 'region_download';
  static const _regionChannelName = 'Offline Map Downloads';
  static const _regionChannelDescription =
      'Offline map region download progress';

  final _plugin = FlutterLocalNotificationsPlugin();
  bool _initialized = false;

  Future<void> _ensureInitialized() async {
    if (_initialized) return;

    const androidSettings = AndroidInitializationSettings(
      '@mipmap/ic_launcher',
    );
    const iosSettings = DarwinInitializationSettings(
      requestAlertPermission: true,
      requestBadgePermission: false,
      requestSoundPermission: false,
    );
    await _plugin.initialize(
      const InitializationSettings(android: androidSettings, iOS: iosSettings),
    );

    if (Platform.isAndroid) {
      await _plugin
          .resolvePlatformSpecificImplementation<
            AndroidFlutterLocalNotificationsPlugin
          >()
          ?.requestNotificationsPermission();
    }

    _initialized = true;
  }

  Future<void> showProgress(String trailName, int done, int total) async {
    await _ensureInitialized();

    final androidDetails = AndroidNotificationDetails(
      _channelId,
      _channelName,
      channelDescription: 'Trail tile download progress',
      importance: Importance.low,
      priority: Priority.low,
      showProgress: true,
      maxProgress: total > 0 ? total : 1,
      progress: done,
      indeterminate: total == 0,
      ongoing: true,
      autoCancel: false,
      onlyAlertOnce: true,
    );
    const iosDetails = DarwinNotificationDetails(
      presentAlert: false,
      presentBadge: false,
      presentSound: false,
    );

    await _plugin.show(
      _notificationId,
      trailName,
      total > 0
          ? 'Downloading trail... ${((done / total) * 100).clamp(0, 100).round()}%'
          : 'Preparing download...',
      NotificationDetails(android: androidDetails, iOS: iosDetails),
    );
  }

  /// Same fixed id-42 notification as [showProgress], but with
  /// caller-supplied [title]/[body] instead of the hardcoded trail-name
  /// title and "Downloading trail... {pct}%" body — lets `DownloadingTrailIds`
  /// show one aggregate notification across the trail download plus any
  /// selected region packages. Deliberately a sibling method, not a
  /// signature change to [showProgress]: the 0-region path must
  /// keep calling the untouched [showProgress] so today's single-source
  /// copy stays byte-for-byte identical.
  // dart format off
  Future<void> showAggregateProgress(String title, String body, int done, int total) async {
    // dart format on
    await _ensureInitialized();

    final androidDetails = AndroidNotificationDetails(
      _channelId,
      _channelName,
      channelDescription: 'Trail tile download progress',
      importance: Importance.low,
      priority: Priority.low,
      showProgress: true,
      maxProgress: total > 0 ? total : 1,
      progress: done,
      indeterminate: total == 0,
      ongoing: true,
      autoCancel: false,
      onlyAlertOnce: true,
    );
    const iosDetails = DarwinNotificationDetails(
      presentAlert: false,
      presentBadge: false,
      presentSound: false,
    );

    await _plugin.show(
      _notificationId,
      title,
      body,
      NotificationDetails(android: androidDetails, iOS: iosDetails),
    );
  }

  Future<void> showSuccess(String trailName) async {
    await _ensureInitialized();
    const androidDetails = AndroidNotificationDetails(
      _channelId,
      _channelName,
      channelDescription: 'Trail tile download progress',
      importance: Importance.defaultImportance,
      priority: Priority.defaultPriority,
      ongoing: false,
      autoCancel: true,
    );
    const iosDetails = DarwinNotificationDetails(
      presentAlert: true,
      presentBadge: false,
      presentSound: false,
    );
    await _plugin.show(
      _notificationId,
      trailName,
      'Saved for offline use',
      const NotificationDetails(android: androidDetails, iOS: iosDetails),
    );
  }

  Future<void> showError(String trailName) async {
    await _ensureInitialized();
    const androidDetails = AndroidNotificationDetails(
      _channelId,
      _channelName,
      channelDescription: 'Trail tile download progress',
      importance: Importance.defaultImportance,
      priority: Priority.defaultPriority,
      ongoing: false,
      autoCancel: true,
    );
    const iosDetails = DarwinNotificationDetails(
      presentAlert: true,
      presentBadge: false,
      presentSound: false,
    );
    await _plugin.show(
      _notificationId,
      trailName,
      'Download failed',
      const NotificationDetails(android: androidDetails, iOS: iosDetails),
    );
  }

  Future<void> dismiss() async {
    if (!_initialized) return;
    await _plugin.cancel(_notificationId);
  }

  /// Ongoing progress notification for ONE region package (vector or DEM),
  /// under a caller-supplied [id] from `regionNotificationId` — never the
  /// fixed id 42 the trail methods above own. Each package download is fully
  /// independent (its own `CancelToken` and progress stream), so each gets its
  /// own notification rather than fighting over a single shared one.
  ///
  /// [percent] is pre-rounded by the caller, which also throttles: this only
  /// gets called when the whole-percent value actually changes, so a large
  /// archive's chunk-rate progress stream can't hammer the platform channel.
  Future<void> showRegionProgress(
    int id,
    String title,
    String body,
    int percent,
  ) async {
    await _ensureInitialized();

    final androidDetails = AndroidNotificationDetails(
      _regionChannelId,
      _regionChannelName,
      channelDescription: _regionChannelDescription,
      importance: Importance.low,
      priority: Priority.low,
      showProgress: true,
      maxProgress: 100,
      progress: percent.clamp(0, 100),
      ongoing: true,
      autoCancel: false,
      onlyAlertOnce: true,
    );
    const iosDetails = DarwinNotificationDetails(
      presentAlert: false,
      presentBadge: false,
      presentSound: false,
    );

    await _plugin.show(
      id,
      title,
      body,
      NotificationDetails(android: androidDetails, iOS: iosDetails),
    );
  }

  /// Replaces [showRegionProgress]'s ongoing notification with a terminal,
  /// dismissable one. [ongoing] is dropped and [autoCancel] set so the user
  /// can swipe it away — an `ongoing: true` notification is not dismissable
  /// and would strand a finished download in the shade forever.
  Future<void> showRegionResult(int id, String title, String body) async {
    await _ensureInitialized();

    const androidDetails = AndroidNotificationDetails(
      _regionChannelId,
      _regionChannelName,
      channelDescription: _regionChannelDescription,
      importance: Importance.defaultImportance,
      priority: Priority.defaultPriority,
      ongoing: false,
      autoCancel: true,
    );
    const iosDetails = DarwinNotificationDetails(
      presentAlert: true,
      presentBadge: false,
      presentSound: false,
    );

    await _plugin.show(
      id,
      title,
      body,
      const NotificationDetails(android: androidDetails, iOS: iosDetails),
    );
  }

  /// Clears one region package's notification outright — used on cancel,
  /// where there is no outcome worth reporting.
  Future<void> dismissRegion(int id) async {
    if (!_initialized) return;
    await _plugin.cancel(id);
  }
}
