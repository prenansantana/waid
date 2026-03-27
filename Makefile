BINARY_NAME=waid
MAIN_PATH=./cmd/waid
BUILD_DIR=./dist
DOCKER_IMAGE=ghcr.io/prenansantana/waid

.PHONY: build run test lint migrate docker-build clean help

## build: compile the binary
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## run: run the server
run:
	go run $(MAIN_PATH)

## test: run all tests
test:
	go test -race -count=1 ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## migrate: run database migrations
migrate:
	go run $(MAIN_PATH) migrate

## docker-build: build the Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):latest -f deploy/docker/Dockerfile .

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean ./...

## tidy: tidy go modules
tidy:
	go mod tidy

## help: show this help
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'
