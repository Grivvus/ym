EXECUTABLE_NAME ?= server

.PHONY: generate_sqlc
generate_sqlc:
	@echo "Generate sqlc"
	@go tool sqlc generate -f ./db/sqlc.yml

.PHONY: generate_api
generate_api:
	@echo "generating server from openapi spec"
	@go tool oapi-codegen -generate 'chi,types' -package api ./api/openapi.yml > internal/api/server.gen.go

.PHONY: generate
generate: generate_sqlc generate_api
	@echo "Generate everything"

.PHONY: build
build: generate
	@echo "Building server"
	@go build -o ${EXECUTABLE_NAME} ./cmd/server

.PHONY: run-docker
run-docker:
	@echo "run docker"
	@docker compose -f compose.yml --profile metrics --profile local-storage up --build -d

.PHONY: stop-docker
stop-docker:
	@echo "stop docker"
	@docker compose -f compose.yml --profile metrics --profile local-storage stop

.PHONY: run-kube
run-kube:
	kubectl apply -f kube/
	helm upgrade --install vls vm/victoria-logs-single -f helm/victorialogs-values.yml
	helm upgrade --install vector vector/vector -f helm/vector-values.yml
	helm upgrade --install vmsingle vm/victoria-metrics-single -f helm/vmsingle-values.yml
	helm upgrade --install grafana grafana-community/grafana -f helm/grafana-values.yml
	helm upgrade --install alertmanager prometheus-community/alertmanager -f helm/alertmanager-values.yml 
	helm upgrade --install vmalert vm/victoria-metrics-alert --namespace default --values helm/vmalert-values.yml

.PHONY: stop-kube
stop-kube:
	helm uninstall vmalert
	helm uninstall alertmanager
	helm uninstall grafana
	helm uninstall vmsingle
	helm uninstall vector
	helm uninstall vls
	kubectl delete -f kube/

.PHONY: test
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

.PHONY: migration_up
migration_up:
	@echo "migration up"
	@go tool goose -env=.env -dir=db/migrations/ up

.PHONY: migration_down
migration_down:
	@echo "migration down"
	@go tool goose -env=.env -dir=db/migrations/ down
