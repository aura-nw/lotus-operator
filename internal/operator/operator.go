package operator

import (
	"context"
	"log/slog"

	"github.com/aura-nw/btc-bridge-operator/config"
)

type Operator struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *slog.Logger
	config *config.Config

	evmVerifier EvmVerifier
}

func NewOperator(ctx context.Context, config *config.Config, logger *slog.Logger) (*Operator, error) {
	ctx, cancel := context.WithCancel(ctx)
	op := &Operator{
		ctx:    ctx,
		cancel: cancel,
		config: config,
		logger: logger,
	}
	return op, nil
}

func (op *Operator) Start() {

}

func (op *Operator) readContractEvents() error {
	return nil
}

func (op *Operator) Stop() {
	op.cancel()
}
