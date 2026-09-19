PLUGIN_ID := brouter
DIST_DIR := dist/$(PLUGIN_ID)

.PHONY: build manifest clean

build: manifest
	tinygo build -target=wasi -scheduler=none -no-debug -o $(DIST_DIR)/plugin.wasm .

manifest:
	mkdir -p $(DIST_DIR)
	go run github.com/open-wanderer/wanderer/plugins/sdk/cmd/manifestcheck > $(DIST_DIR)/plugin.json
	rm -rf $(DIST_DIR)/profiles
	cp -a profiles $(DIST_DIR)/profiles
	cp assets/icon.svg $(DIST_DIR)/icon.svg

clean:
	rm -rf dist
