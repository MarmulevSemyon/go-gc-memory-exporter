APP_NAME := gc-mem-exporter
BIN_DIR  := bin
CMD_DIR  := ./cmd/gc-mem-exporter
IMAGE    := gc-mem-exporter:latest

GO     := go
GOFMT  := gofmt
GOLINT := golint

.PHONY: all fmt vet lint test build run clean docker-build docker-run docker-up docker-down docker-logs docker-restart

all: fmt vet test build

fmt:
	$(GOFMT) -w ./cmd ./internal

vet: fmt
	$(GO) vet ./...

lint:
	$(GOLINT) ./...

test:
	$(GO) test ./...

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)

run:
	ADDR=:8080 GC_PERCENT=100 $(GO) run $(CMD_DIR)

clean:
	rm -rf $(BIN_DIR)

docker-build:
	docker build -t $(IMAGE) .

docker-run: docker-build
	docker run --rm -p 8080:8080 -e ADDR=:8080 -e GC_PERCENT=100 $(IMAGE)

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f app

docker-restart:
	docker compose down
	docker compose up --build -d
