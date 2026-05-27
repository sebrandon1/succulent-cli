APP_NAME = succulent-cli

vet:
	go vet ./...

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(APP_NAME)

install: build
	cp $(APP_NAME) $(GOPATH)/bin/$(APP_NAME) 2>/dev/null || cp $(APP_NAME) $(HOME)/go/bin/$(APP_NAME)

lint:
	golangci-lint run ./...

test:
	go test ./... -v

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -f $(APP_NAME) coverage.out coverage.html

.PHONY: vet build install lint test coverage coverage-html fmt clean
