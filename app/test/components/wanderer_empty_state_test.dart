import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/base/wanderer_empty_state.dart';
import 'package:wanderer/util/format.dart';

void main() {
  Widget harness(Widget child) => MaterialApp(home: Scaffold(body: child));

  testWidgets('shows no button without an action', (tester) async {
    await tester.pumpWidget(
      harness(
        const WandererEmptyState(
          icon: FontAwesomeIcons.route,
          title: 'Title',
          body: 'Body',
        ),
      ),
    );

    expect(find.text('Title'), findsOneWidget);
    expect(find.text('Body'), findsOneWidget);
    expect(find.byType(WandererButton), findsNothing);
  });

  testWidgets('runs the action from its button', (tester) async {
    var taps = 0;
    await tester.pumpWidget(
      harness(
        WandererEmptyState(
          icon: FontAwesomeIcons.route,
          title: 'Title',
          body: 'Body',
          actionLabel: 'Act',
          onAction: () => taps++,
        ),
      ),
    );

    await tester.tap(find.text('Act'));
    expect(taps, 1);
  });

  // REGRESSION GUARD. The profile screens used a bare Center here, which gave
  // the surrounding RefreshIndicator nothing to pull on.
  testWidgets('can be pulled to refresh although the content fits', (
    tester,
  ) async {
    var refreshes = 0;
    await tester.pumpWidget(
      harness(
        RefreshIndicator(
          onRefresh: () async => refreshes++,
          child: const WandererEmptyState(
            icon: FontAwesomeIcons.route,
            title: 'Title',
            body: 'Body',
          ),
        ),
      ),
    );

    await tester.fling(find.text('Title'), const Offset(0, 400), 1000);
    await tester.pumpAndSettle();

    expect(refreshes, 1);
  });

  // REGRESSION GUARD. A Scaffold body hands down a loose width, so without a
  // minimum width the column shrinks to short copy and sits at the left edge.
  testWidgets('centers short copy under a loose width', (tester) async {
    await tester.pumpWidget(
      harness(
        RefreshIndicator(
          onRefresh: () async {},
          child: const WandererEmptyState(
            icon: FontAwesomeIcons.route,
            title: 'Title',
            body: 'Body',
          ),
        ),
      ),
    );

    final screenCenter = tester.getCenter(find.byType(Scaffold)).dx;
    expect(tester.getCenter(find.text('Title')).dx, screenCenter);
    expect(tester.getCenter(find.text('Body')).dx, screenCenter);
  });

  test('usernameFromHandle drops the prefix and the domain', () {
    expect(usernameFromHandle('@alice'), 'alice');
    expect(usernameFromHandle('@alice@example.com'), 'alice');
  });
}
