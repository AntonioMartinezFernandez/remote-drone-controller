package configs

import (
	"context"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	LogLevel             string        `env:"LOG_LEVEL,default=debug"`
	UdpReceiverAddr      string        `env:"UDP_RECEIVER_ADDR,required"`
	TransmitIntervalMs   time.Duration `env:"TRANSMIT_INTERVAL_MS,default=5ms"`
	ControlStepPercent   int           `env:"CONTROL_STEP_PERCENT,default=5"`
	FirstThrottlePercent int           `env:"FIRST_THROTTLE_PERCENT,default=40"`
	CsrfChannelValueMin  int           `env:"CSRF_CHANNEL_VALUE_MIN,default=172"`
	CsrfChannelValueMid  int           `env:"CSRF_CHANNEL_VALUE_MID,default=992"`
	CsrfChannelValueMax  int           `env:"CSRF_CHANNEL_VALUE_MAX,default=1811"`
}

func LoadEnvConfig() Config {
	var config Config

	_ = godotenv.Load(".env")
	ctx := context.Background()
	if err := envconfig.Process(ctx, &config); err != nil {
		panic(err)
	}

	return config
}
