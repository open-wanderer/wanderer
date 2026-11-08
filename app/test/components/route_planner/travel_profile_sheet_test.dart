import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/route_planner/travel_profile_sheet.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/models/route_travel_bucket.dart';

/// A `CategoryNotifier` fake returning an empty list synchronously — avoids
/// hitting the network or ObjectBox in this widget test.
class _FakeCategoryNotifier extends CategoryNotifier {
  @override
  FutureOr<List<Category>> build() async => const [];
}

/// A `SubcategoryNotifier` fake returning an empty list — the sheet resolves
/// its bucket icons through `categorySelectionForBucket`, which now consults
/// subcategories, and the real notifier would reach for ObjectBox.
class _FakeSubcategoryNotifier extends SubcategoryNotifier {
  @override
  List<Subcategory> build() => const [];
}


/// Opens [showTravelProfileSheet] on tap and stashes the awaited result in
/// [onResult] so the test can assert on it after the sheet closes.
class _Host extends StatelessWidget {
  final void Function(RouteTravelBucket? result) onResult;

  const _Host({required this.onResult});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Builder(
        builder: (context) {
          return ElevatedButton(
            onPressed: () async {
              final result = await showTravelProfileSheet(context);
              onResult(result);
            },
            child: const Text('Open'),
          );
        },
      ),
    );
  }
}

void main() {
  testWidgets('renders all 5 bucket labels', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          categoryProvider.overrideWith(() => _FakeCategoryNotifier()),
          subcategoryProvider.overrideWith(() => _FakeSubcategoryNotifier()),
        ],
        child: MaterialApp(home: _Host(onResult: (_) {})),
      ),
    );

    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    expect(find.text('Hiking'), findsOneWidget);
    expect(find.text('Biking / Hybrid'), findsOneWidget);
    expect(find.text('Biking / Mountain'), findsOneWidget);
    expect(find.text('Biking / Cross'), findsOneWidget);
    expect(find.text('Biking / Road'), findsOneWidget);
  });

  testWidgets(
    'tapping the Mountain card resolves to RouteTravelBucket.bikingMountain',
    (tester) async {
      RouteTravelBucket? captured;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            categoryProvider.overrideWith(() => _FakeCategoryNotifier()),
            subcategoryProvider.overrideWith(() => _FakeSubcategoryNotifier()),
          ],
          child: MaterialApp(
            home: _Host(onResult: (result) => captured = result),
          ),
        ),
      );

      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Biking / Mountain'));
      await tester.pumpAndSettle();

      expect(captured, RouteTravelBucket.bikingMountain);
      expect(captured?.costing, 'bicycle');
      expect(captured?.costingOptions['bicycle_type'], 'Mountain');
    },
  );

  testWidgets('dismissing the sheet resolves to null', (tester) async {
    RouteTravelBucket? captured = RouteTravelBucket.bikingRoad;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          categoryProvider.overrideWith(() => _FakeCategoryNotifier()),
          subcategoryProvider.overrideWith(() => _FakeSubcategoryNotifier()),
        ],
        child: MaterialApp(
          home: _Host(onResult: (result) => captured = result),
        ),
      ),
    );

    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    // Tap outside the sheet (top of the screen) to dismiss it.
    await tester.tapAt(const Offset(200, 50));
    await tester.pumpAndSettle();

    expect(captured, isNull);
  });
}
