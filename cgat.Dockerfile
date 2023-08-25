# syntax=docker/dockerfile:1
FROM golang:1.21-alpine as GOBUILDER
RUN apk add build-base git
WORKDIR /src
COPY . .

RUN go mod tidy
RUN go build -o pggat ./cmd/cgat

FROM alpine:latest
WORKDIR /bin
COPY --from=GOBUILDER /src/pggat pggat

ENTRYPOINT ["/bin/pggat"]
