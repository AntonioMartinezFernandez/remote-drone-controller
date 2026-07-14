UDPLISTENER=cmd/udp-listener/main.go

.PHONY: listener keyboard hid list-devices test

listener:
	go run $(UDPLISTENER)

keyboard:
	go run cmd/keyboard-controller/main.go

hid:
	go run cmd/hid-controller/main.go

list-devices:
	@command -v lshid >/dev/null 2>&1 || { echo "Installing lshid..."; go install github.com/sstallion/go-hid/cmd/lshid@latest; }
	lshid

test:
	go test -race -cover ./...
