BINARY=imagine
PKG=./cmd/imagine
DIST_DIR=dist

.PHONY: all build test sync-obsidian build-mac build-all install clean

all: build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) $(PKG)

test:
	go test -v ./...

# Re-import progressions tagged [music, chords] from Obsidian and update the embedded JSON seed
sync-obsidian: build
	./$(BINARY) import-obsidian /home/james/brain
	./$(BINARY) export-seed pkg/db/obsidian_seed.json
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) $(PKG)

# Cross-compile standalone zero-dependency binaries for macOS (both x86_64 Mac Mini and Apple Silicon)
build-mac:
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-darwin-amd64 $(PKG)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-darwin-arm64 $(PKG)
	@ls -lh $(DIST_DIR)/$(BINARY)-darwin-*

build-all: build-mac
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-linux-amd64 $(PKG)

install:
	CGO_ENABLED=0 go install $(PKG)

clean:
	rm -rf $(DIST_DIR) $(BINARY)
