.PHONY: build build-coordinator build-agent build-cli test clean

VERSION ?= 0.1.0
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

build: build-coordinator build-agent build-cli

build-coordinator:
	go build $(LDFLAGS) -o bin/coordinator ./cmd/coordinator

build-agent:
	go build $(LDFLAGS) -o bin/agent ./cmd/agent

build-cli:
	go build $(LDFLAGS) -o bin/mycelctl ./cmd/mycelctl

test:
	go test ./...

clean:
	rm -rf bin/
	go clean
