#!/usr/bin/env bash
flyway \
  -url="jdbc:postgresql://${PGHOST}:${PGPORT}/${PGDATABASE}" \
  -user="${PGUSER}" \
  -password="${PGPASSWORD}" \
  -schemas="${NF_PGSCHEMAS}" \
  -group=true \
  -locations="filesystem:${NF_SHARE}/migrations" \
  $@
