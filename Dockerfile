FROM golang:1.11 AS builder
WORKDIR /go/src/github.com/ryotarai/prometheus-query-proxy
COPY . .
RUN go build -o /usr/bin/prometheus-query-proxy .

###############################################

FROM ubuntu:16.04
COPY --from=builder /usr/bin/prometheus-query-proxy /usr/bin/prometheus-query-proxy
ENTRYPOINT ["/usr/bin/prometheus-query-proxy"]
