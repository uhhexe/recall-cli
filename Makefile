.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/recall$(BIN_EXT) ./cmd/recall

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/recall

clean:
	rm -rf bin/
