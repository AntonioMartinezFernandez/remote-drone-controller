package di

import (
	"context"
	"log/slog"
)

type KeyboardControllerDi struct {
	CommonServices *CommonServices
}

func InitKeyboardControllerDi(ctx context.Context) *KeyboardControllerDi {
	commonServices := InitCommonServices(ctx)

	return &KeyboardControllerDi{
		CommonServices: commonServices,
	}
}

func (ad *KeyboardControllerDi) ErrorShutdown(ctx context.Context, cancel context.CancelFunc, err error) {
	defer cancel()
	if err == nil {
		return
	}

	ad.CommonServices.Logger.Error(
		ctx,
		"error on starting service",
		slog.String("error", err.Error()),
	)
}

func (ad *KeyboardControllerDi) GracefulShutdown(ctx context.Context) {
	ad.CommonServices.Logger.Info(
		ctx,
		"service stopped",
	)
}
