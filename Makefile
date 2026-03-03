BINARY    = wire-fix
MODULE    = github.com/shapestone/flow-wire-diagram
BUILD_DIR = bin
CMD       = ./cmd/wire-fix

.PHONY: all build test lint clean install run

all: build

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

install:
	go install $(CMD)

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run against a file: make run FILE=testdata/nested_box.md
run:
	go run $(CMD) $(FILE)
