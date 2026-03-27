VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DEV_BIN := $(HOME)/.local/bin

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o koll ./cmd/koll

install: build
	sudo cp koll /usr/local/bin/koll

dev: build
	@mkdir -p $(DEV_BIN)
	ln -sf $(CURDIR)/koll $(DEV_BIN)/koll-dev
	@echo "koll-dev -> $(CURDIR)/koll"
	@echo "Make sure $(DEV_BIN) is in your PATH"

.PHONY: build install dev
