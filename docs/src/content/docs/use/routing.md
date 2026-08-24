---
title: Route planning
description: Configure route planning and use routing modes, variants, and round trips
---

The route editor calculates paths between the anchors you place on the map. It
can also add elevation data, compare alternatives, or generate round trips when
an enabled routing plugin provides the required capability.

Route planning requires at least one plugin with route support to be installed
by the server administrator and enabled for your account. See
[Routing plugins](/use/plugins/routing/) for available capabilities and
plugin-specific setup, or [Plugin installation](/run/installation/plugins/) for
administrator setup.

## Configure route planning

Open **Settings → Routing** to configure how the route editor uses your enabled
plugins.

### Enable or disable routing

**Enable routing** controls automatic route and elevation calculation in the
route editor. At least one enabled plugin with route support is required before
you can turn it on.

Disabling routing hides the routing controls and stops route, variant, profile,
and elevation operations. Your selected engines are retained for the next time
you enable routing.

### Select engines

When more than one suitable plugin is enabled, the settings page lets you select
engines for separate roles:

- **Routing engine** calculates the route geometry between anchors.
- **Elevation engine** adds missing height values to route points. It can be
  different from the routing engine.
- **Navigation engine** generates turn-by-turn maneuvers for navigation clients.

An engine selector is shown only when there is more than one enabled plugin for
that role. If you do not make an explicit selection, Wanderer chooses a suitable
enabled plugin automatically. It prefers the primary routing engine for
elevation and navigation when that engine supports the corresponding
capability.

:::note
You cannot disable or remove a plugin while it is explicitly selected as your
routing or elevation engine. Select another engine first, or disable routing.
:::

### Default editor behavior

The same settings page controls:

- the default routing mode for newly opened route editors;
- whether route variants are available;
- the default number of variants to request;
- whether additional enabled routing engines participate in variant searches;
- whether a new route may begin at your current location.

Instance administrators can restrict some routing features globally. A feature
disabled by the administrator cannot be re-enabled in personal settings.

## Categories determine the routing profile

Normal route planning is category-driven. The category and optional
subcategory selected in the trail form determine the routing profile and the
general mode of transport.

For example, a plugin can map:

- **Hiking** to a pedestrian or hiking profile;
- **Biking / Road** to a road-bike profile;
- **Biking / Gravel** to a gravel profile;
- **Biking / MTB** to a mountain-bike profile.

This is separate from the editor's **Routing mode**. The category answers
*which kind of route should be calculated*; the routing mode controls *how a
sequence of anchors is sent to the provider*.

To change profile mappings and provider-specific options, open the routing
plugin under **Settings → Plugins**. See
[Routing plugins](/use/plugins/routing/#profiles-by-category) for details.

## Draw or edit a routed trail

Start a new trail and select **Draw a route**, or open an existing trail and
choose **Edit**.

1. Select the trail category and, if appropriate, a subcategory.
2. Leave **Enable auto-routing** turned on.
3. Click the map to place the first anchor.
4. Add more anchors. Wanderer calculates a path between them with the selected
   routing engine.
5. Drag, reorder, or delete anchors to adjust the route.
6. Select **Stop drawing** or **Stop editing** when the route is complete.

Turn off **Enable auto-routing** if you want straight lines between anchors
instead of calculated paths.

The route toolbar also lets you reverse the route, undo or redo changes, show or
hide waypoint markers, and recalculate elevation data. The additional route
settings button appears when the active category and profile expose adjustable
routing controls.

### Segment and via routing

The editor offers up to two routing modes:

- **Route each segment** calculates every adjacent anchor pair separately.
  Moving or replacing one anchor therefore affects only its neighboring
  segments.
- **Route through all waypoints** sends the complete ordered anchor list to one
  capable engine. This allows the provider to optimize the route while treating
  the intermediate anchors as via points.

Via routing is shown only when an enabled route engine supports it. If it is no
longer available, Wanderer can fall back to segment routing and displays a
warning.

### Change the engine while editing

If several routing engines are enabled, the route settings panel contains a
**Routing engine** selector. Changing it affects new or recalculated route
segments. When existing segments were calculated with different planning
settings, Wanderer asks whether to re-route them or keep the existing geometry.

## Compare route variants

When variants are enabled under **Settings → Routing**, the route editor shows a
**Route variants** section after the route has at least two anchors.

1. Expand **Route variants** to calculate alternatives.
2. Hover or focus a result to preview it on the map.
3. Compare its distance, estimated duration, and elevation gain with the
   original route.
4. Select the preferred candidate and choose **Use variant**.
5. Use **Find more variants** when available to request another alternative.

If **Include additional routing engines** is enabled, other suitable and enabled
plugins can participate in the search. A provider may return fewer alternatives
than requested, especially when the route is short or the candidates would be
too similar to the original route. A failure in one additional engine does not
necessarily prevent results from the remaining engines.

Changing anchors or route settings makes existing variant results stale. Run the
variant search again before applying one.

## Generate a round trip

The round-trip controls appear when at least one enabled routing engine supports
round-trip generation.

1. Enable a plugin with round-trip support and choose an appropriate trail
   category or profile. It does not have to be your primary routing engine;
   Wanderer automatically selects an enabled compatible engine.
2. Place the starting anchor.
3. Expand **Generate round trip**.
4. Choose the target distance.
5. Optionally choose a preferred compass direction, or leave it at
   **Automatic**.
6. Select **Generate loop**.

Wanderer turns the generated loop into normal route segments and anchors. You
can then continue editing it like any other routed trail.

## Troubleshooting

### Routing cannot be enabled

Open **Settings → Plugins** and make sure at least one installed plugin with
route support is enabled. If none is listed, the server administrator must
install a compatible plugin bundle first.

### Automatic routing draws straight lines

Check that routing is enabled under **Settings → Routing** and that **Enable
auto-routing** is turned on in the route editor. Straight lines are expected
when auto-routing is off.

### Via routing, variants, or round trips are not shown

These controls are capability-dependent. Select and enable a plugin that
supports the required feature. See [Routing plugins](/use/plugins/routing/) to
compare the currently documented providers.

### No profile is available for a category

Open the routing plugin under **Settings → Plugins** and review **Profiles by
category**. Assign an enabled native or custom profile to the selected category.
For a subcategory, remember that its own mapping overrides the broad category.

### A provider is unavailable or times out

Try the request again, shorten the route, request fewer variants, or select
another engine. For a self-hosted endpoint, the administrator must verify that
the Wanderer backend—not only your browser—can resolve and reach it. Container
networking and connector permissions are common causes. Provider-specific
guidance is available under [Routing plugins](/use/plugins/routing/).
