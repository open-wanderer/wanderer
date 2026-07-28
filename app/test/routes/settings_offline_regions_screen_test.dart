import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/region_download_state.dart';
import 'package:wanderer/models/region_hierarchy_row.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/provider/region/tile_repository_provider.dart';
import 'package:wanderer/routes/settings_offline_regions_screen.dart';
import 'package:skeletonizer/skeletonizer.dart';

// ---------------------------------------------------------------------------
// Widget tests for the flat->tree conversion (31-02). This screen has no
// prior widget-test baseline (RESEARCH.md Pitfall 5) -- these tests are
// written from scratch, focused on the two APPUI-02 regression surfaces
// (leaf download actions, disk-usage summary) plus the new APPUI-01
// hierarchy behavior (chevron expand/collapse).
//
// No real ObjectBox Store or network is available in a widget test, so:
// - `regionListNotifierProvider` is overridden with a fixed leaf-entity list.
// - `tileRepositoryStatusProvider` is overridden with an empty download-state
//   map.
// - `regionRepositoryProvider` is overridden with a stub `RegionRepository`
//   whose `refreshCatalogAndFetchHierarchy()` returns a fixed group+leaf
//   fixture without touching a real Store, so a bare (never-called)
//   `_FakeStore` is a safe stand-in for its constructor's `Store` parameter.
// ---------------------------------------------------------------------------

/// Never invoked -- `_StubRegionRepository` overrides every method the
/// screen calls, so `RegionRepository`'s `_store` field is never
/// dereferenced. Only exists to satisfy the constructor's `Store` parameter
/// type without opening a real (native) ObjectBox store in a widget test.
class _FakeStore implements Store {
  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _StubRegionRepository extends RegionRepository {
  _StubRegionRepository(this._hierarchyRows) : super(Dio(), _FakeStore());

  final List<RegionHierarchyRow> _hierarchyRows;

  @override
  Future<List<RegionHierarchyRow>> refreshCatalogAndFetchHierarchy() async =>
      _hierarchyRows;
}

/// Like `_StubRegionRepository` but leaves `refreshCatalogAndFetchHierarchy()`
/// pending on an external `Completer`, so a test can observe the screen
/// mid-load (before the initial fetch resolves) and drive resolution
/// deterministically.
class _DeferredRegionRepository extends RegionRepository {
  _DeferredRegionRepository(this._completer) : super(Dio(), _FakeStore());

  final Completer<List<RegionHierarchyRow>> _completer;

  @override
  Future<List<RegionHierarchyRow>> refreshCatalogAndFetchHierarchy() =>
      _completer.future;
}

class _StubRegionListNotifier extends RegionListNotifier {
  _StubRegionListNotifier(this._regions);

  final List<RegionEntity> _regions;

