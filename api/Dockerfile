# syntax=docker/dockerfile:1

FROM golang:1.25.3-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG APP_VERSION=v1.0.0
RUN echo "Building for $TARGETOS/$TARGETARCH$TARGETVARIANT"

WORKDIR /src

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN set -eux; \
    export GOOS="$TARGETOS" GOARCH="$TARGETARCH"; \
    if [ "$TARGETARCH" = "arm" ] && [ -n "$TARGETVARIANT" ]; then export GOARM="${TARGETVARIANT#v}"; fi; \
    if [ "$TARGETARCH" = "amd64" ] && [ -n "$TARGETVARIANT" ]; then export GOAMD64="${TARGETVARIANT#v}"; fi; \
    go build -trimpath -tags release -ldflags="-s -w -X github.com/dujiao-next/internal/version.Version=${APP_VERSION}" -o /out/dujiao-api ./cmd/server

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata \
    && mkdir -p /app/db /app/uploads /app/logs

COPY --from=builder /out/dujiao-api /app/dujiao-api
COPY config.yml.example /app/config.yml.example

EXPOSE 8080

CMD ["./dujiao-api"]
