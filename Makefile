## db

.PHONY: db-fmt
db-fmt:
	cd db && go fmt ./...

.PHONY: db-vet
db-vet:
	cd db && go vet ./...

.PHONY: install-golangci-lint
install-golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $HOME/go/bin v2.10.1

.PHONY: db-lint
db-lint:
	cd db && golangci-lint run --verbose --show-stats ./...

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
