import 'dart:io';

import 'package:flutter_local_notifications/flutter_local_notifications.dart';

class DownloadNotificationService {
  static const _notificationId = 42;
  static const _channelId = 'trail_download';
  static const _channelName = 'Trail Downloads';

  final _plugin = FlutterLocalNotificationsPlugin();
  bool _initialized = false;

  Future<void> _ensureInitialized() async {
    if (_initialized) return;

    const androidSettings =
        AndroidInitializationSettings('@mipmap/ic_launcher');
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
              AndroidFlutterLocalNotificationsPlugin>()
          ?.requestNotificationsPermission();
    }

    _initialized = true;
  }

  /// Indeterminate state shown while a trail's map tiles are still being
  /// generated on the server — there's no measurable percentage for this
  /// phase, so it's a distinct spinner rather than a fake progress value.
  Future<void> showGenerating(String trailName) async {
    await _ensureInitialized();

    final androidDetails = AndroidNotificationDetails(
      _channelId,
      _channelName,
      channelDescription: 'Trail tile download progress',
      importance: Importance.low,
      priority: Priority.low,
      showProgress: true,
      indeterminate: true,
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
      'Generating map tiles...',
      NotificationDetails(android: androidDetails, iOS: iosDetails),
    );
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
  /// selected region packages (D-10). Deliberately a sibling method, not a
  /// signature change to [showProgress]: the 0-region GUARD-01 path must
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
}
