.PHONY: build test lint

build:
	go build -o bin/huh .

test:
	go test ./...

lint:
	go vet ./...
