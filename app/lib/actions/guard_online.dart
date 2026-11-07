import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

/// Shared pre-flight gate for a network-mutating action: call it first,
/// bail on `false`.
///
/// ```dart
/// if (!guardOnline(ref, l10n)) return;
/// ```
///
/// Returns `true` and does nothing else when [onlineStatusProvider] reads
/// `true`. Returns `false` when offline, after queueing exactly one
/// `ToastMessage` (type `warning`, not `error` — this is an expected,
/// recoverable state, not a failure) carrying [AppLocalizations.offline_action_unavailable].
///
/// Synchronous and `ref.read`-based — never `ref.watch`, and never a fresh
/// network probe. This is called from event handlers (button taps, submit
/// handlers), and re-probing there would reintroduce the multi-second hang
/// this whole plan exists to remove; it only reads the already-maintained
/// provider state, so it is instant and cannot itself hang while offline.
///
/// Holds no dedupe state — two calls while offline queue two distinct
/// toasts, so feedback is never silently suppressed on a second tap.
bool guardOnline(WidgetRef ref, AppLocalizations l10n) {
  if (ref.read(onlineStatusProvider)) return true;

  ref
      .read(toastProvider.notifier)
      .add(
        ToastMessage(
          type: ToastType.warning,
          icon: FontAwesomeIcons.linkSlash,
          text: l10n.offline_action_unavailable,
        ),
      );
  return false;
}
