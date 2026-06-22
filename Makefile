## db

.PHONY: db-fmt
db-fmt:
	cd db && go fmt ./...

.PHONY: db-vet
db-vet:
	cd db && go vet ./...

.PHONY: db-test
db-test:
	cd db && go test ./...

.PHONY: db-build
db-build:
	cd db && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pocketbase_amd64

.PHONY: db-build-docker
db-build-docker: db-build
	docker buildx build db/ --no-cache -t flomp/wanderer-db:latest

## Web

.PHONY: web-install
web-install:
	cd web && npm install

.PHONY: web-playwright-install
web-playwright-install:
	cd web && npx playwright install --with-deps chromium

.PHONY: web-check
web-check:
	cd web && npm run check

.PHONY: web-test
web-test:
	cd web && npm run test

.PHONY: web-build-docker
web-build-docker:
	docker buildx build web/ --no-cache -t flomp/wanderer-web:latest

## Plugins

.PHONY: plugins-test
plugins-test:
	cd plugins/sdk && go test ./...
	cd plugins/hammerhead && go test ./...
	cd plugins/komoot && go test ./...
	cd plugins/strava && go test ./...

.PHONY: plugins-build
plugins-build:
	cd plugins/hammerhead && XDG_CACHE_HOME=$${XDG_CACHE_HOME:-/tmp/wanderer-tinygo-cache} make build
	cd plugins/komoot && XDG_CACHE_HOME=$${XDG_CACHE_HOME:-/tmp/wanderer-tinygo-cache} make build
	cd plugins/strava && XDG_CACHE_HOME=$${XDG_CACHE_HOME:-/tmp/wanderer-tinygo-cache} make build

.PHONY: plugins-install-local
plugins-install-local: plugins-build
	mkdir -p data/plugins
	rm -rf data/plugins/hammerhead data/plugins/komoot data/plugins/strava
	cp -a plugins/hammerhead/dist/hammerhead data/plugins/
	cp -a plugins/komoot/dist/komoot data/plugins/
	cp -a plugins/strava/dist/strava data/plugins/

.PHONY: plugins-package
plugins-package: plugins-build
	rm -rf plugin_dist
	mkdir -p plugin_dist
	tar -C plugins/hammerhead/dist -czf plugin_dist/wanderer-plugin-hammerhead.tar.gz hammerhead
	tar -C plugins/komoot/dist -czf plugin_dist/wanderer-plugin-komoot.tar.gz komoot
	tar -C plugins/strava/dist -czf plugin_dist/wanderer-plugin-strava.tar.gz strava
	cd plugin_dist && sha256sum *.tar.gz > SHA256SUMS
