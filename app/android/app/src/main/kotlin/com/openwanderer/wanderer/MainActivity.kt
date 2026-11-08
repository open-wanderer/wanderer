package com.openwanderer.wanderer

import android.os.Bundle
import android.util.Log
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.security.KeyStore
import org.maplibre.android.MapLibre

class MainActivity : FlutterActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // MapLibre suppresses every online-file-source request when its
        // ConnectivityReceiver reports no network -- including requests to our
        // loopback tile proxy. Every style routes through that proxy, so this
        // must stay pinned `true` or the proxy becomes unreachable the moment
        // the radio drops.
        //
        // Accepted consequence: a permanent pin means MapLibre never sees the
        // false->true edge networkIsReachableAgain() needs, so it never
        // retries a Connection-failed tile on its own. Recovery is
        // user-initiated -- a pan or zoom issues fresh requests at new
        // coordinates. Driving that edge deliberately was tried on a device
        // and does not work: MapLibre re-schedules nothing.
        MapLibre.getInstance(applicationContext)
        MapLibre.setConnected(true)
    }

    // Dart's TLS stack (BoringSSL in dart:io) reads only the system CA store
    // on Android and ignores network_security_config, so a user-installed CA
    // never reaches it and self-hosted instances behind a private root CA fail
    // TLS (#1255). This hands the user CAs' DER bytes to Dart at startup,
    // where user_ca_trust.dart adds them to SecurityContext.defaultContext.
    // Only `user:` aliases are read because system CAs are already trusted.
    // Failures return an empty list because a missing CA must never stop the
    // app from starting.
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, USER_CA_CHANNEL)
            .setMethodCallHandler { call, result ->
                if (call.method == "getUserCaCertificates") {
                    result.success(readUserCaCertificates())
                } else {
                    result.notImplemented()
                }
            }
    }

    private fun readUserCaCertificates(): List<ByteArray> {
        return try {
            val keyStore = KeyStore.getInstance("AndroidCAStore")
            keyStore.load(null)
            keyStore.aliases().toList()
                .filter { it.startsWith("user:") }
                // One unreadable certificate is dropped instead of emptying
                // the whole list.
                .mapNotNull { alias ->
                    runCatching { keyStore.getCertificate(alias)?.encoded }.getOrNull()
                }
        } catch (e: Exception) {
            Log.w(TAG, "Could not read user CA certificates", e)
            emptyList()
        }
    }

    companion object {
        private const val USER_CA_CHANNEL = "com.openwanderer.wanderer/user_ca"
        private const val TAG = "UserCaTrust"
    }
}
