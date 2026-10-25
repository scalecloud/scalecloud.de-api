# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.27 AS build

WORKDIR /build

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY ./ ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -v -ldflags="-s -w" -o /scalecloud.de-api ./cmd/scalecloud.de-api

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -v -ldflags="-s -w" -o /opencloud-usage-collector ./cmd/opencloud-usage-collector

##
## Deploy
##
FROM gcr.io/distroless/static-debian13:nonroot AS deploy

WORKDIR /app

COPY --from=build /scalecloud.de-api /app/scalecloud-api.de
COPY --from=build /opencloud-usage-collector /app/opencloud-usage-collector

EXPOSE 15000

USER nonroot:nonroot

ENTRYPOINT ["/app/scalecloud-api.de"]