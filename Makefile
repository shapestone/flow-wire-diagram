BINARY    = wire-fix
MODULE    = github.com/shapestone/flow-wire-diagram
BUILD_DIR = bin
CMD       = ./cmd/wire-fix
VERSION   = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   = -ldflags "-X main.version=$(VERSION)"

.PHONY: all build test lint clean install run

all: build

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

install:
	go install $(LDFLAGS) $(CMD)

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run against a file: make run FILE=testdata/nested_box.md
run:
	go run $(CMD) $(FILE)
