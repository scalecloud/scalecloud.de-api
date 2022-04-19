# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.18 AS build

WORKDIR /app

COPY web-service-gin/go.mod ./
COPY web-service-gin/go.sum ./
RUN go mod download

COPY web-service-gin/*.go ./

RUN go build -v -o /web-service-gin

##
## Deploy
##
FROM gcr.io/distroless/base-debian11:latest AS deploy

WORKDIR /app

COPY --from=build /web-service-gin ./

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/web-service-gin"]