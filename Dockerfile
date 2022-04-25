# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.18 AS build

WORKDIR /build

COPY ./ ./

RUN go mod download -json

RUN go build -v -o ./installer/scalecloud.de-api/ ./cmd/scalecloud.de-api

##
## Deploy
##
FROM gcr.io/distroless/base-debian11:latest AS deploy

WORKDIR /app

COPY --from=build /build/installer/scalecloud.de-api /app/scalecloud.de-api

EXPOSE 15000

USER nonroot:nonroot

ENTRYPOINT ["/app/scalecloud.de-api"]