src = src/novm.go src/cli.go
target = novm
db = novm.db
flags = -a -gcflags=all="-l -B" -ldflags="-s -w" -trimpath

build:
	go build $(flags) $(src)

init:
	go mod init novm
	go get github.com/mattn/go-sqlite3
	go get golang.org/x/crypto/bcrypt
	go get github.com/yuin/goldmark

run:
	go run $(src)

clean:
	rm $(db) $(target)
