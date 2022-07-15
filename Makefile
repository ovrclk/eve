BIN_DIR = bin

test:
	go test ./...

build:
	go build -o $(BIN_DIR)/eve

install:
	go install 

.PHONY: build test install