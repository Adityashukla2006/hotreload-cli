.PHONY: build test clean demo

build:
	go build -o ./bin/hotreload .

test:
	go test ./...

clean:
	rm -rf ./bin

demo: build
	go build -o ./bin/testserver ./testserver
	./bin/hotreload --root ./testserver --build "go build -o ./bin/testserver ./testserver" --exec "./bin/testserver"

# Windows: use backslashes for paths in --exec
demo-windows: build
	go build -o ./bin/testserver.exe ./testserver
	./bin/hotreload.exe --root ./testserver --build "go build -o ./bin/testserver.exe ./testserver" --exec "bin\\testserver.exe"

install:
	go install .
