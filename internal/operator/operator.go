package operator

import (
	"context"
	"log/slog"
	"time"

	"github.com/aura-nw/btc-bridge-core/clients/evm/contracts"
	"github.com/aura-nw/btc-bridge-operator/config"
	"github.com/aura-nw/btc-bridge-operator/internal/operator/bitcoin"
	"github.com/aura-nw/btc-bridge-operator/internal/operator/evm"
)

type Operator struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *slog.Logger
	config *config.Config

	evmVerifier evm.Verifier
	btcVerifier bitcoin.Verifier
}

func NewOperator(ctx context.Context, config *config.Config, logger *slog.Logger) (*Operator, error) {
	ctx, cancel := context.WithCancel(ctx)
	op := &Operator{
		ctx:    ctx,
		cancel: cancel,
		config: config,
		logger: logger,
	}

	if err := op.init(); err != nil {
		return nil, err
	}

	return op, nil
}

func (op *Operator) init() error {
	// Init evm verifier
	evmVerifier, err := evm.NewVerifier(op.ctx, op.logger, op.config.Evm)
	if err != nil {
		return err
	}
	op.evmVerifier = evmVerifier

	return nil
}

func (op *Operator) Start() {
	op.logger.Info("starting operator service")
	go op.incomingEventsLoop()
	go op.outgoingEventsLoop()
}

func (op *Operator) incomingEventsLoop() {
	op.logger.Info("starting incoming events loop")
	nextId, err := op.evmVerifier.GetNextIdVerifyIncomingInvoice(op.evmVerifier.GetAddress())
	if err != nil {
		op.logger.Error("incomingEventsLoop: get last id failed", "err", err)
		return
	}
	op.logger.Info("incomingEventsLoop", "next_id", nextId, "query_interval", op.config.Evm.QueryInterval)

	ticker := time.NewTicker(time.Duration(op.config.Evm.QueryInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-op.ctx.Done():
			op.logger.Info("incomingEventsLoop: context done")
			return
		case <-ticker.C:
			c, err := op.evmVerifier.GetIncomingInvoiceCount()
			if err != nil {
				op.logger.Error("incomingEventsLoop: get incoming invoice count failed", "err", err)
				continue
			}
			id := nextId.Uint64()
			count := c.Uint64()
			if id > count {
				op.logger.Info("incomingEventsLoop: waiting for next incoming id", "err", err, "next_id", id, "count", count)
				continue
			}
			// Process next id
			invoice, err := op.evmVerifier.GetIncomingInvoice(id)
			if err != nil {
				op.logger.Error("incomingEventsLoop: get incoming invoice failed", "id", id, "err", err)
				continue
			}

			if evm.InvoiceStatus(invoice.Status) != evm.Pending {
				op.logger.Info("incomingEventsLoop: incoming invoice no need verify", "id", id, "err", err)
				id++
				continue
			}

			if op.isVerified(invoice) {
				op.logger.Info("incomingEventsLoop: operator has verified", "address", op.evmVerifier.GetAddress().Hex())
				id++
				continue
			}

			// Verify invoice
			valid, err := op.btcVerifier.VerifyBtcDeposit(invoice.Utxo, invoice.Amount.Uint64(), invoice.Recipient.Hex())
			if err != nil {
				op.logger.Error("incomingEventsLoop: verify btc deposit failed", "err", err)
				continue
			}
			if !valid {
				op.logger.Info("incomingEventsLoop: btc deposit not vaild", "id", invoice.InvoiceId)
				// Vote no and wait
				if err := op.evmVerifier.VerifyIncomingInvoice(invoice.InvoiceId.Uint64(), invoice.Utxo, invoice.Amount, invoice.Recipient, false); err != nil {
					op.logger.Error("verify incomming invoice error", "err", err)
					continue
				}
				id++
				continue
			}
			// Vote yes and wait
			op.logger.Info("incomingEventsLoop: btc deposit vaild", "id", invoice.InvoiceId)
			if err := op.evmVerifier.VerifyIncomingInvoice(invoice.InvoiceId.Uint64(), invoice.Utxo, invoice.Amount, invoice.Recipient, true); err != nil {
				op.logger.Error("verify incomming invoice error", "err", err)
				continue
			}
			id++
		}

	}
}

func (op *Operator) outgoingEventsLoop() {
	op.logger.Info("starting outgoing events loop")
}

func (op *Operator) Stop() {
	op.logger.Info("stopping operator service")
	op.cancel()
}

func (op *Operator) isVerified(invoice contracts.IGatewayIncomingInvoiceResponse) bool {
	myIndex := op.indexOnIncommingInvoice(invoice)
	if myIndex == -1 || myIndex >= len(invoice.Confirmations) {
		return false
	}
	return invoice.Confirmations[myIndex]
}

func (op *Operator) indexOnIncommingInvoice(invoice contracts.IGatewayIncomingInvoiceResponse) int {
	for index, address := range invoice.Validators {
		if op.evmVerifier.GetAddress() == address {
			return index
		}
	}
	return -1
}
