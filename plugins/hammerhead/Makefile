PLUGIN_ID := hammerhead
DIST_DIR := dist/$(PLUGIN_ID)

.PHONY: build manifest test-integration test-integration-built clean

build: manifest
	tinygo build -target=wasi -scheduler=none -no-debug -o $(DIST_DIR)/plugin.wasm .

manifest:
	mkdir -p $(DIST_DIR)
	go run github.com/open-wanderer/wanderer/plugins/sdk/cmd/manifestcheck > $(DIST_DIR)/plugin.json
	cp assets/icon.svg $(DIST_DIR)/icon.svg
	cp assets/icon_dark.svg $(DIST_DIR)/icon_dark.svg

test-integration: build
	$(MAKE) test-integration-built

test-integration-built:
	cd ../../db && WANDERER_HAMMERHEAD_WASM="$(CURDIR)/$(DIST_DIR)/plugin.wasm" go test ./routes -run '^TestHammerheadBuiltWASM' -count=1

clean:
	rm -rf dist
