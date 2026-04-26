FROM golang:1.25 AS builder

WORKDIR /app

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY db/migrations/ ./migrations/

ENTRYPOINT ["goose", "-env=none", "-dir=/app/migrations/", "up"]
