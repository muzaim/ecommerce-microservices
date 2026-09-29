DB_URL=postgres://postgres:postgrespassword@localhost:5432/ecommerce_auth?sslmode=disable

.PHONY: dev build run

dev:
	air

build:
	go build -o bin/api cmd/api/main.go

run: build
	./bin/api

migrate-up:
	migrate -path migrations -database "$(DB_URL)" -verbose up
	
migrate-down:
	migrate -path migrations -database "$(DB_URL)" -verbose down 1