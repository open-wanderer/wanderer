/// Mint / read / persist the loopback tile proxy's stable-but-random port and
/// per-install secret path segment.
///
/// Split into a pure half (minting/validation) and a store-backed half, so the
/// pure half stays unit-testable without a live [Store]. Takes a raw [Store]
/// rather than a provider: `TileProxyServer.start` runs before `ProviderScope`
/// exists.
library;

import 'dart:math';

import 'package:wanderer/entities/local_settings_entity.dart';
import 'package:wanderer/objectbox.g.dart';

/// IANA dynamic/private port range — the same range an ephemeral bind would
/// have drawn from, so a persisted value is no more guessable. It is simply
/// drawn once and remembered instead of redrawn every launch.
const int kTileProxyPortMin = 49152;
const int kTileProxyPortMax = 65535;

/// A stable loopback tile proxy identity: the port the server binds to and
/// the per-install secret required as the first path segment of every
/// request.
class TileProxyIdentity {
  const TileProxyIdentity({required this.port, required this.secret});

  final int port;
  final String secret;

  @override
  bool operator ==(Object other) =>
      other is TileProxyIdentity &&
      other.port == port &&
      other.secret == secret;

  @override
  int get hashCode => Object.hash(port, secret);
}

/// Exactly 32 lowercase hex characters. Anchored at both ends so a corrupted
/// or truncated persisted value — or one containing `/` or `..` — can never
/// become a live route prefix.
final RegExp _secretPattern = RegExp(r'^[0-9a-f]{32}$');

/// Gate applied before accepting a persisted secret and before comparing a
/// request's first path segment against it.
bool isValidProxySecret(String secret) => _secretPattern.hasMatch(secret);

/// Mints a fresh [TileProxyIdentity]: a random port in
/// `[kTileProxyPortMin, kTileProxyPortMax]` and a random 128-bit secret
/// rendered as 32 lowercase hex characters.
///
/// [random] defaults to `Random.secure()`, which is mandatory in production:
/// the port and secret must be unguessable to a co-resident app. A seeded
/// `Random` is accepted only so tests can assert deterministic output.
TileProxyIdentity mintTileProxyIdentity({Random? random}) {
  final rng = random ?? Random.secure();
  final port =
      kTileProxyPortMin + rng.nextInt(kTileProxyPortMax - kTileProxyPortMin + 1);
  final buffer = StringBuffer();
  for (var i = 0; i < 16; i++) {
    buffer.write(rng.nextInt(256).toRadixString(16).padLeft(2, '0'));
  }
  return TileProxyIdentity(port: port, secret: buffer.toString());
}

/// Reads the persisted proxy identity from [store], minting and persisting a
/// fresh one if the persisted row is missing or invalid.
///
/// A valid identity is returned unchanged — that stability is what keeps
/// MapLibre's ambient tile cache alive across cold starts. An invalid row is
/// never partially repaired; port and secret are re-minted together.
TileProxyIdentity resolveTileProxyIdentity(Store store) {
  final box = store.box<LocalSettingsEntity>();
  final entity = box.getAll().firstOrNull ?? LocalSettingsEntity();

  final persistedPort = entity.tileProxyPort;
  final persistedSecret = entity.tileProxySecret;
  if (persistedPort >= kTileProxyPortMin &&
      persistedPort <= kTileProxyPortMax &&
      isValidProxySecret(persistedSecret)) {
    return TileProxyIdentity(port: persistedPort, secret: persistedSecret);
  }

  final minted = mintTileProxyIdentity();
  entity.tileProxyPort = minted.port;
  entity.tileProxySecret = minted.secret;
  box.put(entity);
  return minted;
}

/// Overwrites only the persisted port, leaving the secret untouched.
///
/// Used by the bind-retry path when the persisted port is already occupied.
/// The secret must survive a rebind — re-minting it would churn every cache
/// key for no reason.
void persistTileProxyPort(Store store, int port) {
  final box = store.box<LocalSettingsEntity>();
  final entity = box.getAll().firstOrNull ?? LocalSettingsEntity();
  entity.tileProxyPort = port;
  box.put(entity);
}
