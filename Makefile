.PHONY:
.SILENT:

build:
	go build -o ./.bin/bot cmd/app/main.go

run: build
	./.bin/bot

run-dev:
	go run cmd/app/main.go

test:
	go test -v ./...