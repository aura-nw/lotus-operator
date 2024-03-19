package operator

import (
	"context"
	"log/slog"
	"time"

	"github.com/aura-nw/btc-bridge-operator/config"
	"github.com/ethereum/go-ethereum/common"
)

type Operator struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *slog.Logger
	config *config.Config

	evmVerifier EvmVerifier
	evmSender   EvmSender
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
	go op.incomingEventsLoop()
	go op.outgoingEventsLoop()
}

func (op *Operator) incomingEventsLoop() {
	lastId, err := op.evmVerifier.GetLastIdVerifyIncomingInvoice(op.evmSender.GetAddress())
	if err != nil {
		op.logger.Error("incomingEventsLoop: get last id failed", "err", err)
		return
	}
	op.logger.Info("incomingEventsLoop", "last_id", lastId)

	ticker := time.NewTicker(time.Duration(op.config.Evm.QueryInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-op.ctx.Done():
			op.logger.Info("incomingEventsLoop: context done")
			return
		case <-ticker.C:
			count, err := op.evmVerifier.GetIncomingInvoiceCount()
			if err != nil {
				op.logger.Error("incomingEventsLoop: get incoming invoice count failed", "err", err)
				continue
			}
			if lastId == count {
				op.logger.Info("incomingEventsLoop: waiting for next incoming id", "err", err)
				continue
			}
			// Process next id
			invoice, err := op.evmVerifier.GetIncomingInvoice(lastId + 1)
			if err != nil {
				op.logger.Error("incomingEventsLoop: get incoming invoice failed", "id", lastId+1, "err", err)
				continue
			}
			if IsInAddresses(invoice.Verifieds, op.evmSender.GetAddress()) {
				op.logger.Info("incomingEventsLoop: operator has verified", "address", op.evmSender.GetAddress().Hex())
				lastId++
				continue
			}

			lastId++
		}

	}
}

func (op *Operator) outgoingEventsLoop() {
}

func (op *Operator) Stop() {
	op.cancel()
}

func IsInAddresses(addresses []common.Address, target common.Address) bool {
	for _, addr := range addresses {
		if target.Cmp(addr) == 0 {
			return true
		}
	}
	return false
}
