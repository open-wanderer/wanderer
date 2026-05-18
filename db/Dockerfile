FROM curlimages/curl:8.18.0 AS download-curl

# renovate: datasource=github-releases depName=stunnel/static-curl packageName=stunnel/static-curl
ENV CURL_VERSION=8.18.0

# TARGETARCH is injected by buildx: "amd64" or "arm64"
ARG TARGETARCH

# static-curl uses "x86_64" / "aarch64" in its filenames
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

FROM curlimages/curl:8.18.0 AS download-pmtiles

# renovate: datasource=github-releases depName=go-pmtiles packageName=protomaps/go-pmtiles
ENV PMTILES_VERSION=1.28.0

# TARGETARCH is injected by buildx: "amd64" or "arm64"
ARG TARGETARCH

# go-pmtiles uses "x86_64" / "arm64" in its filenames
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

FROM golang:1.25.0@sha256:5502b0e56fca23feba76dbc5387ba59c593c02ccc2f0f7355871ea9a0852cebe AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/pocketbase

FROM scratch

WORKDIR /

COPY --from=build /app/pocketbase /pocketbase
COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=download-curl /tmp/curl /curl
COPY --from=download-pmtiles /tmp/pmtiles /pmtiles
COPY migrations ./migrations
COPY templates ./templates

ENV MEILI_URL=http://localhost:7700 \
MEILI_MASTER_KEY= \
POCKETBASE_ENCRYPTION_KEY=

EXPOSE 8090

ENTRYPOINT ["/pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/pb_data"]