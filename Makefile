.PHONY: build install test clean

build:
	go build -o bin/agent-sh ./cmd/agent-sh

install:
	go install ./cmd/agent-sh

test:
	go test ./...

clean:
	rm -rf bin/
