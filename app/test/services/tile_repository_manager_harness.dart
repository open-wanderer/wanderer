// On-device driver harness for `TileRepositoryManager` (Plan 23-06).
//
// This is NOT a `flutter test` unit test and NOT part of the production app
// -- it is a standalone Flutter entry point meant to be launched directly
// against a physical device (or emulator) with:
//
//   flutter run -t test/services/tile_repository_manager_harness.dart
//
// It exercises every public `TileRepositoryManager` method
// (`startVectorDownload`, `startDemDownload`, `pauseRegion`, `resumeRegion`,
// `deleteRegion`, `localTilePathsForBounds`) against a real region, a live
// ObjectBox `Store`, and a real network transfer -- the three things that
// cannot be exercised by the pure-seam unit tests in Plans 02-05. It is
// intentionally kept out of `router_provider.dart` / production navigation
// (its location under `app/test/` keeps it off the shipped app).
//
// See this plan's `<verify><human-check>` block for the five on-device
// behaviors (TILE-02/03/04, DEM-01/02, TILE-05) this harness is built to
// confirm.

import 'dart:io';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' show LngLatBounds;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/region/region_provider.dart';
// Drives `tileRepositoryManagerProvider` (the construction-only manager
// seam from 23-05); `TileRepositoryStatus` is the notifier Phase 24's
// Settings UI will subscribe to, referenced here only in doc comments since
// this harness talks to the manager directly to observe raw
// received/total byte counts, which `TileRepositoryStatus` collapses into a
// UI-only progress fraction.
import 'package:wanderer/provider/region/tile_repository_provider.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final appDocDir = await getApplicationDocumentsDirectory();

  final dbPath = p.join(appDocDir.path, 'objectbox');
  final store = await openStore(directory: dbPath);

  final cookiePath = p.join(appDocDir.path, '.cookies');
  final cookieDir = Directory(cookiePath);
  if (!await cookieDir.exists()) {
    await cookieDir.create(recursive: true);
  }
  final jar = PersistCookieJar(
    storage: FileStorage(cookiePath),
    ignoreExpires: false,
  );

  runApp(
    ProviderScope(
      overrides: [
        objectBoxProvider.overrideWithValue(store),
        cookieJarProvider.overrideWithValue(jar),
      ],
      child: const TileRepositoryHarnessApp(),
    ),
  );
}

class TileRepositoryHarnessApp extends StatelessWidget {
  const TileRepositoryHarnessApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'TileRepositoryManager harness',
      home: const TileRepositoryHarnessScreen(),
    );
  }
}

/// Drives every public `TileRepositoryManager` method
/// (`tileRepositoryManagerProvider`) against whatever regions the backend's
/// catalog returns, and lets a human tester exercise resume/pause/disk
/// refusal/DEM independence/bbox query manually.
class TileRepositoryHarnessScreen extends ConsumerStatefulWidget {
  const TileRepositoryHarnessScreen({super.key});

  @override
  ConsumerState<TileRepositoryHarnessScreen> createState() =>
      _TileRepositoryHarnessScreenState();
}

