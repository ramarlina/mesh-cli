.PHONY: build run test clean lint fmt

build:
	go build -o bin/msh ./cmd/msh

run:
	go run ./cmd/msh

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
