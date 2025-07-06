# Simple task runner for dj-set-downloader

.PHONY: build test run vet fmt tidy

build:
	go build -o bin/djset ./cmd

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

run:
	go run ./cmd