import 'package:dio/dio.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Why a server would not issue the `state` for an OAuth login.
enum OAuthAppStateFailure {
  /// The instance does not offer this provider (any more).
  invalidProvider,

  /// Too many requests from this address.
  rateLimited,

  /// The server, or the backend behind it, cannot answer right now.
  serverUnavailable,

  /// Any other refusal, or an answer without a usable state.
  rejected,
}

/// Thrown when `POST /api/v1/auth/oauth/app-state` does not yield a state, so
/// the login ends before the browser opens.
class OAuthAppStateException implements Exception {
  final OAuthAppStateFailure failure;

  /// How long to wait before trying again, when the server says so.
  final int? retryAfterSeconds;

  const OAuthAppStateException(this.failure, {this.retryAfterSeconds});

  /// Classifies an error response. Reverse proxies answer these statuses too,
  /// so nothing here relies on the body or on a well-formed `Retry-After`.
  factory OAuthAppStateException.fromResponse(Response<dynamic> response) {
    final status = response.statusCode;
    final data = response.data;
    if (status == 429) {
      final seconds = int.tryParse(
        response.headers.value('retry-after')?.trim() ?? '',
      );
      return OAuthAppStateException(
        OAuthAppStateFailure.rateLimited,
        retryAfterSeconds: seconds != null && seconds > 0 ? seconds : null,
      );
    }
    if (status == 400 && data is Map && data['message'] == 'invalid_provider') {
      return const OAuthAppStateException(OAuthAppStateFailure.invalidProvider);
    }
    if (status != null && status >= 500) {
      return const OAuthAppStateException(
        OAuthAppStateFailure.serverUnavailable,
      );
    }
    return const OAuthAppStateException(OAuthAppStateFailure.rejected);
  }

  String localizedMessage(AppLocalizations l10n) {
    final wait = retryAfterSeconds;
    return switch (failure) {
      OAuthAppStateFailure.invalidProvider => l10n.oauth_provider_unavailable,
      OAuthAppStateFailure.rateLimited =>
        wait == null
            ? l10n.oauth_login_rate_limited
            : l10n.oauth_login_rate_limited_seconds(wait),
      OAuthAppStateFailure.serverUnavailable =>
        l10n.oauth_login_server_unavailable,
      OAuthAppStateFailure.rejected => l10n.oauth_login_rejected,
    };
  }

  @override
  String toString() =>
      'OAuthAppStateException: ${failure.name}'
      '${retryAfterSeconds == null ? '' : ', retry after ${retryAfterSeconds}s'}';
}
