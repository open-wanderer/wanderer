# Geo benchmark

`geobench` compares four ways of answering “which trails intersect this
radius?” without touching Wanderer's running Meilisearch instance or its
`trails` index.

| Backend | Indexed representation | Query result |
|---|---|---|
| `point` | Wanderer's current `_geo` start point | Meilisearch result |
| `geojson` | GeoJSON `LineString`/`MultiLineString` | ADR-accepted approximate Direct product path plus separately measured candidate/exact variants |
| `h3` | Route cells at H3 resolutions 5, 6, 7 and 8 | H3 candidates, then SQLite load/decode and exact verification |
| `rtree` | One SQLite RTree box per line segment | RTree candidates, then geometry-BLOB load/decode and exact verification |

Every Meilisearch strategy stores the same compact encoded `polyline` once,
using the same encoder as Wanderer's production projection. The `_geojson` or
H3 cell fields are additional strategy-specific load, so comparisons do not
give the point or H3 variants a different common route payload. Absolute
Meilisearch build time, peak RAM and disk figures -- including NAS results --
are therefore substantially closer to the production index footprint than a
raw-coordinate benchmark would be. They remain workload estimates rather than
exact forecasts: the synthetic trails are supplied as already prepared
geometry instead of passing through Wanderer's real GPX import. The optional
`--geojson-simplify-meters` projection lets the indexed GeoJSON reproduce
Wanderer's 50 m Ramer-Douglas-Peucker step while the returned polyline and all
accuracy oracles continue to use the original geometry. After simplification,
`--geojson-max-segment-meters` (default `5000`) inserts points along the
spherical great-circle path and splits antimeridian crossings into separate
RFC 7946-compatible LineStrings. This normalization affects `_geojson` only;
it does not alter the compact product polyline or the reference geometry.

All Meilisearch strategies also index the same product-shaped text, ACL,
taxonomy, numeric and sortable fields with Wanderer-like settings. Their full
index time and disk/RAM figures therefore include the non-spatial search index
needed by the measured product queries.

The exact reference is the minimum spherical point-to-polyline distance. Parts
of a `MultiLineString` stay separate, so no imaginary segment crosses a gap.

## Quick start

Running from source requires a Linux host, a local Unix-socket Docker engine,
Go 1.25 or newer, and a C compiler (`h3-go` uses CGO).

```bash
# Fast development run: 500 trails, 24 query points, Meili 1.36, 1.44 and 1.53.1
make geo-bench

# Regular comparison: 5,000 trails and 36 query points
make geo-bench GEO_BENCH_ARGS='--profile standard --out /tmp/wanderer-geo-standard'

# Deliberately hard run: 25,000 trails and 60 query points
make geo-bench GEO_BENCH_ARGS='--profile stress --out /tmp/wanderer-geo-stress'

# Optional DS923+ stress shape: 50,000 trails, 5,040 warm product samples,
# four clients, one pinned Meilisearch version and one backend at a time.
make geo-bench GEO_BENCH_ARGS='\
  --profile nas --backends geojson \
  --geojson-simplify-meters 50 \
  --geojson-max-segment-meters 5000 \
  --batch-size 50000 --batch-bytes-mib 90 \
  --meili-images getmeili/meilisearch:v1.53.1 \
  --backend-timeout 0 --fail-on-incorrect --fail-on-ux \
  --out /tmp/wanderer-geo-ds923plus'

# Exact H3 comparison on the same NAS and dataset/profile.
make geo-bench GEO_BENCH_ARGS='\
  --profile nas --backends h3 \
  --meili-images getmeili/meilisearch:v1.53.1 \
  --backend-timeout 0 --fail-on-incorrect \
  --out /tmp/wanderer-h3-ds923plus'

# Lock down LineString/MultiLineString behavior and both additive GeoJSON diagnostics
make geo-bench-contract
```

For a Linux host such as DSM that has Docker but no Go toolchain, build the
native binary on a compatible Linux machine and copy it to the target. The
build target uses CGO and does not cross-compile; for a DS923+ it must therefore
run on Linux/amd64 with a libc compatible with DSM.

```bash
make geo-bench-build GEO_BENCH_BINARY=/tmp/wanderer-geobench
sha256sum /tmp/wanderer-geobench

# After copying it to the NAS:
chmod +x ./wanderer-geobench
./wanderer-geobench --help
```

The executable is intentionally not committed. Every report records the
actual executable SHA-256 and normalized command-line arguments, so the copied
binary can be tied to its run independently of the source-tree digest.

The default report is written to a new `wanderer-geobench-results-*` directory
under `/tmp`. `--out` accepts either a JSON filename or a directory. Every run
writes both `report.json` (complete raw measurements) and `report.md` (tables).

Meilisearch images are pulled if necessary. Each Meilisearch backend gets a
fresh container, port, index, and data directory. Every run gets a unique work
directory. It is removed afterward unless `--keep-work-dir` is set;
`--work-dir` selects the parent directory for that unique run. It never
connects to port 7700 unless Docker happens to assign that isolated port, and
it never uses the index name `trails`.

Each backend/version cell has its own 15-minute budget by default. If one cell
reaches it, the report records `timed_out` and its elapsed time as a lower bound,
then continues with a fresh context for the remaining cells. This matters for
the dense 25,000-trail stress profile, where GeoJSON indexing can remain busy
for a long time. Use `--backend-timeout 0` to wait without a per-cell limit.
`--timeout` is a separate optional overall limit and defaults to disabled.
The harness still runs the remaining cells and writes the partial report, but
exits non-zero after any timeout so automation cannot mistake it for a complete
comparison.

