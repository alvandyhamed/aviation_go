# syntax=docker/dockerfile:1

## ---- Build stage ----
# نسخه Go را با go.mod هماهنگ کن
FROM golang:1.24.5-alpine AS build
WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -ldflags="-s -w" -o /out/api    ./cmd/api && \
    go build -ldflags="-s -w" -o /out/ingest ./cmd/ingest

## ---- Runtime stage ----
FROM gcr.io/distroless/static:nonroot
WORKDIR /srv
COPY --from=build /out/api /srv/api
COPY --from=build /out/ingest /srv/ingest
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/srv/api"]
