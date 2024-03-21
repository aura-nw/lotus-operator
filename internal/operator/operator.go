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

const (
	verifyIncomingInvoiceMethod = "verifyIncomingInvoice"
)

type Operator struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *slog.Logger
	config *config.Config

	evmVerifier evm.Verifier
	evmSender   evm.Sender
	contracts   *evm.ContractOptions

	btcVerifier bitcoin.BtcVerifier
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
	return nil
}

func (op *Operator) Start() {
	op.logger.Info("starting operator service")
	go op.incomingEventsLoop()
	go op.outgoingEventsLoop()
}

func (op *Operator) incomingEventsLoop() {
	op.logger.Info("starting incoming events loop")
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

			if evm.InvoiceStatus(invoice.Status) != evm.Pending {
				op.logger.Info("incomingEventsLoop: incoming invoice no need verify", "id", lastId+1, "err", err)
				lastId++
				continue
			}

			if op.isVerified(invoice) {
				op.logger.Info("incomingEventsLoop: operator has verified", "address", op.evmSender.GetAddress().Hex())
				lastId++
				continue
			}

			// Verify invoice
			verified, err := op.btcVerifier.VerifyBtcDeposit("", invoice.Utxo)
			if err != nil {
				op.logger.Error("incomingEventsLoop: verify btc deposit failed", "err", err)
				continue
			}
			if !verified {
				op.logger.Info("incomingEventsLoop: btc deposit not vaild", "id", invoice.InvoiceId)
				// Vote no
				tx, err := op.contracts.GatewayContract.Call(verifyIncomingInvoiceMethod,
					invoice.InvoiceId,
					invoice.Utxo,
					invoice.Amount,
					invoice.Recipient,
					false,
				).ToTxCandidate()
				if err != nil {
					op.logger.Error("contract call to tx candidate error", "err", err)
					continue
				}
				receipt, err := op.evmSender.SendAndWait(tx)
				if err != nil {
					op.logger.Error("incomingEventsLoop: send vote no failed", "err", err)
					continue
				}
				op.logger.Info("incomingEventsLoop: send vote successed", "receipt", receipt.TxHash.String())
				lastId++
				continue
			}
			// Vote yes
			tx, err := op.contracts.GatewayContract.Call(verifyIncomingInvoiceMethod,
				invoice.InvoiceId,
				invoice.Utxo,
				invoice.Amount,
				invoice.Recipient,
				true,
			).ToTxCandidate()
			if err != nil {
				op.logger.Error("contract call to tx candidate error", "err", err)
				continue
			}
			receipt, err := op.evmSender.SendAndWait(tx)
			if err != nil {
				op.logger.Error("incomingEventsLoop: send vote yes failed", "err", err)
				continue
			}
			op.logger.Info("incomingEventsLoop: send vote successed", "receipt", receipt.TxHash.String())
			lastId++
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

func (op *Operator) isVerified(invoice *contracts.IGatewayIncomingInvoiceResponse) bool {
	myIndex := op.indexOnIncommingInvoice(invoice)
	if myIndex == -1 || myIndex >= len(invoice.Confirmations) {
		return false
	}
	return invoice.Confirmations[myIndex]
}

func (op *Operator) indexOnIncommingInvoice(invoice *contracts.IGatewayIncomingInvoiceResponse) int {
	for index, address := range invoice.Validators {
		if op.evmSender.GetAddress() == address {
			return index
		}
	}
	return -1
}
