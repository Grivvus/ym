FROM golang:1.25 AS builder

WORKDIR /app
RUN apt-get update && apt-get install libwebp-dev -y

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux go build -o server ./cmd/server/

FROM debian:latest

RUN apt-get update && apt-get install libwebp-dev -y

COPY --from=builder /app/server /usr/bin/server
COPY --from=builder /app/api/openapi.yml /api/openapi.yml
COPY --from=builder /app/.env /.env
COPY --from=builder /app/.env.minio /.env.minio
RUN chmod +x /usr/bin/server

ENTRYPOINT ["/usr/bin/server"]
