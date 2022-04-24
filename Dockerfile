# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.18 AS build

WORKDIR /app

COPY cmd/scalecloud.de-api/go.mod ./
COPY cmd/scalecloud.de-api/go.sum ./
RUN go mod download

COPY cmd/scalecloud.de-api/*.go ./

RUN go build -v -o /scalecloud.de-api

##
## Deploy
##
FROM gcr.io/distroless/base-debian11:latest AS deploy

WORKDIR /app

COPY --from=build /scalecloud.de-api ./

EXPOSE 15000

USER nonroot:nonroot

ENTRYPOINT ["/app/scalecloud.de-api"]