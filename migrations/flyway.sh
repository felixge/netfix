#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"
NF_MIGRATIONS_DIR="${NF_MIGRATIONS_DIR:-${DIR}/migrations}"
NF_FLYWAY_BIN="${NF_FLYWAY_BIN:-flyway}"

"${NF_FLYWAY_BIN}" \
  -url="jdbc:postgresql://${PGHOST}:${PGPORT}/${PGDATABASE}" \
  -user="${PGUSER}" \
  -password="${PGPASSWORD}" \
  -schemas="${NF_PGSCHEMAS}" \
  -group=true \
  -locations="filesystem:${NF_MIGRATIONS_DIR}" \
  $@
