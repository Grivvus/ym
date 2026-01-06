FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/server /usr/bin/server
COPY --from=builder /app/api/openapi.yml /api/openapi.yml
COPY --from=builder /app/.env /.env
RUN chmod +x /usr/bin/server

ENTRYPOINT ["/usr/bin/server"]
