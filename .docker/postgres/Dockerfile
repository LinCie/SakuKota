FROM docker.io/library/postgres:17-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-17-postgis-3 \
        postgresql-17-postgis-3-scripts \
    && rm -rf /var/lib/apt/lists/* \
    && printf 'CREATE EXTENSION IF NOT EXISTS postgis;\n' \
        > /docker-entrypoint-initdb.d/10-postgis.sql
