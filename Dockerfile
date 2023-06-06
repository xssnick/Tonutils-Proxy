FROM golang:alpine

RUN apk add --no-cache git \
  && git clone https://github.com/xssnick/Tonutils-Proxy.git \
  && cd ./Tonutils-Proxy \
  && go build -o ton-proxy cmd/proxy-cli/main.go


FROM alpine:3.18.0

COPY --from=0 /go/Tonutils-Proxy/ton-proxy /usr/local/bin/

RUN apk add --no-cache bash curl

CMD ["/usr/local/bin/ton-proxy", "-addr", "0.0.0.0:8080"]
