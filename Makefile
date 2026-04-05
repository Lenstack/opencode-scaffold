BINARY     := ocs
VERSION    := 1.0.0
BUILD_DIR  := ./dist
LDFLAGS    := -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all build install clean test tidy

all: build

## Download dependencies
tidy:
	go mod tidy

## Build for current platform
build: tidy
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./main.go
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

## Install to $GOPATH/bin (or ~/go/bin)
install: tidy
	go install $(LDFLAGS) ./...
	@echo "Installed: $(BINARY)"

## Build for all platforms
build-all: tidy
	mkdir -p $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64   ./main.go
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64   ./main.go
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64  ./main.go
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64  ./main.go
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./main.go
	@echo "Cross-compiled binaries in $(BUILD_DIR)/"

## Run tests
test:
	go test ./... -v -race

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean ./...

## Quick scaffold test (creates scaffold in /tmp/test-project)
demo:
	mkdir -p /tmp/test-project
	cd /tmp/test-project && $(BUILD_DIR)/$(BINARY) init --force --verbose
	@echo "\nScaffold demo at /tmp/test-project"
