FROM debian:bookworm-slim

WORKDIR /

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       curl \
    && rm -rf /var/lib/apt/lists/*

COPY migrations ./migrations
COPY templates ./templates

ARG TARGETARCH
RUN echo ${TARGETARCH}
COPY ./pocketbase_${TARGETARCH} /pocketbase
RUN chmod +x /pocketbase

ENV MEILI_URL=http://localhost:7700
ENV MEILI_MASTER_KEY=

EXPOSE 8090

ENTRYPOINT ["/pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/pb_data"]
