EXECUTABLE_NAME ?= server

build:
	@echo "Building server"
	@mkdir -p .bin
	@go build -o ./bin/${EXECUTABLE_NAME} ./cmd/server

clean:
	@echo "deleting binaries"
	@rm ./bin/*
