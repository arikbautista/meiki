VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: build clean release install

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o meiki ./cmd/meiki

install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" ./cmd/meiki

release: clean
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output="dist/meiki-$${os}-$${arch}"; \
		echo "Building $${output}..."; \
		CGO_ENABLED=0 GOOS=$${os} GOARCH=$${arch} go build -ldflags "$(LDFLAGS)" -o $${output} ./cmd/meiki; \
	done
	@cd dist && sha256sum meiki-* > checksums.txt 2>/dev/null || shasum -a 256 meiki-* > checksums.txt

clean:
	rm -rf dist
