.PHONY: build run test docker-build docker-up docker-prod migrate swagger

build:
	go build -o bin/platform cmd/server/main.go

run:
	go run cmd/server/main.go

test:
	go test ./...

docker-build:
	docker build -t platform2:latest .

docker-up:
	docker-compose up -d

docker-prod:
	docker-compose -f docker-compose.prod.yml up -d

migrate:
	psql $(DATABASE_URL) -f projects/strike2/migrations/001_initial.sql

swagger:
	swag init -g cmd/server/main.go