  @override
  List<RegionEntity> build() => _regions;
}

class _StubTileRepositoryStatus extends TileRepositoryStatus {
  @override
  Map<String, RegionDownloadState> build() => {};
}

List<RegionHierarchyRow> _fixtureHierarchyRows() => [
  const RegionHierarchyRow(
    id: 'europe',
    name: 'Europe',
    kind: RegionNodeKind.group,
    parent: '',
    path: 'europe',
    depth: 0,
  ),
  const RegionHierarchyRow(
    id: 'de-nrw',
    name: 'North Rhine-Westphalia',
    kind: RegionNodeKind.leaf,
    parent: 'europe',
    path: 'europe/de-nrw',
    depth: 1,
  ),
];

RegionEntity _fixtureRegionEntity() {
  final region = RegionEntity(
    id: 'de-nrw',
    name: 'North Rhine-Westphalia',
    minLon: 5.9,
    minLat: 50.3,
    maxLon: 9.5,
    maxLat: 52.5,
    version: '2026-07-01',
    vectorUrl: '/api/v1/regions/de-nrw/download',
    vectorSize: 123456,
    demUrl: '/api/v1/regions/de-nrw/download-dem',
    demSize: 654321,
    catalogStatus: CatalogStatus.ready,
  )..vectorPackage.target = DownloadedTilePackageEntity(
      status: PackageStatus.downloaded,
      sizeBytesOnDisk: 123456,
    );
  return region;
}

Future<void> _pumpScreen(WidgetTester tester) async {
  tester.view.physicalSize = const Size(1080, 4000);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        regionRepositoryProvider.overrideWithValue(
          _StubRegionRepository(_fixtureHierarchyRows()),
        ),
        regionListNotifierProvider.overrideWith(
          () => _StubRegionListNotifier([_fixtureRegionEntity()]),
        ),
        tileRepositoryStatusProvider.overrideWith(
          _StubTileRepositoryStatus.new,
        ),
      ],
      child: const MaterialApp(
        localizationsDelegates: [
          AppLocalizations.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        supportedLocales: AppLocalizations.supportedLocales,
        locale: Locale('en'),
        home: SettingsOfflineRegionsScreen(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'a downloaded leaf renders its Vector delete action and DEM tile nested '
    'under its group (APPUI-01, APPUI-02 no-regression)',
    (tester) async {
      await _pumpScreen(tester);

      // Group row renders with its name (APPUI-01).
      expect(find.text('Europe'), findsOneWidget);

      // The leaf has an existing Vector download, so computeDefaultExpanded
      // (D-02) auto-expands its group -- the leaf's own header is visible
      // without any tap.
      expect(find.text('North Rhine-Westphalia'), findsOneWidget);

      // Vector tile: downloaded -> trash (delete) action, same as the flat
      // list rendered before this phase (APPUI-02).
      expect(find.byIcon(FontAwesomeIcons.trash.data), findsOneWidget);

      // DEM tile renders (demUrl != null) with its own not-yet-downloaded
      // download action -- independent of the Vector package (APPUI-02).
      expect(find.byIcon(FontAwesomeIcons.download.data), findsOneWidget);
    },
  );

  testWidgets(
    'the disk-usage summary total still renders in the hierarchical layout '
    '(APPUI-02 no-regression)',
    (tester) async {
      await _pumpScreen(tester);

      // The summary line always renders regardless of the real on-disk byte
      // count (unavailable in a widget test without a real filesystem/path
      // provider plugin) -- its presence, not its exact figure, is what
      // APPUI-02 requires to survive the flat->tree conversion.
      expect(find.textContaining('used across'), findsOneWidget);
    },
  );

  testWidgets(
    'a group row shows a chevron and toggles child-row visibility on tap '
    '(APPUI-01)',
    (tester) async {
      await _pumpScreen(tester);

      // Auto-expanded by default (D-02: the leaf has a Vector download) --
      // the chevron reads "expanded" and the leaf is visible.
      expect(find.byIcon(FontAwesomeIcons.angleDown.data), findsOneWidget);
      expect(find.byIcon(FontAwesomeIcons.angleRight.data), findsNothing);
      expect(find.text('North Rhine-Westphalia'), findsOneWidget);

      await tester.tap(find.text('Europe'));
      await tester.pumpAndSettle();

      // Collapsed: chevron flips, leaf row disappears.
      expect(find.byIcon(FontAwesomeIcons.angleRight.data), findsOneWidget);
      expect(find.byIcon(FontAwesomeIcons.angleDown.data), findsNothing);
      expect(find.text('North Rhine-Westphalia'), findsNothing);

      await tester.tap(find.text('Europe'));
      await tester.pumpAndSettle();

      // Expands again on a second tap.
      expect(find.byIcon(FontAwesomeIcons.angleDown.data), findsOneWidget);
      expect(find.text('North Rhine-Westphalia'), findsOneWidget);
    },
  );

  testWidgets(
    'while the initial fetch is in flight the tree area shows a loading '
    'skeleton, not the offline empty state',
    (tester) async {
      tester.view.physicalSize = const Size(1080, 4000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      // Leave the hierarchy fetch pending so we can observe the load state.
      final completer = Completer<List<RegionHierarchyRow>>();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            regionRepositoryProvider.overrideWithValue(
              _DeferredRegionRepository(completer),
            ),
            // Fresh-install shape: zero cached regions. Before the fix this is
            // exactly when the offline "Can't load regions" state flashed
            // during load (regions empty AND _treeRoots still null).
            regionListNotifierProvider.overrideWith(
              () => _StubRegionListNotifier(const []),
            ),
            tileRepositoryStatusProvider.overrideWith(
              _StubTileRepositoryStatus.new,
            ),
          ],
          child: const MaterialApp(
            localizationsDelegates: [
              AppLocalizations.delegate,
              GlobalMaterialLocalizations.delegate,
              GlobalWidgetsLocalizations.delegate,
              GlobalCupertinoLocalizations.delegate,
            ],
            supportedLocales: AppLocalizations.supportedLocales,
            locale: Locale('en'),
            home: SettingsOfflineRegionsScreen(),
          ),
        ),
      );

      // One frame after initState fired the (still-pending) fetch: the loading
      // skeleton renders and the offline empty state does NOT. `Skeletonizer`
      // is an abstract factory (concrete type `_Skeletonizer`), so match its
      // always-present public `SkeletonizerScope` instead of the factory type.
      await tester.pump();
      expect(find.byType(SkeletonizerScope), findsOneWidget);
      expect(find.text("Can't load regions"), findsNothing);
      expect(find.text('Europe'), findsNothing);

      // Resolving the fetch swaps the skeleton for the real tree.
      completer.complete(_fixtureHierarchyRows());
      await tester.pumpAndSettle();
      expect(find.byType(SkeletonizerScope), findsNothing);
      expect(find.text('Europe'), findsOneWidget);
    },
  );
}
