#!/bin/sh

set -eu

DATA_DIR="${MEILI_DATA_DIR:-/meili_data/data.ms}"
VERSION_FILE="${DATA_DIR}/VERSION"
TARGET_VERSION="${MEILI_TARGET_VERSION:-}"

if [ -z "${TARGET_VERSION}" ]; then
  echo "MEILI_TARGET_VERSION is not set; refusing to start." >&2
  exit 1
fi

mkdir -p "${DATA_DIR}"

CURRENT_VERSION=""
if [ -f "${VERSION_FILE}" ]; then
  CURRENT_VERSION="$(tr -d '\r\n' < "${VERSION_FILE}")"
fi
CURRENT_VERSION_STRIPPED="${CURRENT_VERSION#v}"
TARGET_VERSION_STRIPPED="${TARGET_VERSION#v}"

if [ "${CURRENT_VERSION_STRIPPED}" != "${TARGET_VERSION_STRIPPED}" ]; then
  BACKUP_SUFFIX="$(date +%Y%m%d%H%M%S)"
  BACKUP_ROOT="${MEILI_BACKUP_DIR:-/meili_data/backups}"
  BACKUP_DIR="${BACKUP_ROOT}/data.ms-${BACKUP_SUFFIX}"

  if [ "$(ls -A "${DATA_DIR}")" ]; then
    echo "Detected meilisearch data for version '${CURRENT_VERSION}', backing up to '${BACKUP_DIR}'..."
    mkdir -p "${BACKUP_DIR}"
    cp -a "${DATA_DIR}/." "${BACKUP_DIR}/"
    rm -rf "${DATA_DIR:?}/"*
  fi
fi

exec meilisearch "$@"
