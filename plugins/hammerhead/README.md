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

## Development

```sh
GOCACHE=/tmp/wanderer-go-cache go test ./...
make manifest
```
