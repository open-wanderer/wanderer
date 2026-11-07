import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:skeletonizer/skeletonizer.dart';
import 'package:wanderer/components/async_loader.dart';

void main() {
  Future<bool> skeletonOn(
    WidgetTester tester,
    AsyncValue<String> value, {
    bool keep = false,
  }) async {
    late bool enabled;
    await tester.pumpWidget(
      MaterialApp(
        home: AsyncLoader<String>(
          asyncValue: value,
          mockData: 'mock',
          keepValueWhileLoading: keep,
          builder: (data) => Builder(
            builder: (context) {
              enabled = Skeletonizer.maybeOf(context)?.enabled ?? false;
              return Text(data);
            },
          ),
        ),
      ),
    );
    return enabled;
  }

  // ignore: invalid_use_of_internal_member
  final reloading = const AsyncLoading<String>().copyWithPrevious(
    const AsyncData('real'),
  );

  testWidgets('skeletonizes the mock on a first load', (tester) async {
    expect(await skeletonOn(tester, const AsyncLoading(), keep: true), isTrue);
    expect(find.text('mock'), findsOneWidget);
  });

  testWidgets('skeletonizes a reload by default', (tester) async {
    expect(await skeletonOn(tester, reloading), isTrue);
  });

  testWidgets('keepValueWhileLoading leaves the value on screen', (
    tester,
  ) async {
    expect(await skeletonOn(tester, reloading, keep: true), isFalse);
    expect(find.text('real'), findsOneWidget);
  });
}
