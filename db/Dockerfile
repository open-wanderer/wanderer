FROM curlimages/curl:8.18.0 AS download-env

# renovate: datasource=github-releases depName=stunnel/static-curl packageName=stunnel/static-curl
ENV CURL_VERSION=8.18.0

RUN set -eux ; \
    ARCHITECTURE="$(uname -m)" ; \
    case $ARCHITECTURE in \
    x86_64) ARCHITECTURE="x86_64" ;; \
    aarch64 | armv8* | arm64) ARCHITECTURE="aarch64" ;; \
    *) \
    echo "(!) Architecture $ARCHITECTURE unsupported" ; \
    exit 1 \
    ;; \
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
    "https://github.com/stunnel/static-curl/releases/download/${CURL_VERSION}/curl-linux-${ARCHITECTURE}-glibc-${CURL_VERSION}.tar.xz" \
    ; \
    tar -xJf /tmp/curl.tar.xz -C /tmp ; \
    chmod +x /tmp/curl ;

FROM golang:1.24.0@sha256:3f7444391c51a11a039bf0359ee81cc64e663c17d787ad0e637a4de1a3f62a71 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /app/pocketbase

FROM scratch

WORKDIR /

COPY --from=build /app/pocketbase /pocketbase
COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=download-env /tmp/curl /curl
COPY migrations ./migrations
COPY templates ./templates

ENV MEILI_URL=http://localhost:7700 \
    MEILI_MASTER_KEY= \
    POCKETBASE_ENCRYPTION_KEY=

EXPOSE 8090

ENTRYPOINT ["/pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/pb_data"]
