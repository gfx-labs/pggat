FROM golang

WORKDIR /wd
COPY . /wd
RUN go build ./cmd/cgat
ENTRYPOINT ["./cgat"]
