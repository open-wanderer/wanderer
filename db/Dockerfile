# ==========================================
# 1. DOWNLOAD CURL (Native Intel Execution)
# ==========================================
FROM --platform=$BUILDPLATFORM curlimages/curl:8.18.0 AS download-curl

ENV CURL_VERSION=8.18.0

# Explicitly pull target arch since platform is pinned to host
ARG TARGETARCH

RUN set -eux ; \
case "$TARGETARCH" in \
  amd64) ARCHIVE_ARCH="x86_64" ;; \
  arm64) ARCHIVE_ARCH="aarch64" ;; \
esac ; \
curl \
  --connect-timeout 10 \
  --fail \
  --location \
  --max-time 300 \
  --output /tmp/curl.tar.xz \
  --proto '=https' \
  --show-error \
  --silent \
  --tlsv1.2 \
  "https://github.com/stunnel/static-curl/releases/download/${CURL_VERSION}/curl-linux-${ARCHIVE_ARCH}-glibc-${CURL_VERSION}.tar.xz" \
; \
tar -xJf /tmp/curl.tar.xz -C /tmp ; \
chmod +x /tmp/curl ;

# ==========================================
# 2. DOWNLOAD PMTILES (Native Intel Execution)
# ==========================================
FROM --platform=$BUILDPLATFORM curlimages/curl:8.18.0 AS download-pmtiles

ENV PMTILES_VERSION=1.28.0
ARG TARGETARCH

RUN set -eux ; \
case "$TARGETARCH" in \
  amd64) ARCHIVE_ARCH="x86_64" ;; \
  arm64) ARCHIVE_ARCH="arm64" ;; \
esac ; \
curl \
  --connect-timeout 10 \
  --fail \
  --location \
  --max-time 300 \
  --output /tmp/pmtiles.tar.gz \
  --proto '=https' \
  --show-error \
  --silent \
  --tlsv1.2 \
  "https://github.com/protomaps/go-pmtiles/releases/download/v${PMTILES_VERSION}/go-pmtiles_${PMTILES_VERSION}_Linux_${ARCHIVE_ARCH}.tar.gz" \
; \
tar -xzf /tmp/pmtiles.tar.gz -C /tmp pmtiles ; \
chmod +x /tmp/pmtiles ;

# ==========================================
# 3. BUILD POCKETBASE (Native Intel Cross-Compilation)
# ==========================================
FROM golang:1.26@sha256:9d2f36f06329b2a141b9db99ffa32765cf695ee57b813ca29e245e8670bcbfff AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/pocketbase
RUN install -d -m 1777 /rootfs/tmp

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /app/pocketbase

# ==========================================
# 4. FINAL IMAGE (Target Specific: ARM or AMD)
# ==========================================
FROM scratch

WORKDIR /

COPY --from=build /rootfs/ /
COPY --from=build /app/pocketbase /pocketbase
COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=download-curl /tmp/curl /curl
COPY --from=download-pmtiles /tmp/pmtiles /usr/local/bin/pmtiles
COPY migrations ./migrations
COPY templates ./templates

ENV MEILI_URL=http://localhost:7700 \
    MEILI_MASTER_KEY= \
    POCKETBASE_ENCRYPTION_KEY=

EXPOSE 8090

ENTRYPOINT ["/pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/pb_data"]