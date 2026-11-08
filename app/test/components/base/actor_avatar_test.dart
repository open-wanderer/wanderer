import 'package:cached_network_image/cached_network_image.dart';
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

    // No pump: the assertion here is about which URL was handed to the
    // (disk-caching) image provider, before any resolve completes.
    final avatar = tester.widget<CircleAvatar>(find.byType(CircleAvatar));
    expect(avatar.backgroundImage, isA<CachedNetworkImageProvider>());
    expect(
      (avatar.backgroundImage! as CachedNetworkImageProvider).url,
      'https://example.test/avatar.png',
    );
  });

  testWidgets('a load failure falls back to the glyph rather than leaving the '
      'circle blank', (tester) async {
    // CachedNetworkImageProvider's cache manager does real file/network I/O
    // that never completes under the widget test's FakeAsync zone, so route
    // this test through NetworkImage — which uses the test HTTP client
    // (400 for every request) — to exercise the SAME error-listener wiring
    // this test guards.
    debugAvatarImageProviderFactory = NetworkImage.new;
    addTearDown(() => debugAvatarImageProviderFactory = null);

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

    // The image resolve fails against the test HTTP client and the error
    // listener fires.
    await tester.pumpAndSettle();

    expect(find.byType(FaIcon), findsOneWidget);
  });
}
