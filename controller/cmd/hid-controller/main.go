package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	deviceselector "github.com/AntonioMartinezFernandez/remote-drone-controller/internal/device_selector"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/input"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/input/ui"

	tea "charm.land/bubbletea/v2"
)

func main() {
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
		os.Exit(0)

	}()

	// Start the main loop
	go mainLoop(reader, tui)

	// Start the Terminal User Interface (TUI)
	if _, err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
		os.Exit(1)
	}
}

// mainLoop continuously reads input from the HID device and sends the state to the TUI.
func mainLoop(reader *input.HidReader, tui *tea.Program) {
	for {
		state, err := reader.Read()
		if err != nil {
			tui.Send(ui.StateMsg{State: nil})
			return
		}
		tui.Send(ui.StateMsg{State: state})
	}
}
