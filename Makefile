APP_NAME = succulent-cli

vet:
	go vet ./...

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(APP_NAME)

lint:
	golangci-lint run ./...

test:
	go test ./... -v

clean:
	rm -f $(APP_NAME)

.PHONY: vet build lint test clean
