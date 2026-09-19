---
title: Routing plugins
description: Configure routing capabilities, profiles, and first-party routing providers
---

Routing plugins provide the engines used by
[Route planning](/use/routing/). Wanderer discovers their behavior from declared
capabilities and metadata instead of hard-coding provider-specific behavior in
the trail editor.

## Routing capabilities

A routing plugin can implement any supported combination of these roles and
features:

| Capability or metadata | Purpose |
| --- | --- |
| Route calculation | Calculate route geometry between anchors. |
| Elevation | Add missing heights independently of route calculation. |
| Maneuvers | Generate turn-by-turn navigation instructions for a saved trail. |
| Via routing | Route through an ordered list of anchors in one request. |
| Alternatives | Return several candidate routes. |
| Round trips | Generate a closed route from a start, target distance, and optional direction. |
| Custom profiles | Upload or edit provider-native routing profiles. |
| Native controls | Expose profile-specific provider settings. |

Controls in the route editor appear only when an enabled plugin declares the
required capability. A community plugin can expose a different combination from
the first-party plugins described below.

## Activation and engine selection

Enable installed routing plugins under **Settings → Plugins**. Then open
**Settings → Routing** to select the engines used for route calculation,
elevation, and navigation.

When no engine is explicitly selected, Wanderer resolves a suitable enabled
plugin from its executable capabilities. The primary route engine is preferred
for elevation and navigation when it supports those roles.

A plugin selected as the active route or elevation engine cannot be disabled or
removed. Select a replacement first, or disable routing under **Settings →
Routing**.

## Profiles by category

Open a routing plugin's settings to map Wanderer categories and subcategories to
that plugin's profiles. For example, a plugin may offer different native
profiles for hiking, road cycling, gravel cycling, or motor routing.

A subcategory mapping overrides its broad category mapping. Advanced controls
appear when the selected profile exposes standard preferences or
provider-specific options.

Some plugins also support custom profile files:

1. Upload the file under **Custom profiles**.
2. Resolve its transport mode if the plugin cannot detect it automatically.
3. Enable the profile.
4. Assign it to a category or subcategory.

A custom profile cannot be disabled or deleted while a category mapping still
uses it.

## First-party plugin capabilities

The following table describes the current first-party plugins. It is not a
restriction on community plugins.

| Feature | Valhalla | BRouter |
| --- | --- | --- |
| Normal route calculation | Yes | Yes |
| Via routing | Yes | Yes |
| Alternative routes | Yes | Yes |
| Elevation profile in the finished Wanderer route | Yes | Yes |
| Independent elevation calculation | Yes | No |
| Turn-by-turn maneuvers | Yes | No |
| Round-trip generation | No | Yes |
| Custom profile upload | No | Yes, using `.brf` files |

## Valhalla

Valhalla provides route calculation, independent elevation enrichment, and
turn-by-turn maneuvers. Its route response does not embed heights directly;
Wanderer uses the plugin's separate elevation capability to add them to route
points. The finished Wanderer route can therefore contain a complete elevation
profile.

Valhalla is currently the only first-party plugin that Wanderer enables by
default after an administrator installs a valid bundle. Existing instances that
were explicitly disabled remain disabled. Missing route, elevation, and
maneuver selections can be resolved automatically from its capabilities, so a
user does not need to select it manually when it is the only suitable enabled
plugin.

The default connector uses the public, donation-financed FOSSGIS Valhalla
service. If you use it, please consider
[supporting FOSSGIS](https://www.fossgis.de/verein/spenden/). Administrators can
configure another endpoint as described under
[Valhalla routing endpoint](/run/installation/plugins/#valhalla-routing-endpoint).

## BRouter

BRouter provides route calculation, route alternatives, and round-trip
generation. It includes height values in its route result, so Wanderer can build
the elevation profile without a separate elevation capability.

BRouter is opt-in and must be enabled by each user. Its plugin also supports
personal `.brf` profiles. Upload, edit, download, and enable these profiles under
the BRouter plugin settings, then assign them under **Profiles by category**.

The plugin defaults to the public `https://brouter.de` service. Administrators
can connect it to a local instance; see
[BRouter routing endpoint](/run/installation/plugins/#brouter-routing-endpoint).

## Troubleshooting

### A routing plugin cannot be disabled

Check **Settings → Routing**. If the plugin is selected as the route or elevation
engine, choose another engine or disable routing first.

### A profile is unavailable in the route editor

Make sure the profile is enabled and assigned to the selected trail category.
For a custom profile, also check that its transport mode was detected or
selected.

### A custom profile cannot be prepared

Wanderer normally falls back to regular route calculation when preparing a
profile in advance fails. Persistent errors can indicate invalid profile content
or an unavailable provider service.

### A self-hosted provider cannot be reached

The endpoint must be reachable from the Wanderer backend's network namespace.
Container networking, private-network connector permission, DNS resolution, and
TLS trust are common causes. See
[Plugin installation](/run/installation/plugins/) for administrator guidance.
