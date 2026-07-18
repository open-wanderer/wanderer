import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/route_planner/settings_tab.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';

/// Empty seeded state — no anchors means resolveAllSegments never touches
/// apiProvider, so no Dio override is needed for this widget test.
class _SeededRouteAnchors extends RouteAnchors {
  @override
  RouteAnchorsState build() {
    return const RouteAnchorsState(
      anchors: [],
      segments: [],
      autoRoutingEnabled: true,
      travelProfile: 'pedestrian',
      costingOptions: null,
      undoStack: [],
      redoStack: [],
    );
  }
}

/// Fake returning an empty list synchronously — avoids hitting the network
/// or ObjectBox in this widget test.
class _FakeCategoryNotifier extends CategoryNotifier {
  @override
  FutureOr<List<Category>> build() async => const [];
}

void main() {
  testWidgets(
    'SettingsTab renders exactly 5 bucket options and an auto-routing switch',
    (tester) async {
      final container = ProviderContainer(
        overrides: [
          routeAnchorsProvider.overrideWith(() => _SeededRouteAnchors()),
          categoryProvider.overrideWith(() => _FakeCategoryNotifier()),
        ],
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const MaterialApp(home: Scaffold(body: SettingsTab())),
        ),
      );
      await tester.pump();

      expect(find.text('Hiking'), findsOneWidget);
      expect(find.text('Biking / Hybrid'), findsOneWidget);
      expect(find.text('Biking / Mountain'), findsOneWidget);
      expect(find.text('Biking / Cross'), findsOneWidget);
      expect(find.text('Biking / Road'), findsOneWidget);
      expect(find.byType(SwitchListTile), findsOneWidget);
    },
  );

  testWidgets("tapping the 'Biking / Road' card calls switchProfile('bicycle', "
      "roadOpts)", (tester) async {
    final container = ProviderContainer(
      overrides: [
        routeAnchorsProvider.overrideWith(() => _SeededRouteAnchors()),
        categoryProvider.overrideWith(() => _FakeCategoryNotifier()),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: Scaffold(body: SettingsTab())),
      ),
    );
    await tester.pump();

    await tester.tap(find.text('Biking / Road'));
    await tester.pump();

    final state = container.read(routeAnchorsProvider);
    expect(state.travelProfile, 'bicycle');
    expect(state.costingOptions?['bicycle_type'], 'Road');
  });

  testWidgets('toggling the auto-routing switch flips autoRoutingEnabled', (
    tester,
  ) async {
    final container = ProviderContainer(
      overrides: [
        routeAnchorsProvider.overrideWith(() => _SeededRouteAnchors()),
        categoryProvider.overrideWith(() => _FakeCategoryNotifier()),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: Scaffold(body: SettingsTab())),
      ),
    );
    await tester.pump();

    expect(container.read(routeAnchorsProvider).autoRoutingEnabled, isTrue);

    await tester.tap(find.byType(SwitchListTile));
    await tester.pump();

    expect(container.read(routeAnchorsProvider).autoRoutingEnabled, isFalse);
  });

  testWidgets(
    'the option matching the current state is visually marked selected',
    (tester) async {
      final container = ProviderContainer(
        overrides: [
          routeAnchorsProvider.overrideWith(() => _SeededRouteAnchors()),
          categoryProvider.overrideWith(() => _FakeCategoryNotifier()),
        ],
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const MaterialApp(home: Scaffold(body: SettingsTab())),
        ),
      );
      await tester.pump();

      expect(find.byIcon(Icons.check_circle), findsOneWidget);

      await tester.tap(find.text('Biking / Road'));
      await tester.pump();

      expect(find.byIcon(Icons.check_circle), findsOneWidget);
    },
  );
}
