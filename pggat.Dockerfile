# syntax=docker/dockerfile:1
FROM golang:1.21-alpine as GOBUILDER
RUN apk add build-base git
WORKDIR /src
COPY . .

RUN go mod tidy
RUN go build -o caddygat ./cmd/caddygat

FROM alpine:latest
WORKDIR /
RUN apk add --no-cache bash

COPY entrypoint.sh .

COPY --from=GOBUILDER /src/presets /presets
COPY --from=GOBUILDER /src/caddygat /usr/bin/pggat

ENTRYPOINT ["/entrypoint.sh"]
CMD ["pggat"]
