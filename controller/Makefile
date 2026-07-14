UDPLISTENER=cmd/udp-listener/main.go
UDP_TEST_URL=127.0.0.1:4210
UDP_REAL_URL=192.168.4.1:4210

.PHONY: listener keyboard hid real-keyboard real-hid list-devices test

listener:
	go run $(UDPLISTENER)

keyboard:
	UDP_RECEIVER_ADDR=$(UDP_TEST_URL) go run cmd/keyboard-controller/main.go

joystick:
	UDP_RECEIVER_ADDR=$(UDP_TEST_URL) go run cmd/hid-controller/main.go

real-keyboard:
	UDP_RECEIVER_ADDR=$(UDP_REAL_URL) go run cmd/keyboard-controller/main.go

real-joystick:
	UDP_RECEIVER_ADDR=$(UDP_REAL_URL) go run cmd/hid-controller/main.go

list-devices:
	@command -v lshid >/dev/null 2>&1 || { echo "Installing lshid..."; go install github.com/sstallion/go-hid/cmd/lshid@latest; }
	lshid

test:
	go test -race -cover ./...
