.PHONY: dev build run

dev:
	air

build:
	go build -o bin/api cmd/api/main.go

run: build
	./bin/api
