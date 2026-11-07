import 'dart:async';

import 'package:flutter/widgets.dart' show BuildContext;
import 'package:go_router/go_router.dart';

/// Whether [uri] is something the OS handed us rather than a location of this
/// app.
///
/// Every location the app navigates to is a bare path (`/trail/abc`,
/// `/settings/regions`); none carries a scheme. Anything that does is a
/// document or link another app opened us with: an Android "Open with" VIEW
/// intent (`content://media/external/downloads/126256`, `file:///...`), or on
/// iOS an "Open in" file URL or the share extension's
/// `ShareMedia-<bundle id>://` hand-off.
///
/// The app registers no http(s) app links, so no scheme-bearing URI is a route
/// to resolve. If it ever does, this predicate is the place to map them.
bool isInboundUri(Uri uri) => uri.hasScheme;

/// [GoRouter.onEnter] guard that keeps inbound URIs out of the route table.
///
/// Flutter forwards the data URI of every VIEW intent to the router as if it
/// were a deep link: as the platform's initial route on a cold start, and via
/// `pushRouteInformation` when the singleTask activity is already running.
/// The router has no route for it and would show its "Page Not Found" screen.
///
/// The file itself is not lost: `receive_sharing_intent` receives the same
/// intent through its own channel and `MainApp` runs the trail import from
/// there. So the router's only job is to stay out of the way — [Block.stop]
/// leaves the user on the current screen.
///
/// The cold-start half is `overridePlatformDefaultLocation: true` on the
/// router; this guard never sees the initial platform route because the router
/// does not read it. Both are needed.
FutureOr<OnEnterResult> ignoreInboundUris(
  BuildContext context,
  GoRouterState currentState,
  GoRouterState nextState,
  GoRouter router,
) {
  if (isInboundUri(nextState.uri)) return const Block.stop();
  return const Allow();
}
