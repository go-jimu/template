FROM golang:1.24-bookworm AS builder
WORKDIR /go/src
COPY . /go/src/
RUN set -e \
    && export GOPROXY=https://goproxy.cn,direct \
    && go mod download \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s -extldflags '-static'" -tags netgo -o template cmd/main.go \
    && apt update -yqq \
    && apt install -yqq ca-certificates

FROM debian:bookworm
WORKDIR /app
COPY --from=builder /go/src/template .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./configs /app/configs

EXPOSE 8080
CMD ["/app/template"]
