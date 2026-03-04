APP_NAME = mm-guest-audit
VERSION := $(shell cat VERSION)
LDFLAGS = -ldflags="-X main.Version=$(VERSION)"

.PHONY: fmt imports staticcheck vet build build-all clean pre-build-check test test-cover

build:
	go build $(LDFLAGS) -o bin/$(APP_NAME)_local .

pre-build-check:
	@echo "Fetching remote tags..."
	@git fetch --tags
	@if git tag -l "$(VERSION)" | grep -q .; then \
		echo "Error: Tag $(VERSION) already exists. Please update the VERSION file."; \
		exit 1; \
	else \
		echo "Tag $(VERSION) does not exist. Proceeding with the build."; \
	fi

fmt:
	@echo "Running gofmt..."
	@gofmt -d -e -s . 2>&1 | read; if [ $$? == 0 ]; then echo "Code is not formatted, please run 'gofmt -w .'" && exit 1; fi

imports:
	@echo "Running goimports..."
	@goimports -l . 2>&1 | read; if [ $$? == 0 ]; then echo "Imports are not properly organized, please run 'goimports -w .'" && exit 1; fi

staticcheck:
	@echo "Running staticcheck..."
	@staticcheck ./... || (echo "Staticcheck identified problems" && exit 1)

vet:
	@echo "Running go vet..."
	@go vet ./... || (echo "Go vet identified problems" && exit 1)

build-all: pre-build-check fmt imports staticcheck vet
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)_linux_amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)_linux_arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)_macos_intel .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)_macos_apple .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)_windows.exe .

test:
	go test ./... -v

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	@echo "Cleaning up..."
	rm -f bin/$(APP_NAME)_*
