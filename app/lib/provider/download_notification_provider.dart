import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/services/download_notification_service.dart';

final downloadNotificationServiceProvider =
    Provider<DownloadNotificationService>(
      (_) => DownloadNotificationService(),
      name: 'downloadNotificationService',
    );
