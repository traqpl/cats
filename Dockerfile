# syntax=docker/dockerfile:1.7

FROM --platform=linux/amd64 golang:1.26 AS build

WORKDIR /src

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod
ENV GOTMPDIR=/root/.cache/go-tmp

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.cache/go-tmp \
    mkdir -p "$GOCACHE" "$GOTMPDIR" && go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.cache/go-tmp \
    mkdir -p "$GOCACHE" "$GOTMPDIR" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/cats ./server/

FROM scratch

WORKDIR /app

COPY --from=build /out/cats /app/cats
COPY --from=build /src/config.yaml /app/config.yaml

ENV PORT=8071
ENV DB_PATH=/data/cats_scores.db

EXPOSE 8071
VOLUME ["/data"]

ENTRYPOINT ["/app/cats"]
