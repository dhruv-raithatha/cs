BINARY     = cs
CMD        = ./cmd/cs
INSTALL_DIR = $(HOME)/.local/bin

.PHONY: build install test test-integration lint coverage cross-compile hooks

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(CMD)

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) → $(INSTALL_DIR)/$(BINARY)"
	@if [ -d "$(shell go env GOPATH)/bin" ] && [ -f "$(shell go env GOPATH)/bin/$(BINARY)" ]; then \
		cp $(BINARY) "$(shell go env GOPATH)/bin/$(BINARY)"; \
		echo "Updated  $(BINARY) → $(shell go env GOPATH)/bin/$(BINARY)"; \
	fi

hooks:
	@cp .githooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed"

test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./...

lint:
	golangci-lint run

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

cross-compile:
	GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -o $(BINARY)-linux-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o $(BINARY)-darwin-arm64 $(CMD)
