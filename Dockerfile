# syntax=docker/dockerfile:1

FROM golang:1.18

WORKDIR /app

COPY web-service-gin/go.mod ./
COPY web-service-gin/go.sum ./
RUN go mod download

COPY web-service-gin/*.go ./

RUN go build -o /web-service-gin

EXPOSE 8080

CMD [ "/web-service-gin" ]