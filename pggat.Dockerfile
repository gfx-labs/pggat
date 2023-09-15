# syntax=docker/dockerfile:1
FROM golang:1.21-alpine as GOBUILDER
RUN apk add build-base git
WORKDIR /src
COPY . .

RUN go mod tidy
RUN go build -o cgat ./cmd/cgat

FROM alpine:latest
WORKDIR /bin
RUN mkdir /var/run/pgbouncer && addgroup -S pgbouncer && adduser -S pgbouncer && mkdir -p /etc/pgbouncer /var/log/pgbouncer /var/run/pgbouncer
COPY --from=GOBUILDER /src/cgat.sh run.sh
COPY --from=GOBUILDER /src/cgat pggat
RUN apk add openssl
RUN chown -R pgbouncer:pgbouncer /var/log/pgbouncer /var/run/pgbouncer /etc/pgbouncer /etc/ssl/certs /bin/run.sh
USER pgbouncer:pgbouncer

ENTRYPOINT ["/bin/run.sh"]
