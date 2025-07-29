.PHONY: all tidy test

all: tidy test
	go run .

tidy:
	go mod tidy

test:
	go test ./...
