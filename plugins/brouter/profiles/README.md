# BRouter profile presets

These `.brf` files are the upstream BRouter profiles that the curated routing
profiles are rendered from. They are embedded into the WASM module; the plugin
fills in the annotated parameters and uploads the result to the configured
BRouter service.

| File | Origin | Local changes |
| --- | --- | --- |
| `shortest.brf` | `abrensch/brouter` | none |
| `hiking-mountain.brf` | `abrensch/brouter` (Poutnik template) | `uphillcostvalue`, `downhillcostvalue`, `path_preference` annotated |
| `trekking.brf` | `abrensch/brouter` | none |
| `fastbike.brf` | `abrensch/brouter` | `unpavedPenalty` added, mirroring the parameter `trekking.brf` already has |
| `gravel.brf` | `abrensch/brouter` (quaelnix) | none |
| `mtb.brf` | `abrensch/brouter` (Poutnik template) | `hills`, `avoid_unsafe`, `MTB_factor`, `smallpaved_factor`, `path_preference` annotated |
| `car-vario.brf` | `abrensch/brouter` | none |

Annotating means adding a `# %name% | label | type` comment to an assignment
that upstream keeps internal. It changes no cost, only what Wanderer is allowed
to rewrite. Every parameter a template declares must be annotated: rendering
fails on a missing annotation instead of appending an assignment, which would
land in whatever context the file happens to end in.

`fastbike.brf` carries the only behavioural change. Upstream hardcodes its
unpaved surcharges, so `avoidBadSurfaces` would have no target there. The added
`unpavedPenalty` reproduces the upstream costs exactly at its default of `1.0`
and scales them from "surface does not matter" to five times the surcharge.

A parameter that no preference and no control supplies is left untouched, so an
unmodified profile is uploaded byte for byte as it sits here. The contract test
`TestRenderingWithoutValuesKeepsTheBaseVerbatim` pins that down.

## Which base serves which category

| Category | Subcategory | Base |
| --- | --- | --- |
| Walking, Running, Hiking | all | `hiking-mountain` |
| Biking | –, Touring | `trekking` |
| Biking | Road | `fastbike` |
| Biking | Gravel | `gravel` |
| Biking | MTB | `mtb` |
| Biking | E-Bike | `trekking` |

The categories differ only in the preferences the manifest's `categoryMappings`
carry, not in the file that serves them. `shortest` and `car-vario` are offered
as profiles but are not mapped to a category.

## Updating a profile

Refresh the file from `misc/profiles2/` in `abrensch/brouter`, re-apply the
local changes listed above, and run `go test ./...` in the plugin directory. The
contract test renders every template from its base and fails if a declared
parameter lost its annotation.
