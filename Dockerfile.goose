FROM golang:1.27 AS builder

WORKDIR /app

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM debian:trixie-slim

COPY db/migrations/ /app/migrations/
COPY --from=builder /go/bin/goose /usr/bin/goose

ENTRYPOINT ["goose", "-env=none", "-dir=/app/migrations/", "up"]
