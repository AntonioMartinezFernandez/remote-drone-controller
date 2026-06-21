// Command dronectl provides a simple keyboard-driven control loop for the
// ESP32 WiFi-to-CRSF bridge described in the project README.
//
// Controls:
//
//	i / k   pitch forward / back
//	j / l   roll left / right
//	q / a   throttle up / down (arms the drone on first increase)
//	e       emergency stop (throttle to 0, disarm)
//	Ctrl+C  quit
//
// NOTE: the requirements for this program specify movement and throttle
// keys but no dedicated arm/disarm key. Since there's no way to raise
// throttle usefully on a disarmed drone, this program arms automatically
// the first time throttle is increased (q). Emergency (e) is the only way
// to disarm. If you'd rather have an explicit arm key, that's a one-line
// change in increaseThrottle below.
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// CRSF channel tick range used by the ESP32 firmware (matches its
// CRSF_CHANNEL_VALUE_MIN/MID/MAX).
const (
	crsfMin = 172
	crsfMid = 992
	crsfMax = 1811
)

const (
	espAddr = "192.168.4.1:4210" // ESP32 access point + UDP control port
	// espAddr      = "127.0.0.1:4210" // ESP32 access point + UDP control port
	sendInterval  = 20 * time.Millisecond
	controlStep   = 5 // percentage points adjusted per keypress
	firstThrottle = controlStep * 8
)

// DroneControl holds the current commanded state of the drone as
// human-friendly percentages rather than raw CRSF ticks.
//
// Roll, Pitch and Yaw are 0-100, where 50 is centered/neutral and 0/100 are
// full deflection in either direction. Throttle is 0 (idle) to 100 (full).
// All access goes through the methods below, which are safe to call
// concurrently from the key-reader goroutine and the send loop.
type DroneControl struct {
	mu       sync.Mutex
	roll     int
	pitch    int
	yaw      int
	throttle int
	arm      bool
}

// newDroneControl returns a DroneControl in a safe starting state: sticks
// centered, throttle at idle, disarmed.
func newDroneControl() *DroneControl {
	return &DroneControl{roll: 50, pitch: 50, yaw: 50, throttle: 0, arm: false}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (d *DroneControl) adjustRoll(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll = clamp(d.roll+delta, 0, 100)
}

func (d *DroneControl) adjustPitch(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pitch = clamp(d.pitch+delta, 0, 100)
}

// increaseThrottle raises the throttle and arms the drone if it isn't
// already armed -- see the package-level NOTE for why.
func (d *DroneControl) increaseThrottle(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.arm = true
	d.throttle = clamp(d.throttle+delta, 0, 100)
}

func (d *DroneControl) decreaseThrottle(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.throttle = clamp(d.throttle-delta, 0, 100)
}

// emergencyStop immediately disarms the drone and zeroes the throttle.
func (d *DroneControl) emergencyStop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.throttle = 0
	d.arm = false
}

// percentToCRSF linearly maps a 0-100 percentage to the CRSF tick range
// used by the firmware, where 50% maps to the centered tick value.
//
// Rounds to the nearest tick (rather than truncating) so that the
// boundaries map exactly: 0->crsfMin, 50->crsfMid, 100->crsfMax.
func percentToCRSF(percent int) int {
	percent = clamp(percent, 0, 100)
	return crsfMin + ((crsfMax-crsfMin)*percent+50)/100
}

// packet renders the current state as the comma-separated string expected
// by the ESP32 firmware's UDP listener.
//
// IMPORTANT: the firmware parses "roll,pitch,throttle,yaw,arm" -- that
// wire order does not match this struct's field order, so don't reorder
// this without also updating the firmware (or vice versa).
func (d *DroneControl) packet() string {
	d.mu.Lock()
	defer d.mu.Unlock()

	armBit := 0
	if d.arm {
		armBit = 1
	}

	return fmt.Sprintf("%d,%d,%d,%d,%d",
		percentToCRSF(d.roll),
		percentToCRSF(d.pitch),
		percentToCRSF(d.throttle),
		percentToCRSF(d.yaw),
		armBit,
	)
}

// readKeys reads raw single-byte keypresses from r and applies them to
// control until r returns an error (EOF, closed stdin, ...) or Ctrl+C is
// pressed, then closes done.
//
// Taking an io.Reader instead of hardcoding os.Stdin keeps this testable:
// tests can feed it a strings.Reader instead of needing a real terminal.
func readKeys(r io.Reader, control *DroneControl, done chan<- struct{}) {
	defer close(done)

	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if err != nil || n == 0 {
			return
		}

		switch buf[0] {
		case 'i': // forward
			control.adjustPitch(controlStep)
		case 'k': // back
			control.adjustPitch(-controlStep)
		case 'j': // left
			control.adjustRoll(-controlStep)
		case 'l': // right
			control.adjustRoll(controlStep)
		case 'q': // throttle up
			control.increaseThrottle(controlStep)
		case 'a': // throttle down
			control.decreaseThrottle(controlStep)
		case 'e': // emergency stop
			control.emergencyStop()
		case 0x03: // Ctrl+C -- raw mode disables normal SIGINT handling
			return
		}
	}
}

// sendPacket writes the current control packet to w. Taking an io.Writer
// instead of net.Conn keeps this testable with an in-memory buffer.
func sendPacket(w io.Writer, control *DroneControl) {
	if _, err := w.Write([]byte(control.packet())); err != nil {
		log.Printf("send error: %v", err)
	}
}

func printControls() {
	fmt.Println("Drone control - keys:")
	fmt.Println("  i/k  forward/back   j/l  left/right")
	fmt.Println("  q/a  throttle up/down (q also arms)")
	fmt.Println("  e    EMERGENCY STOP")
	fmt.Println("  Ctrl+C  quit")
}

func main() {
	conn, err := net.Dial("udp", espAddr)
	if err != nil {
		log.Fatalf("failed to connect to ESP32: %v", err)
	}
	defer conn.Close()

	stdinFd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		log.Fatalf("failed to set terminal to raw mode: %v", err)
	}
	defer term.Restore(stdinFd, oldState)

	control := newDroneControl()
	done := make(chan struct{})
	go readKeys(os.Stdin, control, done)

	printControls()

	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			// Make sure we leave the drone disarmed before exiting.
			control.emergencyStop()
			sendPacket(conn, control)
			return
		case <-ticker.C:
			sendPacket(conn, control)
		}
	}
}
