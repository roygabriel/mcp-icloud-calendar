VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY_NAME = mcp-icloud-calendar
LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: build test lint clean docker run

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) .

test:
	go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME) coverage.out

docker:
	docker build --build-arg VERSION=$(VERSION) -t $(BINARY_NAME):$(VERSION) .

run: build
	./$(BINARY_NAME)
