package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/AntonioMartinezFernandez/remote-drone-controller/cmd/di"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/drone"
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

	droneState := drone.NewDroneState(CrsfMin, CrsfMid, CrsfMax)

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
			droneState.EmergencyStop()
			sendPacket(ctx, conn, droneState, di.CommonServices.Logger)
			return
		case <-ticker.C:
			sendPacket(ctx, conn, droneState, di.CommonServices.Logger)
			redrawScreen(droneUI, droneState)
		}
	}
}

// redrawScreen pulls a consistent snapshot of the drone state and pushes it to the screen
func redrawScreen(scr *screen.Screen, droneState *drone.DroneState) {
	roll, pitch, yaw, throttle, arm := droneState.Snapshot()
	scr.Refresh(roll, pitch, yaw, throttle, arm)
}

func readKeys(droneUI *screen.Screen, droneState *drone.DroneState, done chan<- struct{}) {
	defer close(done)

	for {
		ev := droneUI.NextKey()
		if ev.Quit {
			return
		}

		switch ev.Rune {
		case 'i': // forward
			droneState.AdjustPitch(ControlStepPercent)
		case 'k': // back
			droneState.AdjustPitch(-ControlStepPercent)
		case 'j': // left
			droneState.AdjustRoll(-ControlStepPercent)
		case 'l': // right
			droneState.AdjustRoll(ControlStepPercent)
		case 'q': // throttle up
			droneState.IncreaseThrottle(ControlStepPercent)
		case 'a': // throttle down
			droneState.DecreaseThrottle(ControlStepPercent)
		case 'e': // emergency stop
			droneState.EmergencyStop()
		}
	}
}

// sendPacket writes the current control packet to w. Taking an io.Writer
// instead of net.Conn keeps this testable with an in-memory buffer.
func sendPacket(ctx context.Context, w io.Writer, control *drone.DroneState, log logger.Logger) {
	if _, err := w.Write([]byte(control.SerializePacket())); err != nil {
		log.Error(ctx, fmt.Sprintf("send error: %v", err))
	}
}
