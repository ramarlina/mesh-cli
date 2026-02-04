.PHONY: build run test clean lint fmt

build:
	go build -o bin/mesh ./cmd/mesh

run:
	go run ./cmd/mesh

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-e2e: build
	./scripts/e2e-happy-path.sh

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

lint:
	$(shell go env GOPATH)/bin/golangci-lint run ./...

fmt:
	go fmt ./...
