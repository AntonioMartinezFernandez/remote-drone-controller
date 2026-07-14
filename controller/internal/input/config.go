package input

import "time"

type AxisConfig struct {
	Name     AxisName
	BitStart int
	BitLen   int
	Inverted bool
}

type ButtonConfig struct {
	Bit  int
	Name string
}

type Config struct {
	Axes        []AxisConfig
	Buttons     []ButtonConfig
	ReadTimeout time.Duration
}

func GetConfig() Config {
	return Config{
		Axes: []AxisConfig{
			{AxisRoll, 0, 10, false},
			{AxisPitch, 10, 10, true},
			{AxisThrottle, 32, 8, true},
			{AxisYaw, 22, 10, false},
		},
		Buttons: []ButtonConfig{
			{0, "BTN 0"},
			{1, "BTN 1"},
			{8, "BTN 8"},
			{9, "BTN 9"},
			{10, "BTN 10"},
			{11, "BTN 11"},
			{12, "BTN 12"},
			{13, "BTN 13"},
		},
		ReadTimeout: 3 * time.Millisecond,
	}
}
