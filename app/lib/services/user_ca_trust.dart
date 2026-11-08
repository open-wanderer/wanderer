/// Makes dart:io trust the CA certificates a user installed on Android.
///
/// dart:io's BoringSSL on Android reads only the system CA store, and
/// `network_security_config` does not apply to it. A self-hosted instance
/// behind a user-installed private root CA therefore fails TLS (#1255).
/// Native code hands over the user CAs at launch and they are added to
/// [SecurityContext.defaultContext], which every dart:io client in the app
/// (Dio, the image caches, the tile proxy) uses.
///
/// A CA installed while the app is running only takes effect after a restart
/// (accepted tradeoff).
library;

import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

/// Name must match `USER_CA_CHANNEL` in MainActivity.kt.
const MethodChannel userCaChannel = MethodChannel(
  'com.openwanderer.wanderer/user_ca',
);

const String _getUserCaCertificatesMethod = 'getUserCaCertificates';

/// Converts DER bytes to a PEM block. `setTrustedCertificatesBytes` documents
/// PEM/PKCS12 input, not raw DER, so the native bytes are re-wrapped here.
@visibleForTesting
String derToPem(List<int> der) {
  final encoded = base64Encode(der);
  final buffer = StringBuffer('-----BEGIN CERTIFICATE-----\n');
  for (var i = 0; i < encoded.length; i += 64) {
    final end = i + 64 < encoded.length ? i + 64 : encoded.length;
    buffer.write(encoded.substring(i, end));
    buffer.write('\n');
  }
  buffer.write('-----END CERTIFICATE-----\n');
  return buffer.toString();
}

/// Adds each DER certificate to [context] and returns how many were accepted.
///
/// Each certificate gets its own try/catch: one invalid or duplicate
/// certificate (a TlsException) must not stop the rest from being added.
@visibleForTesting
int trustDerCertificates(Iterable<List<int>> ders, SecurityContext context) {
  var added = 0;
  for (final der in ders) {
    try {
      context.setTrustedCertificatesBytes(utf8.encode(derToPem(der)));
      added++;
    } catch (error) {
      // Never log the certificate bytes, only the failure.
      debugPrint('user CA trust: skipped a certificate: $error');
    }
  }
  return added;
}

/// Adds the Android user-installed CA certificates to [context] (default:
/// [SecurityContext.defaultContext]) and returns how many were added.
///
/// Best-effort and never throws: a missing handler, a native failure, or a
/// stalled reply is logged and yields 0, since a missing CA must never stop
/// the app from starting. The [timeout] guarantees that even if the native
/// side never replies.
Future<int> trustUserCaCertificates({
  bool? isAndroid,
  MethodChannel channel = userCaChannel,
  SecurityContext? context,
  Duration timeout = const Duration(seconds: 2),
}) async {
  if (!(isAndroid ?? Platform.isAndroid)) return 0;

  final List<Uint8List> certs;
  try {
    final raw = await channel
        .invokeMethod<List<Object?>>(_getUserCaCertificatesMethod)
        .timeout(timeout);
    // whereType instead of invokeListMethod<Uint8List>: a lazy cast would
    // throw on a bad element outside the per-certificate try.
    certs = (raw ?? const <Object?>[]).whereType<Uint8List>().toList();
  } catch (error) {
    debugPrint('user CA trust: could not read user certificates: $error');
    return 0;
  }
  if (certs.isEmpty) return 0;

  final added = trustDerCertificates(
    certs,
    context ?? SecurityContext.defaultContext,
  );
  debugPrint(
    'user CA trust: trusted $added of ${certs.length} user certificate(s)',
  );
  return added;
}
