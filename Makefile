.PHONY: help build run dev clean test install serve migrate generate docs lint install-hooks

APP_NAME=kas
BUILD_DIR=.

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -trimpath -ldflags="-s -w" -o $(APP_NAME) .

run: build ## Build and run the application
	./$(APP_NAME) serve

dev: ## Run in development mode with auto-reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air is not installed. Install with: go install github.com/air-verse/air@latest"; \
		echo "Running without auto-reload..."; \
		$(MAKE) run; \
	fi

serve: ## Run the PocketBase server
	go run main.go serve

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f $(APP_NAME)
	go clean

migrate-up: ## Apply pending migrations
	go run main.go migrate up

migrate-down: ## Revert last migration
	go run main.go migrate down 1

migrate-create: ## Create new migration (usage: make migrate-create NAME="migration_name")
	go run main.go migrate create $(NAME)

migrate-collections: ## Generate collections snapshot migration
	go run main.go migrate collections

migrate-sync: ## Sync migration history (remove entries without files)
	go run main.go migrate history-sync

admin: ## Create admin account (interactive)
	go run main.go admin create

backup: ## Backup PocketBase data
	@echo "Creating backup of pb_data..."
	@mkdir -p pb_data_backup
	@cp -r pb_data pb_data_backup/pb_data_$(shell date +%Y%m%d_%H%M%S)
	@echo "Backup created successfully!"

update: ## Update dependencies
	go get -u ./...
	go mod tidy

generate: ## Generate type-safe proxies from PocketBase schema
	@echo "Generating template from pb_data..."
	pocketbase-gogen template ./pb_data ./pbschema/template.go
	@echo "Generating proxies with utils and hooks..."
	pocketbase-gogen generate ./pbschema/template.go ./generated/proxies.go --utils --hooks
	@echo "Code generation completed!"

lint: ## Run go vet + staticcheck (excludes generated/)
	go vet ./...
	@if command -v staticcheck > /dev/null; then \
		staticcheck -checks "all,-ST1000,-ST1003,-ST1020,-ST1021,-ST1022" $$(go list ./... | grep -v 'kas/generated$$' | grep -v 'kas/pbschema$$'); \
	else \
		echo "staticcheck not installed. Run: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

install-hooks: ## Install git hooks from scripts/ into .git/hooks/
	@chmod +x scripts/pre-commit scripts/install-hooks.sh
	@bash scripts/install-hooks.sh

docs: ## Generate OpenAPI spec with swaggo/swag
	@echo "Generating OpenAPI spec..."
	@tmp_dir="$$(mktemp -d)"; \
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init \
		--generalInfo main.go \
		--dir . \
		--output "$$tmp_dir" \
		--parseDependency \
		--parseInternal; \
	cp "$$tmp_dir/swagger.yaml" pb_public/openapi.yaml; \
	rm -rf "$$tmp_dir"; \
	echo "OpenAPI spec written to pb_public/openapi.yaml"
