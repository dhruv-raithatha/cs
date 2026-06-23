BINARY     = cs
CMD        = ./cmd/cs
INSTALL_DIR = $(HOME)/.local/bin

.PHONY: build install test test-integration lint coverage cross-compile hooks

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(CMD)
	@codesign --sign - --force $(BINARY) 2>/dev/null || true

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@codesign --sign - --force $(INSTALL_DIR)/$(BINARY) 2>/dev/null || true
	@echo "Installed $(BINARY) → $(INSTALL_DIR)/$(BINARY)"

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
