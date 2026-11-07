import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/auth_methods_provider.dart';

void main() {
  test('reads signupDisabled and the OAuth providers', () {
    final methods = AuthMethods.fromJson({
      'signupDisabled': true,
      'oauth2': {
        'providers': [
          {
            'name': 'google',
            'displayName': 'Google',
            'state': 's',
            'codeVerifier': 'v',
            'url': 'https://accounts.example/auth',
          },
        ],
      },
    });

    expect(methods.signupDisabled, isTrue);
    expect(methods.oauthProviders.single.name, 'google');
  });

  test('a server without the flag keeps signup enabled', () {
    final methods = AuthMethods.fromJson({
      'oauth2': {'providers': []},
    });

    expect(methods.signupDisabled, isFalse);
    expect(methods.oauthProviders, isEmpty);
  });
}
