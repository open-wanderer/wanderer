import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/provider/auth_provider.dart';

// ---------------------------------------------------------------------------
// Regression guard for the "author avatar is a blank grey circle" bug.
//
// The inline `CircleAvatar`s these replaced wrote
// `onBackgroundImageError: (_, _) => FaIcon(FontAwesomeIcons.user)`. That
// callback is an `ImageErrorListener` returning void, so the icon was built
// and discarded and the circle stayed empty. These tests assert on the glyph
// actually being in the tree.
// ---------------------------------------------------------------------------

class _StubAuth extends Auth {
  _StubAuth(this.user);

  final UserEntity? user;

  @override
  Future<UserEntity?> build() async => user;
}

Widget _harness(Widget child, {UserEntity? user}) {
  return ProviderScope(
    overrides: [authProvider.overrideWith(() => _StubAuth(user))],
    child: MaterialApp(home: Scaffold(body: child)),
  );
}

void main() {
  testWidgets(
    'an unresolved author renders the person glyph, not an empty circle',
    (tester) async {
      await tester.pumpWidget(
        _harness(
          const ActorAvatar(
            actorId: null,
            imageUrl: '',
            nameSeed: 'Unknown',
            radius: 12,
          ),
        ),
      );
      await tester.pump();

      expect(find.byType(FaIcon), findsOneWidget);
    },
  );

  testWidgets(
    'no initials fallback is attempted without a resolved author -- the seed '
    'alone must not trigger a network fetch',
    (tester) async {
      await tester.pumpWidget(
        _harness(
          const ActorAvatar(
            actorId: null,
            imageUrl: null,
            nameSeed: 'Unknown',
            radius: 12,
          ),
        ),
      );
      await tester.pump();

      final avatar = tester.widget<CircleAvatar>(find.byType(CircleAvatar));
      expect(avatar.backgroundImage, isNull);
      expect(find.byType(FaIcon), findsOneWidget);
    },
  );

  testWidgets('a resolved author with an avatar URL renders an image circle', (
    tester,
  ) async {
    await tester.pumpWidget(
      _harness(
        const ActorAvatar(
          actorId: 'actor-abc',
          imageUrl: 'https://example.test/avatar.png',
          nameSeed: 'hiker',
          radius: 12,
        ),
      ),
    );

    // No pump: the test HTTP client fails every request, so the widget swaps
    // to the glyph as soon as the image resolve completes. The assertion here
    // is about which URL was handed to the image, before that happens.
    final avatar = tester.widget<CircleAvatar>(find.byType(CircleAvatar));
    expect(avatar.backgroundImage, isA<NetworkImage>());
    expect(
      (avatar.backgroundImage! as NetworkImage).url,
      'https://example.test/avatar.png',
    );
  });

  testWidgets('a load failure falls back to the glyph rather than leaving the '
      'circle blank', (tester) async {
    await tester.pumpWidget(
      _harness(
        const ActorAvatar(
          actorId: 'actor-abc',
          imageUrl: 'https://example.test/missing.png',
          nameSeed: 'hiker',
          radius: 12,
        ),
      ),
    );

    // The default test HTTP client returns 400 for every request, so the image
    // resolve fails and the error listener fires.
    await tester.pumpAndSettle();

    expect(find.byType(FaIcon), findsOneWidget);
  });
}
