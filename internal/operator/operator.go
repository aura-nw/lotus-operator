package operator

import (
	"bytes"
	"context"
	"encoding/hex"
	"github.com/aura-nw/lotus-operator/internal/operator/types"
	"github.com/btcsuite/btcd/wire"
	"log/slog"
	"math/big"
	"time"

	"github.com/aura-nw/lotus-core/clients/evm/contracts"
	"github.com/aura-nw/lotus-operator/config"
	"github.com/aura-nw/lotus-operator/internal/operator/bitcoin"
	"github.com/aura-nw/lotus-operator/internal/operator/evm"
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

	// Init bitcoin verifier
	btcVerifier, err := bitcoin.NewVerifier(op.ctx, op.logger, op.config.Bitcoin)
	if err != nil {
		return err
	}
	op.btcVerifier = btcVerifier

	return nil
}

func (op *Operator) Start() {
	op.logger.Info("starting operator service", "evm address", op.evmVerifier.GetAddress().Hex())
	go op.incomingEventsLoop()
	go op.outgoingEventsLoop()
}

func (op *Operator) incomingEventsLoop() {
	op.logger.Info("starting incoming events loop")
	nextId, err := op.evmVerifier.GetNextIdVerifyIncomingInvoice(op.evmVerifier.GetAddress())
	if err != nil {
		op.logger.Error("incomingEventsLoop: get next id error", "err", err)
		return
	}
	op.logger.Info("incomingEventsLoop", "next_id", nextId.Uint64(), "query_interval", op.config.Evm.QueryInterval)

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
				op.logger.Error("incomingEventsLoop: get incoming invoice count error", "err", err)
				continue
			}
			id := nextId.Uint64()
			count := c.Uint64()
			if id > count {
				op.logger.Info("incomingEventsLoop: waiting for next incoming id", "next_id", id, "count", count)
				continue
			}
			// Process next id
			invoice, err := op.evmVerifier.GetIncomingInvoice(id)
			if err != nil {
				op.logger.Error("incomingEventsLoop: get incoming invoice error", "err", err, "id", id, "err", err)
				continue
			}

			op.logger.Info("incomingEventsLoop: found invoice", "id", id)

			if evm.InvoiceStatus(invoice.Status) != evm.Pending {
				op.logger.Info("incomingEventsLoop: incoming invoice no need verify", "id", id, "status", invoice.Status)
				id++
				continue
			}

			if op.isVerified(invoice) {
				op.logger.Info("incomingEventsLoop: invoice has self-verified", "address", op.evmVerifier.GetAddress().Hex())
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
	ticker := time.NewTicker(time.Duration(op.config.Evm.QueryInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-op.ctx.Done():
			op.logger.Info("outgoingEventsLoop: context done")
			return
		case <-ticker.C:
			lastId, err := op.evmVerifier.GetOutgoingTxCount()
			if err != nil {
				op.logger.Error("outgoingEventsLoop: get last id failed", "err", err)
				continue
			}
			if lastId == nil || lastId.Cmp(big.NewInt(0)) == 0 {
				op.logger.Info("outgoingEventsLoop: no outgoing tx")
				continue
			}
			op.logger.Info("outgoingEventsLoop", "last_id", lastId.Uint64())

			// Process next id
			txOutgoing, err := op.evmVerifier.GetOutgoingTx(lastId)
			if err != nil {
				op.logger.Error("outgoingEventsLoop: get outgoing invoice error", "err", err, "id", lastId.Uint64())
				continue
			}

			op.logger.Info("outgoingEventsLoop: found invoice", "id", lastId)

			if evm.InvoiceStatus(txOutgoing.Status) != evm.Pending {
				op.logger.Info("outgoingEventsLoop: outgoing invoice no need verify", "id", lastId, "status", txOutgoing.Status)
				continue
			}

			isValidate := true
			outputs := make([]types.Utxo, 0)
			for _, invoiceId := range txOutgoing.InvoiceIds {
				invoice, err := op.evmVerifier.GetOutgoingInvoice(invoiceId.Uint64())
				if err != nil {
					op.logger.Error("outgoingEventsLoop: get outgoing invoice error", "err", err, "id", invoiceId)
					isValidate = false
					continue
				}

				if evm.InvoiceStatus(invoice.Status) != evm.Pending {
					op.logger.Info("outgoingEventsLoop: outgoing invoice no need verify", "id", invoiceId, "status", invoice.Status)
					continue
				}

				outputs = append(outputs, types.Utxo{
					Address: invoice.Recipient,
					Amount:  invoice.Amount.Int64(),
				})
			}
			if !isValidate {
				op.logger.Info("outgoingEventsLoop: outgoing invoice not validate", "id", lastId)
				// submit verify failed to contract
				if err := op.evmVerifier.VerifyOutgoingTx(lastId.Uint64(), false, ""); err != nil {
					op.logger.Error("verify outgoing tx error", "err", err)
					continue
				}
			}

			// Verify and sign btc
			signature, err := op.verifyAndSignBtc(txOutgoing.TxContent, outputs)
			if err != nil {
				op.logger.Error("outgoingEventsLoop: verify and sign btc error", "err", err)
				continue
			}

			// submit verify success to contract
			if err := op.evmVerifier.VerifyOutgoingTx(lastId.Uint64(), true, hex.EncodeToString(signature)); err != nil {
				op.logger.Error("verify outgoing tx error", "err", err)
				continue
			}
		}
	}
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

func (op *Operator) verifyAndSignBtc(txContext string, outputs []types.Utxo) ([]byte, error) {
	txBytes, err := hex.DecodeString(txContext)
	if err != nil {
		op.logger.Error("verifyAndSignBtc: decode tx context error", "err", err)
		return nil, err
	}

	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		op.logger.Error("verifyAndSignBtc: deserialize tx error", "err", err)
		return nil, err
	}

	// Verify
	allHasUtxo := 0
	for _, output := range outputs {
		for _, uxto := range msgTx.TxOut {
			receiver, err := op.btcVerifier.ConvertToAddress(uxto.PkScript)
			if err != nil {
				op.logger.Error("verifyAndSignBtc: convert to address error", "err", err)
				return nil, err
			}
			if output.Address == receiver && output.Amount == uxto.Value {
				allHasUtxo++
			}
		}
	}
	if allHasUtxo != len(outputs) {
		op.logger.Error("verifyAndSignBtc: not all outputs has utxo")
		return nil, err
	}

	// Sign
	signature, err := op.btcVerifier.Sign(&msgTx)
	if err != nil {
		op.logger.Error("verifyAndSignBtc: sign tx error", "err", err)
		return nil, err
	}

	return signature, nil
}
