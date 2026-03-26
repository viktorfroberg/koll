VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o koll ./cmd/koll

install: build
	cp koll /usr/local/bin/koll

.PHONY: build install
