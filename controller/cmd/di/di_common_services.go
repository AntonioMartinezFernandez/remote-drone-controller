package di

import (
	"context"
	"os"
	"os/signal"

	"github.com/AntonioMartinezFernandez/remote-drone-controller/configs"

	pkg_logger "github.com/AntonioMartinezFernandez/remote-drone-controller/pkg/logger"

	"github.com/joho/godotenv"
)

type CommonServices struct {
	Config configs.Config
	Logger pkg_logger.Logger
}

func InitCommonServices(ctx context.Context) *CommonServices {
	config := initConfig()

	logger := pkg_logger.NewLogger(config.LogLevel)

	return &CommonServices{
		Config: config,
		Logger: logger,
	}
}

/* HELPERS */

func InitCommonServicesWithEnvFiles(envFiles ...string) *CommonServices {
	ctx := context.Background()
	err := godotenv.Overload(envFiles...)
	if err != nil {
		panic(err)
	}

	return InitCommonServices(ctx)
}

func RootContext() (context.Context, context.CancelFunc) {
	rootCtx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt, os.Kill,
	)
	return rootCtx, cancel
}

func initConfig() configs.Config {
	return configs.LoadEnvConfig()
}
