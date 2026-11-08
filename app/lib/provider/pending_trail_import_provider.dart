import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'pending_trail_import_provider.g.dart';

/// Path of a trail file handed to the app by the OS ("Open with" or the share
/// sheet), waiting for `TrailSourceSelectScreen` to import it.
///
/// The router's post-splash redirect reads this to land on `/trail/create`
/// instead of `/map`, so an inbound file goes through the same screen and
/// loading state as + → Import file. Kept alive because the file must survive
/// a detour through the auth routes when the cached session turns out to be
/// invalid.
@Riverpod(keepAlive: true)
class PendingTrailImport extends _$PendingTrailImport {
  @override
  String? build() => null;

  void set(String path) => state = path;

  /// Returns the pending path and clears it, so each file is imported once.
  String? take() {
    final path = state;
    state = null;
    return path;
  }
}
