package input

type AxisConfig struct {
	Name     string
	BitStart int
	BitLen   int
	Inverted bool
}

type ButtonConfig struct {
	Bit  int
	Name string
}

type Config struct {
	Axes    []AxisConfig
	Buttons []ButtonConfig
}

func GetConfig() Config {
	return Config{
		Axes: []AxisConfig{
			{"Roll", 0, 10, false},
			{"Pitch", 10, 10, true},
			{"Throttle", 32, 8, true},
			{"Yaw", 22, 10, false},
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
	}
}
