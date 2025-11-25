.PHONY: build run test clean docker-build docker-up docker-down

# Build the simulator
build:
	go build -o bin/simulator main.go

# Run the simulator
run: build
	./bin/simulator

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Initialize go modules
init:
	go mod download
	go mod tidy

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up

docker-down:
	docker-compose down

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	go vet ./...

# Install dependencies
deps:
	go get gopkg.in/yaml.v3

# Full build and test
all: clean deps build test