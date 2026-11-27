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

## Updating the plugin bundle

Historical note: Version 0.1.2 fixed the elevation scale in new Dashboard activity
imports and used the existing manifest 1.0 contract. That version was tested with
wanderer v0.21.0.

Download and extract the Hammerhead archive from the release assets, then
replace the installed `data/plugins/hammerhead` bundle, including both
`plugin.json` and `plugin.wasm`. Restart the backend and open Hammerhead's
information dialog in Plugins settings to verify that the displayed version
matches the downloaded bundle. Updating the Docker image alone does not replace
an installed plugin bundle. Previously imported GPX files are preserved.

## Development

```sh
GOCACHE=/tmp/wanderer-go-cache go test ./...
make manifest
```
