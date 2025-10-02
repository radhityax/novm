src = src/novm.go src/cli.go src/front.go
target = novm
db = novm.db
flags = -a -gcflags=all="-l -B" -ldflags="-s -w" -trimpath

build:
	go build $(src)

optimize:
#	go build $(flags) $(src)
	CGO_ENABLED=1 CGO_CFLAGS="-O3" CGO_LDFLAGS="-static" go build -a -gcflags=all="-l -B" -ldflags="-s -w -extldflags '-static -s -w'" -trimpath $(src)

init:
	go mod init novm
	go get github.com/mattn/go-sqlite3
	go get golang.org/x/crypto/bcrypt
	go get github.com/yuin/goldmark

run:
	go run $(src)

clean:
	rm $(db) $(target)
