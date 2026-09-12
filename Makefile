run:
	go run ./cmd/api

test:
	go test ./...

test-integration:
	go test -v -tags=integration ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run

build:
	go build -o bin/api ./cmd/api

docker-up:
	docker-compose -f docker/docker-compose.yml up -d

docker-down:
	docker-compose -f docker/docker-compose.yml down
