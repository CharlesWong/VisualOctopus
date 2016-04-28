.PHONY: build proto run test

default: build

proto:
	protoc -I=./model/ --go_out=plugins=grpc:./model/ ./model/*.proto

build: proto 
	GOARCH=amd64 GOOS=linux go build -v -o bin/scraper-amd64-linux scraper.go
	GOARCH=amd64 GOOS=linux go build -v -o bin/teleport-amd64-linux teleport/teleport.go

run: build
	./bin/scraper-amd64-linux

test:
	go test ./...
