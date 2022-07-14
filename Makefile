BIN_DIR = bin

test:
	go test ./...

build:
	go build -o $(BIN_DIR)/eve

.PHONY: build
