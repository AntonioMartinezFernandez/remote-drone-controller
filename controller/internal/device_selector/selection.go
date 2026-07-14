package deviceselector

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sstallion/go-hid"
)

func DeviceSelection() (*hid.DeviceInfo, error) {
	devices, err := enumerateDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enumerating devices: %v\n", err)
		return nil, err
	}
	if len(devices) == 0 {
		fmt.Println("No HID controller found")
		return nil, err
	}

	fmt.Printf("Available HID controllers:\n\n")

	for i, dev := range devices {
		fmt.Printf("  [%d] %s (VID: 0x%04X, PID: 0x%04X) — %s\n",
			i, dev.ProductStr, dev.VendorID, dev.ProductID, dev.Path,
		)
	}

	fmt.Print("\nSelect a device (number): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return nil, err
	}

	input = strings.TrimSpace(input)
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 0 || idx >= len(devices) {
		fmt.Println("Invalid selection")
		return nil, fmt.Errorf("invalid selection")
	}

	// Clear the terminal screen
	fmt.Print("\033[H\033[2J")

	return devices[idx], nil
}

func enumerateDevices() ([]*hid.DeviceInfo, error) {
	var devices []*hid.DeviceInfo
	err := hid.Enumerate(hid.VendorIDAny, hid.ProductIDAny, func(info *hid.DeviceInfo) error {
		if info.UsagePage == 0x01 && (info.Usage == 0x04 || info.Usage == 0x05) {
			devices = append(devices, info)
		}
		return nil
	})
	return devices, err
}
