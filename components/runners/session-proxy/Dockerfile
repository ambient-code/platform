# Session Proxy - ADR-0006 streaming exec sidecar
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o session-proxy .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/session-proxy /usr/local/bin/session-proxy

USER 1000:1000
ENTRYPOINT ["/usr/local/bin/session-proxy"]
