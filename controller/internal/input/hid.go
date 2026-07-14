package input

import (
	"github.com/sstallion/go-hid"
)

type Reader interface {
	Read() (*InputState, error)
	Close()
}

type HidReader struct {
	device *hid.Device
	config Config
}

func Open(path string, config Config) (*HidReader, error) {
	device, err := hid.OpenPath(path)
	if err != nil {
		return nil, err
	}
	if _, err = device.GetDeviceInfo(); err != nil {
		device.Close()
		return nil, err
	}
	return &HidReader{
		device: device,
		config: config,
	}, nil
}

func (r *HidReader) Read() (*InputState, error) {
	buffer := make([]byte, 64)
	n, err := r.device.ReadWithTimeout(buffer, 50)
	if err != nil {
		if err == hid.ErrTimeout {
			return nil, nil
		}
		return nil, err
	}
	if n < 10 {
		return nil, nil
	}

	state := &InputState{
		Axes:    make([]Axis, len(r.config.Axes)),
		Buttons: make([]Button, len(r.config.Buttons)),
	}

	for i, ax := range r.config.Axes {
		raw := extractBits(buffer, ax.BitStart, ax.BitLen)
		maxVal := uint16((1 << ax.BitLen) - 1)
		if ax.Inverted {
			raw = maxVal - raw
		}
		state.Axes[i] = Axis{
			Name:  ax.Name,
			Value: float64(raw) / float64(maxVal) * 100,
		}
	}

	for i, btn := range r.config.Buttons {
		state.Buttons[i] = Button{
			Bit:     btn.Bit,
			Name:    btn.Name,
			Pressed: isButtonPressed(buffer, btn.Bit),
		}
	}

	return state, nil
}

func (r *HidReader) Close() {
	if r.device != nil {
		r.device.Close()
	}
}

func (r *HidReader) DeviceInfo() (*hid.DeviceInfo, error) {
	return r.device.GetDeviceInfo()
}

func extractBits(data []byte, bitStart, bitLen int) uint16 {
	byteOffset := bitStart / 8
	bitOffset := bitStart % 8
	if byteOffset+1 >= len(data) {
		return 0
	}
	var raw uint16
	if bitOffset+bitLen <= 8 {
		mask := (uint16(1) << bitLen) - 1
		raw = (uint16(data[byteOffset]) >> bitOffset) & mask
	} else {
		low := uint16(data[byteOffset]) >> bitOffset
		high := uint16(data[byteOffset+1]) << (8 - bitOffset)
		mask := (uint16(1) << bitLen) - 1
		raw = (low | high) & mask
	}
	return raw
}

func isButtonPressed(data []byte, bit int) bool {
	globalBit := 64 + bit
	byteIdx := globalBit / 8
	bitIdx := globalBit % 8
	if byteIdx >= len(data) {
		return false
	}
	return (data[byteIdx] & (1 << bitIdx)) != 0
}
