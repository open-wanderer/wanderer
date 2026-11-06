import 'dart:async';
import 'dart:io' show Platform;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:tracelet/tracelet.dart' as tl;
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/local_settings_provider.dart';

/// Offers to upgrade a `whileInUse` grant to background ("Allow all the time")
/// before a tracking session starts.
///
/// Load-bearing, not a nicety: tracelet-sdk stops tracking on task removal
/// whenever ACCESS_BACKGROUND_LOCATION is missing — "ACCESS_BACKGROUND_LOCATION
/// not granted — stopping tracking on task removal" — regardless of
/// `stopOnTerminate: false`, and `AppConfig` exposes no way to opt out. Without
/// the permission, clearing the app from recents silently ends an in-progress
/// recording and takes the foreground notification with it.
///
/// On Android the permission cannot be granted from a runtime prompt at all:
/// since Android 11 the system dialog only offers "While using the app" /
/// "Only this time" / "Deny", and "Allow all the time" lives solely on the
/// app's settings page. So the disclosure Play requires doubles as the handoff
/// to settings — calling [Geolocator.requestPermission] here would show the
/// same foreground dialog again and never yield background access.
///
/// Because the answer therefore arrives out-of-band, the prompt is shown at
/// most once ([LocalSettingsEntity.backgroundLocationAsked]); re-offering it
/// every recording would nag anyone who chose not to go to settings.
///
/// iOS raises its "Change to Always Allow?" prompt directly, and the Info.plist
/// usage string serves as the disclosure. It has to go through tracelet:
/// [Geolocator.requestPermission] returns without prompting once the status is
/// anything but `notDetermined`. The system shows this prompt at most once.
///
/// Best-effort throughout. Declining leaves tracking fully working; it just
/// won't survive task removal. Returns the resulting permission so callers can
/// keep their local view in sync.
Future<LocationPermission> requestBackgroundLocation(
  BuildContext context,
  WidgetRef ref,
  LocationPermission permission,
) async {
  // `always` needs nothing; anything weaker than `whileInUse` means the
  // foreground grant itself is still missing, and that is the caller's to
  // resolve before this makes sense.
  if (permission != LocationPermission.whileInUse) return permission;

  if (Platform.isIOS) return _requestAlwaysOnIos(permission);
  if (!Platform.isAndroid) return permission;

  if (ref.read(localSettingsProvider).backgroundLocationAsked) {
    return permission;
  }

  final l10n = AppLocalizations.of(context)!;
  final accepted = await showDialog<bool>(
    context: context,
    builder: (dialogContext) => AlertDialog(
      title: Text(l10n.background_location_title),
      content: Text(l10n.background_location_body),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(dialogContext).pop(false),
          child: Text(l10n.not_now),
        ),
        FilledButton(
          onPressed: () => Navigator.of(dialogContext).pop(true),
          child: Text(l10n.background_location_confirm),
        ),
      ],
    ),
  );

  // Asked either way — a decline is an answer, and the accept path can only
  // hand off to settings, so neither outcome is observable here.
  await ref
      .read(localSettingsProvider.notifier)
      .markBackgroundLocationAsked();

  if (accepted != true) return permission;

  // Returns once the settings screen has been launched, not once the user
  // comes back, so the caller carries on and the session starts underneath.
  // Granting there applies to the next task removal.
  await Geolocator.openAppSettings();
  return permission;
}

/// Raises the iOS Always prompt and completes once it has been answered, so
/// the caller never navigates underneath the system dialog.
///
/// tracelet's own result can't signal that: it resolves after a 1.5s fallback
/// timer whenever the status hasn't changed, which is what happens while the
/// dialog is still up. The dialog makes the app inactive, so waiting for
/// `resumed` is what actually covers the user's answer.
Future<LocationPermission> _requestAlwaysOnIos(
  LocationPermission permission,
) async {
  try {
    await tl.Tracelet.requestLocationAuthorization();
  } catch (_) {
    return permission;
  }
  await _untilResumed();
  return Geolocator.checkPermission();
}

Future<void> _untilResumed() {
  if (WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed) {
    return Future.value();
  }
  final resumed = Completer<void>();
  late final AppLifecycleListener listener;
  listener = AppLifecycleListener(
    onResume: () {
      listener.dispose();
      resumed.complete();
    },
  );
  return resumed.future;
}