class _TileRepositoryHarnessScreenState
    extends ConsumerState<TileRepositoryHarnessScreen> {
  final _serverUrlController = TextEditingController(
    text: 'http://10.0.2.2:8090',
  );

  List<RegionEntity> _regions = [];
  String? _lastQueryResult;
  String? _statusLine;

  @override
  void dispose() {
    _serverUrlController.dispose();
    super.dispose();
  }

  void _connect() {
    ref
        .read(apiProvider.notifier)
        .updateBaseUrl(_serverUrlController.text.trim());
    debugPrint('[harness] api baseUrl -> ${_serverUrlController.text.trim()}');
  }

  Future<void> _refreshCatalog() async {
    try {
      await ref.read(regionRepositoryProvider).refreshCatalog();
      debugPrint('[harness] refreshCatalog succeeded');
    } catch (e) {
      debugPrint('[harness] refreshCatalog failed: $e');
    }
    _reloadRegions();
  }

  void _reloadRegions() {
    final store = ref.read(objectBoxProvider);
    setState(() {
      _regions = store.box<RegionEntity>().getAll();
    });
  }

  void _onProgress(String regionId, String kind, int received, int total) {
    final pct = total > 0
        ? (received / total * 100).toStringAsFixed(1)
        : '?';
    debugPrint(
      '[harness] $regionId:$kind received=$received total=$total ($pct%)',
    );
    setState(() {
      _statusLine = '$regionId:$kind received=$received total=$total';
    });
  }

  Future<void> _downloadVector(RegionEntity region) async {
    final manager = ref.read(tileRepositoryManagerProvider);
    await manager.startVectorDownload(
      region.id,
      onProgress: (received, total) =>
          _onProgress(region.id, 'vector', received, total),
    );
    _reloadRegions();
    debugPrint('[harness] ${region.id} vector RegionStatus=${region.status}');
  }

  Future<void> _downloadDem(RegionEntity region) async {
    final manager = ref.read(tileRepositoryManagerProvider);
    await manager.startDemDownload(
      region.id,
      onProgress: (received, total) =>
          _onProgress(region.id, 'dem', received, total),
    );
    _reloadRegions();
    debugPrint(
      '[harness] ${region.id} dem PackageStatus='
      '${region.demPackage.target?.status ?? PackageStatus.notDownloaded}',
    );
  }

  Future<void> _pause(RegionEntity region) async {
    await ref.read(tileRepositoryManagerProvider).pauseRegion(region.id);
    _reloadRegions();
    debugPrint('[harness] ${region.id} paused, RegionStatus=${region.status}');
  }

  Future<void> _resume(RegionEntity region) async {
    final manager = ref.read(tileRepositoryManagerProvider);
    await manager.resumeRegion(
      region.id,
      onProgress: (received, total) =>
          _onProgress(region.id, 'resume', received, total),
    );
    _reloadRegions();
    debugPrint('[harness] ${region.id} resumed, RegionStatus=${region.status}');
  }

  Future<void> _delete(RegionEntity region) async {
    await ref.read(tileRepositoryManagerProvider).deleteRegion(region.id);
    _reloadRegions();
    debugPrint('[harness] ${region.id} deleted, RegionStatus=${region.status}');
  }

  /// Queries `localTilePathsForBounds` for either a bbox inside [region]
  /// (its center quarter) or well outside every region's bbox.
  void _queryBounds(RegionEntity region, {required bool inside}) {
    final manager = ref.read(tileRepositoryManagerProvider);
    final lonSpan = region.maxLon - region.minLon;
    final latSpan = region.maxLat - region.minLat;
    final bounds = inside
        ? LngLatBounds(
            longitudeWest: region.minLon + lonSpan / 4,
            longitudeEast: region.minLon + lonSpan * 3 / 4,
            latitudeSouth: region.minLat + latSpan / 4,
            latitudeNorth: region.minLat + latSpan * 3 / 4,
          )
        : LngLatBounds(
            longitudeWest: region.maxLon + 10,
            longitudeEast: region.maxLon + 11,
            latitudeSouth: region.maxLat + 10,
            latitudeNorth: region.maxLat + 11,
          );
    final paths = manager.localTilePathsForBounds(bounds);
    final label = inside ? 'inside' : 'outside';
    debugPrint(
      '[harness] localTilePathsForBounds($label ${region.id}) -> $paths',
    );
    setState(() {
      _lastQueryResult = '$label(${region.id}): $paths';
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('TileRepositoryManager harness')),
      floatingActionButton: FloatingActionButton(
        onPressed: _refreshCatalog,
        tooltip: 'Refresh region catalog',
        child: const Icon(Icons.refresh),
      ),
      body: ListView(
        padding: const EdgeInsets.all(8),
        children: [
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _serverUrlController,
                  decoration: const InputDecoration(
                    labelText: 'Backend base URL',
                  ),
                ),
              ),
              const SizedBox(width: 8),
              ElevatedButton(
                onPressed: _connect,
                child: const Text('Connect'),
              ),
            ],
          ),
          const SizedBox(height: 8),
          if (_statusLine != null) Text('Last progress: $_statusLine'),
          if (_lastQueryResult != null) Text('Last query: $_lastQueryResult'),
          const Divider(),
          for (final region in _regions)
            Card(
              margin: const EdgeInsets.symmetric(vertical: 4),
              child: Padding(
                padding: const EdgeInsets.all(8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '${region.name} (${region.id}) -- RegionStatus='
                      '${region.status} demPackageStatus='
                      '${region.demPackage.target?.status ?? PackageStatus.notDownloaded}',
                      style: Theme.of(context).textTheme.titleSmall,
                    ),
                    Wrap(
                      spacing: 8,
                      runSpacing: 4,
                      children: [
                        ElevatedButton(
                          onPressed: () => _downloadVector(region),
                          child: const Text('Download vector'),
                        ),
                        ElevatedButton(
                          onPressed: () => _downloadDem(region),
                          child: const Text('Download DEM'),
                        ),
                        ElevatedButton(
                          onPressed: () => _pause(region),
                          child: const Text('Pause'),
                        ),
                        ElevatedButton(
                          onPressed: () => _resume(region),
                          child: const Text('Resume'),
                        ),
                        ElevatedButton(
                          onPressed: () => _delete(region),
                          child: const Text('Delete'),
                        ),
                        OutlinedButton(
                          onPressed: () => _queryBounds(region, inside: true),
                          child: const Text('Query inside bbox'),
                        ),
                        OutlinedButton(
                          onPressed: () => _queryBounds(region, inside: false),
                          child: const Text('Query outside bbox'),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          if (_regions.isEmpty)
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text(
                'No regions loaded yet. Set the backend URL, tap Connect, '
                'then tap the refresh FAB to fetch the region catalog.',
              ),
            ),
        ],
      ),
    );
  }
}
