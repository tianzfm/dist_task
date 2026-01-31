# Makefile for dist_task

.PHONY: all build run test clean lint docker deps run-docker

# Build
build:
	go build -o bin/server ./cmd/server

# Run development server
run:
	go run ./cmd/server/main.go

# Run tests
test:
	go test -v -race -cover ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f *.out

# Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Download dependencies
deps:
	go mod tidy
	go mod download

# Build Docker image
docker:
	docker build -t dist_task:latest .

# Run with Docker Compose
run-docker:
	docker-compose up -d

# Stop Docker Compose
stop-docker:
	docker-compose down

# View logs
logs:
	docker-compose logs -f dist_task

# Initialize database
init-db:
	mysql -h 127.0.0.1 -P 3306 -u root -proot123 dist_task < migrations/001_init_schema.sql

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  run        - Run development server"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  lint       - Run linter"
	@echo "  deps       - Download dependencies"
	@echo "  docker     - Build Docker image"
	@echo "  run-docker - Run with Docker Compose"
	@echo "  stop-docker - Stop Docker Compose"
	@echo "  logs       - View logs"
	@echo "  init-db    - Initialize database"
	@echo "  help       - Show this help"
