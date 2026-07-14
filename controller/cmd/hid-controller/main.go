package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AntonioMartinezFernandez/remote-drone-controller/cmd/di"
	deviceselector "github.com/AntonioMartinezFernandez/remote-drone-controller/internal/device_selector"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/drone"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/input"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/input/ui"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/pkg/logger"

	tea "charm.land/bubbletea/v2"
)

func main() {
	// Initialize Dependencies
	ctx, cancel := di.RootContext()
	defer cancel()
	di := di.InitKeyboardControllerDi(ctx)
	di.CommonServices.Logger.Info(ctx, "starting drone controller")

	conn, err := net.Dial("udp", di.CommonServices.Config.UdpReceiverAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to receiver: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	droneState := drone.NewDroneState(
		di.CommonServices.Config.CsrfChannelValueMin,
		di.CommonServices.Config.CsrfChannelValueMid,
		di.CommonServices.Config.CsrfChannelValueMax,
	)

	selectedDevice, err := deviceselector.DeviceSelection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error selecting device: %v\n", err)
		os.Exit(1)
	}

	config := input.GetConfig()
	reader, err := input.Open(selectedDevice.Path, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open device: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	info, err := reader.DeviceInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting device info: %v\n", err)
		os.Exit(1)
	}

	deviceInfo := input.DeviceInfo{
		Product: info.ProductStr,
		Vendor:  info.VendorID,
		PID:     info.ProductID,
		Path:    info.Path,
	}

	uiModel := ui.NewModel(config, deviceInfo)
	tui := tea.NewProgram(uiModel)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		droneState.EmergencyStop()
		sendPacket(ctx, conn, droneState, di.CommonServices.Logger)
		os.Exit(0)

	}()

	// Start the main loop
	go mainLoop(ctx, di.CommonServices.Logger, conn, reader, tui, di.CommonServices.Config.TransmitIntervalMs, droneState)

	// Start the Terminal User Interface (TUI)
	if _, err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
		os.Exit(1)
	}
}

// mainLoop continuously reads input from the HID device, sends the state to the TUI, and controls the drone at regular intervals
func mainLoop(
	ctx context.Context,
	log logger.Logger,
	conn net.Conn,
	reader *input.HidReader,
	tui *tea.Program,
	transmitInterval time.Duration,
	droneState *drone.DroneState,
) {
	ticker := time.NewTicker(transmitInterval)
	defer ticker.Stop()

	for range ticker.C {
		state, err := reader.Read()

		if err != nil {
			tui.Send(ui.StateMsg{State: nil})
			return
		}
		tui.Send(ui.StateMsg{State: state})

		//! TODO: UPDATE REAL DRONE STATE FROM HID INPUT
		if state != nil {
			roll := state.GetAxisValue(input.AxisRoll)
			pitch := state.GetAxisValue(input.AxisPitch)
			yaw := state.GetAxisValue(input.AxisYaw)
			throttle := state.GetAxisValue(input.AxisThrottle)
			if state.IsButtonPressed("BTN 0") {
				droneState.SetArm(true)
			}
			if state.IsButtonPressed("BTN 1") {
				droneState.SetArm(false)
			}
			droneState.SetFromPercentages(roll, pitch, yaw, throttle)
		}

		sendPacket(ctx, conn, droneState, log)
	}
}

// sendPacket writes the current control packet to w. Taking an io.Writer
// instead of net.Conn keeps this testable with an in-memory buffer.
func sendPacket(ctx context.Context, w io.Writer, control *drone.DroneState, log logger.Logger) {
	if _, err := w.Write([]byte(control.SerializePacket())); err != nil {
		log.Error(ctx, fmt.Sprintf("send error: %v", err))
	}
}
