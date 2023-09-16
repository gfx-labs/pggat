# syntax=docker/dockerfile:1
FROM golang:1.21-alpine as GOBUILDER
RUN apk add build-base git
WORKDIR /src
COPY . .

RUN go mod tidy
RUN go build -race -o cgat ./cmd/cgat

FROM alpine:latest
WORKDIR /bin
RUN addgroup -S pgbouncer && adduser -S pgbouncer
COPY --from=GOBUILDER /src/cgat.sh run.sh
COPY --from=GOBUILDER /src/cgat pggat
RUN apk add openssl
RUN install -d -m 0755 -o pgbouncer -g pgbouncer /etc/pgbouncer /var/log/pgbouncer /var/run/pgbouncer /etc/ssl/certs
RUN chown -R pgbouncer:pgbouncer /bin/run.sh
USER pgbouncer:pgbouncer

ENTRYPOINT ["/bin/run.sh"]
