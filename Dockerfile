FROM golang

WORKDIR /wd
COPY . /wd
RUN go mod tidy
RUN go build ./cmd/cgat
ENTRYPOINT ["./cgat"]