## Useful runs

```bash
# Demonstrate the current version mismatch. GeoJSON is reported unsupported
# on 1.11.3; point and H3 still run.
make geo-bench GEO_BENCH_ARGS='\
  --meili-images getmeili/meilisearch:v1.11.3,getmeili/meilisearch:v1.36.0,getmeili/meilisearch:v1.44.0,getmeili/meilisearch:v1.53.1 \
  --out /tmp/wanderer-geo-version-matrix'

# Repeat exactly the same generated data later.
make geo-bench GEO_BENCH_ARGS='--dataset-out /tmp/wanderer-dataset.json --out /tmp/run-a'
make geo-bench GEO_BENCH_ARGS='--dataset /tmp/wanderer-dataset.json --out /tmp/run-b'

# Change H3's storage/query tradeoff explicitly.
make geo-bench GEO_BENCH_ARGS='--h3-resolutions 4,6,8,10 --out /tmp/h3-profile-b'

# Change GeoJSON's query-time search space. These 29 variants share one index
# and stay below the hard maximum of 32.
make geo-bench GEO_BENCH_ARGS='\
  --backends geojson --geojson-resolutions 16,32,64,100,125,250,500 \
  --geojson-extra-padding-meters 0,0.5,1 --geojson-finalists 4 \
  --out /tmp/geojson-parameters'

# Reproduce the legacy untuned behavior. Disable the new direct UX workload too.
make geo-bench GEO_BENCH_ARGS='--backends geojson --geojson-tune=false --geojson-direct=false --out /tmp/geojson-legacy'

# Evaluate direct GeoJSON resolutions against a deliberately fuzzy UX contract.
make geo-bench GEO_BENCH_ARGS='\
  --backends geojson --geojson-direct \
  --geojson-direct-resolutions 100,125,250 \
  --geojson-ux-boundary-meters 50 \
  --geojson-ux-min-recall 0.99 --geojson-ux-min-precision 0.99 \
  --geojson-ux-min-top10-recall 0.99 --geojson-ux-min-page-recall 0.99 \
  --geojson-ux-min-page-precision 0.99 \
  --fail-on-ux --out /tmp/geojson-direct-ux'

# Run only the fast engine/geometry measurements, or only the product pipeline.
make geo-bench GEO_BENCH_ARGS='--query-workload engine --out /tmp/geo-engine'
make geo-bench GEO_BENCH_ARGS='--query-workload product --query-clients 12 --out /tmp/geo-product'

# Turn any completed backend's accuracy mismatch into a non-zero exit.
make geo-bench GEO_BENCH_ARGS='--backends geojson,h3,rtree --fail-on-incorrect --out /tmp/geo-gate'

# Keep the benchmark small while debugging one backend.
make geo-bench GEO_BENCH_ARGS='\
  --trails 100 --queries 12 --repetitions 1 \
  --backends geojson --meili-images getmeili/meilisearch:v1.44.0'

# Isolate only the full-index scale curve. No oracle, queries, application
# database, or mutations run, and the composite DS923+ gate is disabled.
make geo-bench GEO_BENCH_ARGS='\
  --profile standard --trails 20000 --index-only \
  --backends geojson --geojson-simplify-meters 50 \
  --batch-size 50000 --batch-bytes-mib 90 \
  --meili-images getmeili/meilisearch:v1.53.1 \
  --backend-timeout 15m --timeout 17m \
  --out /tmp/wanderer-geojson-index-20k'

# Compare stable hash-sharded GeoJSON builds. Run each cell separately so it
# gets a fresh Meilisearch container; reuse one dataset file/digest for a fair
# 1/2/4/8 comparison. Sharding is intentionally index-only until federated query,
# count, sorting and pagination semantics have their own contract tests.
make geo-bench GEO_BENCH_ARGS='--profile standard --trails 20000 --queries 36 --dataset-out /tmp/geo-20k.json --index-only --backends geojson --geojson-shards 1 --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-shards-1'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shards 2 --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-shards-2'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shards 4 --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-shards-4'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shards 8 --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-shards-8'

# Compare append-oriented capacity sharding on the same immutable corpus. The
# number of indexes is derived as ceil(trails/capacity): 20, 8 and 4 for this 20k
# dataset. Run mixed and clustered arrivals separately: mixed is a stable
# seed+ID permutation, while clustered preserves dataset/scenario order.
# `geojson-shards` stays at its default because it applies only to hash mode.
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shard-mode capacity-created --geojson-shard-max-trails 1000 --geojson-capacity-arrival mixed --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-capacity-1000-mixed'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shard-mode capacity-created --geojson-shard-max-trails 1000 --geojson-capacity-arrival clustered --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-capacity-1000-clustered'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shard-mode capacity-created --geojson-shard-max-trails 2500 --geojson-capacity-arrival mixed --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-capacity-2500-mixed'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shard-mode capacity-created --geojson-shard-max-trails 2500 --geojson-capacity-arrival clustered --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-capacity-2500-clustered'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shard-mode capacity-created --geojson-shard-max-trails 5000 --geojson-capacity-arrival mixed --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-capacity-5000-mixed'
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-20k.json --index-only --backends geojson --geojson-shard-mode capacity-created --geojson-shard-max-trails 5000 --geojson-capacity-arrival clustered --geojson-simplify-meters 50 --meili-images getmeili/meilisearch:v1.53.1 --out /tmp/geo-capacity-5000-clustered'

# Short search diagnostic across all 50 capacity shards. This explicit
# r100/product-only measurement rebuilds the same shards but deliberately skips
# the safe-hybrid sweep and mutations. Its 120 warm samples make p95 useful;
# p99 remains diagnostic and this does not exercise the optional long NAS target.
make geo-bench GEO_BENCH_ARGS='--profile standard --dataset /tmp/geo-50k.json --backends geojson --query-workload product --repetitions 2 --query-clients 4 --geojson-shard-mode capacity-created --geojson-shard-max-trails 1000 --geojson-capacity-arrival clustered --geojson-simplify-meters 50 --geojson-max-segment-meters 5000 --geojson-resolutions 100 --geojson-extra-padding-meters 0 --geojson-finalists 1 --geojson-direct-resolutions 100 --geojson-ux-min-recall 0.99 --geojson-ux-min-precision 0.99 --geojson-ux-min-top10-recall 0.99 --geojson-ux-min-page-recall 0.99 --geojson-ux-min-page-precision 0.99 --batch-size 50000 --batch-bytes-mib 90 --meili-images getmeili/meilisearch:v1.53.1 --performance-target none --backend-timeout 25m --timeout 30m --fail-on-incorrect --fail-on-ux --out /tmp/geo-capacity-r100-search-short'

# Optional long DS923+ stress run, normally only after the short diagnostic.
make geo-bench GEO_BENCH_ARGS='--profile nas --dataset /tmp/geo-50k.json --backends geojson --query-workload product --geojson-shard-mode capacity-created --geojson-shard-max-trails 1000 --geojson-capacity-arrival clustered --geojson-simplify-meters 50 --geojson-max-segment-meters 5000 --geojson-resolutions 100 --geojson-extra-padding-meters 0 --geojson-finalists 1 --geojson-direct-resolutions 100 --geojson-ux-min-recall 0.99 --geojson-ux-min-precision 0.99 --geojson-ux-min-top10-recall 0.99 --geojson-ux-min-page-recall 0.99 --geojson-ux-min-page-precision 0.99 --batch-size 50000 --batch-bytes-mib 90 --meili-images getmeili/meilisearch:v1.53.1 --fail-on-incorrect --fail-on-ux --out /tmp/geo-capacity-r100-search-stress-long'

# Let the 25k GeoJSON cells run to completion, even if they take hours.
make geo-bench GEO_BENCH_ARGS='\
  --profile stress --backends geojson --backend-timeout 0 \
  --out /tmp/wanderer-geojson-uncapped'
```

