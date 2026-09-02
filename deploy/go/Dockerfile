# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine AS build

ARG GO_API_DIR=backend-go
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY ${GO_API_DIR}/ ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/chihuo-api ./cmd/server

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S -g 65532 app \
    && adduser -S -D -H -u 65532 -G app app

COPY --from=build --chown=65532:65532 /out/chihuo-api /usr/local/bin/chihuo-api

USER 65532:65532
EXPOSE 4000

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=6 \
  CMD wget -qO- http://127.0.0.1:4000/health/ready >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/chihuo-api"]
