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

RUN go build -o /web-service-gin

##
## Deploy
##
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /web-service-gin /web-service-gin

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/web-service-gin"]