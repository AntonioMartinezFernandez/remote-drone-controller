package input

type AxisName string

const (
	AxisRoll     AxisName = "Roll"
	AxisPitch    AxisName = "Pitch"
	AxisYaw      AxisName = "Yaw"
	AxisThrottle AxisName = "Throttle"
)

type Axis struct {
	Name  AxisName
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
	Axes    []Axis
	Buttons []Button
	Device  DeviceInfo
}

func (s *InputState) GetAxisValue(name AxisName) float64 {
	for _, axis := range s.Axes {
		if axis.Name == name {
			return axis.Value
		}
	}
	return 0
}

func (s *InputState) IsButtonPressed(name string) bool {
	for _, button := range s.Buttons {
		if button.Name == name {
			return button.Pressed
		}
	}
	return false
}
