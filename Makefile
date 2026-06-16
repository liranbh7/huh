.PHONY: build test lint

SRC := src

build:
	cd $(SRC) && go build -o ../bin/huh .

test:
	cd $(SRC) && go test ./...

lint:
	cd $(SRC) && go vet ./...

all:
	$(MAKE) build
	$(MAKE) test
	$(MAKE) lint