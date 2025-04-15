# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.24 AS build

WORKDIR /build

COPY ./ ./

RUN go mod tidy -v

RUN go mod download -json

RUN go mod verify

RUN go build -v -o /scalecloud.de-api ./cmd/scalecloud.de-api

##
## Test
##
RUN go test ./... -json

##
## Deploy
##
FROM gcr.io/distroless/base-debian11:latest AS deploy

WORKDIR /app

COPY --from=build /scalecloud.de-api /app/scalecloud-api.de

EXPOSE 15000

USER nonroot:nonroot

ENTRYPOINT ["/app/scalecloud-api.de"]