import 'dart:io' show Platform;

import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Upgrades a `whileInUse` grant to background ("Allow all the time") before a
/// tracking session starts.
///
/// Load-bearing, not a nicety: tracelet-sdk stops tracking on task removal
/// whenever ACCESS_BACKGROUND_LOCATION is missing — "ACCESS_BACKGROUND_LOCATION
/// not granted — stopping tracking on task removal" — regardless of
/// `stopOnTerminate: false`, and `AppConfig` exposes no way to opt out. Without
/// the permission, clearing the app from recents silently ends an in-progress
/// recording and takes the foreground notification with it.
///
/// On Android a prominent disclosure is shown first, which Play requires for
/// background location: the system prompt must never be the user's first sight
/// of the request. iOS needs none — its Info.plist usage string plays that
/// role — so it re-requests directly, raising the "Change to Always Allow?"
/// prompt.
///
/// Best-effort by design. Declining leaves tracking fully working; it just
/// won't survive task removal. Returns the resulting permission so callers can
/// keep their local view in sync.
Future<LocationPermission> requestBackgroundLocation(
  BuildContext context,
  LocationPermission permission,
) async {
  // `always` needs nothing; anything weaker than `whileInUse` means the
  // foreground grant itself is still missing, and that is the caller's to
  // resolve before this makes sense.
  if (permission != LocationPermission.whileInUse) return permission;

  if (Platform.isIOS) return Geolocator.requestPermission();
  if (!Platform.isAndroid) return permission;

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

  if (accepted != true) return permission;
  return Geolocator.requestPermission();
}
