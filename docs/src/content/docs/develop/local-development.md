---
title: Local development
description: How to install <span class="-tracking-[0.075em]">wanderer</span> for local development
---

If you would like to set up a development environment on your own machine to work on <span class="-tracking-[0.075em]">wanderer</span> please first follow the bare-metal installation steps in the [installation guide](/run/installation#from-source). We will slightly modify the launch script to launch a node server in development mode instead:

```bash
trap "kill 0" EXIT

export ORIGIN=http://localhost:5173
export MEILI_URL=http://127.0.0.1:7700
export MEILI_MASTER_KEY=p2gYZAWODOrwTPr4AYoahCZ9CI8y9bUd0yQLGk-E3m8
export PUBLIC_POCKETBASE_URL=http://127.0.0.1:8090
export PUBLIC_VALHALLA_URL=https://valhalla1.openstreetmap.de
export POCKETBASE_ENCRYPTION_KEY=9ada3c93163812101e50e2bf49e880bc

cd search && ./meilisearch --master-key $MEILI_MASTER_KEY &
cd db && ./pocketbase serve &
cd web && npm run dev &

wait
```

This will bring up a `meilisearch` instance on `http://127.0.0.1:7700`, a `PocketBase` instance on `http://127.0.0.1:8090`, and a `vite` server for the <span class="-tracking-[0.075em]">wanderer</span> frontend on `http://localhost:5173`.

## PocketBase dashboard

It is highly advisable to create an admin user to access PocketBase's dashboard. To do so, please refer to the [backend configuration](/run/backend-configuration/#setup) section of the documentation.

## Building

When you are done with development and would like to build <span class="-tracking-[0.075em]">wanderer</span> for production there are some steps to follow.

### PocketBase

If you modified code in any of the `*.go` files make sure to build an updated binary with `go build`. In case you only edited tables via the `PocketBase` admin panel you don't need to do anything. The database will be migrated automatically.

Since the Docker image is using Alpine linux with musl, you need to compile the binary
using musl or using `CGO_ENABLED=false` as shown below.

```bash
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pocketbase_amd64
```

### Frontend

For the frontend there are no further caveats. Simply run `npm run build`.

### Docker

To create local docker images of <span class="-tracking-[0.075em]">wanderer</span> simply run the script below. These will work as drop-in replacements for the ones hosted on docker hub. This will only work if you have already completed the steps above.

```bash
# db
docker build db/ --no-cache -t flomp/wanderer-db:latest 

# web
docker build web/ --no-cache  -t flomp/wanderer-web:latest 
```

