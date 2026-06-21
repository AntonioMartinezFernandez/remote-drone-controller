package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/AntonioMartinezFernandez/remote-drone-controller/cmd/di"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/screen"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/pkg/logger"
)

var (
	CrsfMin            int = 172
	CrsfMid            int = 992
	CrsfMax            int = 1811
	ControlStepPercent int = 5
)

func main() {
	// Initialize Dependencies
	ctx, cancel := di.RootContext()
	defer cancel()
	di := di.InitKeyboardControllerDi(ctx)
	di.CommonServices.Logger.Info(ctx, "starting keyboard controller")

	// Set global variables with values from environment variables
	CrsfMin = di.CommonServices.Config.CsrfChannelValueMin
	CrsfMid = di.CommonServices.Config.CsrfChannelValueMid
	CrsfMax = di.CommonServices.Config.CsrfChannelValueMax
	ControlStepPercent = di.CommonServices.Config.ControlStepPercent

	conn, err := net.Dial("udp", di.CommonServices.Config.UdpReceiverAddr)
	if err != nil {
		di.CommonServices.Logger.Error(ctx, fmt.Sprintf("failed to connect to receiver: %v", err))
		os.Exit(1)
	}
	defer conn.Close()

	droneState := NewDroneState()

	droneUI, err := screen.NewScreen()
	if err != nil {
		di.CommonServices.Logger.Error(ctx, fmt.Sprintf("failed to initialize screen: %v", err))
		os.Exit(1)
	}
	defer droneUI.Finalize()

	done := make(chan struct{})
	go readKeys(droneUI, droneState, done)

	ticker := time.NewTicker(di.CommonServices.Config.TransmitIntervalMs)
	defer ticker.Stop()

	redrawScreen(droneUI, droneState)

	for {
		select {
		case <-done:
			droneState.emergencyStop()
			sendPacket(conn, droneState, di.CommonServices.Logger, ctx)
			return
		case <-ticker.C:
			sendPacket(conn, droneState, di.CommonServices.Logger, ctx)
			redrawScreen(droneUI, droneState)
		}
	}
}

// redrawScreen pulls a consistent snapshot of the drone state and pushes it to the screen
func redrawScreen(scr *screen.Screen, droneState *DroneState) {
	roll, pitch, yaw, throttle, arm := droneState.snapshot()
	scr.Refresh(roll, pitch, yaw, throttle, arm)
}

type DroneState struct {
	mu       sync.Mutex
	roll     int
	pitch    int
	yaw      int
	throttle int
	arm      bool
}

func NewDroneState() *DroneState {
	return &DroneState{roll: 50, pitch: 50, yaw: 50, throttle: 0, arm: false}
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

func (d *DroneState) adjustRoll(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll = clamp(d.roll+delta, 0, 100)
}

func (d *DroneState) adjustPitch(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pitch = clamp(d.pitch+delta, 0, 100)
}

// increaseThrottle raises the throttle and arms the drone if it isn't
// already armed -- see the package-level NOTE for why.
func (d *DroneState) increaseThrottle(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.arm = true
	d.throttle = clamp(d.throttle+delta, 0, 100)
}

func (d *DroneState) decreaseThrottle(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.throttle = clamp(d.throttle-delta, 0, 100)
}

// emergencyStop immediately disarms the drone and zeroes the throttle.
func (d *DroneState) emergencyStop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.throttle = 0
	d.arm = false
}

// snapshot returns a consistent copy of all fields under a single lock,
// so callers (like the redraw loop) never read a half-updated state.
func (d *DroneState) snapshot() (roll, pitch, yaw, throttle int, arm bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.roll, d.pitch, d.yaw, d.throttle, d.arm
}

// percentToCRSF linearly maps a 0-100 percentage to the CRSF tick range
// used by the firmware, where 50% maps to the centered tick value.
func percentToCRSF(percent int) int {
	percent = clamp(percent, 0, 100)
	return CrsfMin + ((CrsfMax-CrsfMin)*percent+50)/100
}

func (d *DroneState) serializePacket() string {
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

func readKeys(droneUI *screen.Screen, droneState *DroneState, done chan<- struct{}) {
	defer close(done)

	for {
		ev := droneUI.NextKey()
		if ev.Quit {
			return
		}

		switch ev.Rune {
		case 'i': // forward
			droneState.adjustPitch(ControlStepPercent)
		case 'k': // back
			droneState.adjustPitch(-ControlStepPercent)
		case 'j': // left
			droneState.adjustRoll(-ControlStepPercent)
		case 'l': // right
			droneState.adjustRoll(ControlStepPercent)
		case 'q': // throttle up
			droneState.increaseThrottle(ControlStepPercent)
		case 'a': // throttle down
			droneState.decreaseThrottle(ControlStepPercent)
		case 'e': // emergency stop
			droneState.emergencyStop()
		}
	}
}

// sendPacket writes the current control packet to w. Taking an io.Writer
// instead of net.Conn keeps this testable with an in-memory buffer.
func sendPacket(w io.Writer, control *DroneState, log logger.Logger, ctx context.Context) {
	if _, err := w.Write([]byte(control.serializePacket())); err != nil {
		log.Error(ctx, fmt.Sprintf("send error: %v", err))
	}
}
