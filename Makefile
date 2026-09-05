run:
	go run ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run

build:
	go build -o bin/api ./cmd/api
