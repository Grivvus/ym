EXECUTABLE_NAME ?= server

build:
	@echo "Building server"
	@mkdir -p .bin
	@go build -o ./bin/${EXECUTABLE_NAME} ./cmd/server

test:
	@echo "testing"
	go test -shuffle=on -race -coverprofile=coverage.txt ./... -v

clean:
	@echo "deleting binaries"
	@rm ./bin/*
