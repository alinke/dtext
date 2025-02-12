# syntax=docker/dockerfile:1

FROM crazymax/goxx:1.17 AS base

ENV GO111MODULE=auto \
    CGO_ENABLED=1

WORKDIR /go/src/main

FROM base AS build

ARG TARGETPLATFORM

# Install build dependencies
RUN --mount=type=cache,sharing=private,target=/var/cache/apt \
    --mount=type=cache,sharing=private,target=/var/lib/apt/lists \
    goxx-apt-get update && \
    goxx-apt-get install -y \
    binutils \
    gcc \
    g++ \
    pkg-config \
    dpkg-dev

# Copy and build the Go application
COPY . .

RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 GOARCH=arm64 goxx-go build -o /out/dtext main.go

FROM scratch AS artifact

COPY --from=build /out /

FROM scratch

COPY --from=build /out/dtext /dtext

ENTRYPOINT ["/dtext"]
