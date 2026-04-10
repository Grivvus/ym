EXECUTABLE_NAME ?= server

generate_sqlc:
	@echo "Generate sqlc"
	@go tool sqlc generate -f ./db/sqlc.yml

generate_api:
	@echo "generating server from openapi spec"
	@go tool oapi-codegen -generate 'chi,types' -package api ./api/openapi.yml > internal/api/server.gen.go

generate: generate_sqlc generate_api
	@echo "Generate everything"

build: migration_up generate
	@echo "Building server"
	@mkdir -p .bin
	@go build -o ./bin/${EXECUTABLE_NAME} ./cmd/server

serve: build
	@echo "Serve"
	@./bin/${EXECUTABLE_NAME}

test:
	@echo "testing"
	@go test -shuffle=on -race -coverprofile=coverage.txt ./... -v

test_unit:
	@echo "run unit tests"
	@go test -shuffle=on -race -coverprofile=coverage.txt ./internal/... -v

.PHONY: lint
lint:
	@echo "starting golangci-lint in docker"
	@docker run -t --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v2.11.3 golangci-lint run

clean:
	@echo "deleting binaries"
	@rm ./bin/*

migration_up:
	@echo "migration up"
	@go tool goose -env=.env -dir=db/migrations/ up

migration_down:
	@echo "migration down"
	@go tool goose -env=.env -dir=db/migrations/ down
