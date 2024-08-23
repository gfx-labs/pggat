# syntax=docker/dockerfile:1
FROM golang:1.23-alpine as GOBUILDER
RUN apk add build-base git
WORKDIR /src

COPY go.mod go.sum .
RUN go mod download
COPY test test
copy lib lib
copy cmd cmd

RUN go mod tidy
RUN go build -o pggat ./cmd/pggat

FROM alpine:latest
WORKDIR /
RUN apk add --no-cache bash

COPY --from=GOBUILDER /src/pggat /usr/bin/pggat
COPY presets /presets
COPY entrypoint.sh .

ENTRYPOINT ["/entrypoint.sh"]
