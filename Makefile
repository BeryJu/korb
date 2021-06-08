.PHONY: build mover

build:
	go build -v -o bin/korb

build-final:
	GOOS=linux GOARCH=arm go build -v -o bin/korb-linux-arm
	GOOS=linux GOARCH=arm64 go build -v -o bin/korb-linux-arm64
	GOOS=linux GOARCH=amd64 go build -v -o bin/korb-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -v -o bin/korb-darwin-amd64

all: build
