NAME := lucinexus
VERSION := 1.0.0
BUILD_TIME := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X 'github.com/lucinexus/lucinexus/internal/config.Version=$(VERSION)' -X 'github.com/lucinexus/lucinexus/internal/config.BuildTime=$(BUILD_TIME)'

.PHONY: all clean test build-all build-mips build-mipsel build-arm build-arm64 build-amd64

all: test build-amd64

test:
	go test -v -race -coverprofile=coverage.out ./...

build-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(NAME)-amd64 ./cmd/lucinexus

build-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(NAME)-arm ./cmd/lucinexus

build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(NAME)-arm64 ./cmd/lucinexus

build-mips:
	CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(NAME)-mips ./cmd/lucinexus

build-mipsel:
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(NAME)-mipsel ./cmd/lucinexus

build-all: build-amd64 build-arm build-arm64 build-mips build-mipsel

clean:
	rm -rf bin/ coverage.out dist/
