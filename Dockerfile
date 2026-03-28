FROM golang:1.26@sha256:f200f27a113fd26789f07ff95ec1f7e337e295ddb711c693cf5b18a6dc7e88f5 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
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
