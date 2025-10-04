src = src/novm.go src/cli.go src/front.go
target = novm
db = novm.db

base_flags = -tags "osusergo,netgo" -trimpath -buildvcs=false -ldflags="-s -w -buildid= -extldflags '-static -s -w'"
cgo_flags = CGO_ENABLED=1 CGO_CFLAGS="-O3 -march=native -mtune=native -pipe -fomit-frame-pointer" CGO_LDFLAGS="-static -s -w"

ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
    GOARCH = amd64
else ifeq ($(ARCH),aarch64)
    GOARCH = arm64
else
    GOARCH = $(ARCH)
endif

all:
	@echo "Building for $(GOARCH)..."
	$(cgo_flags) GOOS=linux GOARCH=$(GOARCH) go build $(base_flags) -o $(target) $(src)
	@if command -v strip >/dev/null 2>&1; then strip $(target); fi
	@if command -v upx >/dev/null 2>&1; then echo "Compressing with UPX..."; upx --lzma --best $(target) >/dev/null; fi
	@echo "Build complete: $(target)"

run:
	go run $(src)

init:
	go mod init novm
	go get github.com/mattn/go-sqlite3
	go get golang.org/x/crypto/bcrypt
	go get github.com/yuin/goldmark

clean:
	rm -f $(db) $(target)

