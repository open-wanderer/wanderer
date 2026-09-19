---
title: Plugin installation
description: How to install and operate provider plugins
---

Provider integrations are installed as local WASM plugin bundles. A runtime
plugin bundle is a directory with at least:

```text
plugin.json
plugin.wasm
```

Install each extracted bundle as a direct child directory of `data/plugins`:

```text
data/plugins/strava/plugin.json
data/plugins/strava/plugin.wasm
data/plugins/immich/plugin.json
data/plugins/immich/plugin.wasm
data/plugins/valhalla/plugin.json
data/plugins/valhalla/plugin.wasm
```

wanderer discovers plugins from `data/plugins/<plugin-id>/plugin.json`. After
discovery, the plugin appears in the plugin settings page.

## Installing release bundles

Official Wanderer images do not include provider plugins. First-party describes
who maintains a plugin, not whether its runtime bundle is packaged in an image.
Download plugin bundle archives from the
[Wanderer GitHub release assets](https://github.com/open-wanderer/wanderer/releases),
extract them, and copy the extracted plugin directory into the mounted
`./data/plugins` directory.

There is no built-in plugin store. Community plugins can be installed the same
way, but only install plugin bundles from sources you trust.

## Source checkout

When running from a source checkout, first-party plugin source lives under the
repository's `plugins/` directory. That source directory is not the runtime
install location.

Build and install the first-party plugins from the source checkout into
`data/plugins` with:

```sh
make plugins-install-local
```

Use this after a fresh checkout or after changing first-party plugin code.

## Updating plugin bundles

Updating the Wanderer containers or core binaries does not update separately
installed provider bundles. For each installed plugin, use the bundle asset from
the same Wanderer release as the host unless the release notes state otherwise.
Replace the complete `data/plugins/<plugin-id>` directory instead of mixing
files from different bundle versions, then restart the `db` service or process
so the new bundle is discovered. When using a source checkout, rerun
`make plugins-install-local` after updating the checkout.

## Runtime and network model

Plugins run as local WASM modules in a separate worker process. Provider API and
media requests are still executed by the backend through the plugin manifest's
network policy; plugins do not get unrestricted access to your server network.

Self-hosted provider plugins may expose connector settings such as a base URL,
private-network access, storage redirect origins, or a custom CA bundle. Treat
those settings as administrator trust decisions: only enable private-network
access or custom CAs for plugin bundles and endpoints you trust.

Provider plugin connector CAs are configured per connector when a plugin
supports custom TLS. They are not read from `NODE_EXTRA_CA_CERTS`.

## Self-hosted asset plugins

Asset plugins such as Immich commonly connect to self-hosted services. Those
plugins use a configured connector in the plugin manifest. The plugin can
declare that a connector supports private-network access, storage redirects, or
custom TLS, but the actual trust decision is made in wanderer configuration.

For Immich, review the connector settings before enabling the plugin:

| Setting | Meaning |
| --- | --- |
| Base URL | The Immich server URL that wanderer should contact. |
| Private network access | Allows wanderer to reach private IP ranges for this connector. Enable only for trusted self-hosted endpoints. |
| Custom CA | Trusts a custom certificate authority for this connector if the plugin permits it. |
| Storage redirects | Allows media downloads to follow redirects to configured storage origins when the plugin permits it. |

If Immich is only reachable from the server running wanderer, make sure the
backend container can resolve and reach that URL. Browser access from your
desktop is not enough; the backend performs all provider and media requests.
Private-network access, custom CAs, and storage redirect origins require a fixed
administrator-configured connector base URL. When the base URL is instead taken
from a user's Immich plugin setting, wanderer enforces public-network access and
system TLS and does not allow storage redirects.

For a provider running directly on the Wanderer host, configure an explicit
loopback base URL such as `http://127.0.0.1:17777` and enable private-network
access. Loopback remains blocked when a public or other DNS hostname is used.

## Routing plugin activation and selection

Valhalla is the only first-party plugin currently marked default-active by
Wanderer's host-owned policy. After an administrator installs a valid Valhalla
release bundle, its first successful discovery creates enabled instances for
all existing users who do not already have one. Users created while the bundle
is installed also receive an enabled instance. A previously created instance is
not changed, so an explicitly disabled Valhalla instance stays disabled. Later
successful synchronizations repair missing instances without changing existing
ones. Disable the instance for a durable opt-out; deleting its database record
is treated like incomplete provisioning and the next successful synchronization
recreates it.

BRouter and community routing plugins remain opt-in. A community manifest
cannot request automatic activation.

Routing defaults do not contain a Valhalla or other provider ID. When a route,
elevation, or maneuver selection is missing, Wanderer resolves it from enabled
plugins that implement the corresponding executable capability, using stable
plugin setup order. Explicit administrator and user selections remain
authoritative. Consequently, users do not need to activate or select Valhalla
manually when it is the only enabled plugin capable of the requested role.

## Valhalla routing endpoint

The Valhalla routing plugin uses an administrator-controlled connector URL. On
the first successful discovery of a valid Valhalla bundle, a set `VALHALLA_URL`
environment variable on the `db` service/process is imported into the plugin
configuration. An invalid bundle or failed discovery does not import it. Routine
rediscovery never reimports the value, so later environment changes have no
effect; update the endpoint in the plugin configuration instead. New
installations do not need the legacy variable because the plugin supplies its
own connector default. If the bundle is removed, Wanderer retains a
non-executable cache marker and the administrator-owned configuration. A later
reinstallation therefore does not reopen the legacy import.

Users cannot override the connector URL in their Valhalla plugin settings.
This keeps connector targets, private-network access, and TLS trust under
administrator control.

## BRouter routing endpoint

The BRouter plugin defaults to the public `https://brouter.de` service. To use
a local BRouter instance, set the administrator-owned BRouter connector base
URL to a fixed address such as `http://127.0.0.1:17777` and enable
private-network access for that connector. Do not use a public hostname that
resolves to loopback; the connector deliberately rejects that DNS-rebinding
pattern.

The URL is resolved from the backend's network namespace. If Wanderer runs in
a container while BRouter runs on the physical host, `127.0.0.1` points to the
Wanderer container, not automatically to the host. Use an explicitly configured
host-gateway address or place both services on a shared container network and
use the BRouter service name instead. Private service addresses still require
the connector's private-network permission.

Custom and generated BRouter profiles are uploaded through
`/brouter/profile`. Wanderer can prepare that upload while the route editor is
waiting for the first anchor and reuses the opaque provider profile ID for up
to 15 minutes. A provider restart can invalidate an ID early; the host then
coordinates one refresh and retries all waiting route calls with its result.
