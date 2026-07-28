---
title: PocketBase collections are a public API surface — consumer analysis can't stop at Go
date: 2026-07-28
context: Phase 32 planning, caught after a wrong conclusion had already been locked into two decisions
---

## The miss

While designing Phase 32, I concluded that `buildRegion` was the **only** reader of region
polygon geometry, and recommended deleting the `region_polygons` collection outright on the
strength of it. That conclusion came from:

```bash
grep -rn "region_polygons" --include="*.go" db/
```

It was wrong. The admin region picker is an AlpineJS SPA embedded in
`db/routes/regions_ext/regions_ui.html`, and it reads geometry by calling
`/api/collections/region_polygons/records` directly with a superuser JWT — three separate
flows (coverage map, toggle-on draw, hover preview). None of them appear in any `.go` file.

The wrong conclusion survived long enough to be written into two locked decisions before a
verification pass caught it.

## Why it happened

**A PocketBase collection is not an internal data structure — it is a REST endpoint the
moment it is created.** Any client with a valid token can read it without a single line of
Go being written. That inverts the usual assumption behind "find the callers": in a normal
Go codebase, a package-private table has its readers in the same language. Here, the
collection *is* the API, and the most important consumer can live in an HTML file, a Dart
app, or a `curl` invocation in someone's runbook.

The habit that failed is scoping a consumer search by the language the *producer* is written
in. That is exactly backwards for anything reachable over HTTP.

## What to do instead

When assessing whether a PocketBase collection can be changed or removed, search for the
**collection name as a bare string**, unscoped by file type:

```bash
grep -rn "region_polygons" db/ web/ app/ --exclude-dir=node_modules
```

Consumers to expect beyond Go:

| Where | How it reads |
|---|---|
| `db/routes/*_ext/*.html` | Embedded admin SPAs calling `/api/collections/{name}/records` |
| `web/src/**` | SvelteKit server routes and stores via the PocketBase JS SDK |
| `app/lib/**` | Flutter via Dio against the same collection API |
| External | Anything using a `wanderer_key` API token — invisible to any grep |

That last row is the uncomfortable one: a published collection can have consumers that do
not exist in this repository at all. Removing or renaming one is a breaking API change, not
a refactor, even when nothing in the codebase references it.

## The cheap check

Before concluding "nothing reads X," ask what the *shape* of X is. If it is reachable over
HTTP — a PocketBase collection, a route, a published JSON artifact — a language-scoped grep
cannot answer the question, and a confident answer from one is worse than no answer, because
it gets written down and acted on.

Related: [[streamlined-region-definition]] for the region catalog design this nearly broke.
