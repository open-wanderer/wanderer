package com.openwanderer.wanderer

import android.os.Bundle
import io.flutter.embedding.android.FlutterActivity
import org.maplibre.android.MapLibre

class MainActivity : FlutterActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // MapLibre Native suppresses ALL online-file-source HTTP requests when
        // its ConnectivityReceiver reports no network (e.g. airplane mode) —
        // including requests to our in-app loopback tile proxy
        // (http://127.0.0.1). Forcing the connectivity override to `true`
        // disables that gate so the proxy is always queried regardless of
        // radio state. Our own offline gate (trail.isOffline) already decides
        // when the style points at the loopback proxy vs. real online tiles,
        // so this is safe offline — offline styles never carry an online URL
        // to (fail to) reach.
        MapLibre.getInstance(applicationContext)
        MapLibre.setConnected(true)
    }
}
