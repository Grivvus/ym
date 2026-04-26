FROM golang:1.26 AS builder

WORKDIR /app
RUN apt-get update && apt-get install libwebp-dev -y

COPY go.mod go.sum ./
RUN go mod download

COPY api/openapi.yml /app/api/openapi.yml

COPY . .

RUN GOOS=linux go build -o server ./cmd/server/

FROM debian:bookworm-slim

RUN apt-get update && apt-get install ffmpeg libwebp-dev -y

COPY --from=builder /app/server /usr/bin/server
COPY --from=builder /app/api/openapi.yml /api/openapi.yml
RUN chmod +x /usr/bin/server

ENTRYPOINT ["/usr/bin/server"]
