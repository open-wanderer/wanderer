FROM alpine:3.19

WORKDIR /

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