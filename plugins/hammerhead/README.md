# wanderer Hammerhead WASM plugin

WASM/Extism version of the Hammerhead provider for wanderer.

This plugin exports the wanderer plugin-system ABI:

- `list_routes_v1`
- `list_activities_v1`
- `refresh_session_v1`
- `prepare_trail_send_v1`

## Build

Install TinyGo, then run:

```sh
make build
```

The plugin bundle is written to `dist/hammerhead/`. Copy it below
`data/plugins` or run `make plugins-install-local` from the repository root to
install all bundled plugins locally.

## Updating to 0.1.2

Version 0.1.2 fixes the elevation scale in new Dashboard activity imports and
uses the existing manifest 1.0 contract.
0.1.2 is tested with wanderer v0.21.0.
Download and extract the Hammerhead archive from the release assets, then
replace the installed `data/plugins/hammerhead` bundle, including both
`plugin.json` and `plugin.wasm`. Restart the backend and open Hammerhead's
information dialog in Plugins settings to check for version 0.1.2. Updating the
Docker image alone does not replace an installed plugin bundle. Previously
imported GPX files are preserved.

## Development

```sh
GOCACHE=/tmp/wanderer-go-cache go test ./...
make manifest
```

With TinyGo installed, run this from the repository root to build the bundle
and check it through the real backend worker:

```sh
make -C plugins/hammerhead test-integration
```

The target runs `make build` first. For a manual test run, build the guest
before setting the path to its WASM file:

```sh
make -C plugins/hammerhead build
cd db
WANDERER_HAMMERHEAD_WASM="$(pwd)/../plugins/hammerhead/dist/hammerhead/plugin.wasm" \
go test ./routes -run '^TestHammerheadBuiltWASM' -count=1
```

When the bundle has already been built, as in CI after `make plugins-build`,
run `make -C plugins/hammerhead test-integration-built` to reuse it.

This uses only a synthetic local provider and requires a private non-loopback
network interface. Both activity and planned-route exports are checked.
By default, the test binary starts the production worker in a child process.
Optionally set `WANDERER_PLUGIN_WORKER_BIN` to a built wanderer backend to test
the same bundle against that worker.
