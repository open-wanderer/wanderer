---
title: Plugins
description: How installed provider plugins extend wanderer
---

Plugins add optional provider integrations that are not built into the core
application. They can synchronize trails, find media assets, or provide routing
features.

Official Wanderer images do not include provider plugin bundles. A server
administrator must install a bundle before it appears under **Settings →
Plugins**. Installation, updates, and connector trust settings are administrator
tasks; see [Plugin installation](/run/installation/plugins/).

## Plugin types

The plugin settings page groups installed plugins by purpose:

| Type | Purpose |
| --- | --- |
| [Trail plugins](/use/plugins/trail/) | Import, synchronize, or send trails, routes, and activities. |
| [Asset plugins](/use/plugins/asset/) | Find, import, or remotely link photos and other media. |
| [Routing plugins](/use/plugins/routing/) | Calculate routes, elevation, variants, round trips, or maneuvers. |

The available actions and settings come from each plugin's manifest and
capabilities. Two plugins of the same type do not necessarily support the same
features.

## Configure and enable a plugin

1. Open **Settings → Plugins**.
2. Find the plugin in its type group.
3. Open its settings and enter any required credentials or provider options.
4. Save the settings. OAuth plugins must be connected through the provider's
   authorization page before they can be enabled.
5. Toggle the plugin on.

Some plugins can be enabled without credentials, while others require an API
key, an account session, or OAuth authorization. Plugin-specific fields and
validation are shown directly in the settings dialog.

An enabled plugin can still report a setup or provider error. Open its settings
to review the configuration, reconnect it when authorization has expired, or
contact the server administrator if the installed bundle itself is unavailable.

## Disable a plugin

Disabling a plugin stops it from participating in future operations. Existing
trails and locally stored assets are retained.

Some plugin types require an additional decision before they can be disabled:

- A routing plugin selected as the active route or elevation engine must first
  be replaced, or routing must be disabled.
- An asset plugin with linked remote photos asks whether those photos should be
  downloaded or the links removed.

See the documentation for the relevant plugin type for details.
