---
title: Custom categories
description: How to configure trail categories and subcategories
---

<span class="-tracking-[0.075em]">wanderer</span> uses categories to classify what kind of activity a trail belongs to. 
Out of the box you get: Biking, Canoeing, Climbing, Hiking, Running, Skiing and Walking.
Some broad categories also have subcategories, for example Biking can be refined into MTB, Gravel, Road or E-Bike.
You can adapt this taxonomy to your needs in the PocketBase admin panel.

## Modifying categories

![Pocketbase Categories](../../../../assets/guides/pocketbase_categories.png)

In the PocketBase admin panel, click on the `categories` table in the list on the left side. 
All existing categories will be listed here. 
To edit one simply click on the row, edit the data you want to change, and click "Save". 
To delete a category check the box at the beginning of the row and click "Delete selected". 
To create a new category click the "New record" button in the top right corner, give your new category a name, optionally fill in display metadata such as `short_name`, `icon`, or localized `translations`, and click "Save".

The category `name` is the canonical, language-independent identity.
Use stable names such as `Hiking` or `Biking`; display labels in different languages should be stored in `translations`.
Incoming federated trails and integration imports match categories by a normalized version of `name`, so changing a category name can affect future matching.

### Category fields

| Field | Description |
| ----- | ----------- |
| `name` | Canonical category name. This is used for matching across imports and federation. |
| `short_name` | Optional compact label for space-constrained UI. |
| `icon` | Optional Font Awesome Free icon name without the `fa-` prefix, for example `person-hiking`. |
| `translations` | Optional localized display labels. |
| `settings` | Optional JSON settings for category-specific backend behavior. See [Category settings](#category-settings) and [`valhalla_profile`](#valhalla_profile). |

`translations` uses supported base locale codes such as `de`, `en`, `fr`, or `pt` as keys.
Do not use region-specific keys such as `de-CH` or `pt-BR`; the frontend resolves user locales to their base locale before looking up category translations.

Example:

```json
{
  "de": {
    "name": "Radfahren",
    "short_name": "RAD"
  },
  "en": {
    "name": "Biking",
    "short_name": "BIKE"
  }
}
```

## Modifying subcategories

Subcategories live in the `subcategories` table and act as optional refinements below a single parent category. Their names only need to be unique within that parent, so `Road` can exist under both Biking and Running at the same time.

![Pocketbase Subcategories](../../../../assets/guides/pocketbase_subcategories.png)

To add one, create a new record in `subcategories`, choose its parent `category`, set a canonical `name`, and optionally add display metadata.

### Subcategory fields

| Field | Description |
| ----- | ----------- |
| `category` | Required parent category. |
| `name` | Canonical subcategory name, unique within the parent category after normalization. |
| `short_name` | Compact label shown in icon-based filters, for example `MTB`, `GRVL`, or `ROAD`. |
| `icon` | Optional Font Awesome Free icon name. If empty, the parent category icon is used. |
| `badge_icon` | Optional Font Awesome Free overlay icon, for example `snowflake`, `mountain`, `bolt`, or `cross`. |
| `translations` | Optional localized display labels, using the same structure as category translations. |
| `settings` | Optional JSON settings for subcategory-specific backend behavior. Currently used for [`valhalla_profile`](#valhalla_profile). |

Most subcategories should reuse the parent category's icon and rely on `short_name` — plus a `badge_icon` where it helps — to set themselves apart, rather than each carrying a distinct full icon. You can browse available icon names at [fontawesome.com](https://fontawesome.com/search?ic=free-collection).

:::note
Unknown remote categories and subcategories are not automatically created during federation.
Raw remote values are stored on the trail and can be matched later when an admin creates a compatible local category or subcategory.
:::

## Migrating old custom categories

If your instance already had custom categories such as `MTB` or `Gravel` that now overlap with a default subcategory, you can reassign the affected trails in bulk from the web UI. See [Categories](/use/categories/#editing-several-trails-at-once) for the step-by-step migration path.

## Category settings

Categories can optionally define additional settings in the `settings` JSON field.
This field may be left empty.
When no settings are configured, <span class="-tracking-[0.075em]">wanderer</span> uses the built-in defaults.

Currently, the following waypoint-merge settings are supported:

```json
{
  "wp_merge_enabled": true,
  "wp_merge_radius": 50
}
```

`wp_merge_enabled` controls whether geotagged photos are grouped into waypoint clusters.
Set it to `false` to create one waypoint per photo.

`wp_merge_radius` controls how close geotagged photos have to be to each other, in meters, before they are grouped into the same waypoint when adding waypoint photos to a trail.
Set it to `0` to only merge photos with the exact same coordinates, or increase the value to merge photos across a wider area.

### `valhalla_profile`

`valhalla_profile` is an optional string key available on **both** `categories.settings` and `subcategories.settings`.
It tells <span class="-tracking-[0.075em]">wanderer</span> which Valhalla costing model to use when routing, navigating, or downloading a trail in that category.

#### Accepted values

The vocabulary is **open** — it is not a fixed list, and it is never validated against an allowlist.
Any Valhalla costing model name is accepted:

```
valhalla_profile := <costing>                  e.g. "pedestrian", "auto", "truck", "motor_scooter"
                  | "bicycle_" <bicycle_type>  e.g. "bicycle_mountain"
```

- A **bare costing name** is passed through to Valhalla verbatim as its `costing` value.
- The **only** special form is `bicycle_<type>`, where `<type>` is one of Valhalla's four `bicycle_type` values (`road`, `hybrid`, `cross`, `mountain`).
  It maps to the `bicycle` costing with that bike type — for example `bicycle_mountain` becomes `costing: "bicycle"` with `bicycle_type: "Mountain"`.

Because the vocabulary is open, you can adopt a new Valhalla costing model as soon as upstream ships it, without waiting for a <span class="-tracking-[0.075em]">wanderer</span> release.
See Valhalla's own [costing models documentation](https://valhalla.github.io/valhalla/api/turn-by-turn/api-reference/#costing-models) for the full list of names.

#### Example: adding a `Car` category

Create a new category named `Car` and set its `settings` to:

```json
{
  "valhalla_profile": "auto"
}
```

Trails in that category are now routed, navigated, and downloaded using Valhalla's `auto` costing — no code change and no allowlist edit required.

#### Shipped defaults

| (Sub)category | `valhalla_profile` |
| ------------- | ------------------ |
| Category `Hiking` | `pedestrian` |
| Subcategory `Biking / Touring` | `bicycle_hybrid` |
| Subcategory `Biking / MTB` | `bicycle_mountain` |
| Subcategory `Biking / Gravel` | `bicycle_cross` |
| Subcategory `Biking / Road` | `bicycle_road` |
| Subcategory `Biking / E-Bike` | `bicycle_hybrid` |

`Biking / E-Bike` maps to `bicycle_hybrid` because Valhalla has no dedicated e-bike costing model — Hybrid is its closest general-purpose bike profile.
Every other default category and subcategory ships with **no** value, meaning it has no Valhalla costing mapping of its own.
Existing instances receive these defaults retroactively on the next server start, and an operator-set value is never overwritten.

#### Resolution order

For a given trail, the profile is resolved in this order:

1. The selected **subcategory's** `valhalla_profile`, if that subcategory still exists and is not hidden by the user's visibility preferences.
2. The parent **category's** `valhalla_profile`.
3. `bicycle_hybrid`, when the category is biking-shaped (at least one of its subcategories maps to a `bicycle` costing) but nothing above resolved.
   This step is bicycle-specific: Valhalla's four bike variants share one `bicycle` costing and need a sensible `bicycle_type` default.
4. `pedestrian`, when nothing is configured.

An unset or unrecognized value simply falls through this chain rather than erroring, so a typo degrades to walking rather than breaking routing.

#### Route Planner picker

The Route Planner's travel-profile picker still offers only its five built-in options (Hiking and the four bike variants).
A category mapped to a costing outside those five — such as `auto` — is fully honored for routing, navigation, and offline download, but it does not appear as a planner preset, and the picker highlights nothing for it.