Run `cd db/cmd/geobench && go run . --help` for all flags. The benchmark has a
small separate Go module so its CGO-only H3 dependency does not affect
Wanderer's production build or normal database tests.

### Capacity-created sharding contract

`--geojson-shard-mode capacity-created` models the initial assignment of an
existing local catalog. The planner creates a separate monotone local-arrival
timeline and fills consecutive indexes up to
`--geojson-shard-max-trails`. `--geojson-capacity-arrival mixed` deterministically
permutes trails from the dataset seed and trail ID, so spatial scenarios are
spread across shards. `clustered` preserves dataset order, which keeps the
generator's dense Alpine, background, and long-route blocks together and
exposes skew. Equal planner timestamps are resolved by `id` ascending.

The planner is intentionally independent of the indexed product field
`created`: that pre-existing fixture value cycles with `index % 1825` and is
useful for query sorting but is not a local append history. Fields such as an
ActivityPub publication date, `origin_created`, or any other federated
timestamp are likewise never read for assignment. The JSON and Markdown
reports record mode, capacity, arrival model, actual
`ceil(trails/capacity)` shard count, assignment SHA-256, algorithm, and
per-shard document and geometry distribution. Mixed and clustered results are
separate benchmark cells and must not be pooled.

In production, sorting and repacking the remaining database records after a
delete would move later trails between shards. Wanderer therefore needs to
persist the `geo_shard_id` chosen when a trail is first stored locally, plus a
shard manifest with immutable boundaries/state. Updates and federation syncs
must preserve both the local creation timestamp and `geo_shard_id`; an origin
timestamp must never replace them. Only the current open shard accepts new
local trails, and a full shard causes creation of the next one.

Without `--index-only`, capacity mode is a deliberately narrow search gate:
it requires `--query-workload product`, exactly one direct resolution `N`,
at least two derived shards, and exactly one official stable
`getmeili/meilisearch` release at or above 1.53.1. ADR 0002's accepted release
evidence remains pinned to 1.53.1; a newer version is only a qualification
candidate and must pass the same contract, federation and UX gates. Every logical
product search sends one HTTP request to `/multi-search`, with one identical
ACL/text/metadata/`_geoRadius(..., N)` subquery per shard. Top-level
`federation.page` and `hitsPerPage` make Meilisearch return one globally sorted
page plus exhaustive `totalHits` and `totalPages`; `id:asc` is the final stable
tie-breaker. The benchmark strips response-only `_federation` metadata before
materializing the same product payload as the single-index direct path. The
report records shard fan-out, subqueries, HTTP requests, request JSON bytes,
Meilisearch response JSON bytes and final product bytes (HTTP headers are not
counted). The 1.53.1 GeoJSON preflight also creates two miniature indexes and
contracts this exhaustive global page/count/sort/geo-filter wire behavior, so
a future Meilisearch upgrade cannot silently invalidate the raw adapter.

