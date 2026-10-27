OPENAPI_SPEC := contracts/openapi/trip-service.openapi.yaml
OUTPUT_FILE  := internal/generated/api.gen.go

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: generate
generate:
	@mkdir -p internal/generated
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o $(OUTPUT_FILE) \
		$(OPENAPI_SPEC)
	go generate ./...

.PHONY: clean-generated
clean-generated:
	rm -f $(OUTPUT_FILE)

.PHONY: migrate
migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

.PHONY: run
run:
	go run ./cmd/service

.PHONY: test
test:
	go test -race -count=1 -coverprofile=coverage.out \
		./internal/domain/... ./internal/usecase/... ./internal/http/...
	@go tool cover -func=coverage.out | awk '/total:/ {print "coverage domain+usecase+http:", $$3}'
	@go test -race -count=1 -coverprofile=coverage-du.out \
		./internal/domain/... ./internal/usecase/...
	@go tool cover -func=coverage-du.out | awk '/total:/ {print "coverage domain+usecase:", $$3}'

.PHONY: test-integration
test-integration:
	go test -race -count=1 -tags=integration ./internal/repository/...

.PHONY: lint
lint:
	golangci-lint run ./...
