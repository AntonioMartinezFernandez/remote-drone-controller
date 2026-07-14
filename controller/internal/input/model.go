package input

type Axis struct {
	Name  string
	Value float64 // 0-100
}

type Button struct {
	Bit     int
	Name    string
	Pressed bool
}

type DeviceInfo struct {
	Product string
	Vendor  uint16
	PID     uint16
	Path    string
}

type InputState struct {
	Axes   []Axis
	Buttons []Button
	Device DeviceInfo
}
