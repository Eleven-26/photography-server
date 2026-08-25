.PHONY: run build test tidy docker-up docker-down docker-build

run:
	go run ./cmd/server -c config/config.yaml

build:
	go build -o bin/photography-server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
