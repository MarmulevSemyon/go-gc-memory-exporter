FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /out/gc-mem-exporter ./cmd/gc-mem-exporter

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/gc-mem-exporter /app/gc-mem-exporter

ENV ADDR=:8080
ENV GC_PERCENT=100
EXPOSE 8080

ENTRYPOINT ["/app/gc-mem-exporter"]