Before the product cases, an untimed spatial audit queries every dataset point
at all four required radii (0.5, 5, 25 and 100 km), requests complete IDs and
counts across every shard, and compares them with the exact spherical
point-to-polyline oracle. The fixed rN plan qualifies only when both this
complete matrix and the ACL/text/metadata/global-page product audit pass. Warm
latency and client/Meilisearch RSS are then measured separately around only the
concurrent product-page workload; the exhaustive correctness data structures
are excluded from that RSS delta.

That run is explicitly search-only. It skips the single-index parameter sweep,
safe candidate-plus-SQLite refinement, incremental updates, additions and
deletes, and records this scope in both report formats. It therefore measures
the 50-shard query question without pretending that append/delete routing or a
persisted production shard manifest has already been solved.

## Dataset

The generator is deterministic for a given seed and creates:

- densely overlapping routes in the Alps;
- geographically distributed background routes;
- very long outliers, including antimeridian crossings;
- approximately 10% true `MultiLineString` routes with a real gap.

Query points cover every scenario and the required 0.5, 5, 25, and 100 km
radii. Long-outlier queries prefer points far from the start, which makes the
limitations of the current `_geo` baseline visible. Coordinates represent the
source geometry supplied to the benchmark. GPX parsing remains outside the
measured phase. When `--geojson-simplify-meters` is non-zero, simplification is
included in `prepare_ms`. Great-circle densification and RFC 7946 antimeridian
cutting are included in the same phase. The report records source,
pre-normalization and indexed vertices/segments as well as inserted
densification vertices and antimeridian cuts, so their build cost is visible.

A custom dataset uses the JSON form of these structures:

```json
{
  "seed": 42,
  "trails": [{
    "id": "trail-1",
    "scenario": "real",
    "parts": [[{"lat": 46.8, "lon": 8.2}, {"lat": 46.81, "lon": 8.22}]]
  }],
  "query_points": [{
    "id": "query-1",
    "scenario": "real",
    "point": {"lat": 46.805, "lon": 8.21}
  }]
}
```

## Measurements

### Reproducibility metadata

Every JSON and Markdown report identifies the inputs around the measurements:

- the normalized invocation and SHA-256 of the running executable;
- the Git commit and whether the repository had tracked or untracked changes
  when the harness started;
- a `sha256:` harness digest over sorted, length-framed non-test Go/CGO source
  files plus `go.mod` and `go.sum` (documentation and tests are excluded);
- a `sha256:` digest over the dataset's compact JSON representation after it
  has been loaded or generated and validated;
- each configured Meilisearch reference, its local Docker image ID, and all
  available immutable repository digests. Images are inspected after the
  cells run, when Docker has pulled every image that started successfully;
- Go/compiler/CGO details, architecture, logical CPUs, `GOMAXPROCS`, and,
  where Linux exposes them, the kernel, OS release, CPU model, physical-core
  count, total RAM, Docker version, and Docker storage driver.

Metadata discovery is best effort and never changes a benchmark result. A
missing executable/harness/dataset digest is recorded in
`reproducibility.warnings`; an
image that could not be inspected, or has no repository digest, keeps its
reason in that image's `error` field. `--contract-only` intentionally has no
dataset digest. Hostnames and other machine identifiers are not collected.

For full indexing and separate update, add, and delete phases, the report records:

- total backend elapsed time, including isolated container startup, index
  configuration, indexing, queries, update verification, and cleanup;
- `prepare_ms`: client-side GeoJSON/H3/RTree preparation;
- `submit_ms`: time until all batches are accepted; with a non-zero
  `--max-pending-batches`, this includes intermediate barriers;
- `request_ms`, submitted documents/bytes, actual Meilisearch task count and
  completion-barrier count;
- `ready_ms`: wall time until every asynchronous Meilisearch task has
  succeeded (or the synchronous RTree transaction has committed);
- peak Meilisearch container working set and client-process RSS, sampled every
  25 ms, with baseline and delta;
- Meilisearch database/used bytes and the sum of logical data-directory file
  sizes, or the SQLite database plus any journal files.

Before each backend/version run and again before its incremental phase, the
harness forces a Go GC and returns free heap pages to the operating system.
Client RSS remains an in-process measurement, so the raw baseline and absolute
peak are retained alongside the delta in JSON.

The Markdown table also reports `prepare_ms + ready_ms` as the complete phase
time. Mutation phases use roughly 1% of the corpus. Updates change geometry;
adds sample the complete corpus (including its rare tail), get collision-free
IDs and a small deterministic displacement; deletes remove that same add batch
so every backend returns to its original document count. Meilisearch request
batches obey both `--batch-size` and the configurable `--batch-bytes-mib`
safety ceiling. By default the harness submits the complete phase and lets
Meilisearch autobatch it, then polls only the final FIFO task as a queue
barrier. `--max-pending-batches N` inserts a barrier after every N actual
requests for controlled pacing experiments. Every earlier task in each window
is read once to verify that all succeeded. The latest Meilisearch batch
strategy, progress steps, `progressTrace`, duration, and write-channel
congestion are retained for diagnosing slow indexing.
After the timed phase, the harness reads back the first and last changed trail
(including the rare long-outlier end of the dataset) and confirms that a
spatial query finds it at its new start.

`used_disk_bytes` is Meilisearch's used-database statistic or SQLite's
`page_size × (page_count - freelist_count)`. Use `filesystem_bytes` for the
closest cross-backend full-footprint comparison.

