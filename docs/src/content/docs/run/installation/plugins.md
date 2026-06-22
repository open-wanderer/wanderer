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
```

wanderer discovers plugins from `data/plugins/<plugin-id>/plugin.json`. After
discovery, the plugin appears in the plugin settings page.

## Installing release bundles

Official Docker images do not include provider plugins. Download plugin bundle
archives from the GitHub release assets, extract them, and copy the extracted
plugin directory into the mounted `./data/plugins` directory.

There is no built-in plugin store. Community plugins can be installed the same
way, but only install plugin bundles from sources you trust.

## Source checkout

When running from a source checkout, first-party plugin source lives under the
repository's `plugins/` directory. That source directory is not the runtime
install location.

Build and install the bundled plugins into `data/plugins` with:

```sh
make plugins-install-local
```

Use this after a fresh checkout or after changing first-party plugin code.

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
