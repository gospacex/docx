.PHONY: up down test build clean verify fmt vet lint

up:
	docker-compose up -d

down:
	docker-compose down

build:
	go build ./...

test:
	go test -race -count=1 ./...

verify:
	go mod verify

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	docker-compose down -v
