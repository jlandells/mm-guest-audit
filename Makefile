BINARY  = mm-guest-audit
VERSION ?= dev
LDFLAGS = -ldflags="-X main.version=$(VERSION)"

.PHONY: build build-all test test-cover lint clean

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY) .

build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe .

test:
	CGO_ENABLED=0 go test ./... -v

test-cover:
	CGO_ENABLED=0 go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

lint:
	go vet ./...

clean:
	rm -rf bin/ coverage.out
