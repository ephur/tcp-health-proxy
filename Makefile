.PHONY: test
export GOBIN := $(PWD)/bin
export PATH := $(GOBIN):$(PATH)
export INSTALL_FLAG=
VERSION?=$(shell git describe --all --dirty | cut -d / -f2,3,4)

# Determine which OS.
OS?=$(shell uname -s | tr A-Z a-z)

default: build

dep:
	GO111MODULE=on go mod tidy && GO111MODULE=on go mod vendor

build:
	CGO_ENABLED=0 GOOS=$(OS) go build $(INSTALL_FLAG) -o ./bin/tcp-health-proxy

build-linux:
	CGO_ENABLED=0 GOOS=linux go build $(INSTALL_FLAG) -o ./bin/tcp-health-proxy

test:
	go test -timeout 60s -v . -coverprofile=coverage.txt -covermode=atomic ; go tool cover -html=coverage.txt -o coverage.html

clean:
	@go clean
