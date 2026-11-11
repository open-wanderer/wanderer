import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_photo.dart';
import 'package:wanderer/components/base/wanderer_photo_picker.dart';
import 'package:wanderer/i18n/app_localizations.dart';

Widget app(Widget child) => MaterialApp(
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);

void main() {
  testWidgets('shows preview but reserves cover control for real photos', (
    tester,
  ) async {
    String? selected;
    await tester.pumpWidget(
      app(
        WandererPhotoPicker(
          initialWebPhotos: const [
            'https://a.example/generated.svg',
            'https://a.example/user.svg',
          ],
          generatedPreviews: const {'https://a.example/generated.svg'},
          initialThumbnailPhoto: 'https://a.example/generated.svg',
          onChanged: (_) {},
          onWebPhotosChanged: (_) {},
          onThumbnailChanged: (photo) => selected = photo,
        ),
      ),
    );
    expect(find.byType(WandererPhoto), findsNWidgets(2));
    expect(find.byTooltip('Use as cover'), findsOneWidget);
    expect(find.byTooltip('Cover photo'), findsNothing);
    await tester.tap(find.byTooltip('Use as cover'));
    await tester.pump();
    expect(selected, 'https://a.example/user.svg');
    expect(find.byTooltip('Cover photo'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is FaIcon && widget.icon == FontAwesomeIcons.xmark.data,
      ),
      findsNWidgets(2),
    );
  });

  testWidgets('a lone generated preview keeps its cover control', (
    tester,
  ) async {
    await tester.pumpWidget(
      app(
        WandererPhotoPicker(
          initialWebPhotos: const ['https://a.example/generated.svg'],
          generatedPreviews: const {'https://a.example/generated.svg'},
          initialThumbnailPhoto: 'https://a.example/generated.svg',
          onChanged: (_) {},
          onThumbnailChanged: (_) {},
        ),
      ),
    );
    expect(find.byTooltip('Cover photo'), findsOneWidget);
    expect(find.byType(WandererPhoto), findsOneWidget);
  });
}
