import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'pending_launch_location_provider.g.dart';

/// Route an app shortcut asked for before the router could show it — on a
/// cold start, or while the user is signed out.
///
/// The router's post-splash redirect takes it to land there instead of `/map`.
/// Kept alive for the same reason as `PendingTrailImport`: the target must
/// survive a detour through the auth routes.
@Riverpod(keepAlive: true)
class PendingLaunchLocation extends _$PendingLaunchLocation {
  @override
  String? build() => null;

  void set(String location) => state = location;

  /// Returns the pending location and clears it, so it is opened once.
  String? take() {
    final location = state;
    state = null;
    return location;
  }
}
