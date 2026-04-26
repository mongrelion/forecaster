.PHONY: build test run clean tidy

build:
	go build -o server ./cmd/server

test:
	go test ./...

run:
	go run ./cmd/server

clean:
	rm -f server

tidy:
	go mod tidy