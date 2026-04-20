BINARY = cs
CMD = ./cmd/cs

.PHONY: build test test-integration lint coverage cross-compile

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(CMD)

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
