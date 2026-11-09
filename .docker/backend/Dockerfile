# Build from the repository root:
#   docker build -f .docker/backend/Dockerfile .
FROM docker.io/library/golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ENV CGO_ENABLED=0
RUN for cmd in api migrate seed cleanup; do \
        go build -trimpath -ldflags="-s -w" -o "/out/$cmd" "./cmd/$cmd" || exit 1; \
    done

# One-shot setup for compose: migrations and Garage (see .docker/init).
FROM docker.io/dxflrs/garage:v2.5.0 AS garage

FROM docker.io/library/alpine:3.22 AS init

WORKDIR /app

RUN apk add --no-cache aws-cli

COPY --from=garage /garage /usr/local/bin/garage
COPY --from=build /out/migrate /usr/local/bin/
COPY migrations ./migrations
COPY .docker/init/init.sh /init.sh

ENTRYPOINT ["sh", "/init.sh"]

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/ /app/
COPY migrations ./migrations

EXPOSE 8080

# Run the other commands by overriding the entrypoint, for example:
#   docker compose run --rm --entrypoint /app/migrate api up
ENTRYPOINT ["/app/api"]
