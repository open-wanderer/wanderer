import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/util/oauth_app_state.dart';

void main() {
  final l10n = lookupAppLocalizations(const Locale('en'));

  String message(OAuthAppStateFailure failure, {int? retryAfterSeconds}) =>
      OAuthAppStateException(
        failure,
        retryAfterSeconds: retryAfterSeconds,
      ).localizedMessage(l10n);

  test('names the wait when the server gave one', () {
    expect(
      message(OAuthAppStateFailure.rateLimited, retryAfterSeconds: 42),
      contains('Try again in 42 seconds.'),
    );
    expect(
      message(OAuthAppStateFailure.rateLimited, retryAfterSeconds: 1),
      contains('Try again in 1 second.'),
    );
    expect(
      message(OAuthAppStateFailure.rateLimited),
      contains('Try again in a minute.'),
    );
  });

  test('shows no raw server code for any failure', () {
    for (final failure in OAuthAppStateFailure.values) {
      final text = message(failure);
      expect(text, isNotEmpty);
      expect(text, isNot(contains('_')));
    }
  });
}
