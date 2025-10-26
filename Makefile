EXECUTABLE_NAME ?= server

generate_swagger:
	@echo "Generate swagger"
	@swag init -d cmd/server/,internal/api

build: generate_swagger
	@echo "Building server"
	@mkdir -p .bin
	@go build -o ./bin/${EXECUTABLE_NAME} ./cmd/server

serve: build
	@echo "Serve"
	@./bin/${EXECUTABLE_NAME}

test:
	@echo "testing"
	@go test -shuffle=on -race -coverprofile=coverage.txt ./... -v

clean:
	@echo "deleting binaries"
	@rm ./bin/*
