.PHONY: help build run test clean migrate docker-up docker-down

# Default target
help:
	@echo "Available commands:"
	@echo "  build      - Build the Go application"
	@echo "  run        - Run the server"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  migrate    - Run database migrations"
	@echo "  docker-up  - Start Docker services"
	@echo "  docker-down- Stop Docker services"
	@echo "  dev        - Start development environment"

# Build the application
build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/migrate cmd/migrate/main.go
	go build -o bin/admin cmd/admin/main.go

# Run the server
run: build
	./bin/server

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf uploads/

# Run database migrations
admin: build
	./bin/admin $(ARGS)

# Run migrations
migrate: build
	./bin/migrate

# Start Docker services
docker-up:
	docker-compose up -d

# Stop Docker services
docker-down:
	docker-compose down

# Start development environment
dev: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Running migrations..."
	@make migrate
	@echo "Starting server..."
	@make run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate API documentation (if using swagger)
docs:
	@echo "API documentation generation not implemented yet"

# Create uploads directory
setup:
	mkdir -p uploads
	cp env.example .env

# Full setup for new development environment
setup-dev: setup deps docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Running migrations..."
	@make migrate
	@echo "Development environment ready!"
	@echo "Run 'make run' to start the server" 