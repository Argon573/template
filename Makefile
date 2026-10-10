OPENAPI_SPEC := contracts/openapi/trip-service.openapi.yaml
OUTPUT_FILE  := internal/generated/api.gen.go

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: generate
generate:
	@echo "Генерация кода по спецификации OpenAPI..."
	@mkdir -p internal/generated
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o $(OUTPUT_FILE) \
		$(OPENAPI_SPEC)
	@echo "Успешно сгенерировано в $(OUTPUT_FILE)"

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
	go run ./cmd/api
