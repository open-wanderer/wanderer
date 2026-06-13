---
title: Categories
description: How to use trail categories, subcategories, and category visibility settings
---

Every trail has a category that describes its broad activity type — Hiking, Biking, Running, Skiing, and so on. Many categories can be narrowed down further with a subcategory, such as Biking / Gravel or Hiking / Snowshoeing, whenever you want to be more specific.

## Choosing a category

When you create or edit a trail, pick the activity type with the **Category** selector in the trail form. It lists the broad categories together with their subcategories, so you can stay general or get specific:

- Choose **Hiking** for an ordinary hiking trail.
- Choose **Hiking / Snowshoeing** to mark it as a snowshoe route.
- Choose **Biking / Gravel** for a gravel ride.

A broad category on its own is always enough; a subcategory is optional. On trail cards and in lists, the category icon carries a small badge for subcategories that need one — for example a snowflake for winter variants — so you can tell refinements apart at a glance.

## Filtering trails

The filter panel shows each category as an icon. Click an icon to add that category to the filter.

![Trail category filter with the subcategory overlay open](../../../assets/guides/wanderer_trails_category_filter.png)

Categories that have subcategories reveal a subcategory overlay when you hover or focus the icon; on touch devices, long-press it instead. From there you can filter by:

- the whole category,
- only trails that have no subcategory, or
- one or more specific subcategories.

When a subcategory filter is active, a small indicator appears on the category icon — that's how you tell "all Biking trails" apart from "only the Biking subcategories I picked".

## Editing several trails at once

To reclassify many trails in one go, select them in the trail list, open the action menu, and choose **Adjust**. The modal lets you set a new category, subcategory, or difficulty for the whole selection.

![The bulk edit modal for adjusting selected trails](../../../assets/guides/wanderer_trails_adjust.png)

This is handy after upgrading an instance, when an older standalone category overlaps with a new subcategory: every trail previously filed under `Gravel`, for instance, can be moved to Biking / Gravel in a single step.

## Category preferences

Open **Settings → Categories** to control how categories behave for your account.

![The category preferences page in Settings](../../../assets/guides/wanderer_settings_categories.png)

Each category has two toggles:

- **Show** controls whether a category is part of your exploration and planning. While it is on, the category appears in search and discovery and is offered in the category picker when you create or edit a trail. Turn it off to hide the category from those places entirely — which also hides its federated trails.
- **Remote trails** controls federated content. While it is on, trails in that category from other instances appear alongside your own. Turn it off to hide those remote trails while keeping your own trails in the category visible.

Beyond the toggles, you can **reorder** categories by dragging them — the order carries over to the picker and filters — and **hide individual subcategories** by clicking a subcategory label below its parent; hidden subcategories appear muted and drop out of your pickers and filters. Subcategories otherwise inherit their parent category's settings.

These settings are personal. They never delete categories, change other users' settings, or remove category assignments that already exist on trails.

:::note
Categories and subcategories themselves are defined by the instance administrator in PocketBase. As a regular user you choose from the available taxonomy and set your own visibility preferences, but you cannot create global categories from the web UI.
:::
