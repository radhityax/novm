src = src/novm.go
target = novm

build:
	go build $(src)
run:
	go run $(src)

clean:
	rm $(target)
