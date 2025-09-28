src = src/novm.go
target = novm
db = novm.db

build:
	go build $(src)

init:
	go mod init novm
	go get github.com/mattn/go-sqlite3
	go get golang.org/x/crypto/bcrypt

run:
	go run $(src)

clean:
	rm $(db) $(target)