Search results contain the first sample plus subsequent-sample p50/p95/p99,
candidate and exact verification time, candidate counts, precision, recall,
and candidate recall per radius and scenario. “First sample” deliberately does
not claim a fully cold operating-system or engine cache. Trails within 0.5 m
(or `1e-7 × radius`) of the exact boundary are excluded from accuracy totals to
avoid turning harmless rounding differences into the main result.

GeoJSON is the deliberate exception to that last scoring shortcut. By default,
the harness screens the undocumented three-argument default plus explicit
polygon resolutions 3, 8, 16, 32, 64, 100, 125, 250, 500 and 1,000. At every resolution it
tests the requested radius itself for characterization and conservative
candidate radii of

```text
requested_radius / cos(pi / resolution)
  + extra_padding
  + index_simplification_tolerance
```

with 0 and 0.5 m extra padding. The final term is zero for an unsimplified
index. For a simplified candidate-plus-exact path it covers the configured
projection tolerance; a direct exact-radius plan receives no such expansion, so its UX audit
still measures the real final result. `_geoRadius` approximates a circle with
an inscribed polygon; the expansion is intended to circumscribe the target circle
in the corresponding planar model. It remains a candidate strategy verified by
contracts and the exact oracle, not a general spherical proof.
The behavior comes from
[Cellulite's circle reader](https://github.com/meilisearch/cellulite/blob/v0.3.2/src/reader.rs#L139-L176);
Meilisearch 1.53.1 currently validates the optional resolution as 3..1,000 in
[its filter implementation](https://github.com/meilisearch/meilisearch/blob/v1.53.1/crates/milli/src/search/facet/filter/index_filter.rs#L481-L527).

Every variant is queried once for every query point and radius, in rotated order
on the same already-built index. The complete numerical boundary zone is
required in the candidate set. A variant with one missing candidate or a
truncated response cannot become a finalist. Direct-radius variants remain
visible as characterization data but are never selected by this
candidate-plus-exact screen. Up to `--geojson-finalists` safe screen winners
then run through the full SQLite/exact engine path and the product workload.
The selected comparison plan is the fully correct finalist with the lowest
measured end-to-end p95 (then p99, candidate fan-out and resolution). This is
an optimum among the measured finalists on that corpus, host and Meilisearch
version, not a universal constant.
Safe-sweep selections with fewer than 5,000 full-workload samples retain the
historical `provisional` tuning label. That label describes the statistical
depth of this optional comparison; it is not Wanderer's release status.
Index and mutation costs are reported once because resolution and radius padding
are query-time parameters.

### Direct GeoJSON UX contract

`--geojson-direct` is enabled by default when GeoJSON tuning is enabled. It
measures the approximate Direct product path alongside the candidate-plus-exact
comparison; it does not change that comparison's correctness status. ADR 0002
fixes the released profile at r100. The generic diagnostic screen also
considers resolutions from `--geojson-direct-resolutions` (default
`100,125,250`); a result for another resolution does not change the released
profile without a new contract decision. Every direct resolution must also
occur in `--geojson-resolutions`, because both evaluations reuse the same rotated
parameter screen rather than issuing a second set of spatial queries.

The default UX policy is configurable:

- `--geojson-ux-boundary-meters 50` creates a neutral band 50 m inside and
  outside the requested radius;
- `--geojson-ux-min-recall 0.99` requires at least 99% aggregate raw recall,
  clear-interior recall, nearest-result recall and non-empty-request success;
- `--geojson-ux-min-precision 0.99` requires at least 99% aggregate material
  precision outside the neutral band;
- `--geojson-ux-min-top10-recall 0.99` requires at least 99% aggregate coverage of
  the ten nearest clear-interior reference trails;
- `--geojson-ux-min-page-recall 0.99` requires at least 99% aggregate recall across
  comparable product pages;
- `--geojson-ux-min-page-precision 0.99` requires at least 99% aggregate material
  precision on those pages;
- `--fail-on-ux` turns a missing qualifying direct plan or a failed direct
  product workload into a final non-zero exit. It remains independent of
  `--fail-on-incorrect`, which gates the exact backend contract.

The 50 m band reflects the scale of Wanderer's already simplified index
geometry and prevents a route a few metres either side of an arbitrary radius
edge from deciding the architecture. It is not discarded from the report:
raw recall, boundary misses, allowed boundary false positives and count error
still include it. Outside the band, material misses and false positives count
against the configured aggregate rates (99% by default). A request with clear
expected hits but no result counts against a separate non-empty-request success
rate governed by the same recall threshold. Semantic/ACL false positives and
truncated responses remain unconditional hard failures. Worst-request recall,
worst-page recall/precision, false-empty count and maximum miss/hit depth remain
diagnostics; one poor case does not override an aggregate pass by itself.
For that reason each retained material violation now includes the bounded worst
trail IDs, exact oracle distances and radius deltas. The report must be read
alongside the aggregate status: a passing 99% gate can still contain a rare,
deep miss. A current upstream report describes related silently missing
`_geojson` cells; the harness does not assume it has the same root cause and
keeps its independent oracle authoritative:
<https://github.com/meilisearch/meilisearch/issues/6575>.

Every spatially qualified direct plan is first audited with a product-shaped
Meilisearch request. For each unique product case, the harness prepares one
shared, backend-independent reference with `RunProductQueryExhaustive` over
every synthetic application document. That oracle independently applies the
public/owner/share ACL, deterministic AND-text semantics, federation, taxonomy
and numeric filters, exact point-to-polyline distance, total count, stable sort
and pagination. Its complete ordering and requested page are cross-checked for
internal consistency. The reference is built once outside timed requests and
reused by all direct resolutions; no Meilisearch result defines its expected
hits or order.

Each resolution returns both a complete geo-filtered ID set for the accuracy
audit and the real filtered, sorted and paginated product page. For every
supported sort, the complete returned subset is independently put into canonical
`ProductQuery` order; the Meilisearch full ordering must agree, and the page IDs,
`totalHits`, `totalPages`, page number and page size must match the corresponding
slice and metadata exactly. This consistency check is separate from the
approximate UX page recall/precision: those metrics conservatively compare the
page with the exhaustive radius oracle while treating only the configured
boundary band (50 m by default) as neutral. Raw recall and count error still
include that band.

The count audit keeps the same two levels separate. First, `totalHits` and
`totalPages` must match the independently enumerated Direct result set without
tolerance. Second, the Direct total is compared with the exact raw-geometry
oracle using `abs(got - oracle) / max(1, oracle)`. Its relative p95 must be at
most 1%; this is the normative ADR 0002 count gate. Exact-count rate, absolute
error, per-case deltas and count false-empty cases remain visible diagnostics;
they are not additional hard count gates. Future facet or histogram groups are
not synthesized by this workload and must receive their own contract coverage
when they become part of the product request.

Only plans that pass the shared reference audit remain eligible. Their paginated
audit requests provide a short product-page calibration, but calibration alone
does not choose the architecture: every plan whose audit completed, including
an accuracy failure, receives the complete repeated/concurrent workload with
its own clients, latency distribution, throughput and stability result. This
keeps performance evidence available without allowing an inaccurate plan to be
selected. With an active NAS target, each eligible plan is also gated
independently. The winner is selected only among spatial-accuracy- and
relative-count-p95-passing, warm-stable and, when required, SLO-passing plans
by warm p95/p99; calibration and the spatial ranking are tie-breakers.

Because every eligible direct resolution now runs the full warm workload, an
optional NAS comparison with three resolutions spends roughly three query-workload
budgets on the direct path (the shared index build is not repeated). For a
short diagnostic run, narrow `--geojson-direct-resolutions`; for a broad
comparison, keep every resolution that could realistically be selected.

Each direct plan's returned documents are treated as the final answer: its
timed request neither fetches a candidate superset from the application SQLite
fixture nor runs exact point-to-polyline refinement afterward. Direct and
hybrid paths materialize and serialize Wanderer's same 32-field default trail
list projection inside their measured path. The compact polyline remains an
indexed field but is not part of that default list response; final response
bytes remain separate from candidate-ID and SQLite bytes. This makes `direct
total` comparable with `candidate + SQLite load/decode + exact finalizer +
response projection total`, while raw candidate latency remains diagnostic
only.

The benchmark continues to use deterministic synthetic text semantics, so it
does not claim production Meilisearch typo-ranking equivalence. Sorts whose
meaning can be expressed by the indexed product fields are comparable with the
exhaustive reference. Exact line `proximity` sorting is deliberately marked
unsupported: computing it would reintroduce the geometry calculation that the
direct design removes. Meilisearch likewise documents that `_geoPoint` sorting
does not work with `_geojson` in its
[GeoJSON guide](https://www.meilisearch.com/docs/capabilities/geo_search/how_to/use_geojson_format).
Such cases are counted and listed but excluded from the comparable-page
denominator; they are never silently presented as successful direct pages.
Top-10 in the spatial screen likewise means nearest-ten
*candidate coverage*, not Meilisearch result ranking.

To reproduce reports from before this UX workload existed, pass both
`--geojson-tune=false` and `--geojson-direct=false`. The second flag is
explicit even though direct evaluation depends on the tuning screen; it keeps
legacy commands unambiguous if defaults evolve.

The exact distance matrix is prepared once and its duration is reported as
`oracle_duration_ms`. H3 refinement fetches complete encoded application
records by candidate ID from a separate SQLite database, scans and JSON-decodes
them, then runs the exact geometry check. RTree does the equivalent against
the geometry BLOB in its own `trails` table. The JSON report separates database
load/decode time from exact distance time and records rows and encoded bytes
loaded. The reference matrix remains in memory, but it is used only outside the
timed backend result path to score correctness.

### Product query workload

`--query-workload both` is the default. `engine` keeps only the radius/accuracy
matrix; `product` keeps only the end-to-end-style workload. Meilisearch product
queries combine the spatial predicate, ACL, text and metadata filters in one
unpaginated candidate request. H3 and safe GeoJSON then load those candidate
records from SQLite and defensively reapply, in order:

- public/owner/share ACLs and local/federated selection;
- deterministic multi-field AND text matching;
- category/subcategory plus distance, duration, elevation and difficulty
  filters;
- exact spherical point-to-polyline distance;
- exact total count, global stable sorting with an ID tie-breaker, and only
  then pagination (including deep-page cases).

Every case is first checked against an exhaustive reference over all records.
The report retains expected/actual ordered page IDs and counts for failures.
Warm samples then run with 2, 8, or 16 clients in the smoke, standard, and
stress profiles; `--query-clients` overrides this. Stage p50/p95/p99, overall
latency, throughput, candidate fan-out, database rows/bytes and exact-check
counts are recorded. The immutable application-database fixture's setup time
and size are reported separately and are not folded into index-build time.
Any spatial or product-semantic mismatch gives that backend status
`incorrect`; the report is still written and the run normally continues so the
known-inexact `point` baseline remains usable. `--fail-on-incorrect` turns one
or more such completed cells into a final non-zero exit for an ADR/CI gate.

The generic hybrid workload remains a diagnostic precursor for architectures
that require an exact finalizer. The capacity-created GeoJSON Direct workload
supplied the accepted G1 release evidence within its explicitly documented
scope.
The source record is synthetic JSON rather than a PocketBase record/GPX access;
Meilisearch paths push ACL, text and metadata predicates into candidate
acquisition and recheck them in the application finalizer, while RTree still
applies them only after candidate loading instead of joining them into SQL;
text matching is deterministic and does not imitate Meilisearch typo tolerance
or ranking; the federation flag does not perform remote network calls. Like the
original engine benchmark, candidate acquisition raises `maxTotalHits` to the
catalog size and requests the complete ID superset. This makes missing
candidates observable and final counts/pages genuinely exact for the measured
corpus, but it does not solve production caps, bounded-memory overflow, or the
cross-shard deep-pagination problem highlighted by PR 731. The dedicated
capacity-created direct search gate separately exercises the official
Meilisearch federation path, including the accepted 1.53.1 release and newer
qualification candidates, with an exhaustive page/count merge across every
shard; it does not use this candidate-plus-RAM/SQLite architecture. Those
limits are explicit so
GeoJSON/H3/RTree cannot look production-exact merely because the harness
already had every `Trail` in RAM. The benchmark's selected safe GeoJSON
comparison plan, like H3 and RTree, is a two-stage candidate-plus-exact plan.
The separately labelled `geojson_direct` report represents the ADR
0002-accepted product path and treats GeoJSON as the final answer without exact
geometry refinement.

### Optional DS923+ stress target

The `nas` profile is an optional stress workload, not an emulation or a release
gate. It uses 50,000 trails, 60 query points, four product clients and 84 warm repetitions (5,040
product samples). `--performance-target ds923plus` is implied and evaluates
each backend as follows:

- normal radii up to 25 km: p50 <= 250 ms, p95 <= 750 ms, p99 <= 1.5 s;
- broad radii above 25 km: p50 <= 500 ms, p95 <= 2 s, p99 <= 4 s;
- zero incorrect first or warm product results;
- at least 50,000 trails, four clients, 5,000 warm samples and 4 queries/s overall;
- one complete index build in no more than 30 minutes.

When this optional performance target is active, every backend cell must
expose at least one complete passing product architecture. GeoJSON reports the conservative
exact and direct UX paths separately; either can satisfy the backend
performance gate, while `--fail-on-ux` additionally requires the direct path
itself to pass accuracy, stability and the active DS923+ target. These rules
qualify that requested stress run; they do not add a release prerequisite.

The profile models a DS923+ with 8 GB ECC RAM, ordinary SATA storage, no NVMe
cache and the complete DSM/Wanderer/SQLite/Meilisearch stack.
Synology specifies a two-core/four-thread AMD Ryzen R1600 at 2.6/3.1 GHz and
4 GB ECC RAM expandable to 32 GB in the
[DS923+ data sheet](https://global.download.synology.com/download/Document/Hardware/DataSheet/DiskStation/23-year/DS923%2B/enu/Synology_DS923%2B_Data_Sheet_enu.pdf).
Eight GB is the profile reference so DSM and the application retain useful
page-cache headroom; stock 4 GB can be covered by a separate small-catalog
smoke/no-OOM diagnostic.

The Markdown/JSON target result covers the query and isolated full-build SLO.
When using this optional profile operationally, also observe at least 25%
available RAM, no swap thrashing or OOM, sustained CPU headroom, and
non-saturated I/O on the device. A 50k shadow rebuild should finish within 30
minutes without blocking startup or taking the live index away;
single add/update/delete visibility should stay below 2 s p95 and 5 s p99.
The current harness reports build/mutation times and RSS, but it cannot prove
DSM-wide memory, swap or I/O headroom from inside a developer workstation.

### G1 release decision

[ADR 0002](../../../docs/concepts/adr/0002-geojson-direct-route-radius.md)
accepts and release-qualifies the official Meilisearch GeoJSON Direct path
with r100, an S50 index projection, geodesically densified segments of at most 5 km, and
capacity-created shards of at most 1,000 trails. The Direct response is final;
there is no hidden SQLite hydration or exact point-to-polyline finalizer. Its
spatial contract is intentionally aggregate and approximate, while ACL/text/
filter semantics and truncation remain hard correctness gates. H3 plus exact
verification is the first fallback, not a parallel production index. RTree is
no longer an ADR candidate.

The accepted 10k decision artifact used Meilisearch 1.53.1, 10 shards, 240
untimed spatial cases and 240 warm product samples. It achieved 99.97% raw
spatial recall, 100% clear-interior recall and material precision, 65/183/338
ms p50/p95/p99 and 49 queries/s. Its full build took 10.2 seconds, used about
52.7 MiB of Meilisearch database space and measured about 1.2 GiB engine
peak-RSS delta. The evidence identity anchors are:

- harness `sha256:b65bfd9c31f15edb2d697743551237951ab6642a3a1da5251a2d86d1c82c4867`;
- dataset `sha256:9f94f324539c0c231f83b1c21f7fd3a2ec59eacc5c9ef375230329f0e62bbdc0`;
- executable `sha256:acc700fe190faf67371b6224b22bd241c3349257a7c2eb2dc049e1b0637fe535`;
- image `getmeili/meilisearch@sha256:8d6643d86d71fad6ad3cba92cde7ccfce9e4d6c384bda67598eb553571c32431`.

These digests identify the measured inputs; they do not by themselves archive
the corresponding source, dataset or executable bytes.

That closes G1 and qualifies the path for the intended release. No additional
50k run, 8-GB RAM configuration or specific NAS model is required. The 50k and
250k profiles remain optional scaling diagnostics if an operator wants to
evaluate unusually large catalogs or host-specific capacity.

The GeoJSON preflight separately proves all of the following:

- a radius intersects the middle of a `LineString` whose endpoints are far
  outside it;
- a radius intersects the second part of a `MultiLineString`;
- a radius in a `MultiLineString` gap does not match;
- a bounding box intersects a `LineString` whose endpoints are outside it;
- overlapping geometry and viewport bounding boxes alone do not produce a
  match when the actual line is disjoint;
- a bounding box in a `MultiLineString` gap does not match;
- touching the viewport boundary counts as an intersection;
- a viewport crossing the antimeridian matches an RFC 7946-style split
  `MultiLineString`;
- coordinates use GeoJSON order `[longitude, latitude]`.
- the explicit fourth `_geoRadius` resolution argument works at 125 and 1,000;
- conservative queries still include the five deterministic near-boundary
  regressions previously missed by the default three-argument query.

It also creates an isolated index below Cellulite's 200-document split
threshold and stores native `_geo` and GeoJSON points at identical coordinates.
The same radius filter finds both controls in Zurich, but on Meilisearch 1.36,
1.44 and 1.53.1 it finds only `_geo` at `20 N / 15 E` for 500 m and at
`79.101109 N / -17.696735 E` through 100 km. The latter also fails with a
correctly ordered bounding box and with forward, reversed and densified
LineStrings. This is reported as the additive `geojson_cell_coverage`
diagnostic rather than mislabeled as a polar error.

The implementation explains the location dependence: Cellulite's query path
uses h3o coverage and planar `geo::Relate` checks against latitude/longitude
polygons made from coarse H3 cell boundaries. At the failing coordinates this
planar polygon disagrees with the spherical H3 cell assignment, so the stored
cell can be omitted. The query path does not call Local IJ, `gridDistance` or
`gridPathCells`; Pentagon and Face-unfolding limitations therefore do not
explain this reproducer. See the pinned
[Cellulite 0.3.2 reader](https://github.com/meilisearch/cellulite/blob/v0.3.2/src/reader.rs)
and [h3o 0.9.0 tiler](https://github.com/HydroniumLabs/h3o/blob/v0.9.0/src/geom/tiler.rs).
The diagnostic remains non-gating because the declared Direct UX policy is
aggregate and explicitly tolerates rare misses; it stays visible so a future
engine upgrade cannot silently change the evidence.

A second isolated diagnostic exercises Cellulite's 200-document split path.
It indexes 5,000 deterministic points in a dense 0.6-degree square, with every
coordinate duplicated into native `_geo` and GeoJSON. On Meilisearch 1.36,
1.44 and 1.53.1 the native bounding box returns 5,000/5,000 while an
`_geoPolygon` over the same extent returns 4,980/5,000. This is reported as
`geojson_hierarchical_split` and is independent of the planar boundary error.

The reduced Cellulite reproducer shows that those 20 documents are present in
the index. Its contained-cell optimization pre-marks formal H3 children as
explored even though H3 hierarchy does not guarantee strict geometric
containment. A child still required by geometric tiling can therefore be
skipped when it is reached later through an intersecting cell. Removing that
pre-mark restores
5,000/5,000 (and 50,000/50,000 in the larger reduction). Densifying cell
boundaries alone does not repair this split-reader failure, so an upstream fix
must pass both diagnostics.

Wanderer treats both diagnostics as accepted upstream limitations of the
official Meilisearch GeoJSON implementation. They do not justify maintaining a
custom Meilisearch, Cellulite or h3o build and remain non-gating while the
selected direct plan satisfies its aggregate UX contract. Every Meilisearch
upgrade must rerun the diagnostics. A newer image removes the limitation only
after the corresponding diagnostic passes; an image-version increase alone is
not evidence of a fix. A completed `known_engine_limitation` remains non-gating;
`diagnostic_error` means the probe produced no conclusion and makes
`--contract-only` exit non-zero.

For the separate planar mismatch, an isolated h3o/Cellulite prototype
geodetically densified cell edges to at most 25 km before planar relation
tests. It removed all tested non-polar misses: all 120 non-polar resolution-0
cells in a global raster, 67,721 DACH/Alps samples and 101,950 near-boundary
500 m circle samples. The two cells containing the geographic poles require a
special cap closure after longitude unwrapping; further densification alone
cannot represent that topology in a flat longitude/latitude polygon. These
upstream prototype results are not evidence that an official Meilisearch image
already contains the fix.

Meilisearch documents `_geoBoundingBox` as top-right `[north, east]` followed
by bottom-left `[south, west]`, with the western longitude smaller than the
eastern longitude. Versions 1.36 and 1.44 currently also accept a
single reversed-longitude box, but that behavior is undocumented and is not a
contract the benchmark relies on. The benchmark's `GeoViewport` primitive
therefore treats `west > east` as an antimeridian crossing and emits two
documented `_geoBoundingBox` filters joined with `OR` (`west..180` and
`-180..east`). A zero-width `west == east` viewport is rejected as ambiguous.
Dateline-crossing route geometry is likewise cut into separate line parts at
`+180/-180`; an uncut segment from `+179` to `-179` is outside this contract.

Numbers are host- and Docker-version-dependent. Compare backends on the same
machine, with no other heavy workload, and keep the JSON report with its
`environment` and `reproducibility` sections.
