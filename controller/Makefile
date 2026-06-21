APP=cmd/keyboard-controller/main.go
UDPLISTENER=cmd/udp-listener/main.go
BIN=bin/keyboard-controller

.PHONY: all build run race

all: build

build:
	go build -o $(BIN) $(APP)

run:
	go run $(APP)

test:
	go test -race -cover ./...

listener:
	go run $(UDPLISTENER)