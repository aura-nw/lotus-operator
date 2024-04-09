package operator

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/aura-nw/lotus-core/clients/evm/contracts"
	"github.com/aura-nw/lotus-operator/config"
	"github.com/aura-nw/lotus-operator/internal/operator/bitcoin"
	"github.com/aura-nw/lotus-operator/internal/operator/evm"
	"github.com/aura-nw/lotus-operator/internal/operator/types"
	"github.com/btcsuite/btcd/wire"
)

type Operator struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *slog.Logger
	config *config.Config

	evmVerifier evm.Verifier
	btcVerifier bitcoin.Verifier

	server *Server
}

func NewOperator(ctx context.Context, config *config.Config, logger *slog.Logger) (*Operator, error) {
	ctx, cancel := context.WithCancel(ctx)
	op := &Operator{
		ctx:    ctx,
		cancel: cancel,
		config: config,
		logger: logger,
	}

	if err := op.initVerifier(); err != nil {
		return nil, err
	}

	server, err := NewServer(ctx, op.logger, op.config.Server)
	if err != nil {
		return nil, err
	}
	op.server = server

	return op, nil
}

func (op *Operator) initVerifier() error {
	// Init evm verifier
	evmVerifier, err := evm.NewVerifier(op.logger, op.config.Evm)
	if err != nil {
		op.logger.Error("init evm verifier failed", "err", err)
		return err
	}
	op.evmVerifier = evmVerifier

	// Init bitcoin verifier
	btcVerifier, err := bitcoin.NewVerifier(op.logger, op.config.Bitcoin)
	if err != nil {
		op.logger.Error("init bitcoin verifier failed", "err", err)
		return err
	}
	op.btcVerifier = btcVerifier

	return nil
}

func (op *Operator) Start() {
	op.logger.Info("starting operator service", "evm_address", op.evmVerifier.GetAddress().Hex())
	go op.incomingEventsLoop()
	go op.outgoingEventsLoop()

	op.logger.Info("starting operator server", "port", op.config.Server.HttpPort)
	go op.server.Start()
}

func (op *Operator) findNextIncomingIdNeedVerify() (uint64, error) {
	address := op.evmVerifier.GetAddress()
	nextId, err := op.evmVerifier.GetNextIdVerifyIncomingInvoice(address)
	if err != nil {
		op.logger.Error("get next id verify incomint invoice error", "err", err)
		return 0, err
	}
	id := nextId.Uint64()
	for {
		count, err := op.evmVerifier.GetIncomingInvoiceCount()
		if err != nil {
			op.logger.Error("get incoming invoice count error", "err", err)
			return 0, err
		}
		if id > count.Uint64() {
			op.logger.Info("no incoming invoice need verify")
			return 0, fmt.Errorf("no incoming invoice need verify")
		}
		invoice, err := op.evmVerifier.GetIncomingInvoice(id)
		if err != nil {
			op.logger.Error("get incoming invoice error", "err", err, "id", id, "err", err)
			return 0, err
		}
		if evm.InvoiceStatus(invoice.Status) != evm.Pending {
			op.logger.Info("incoming invoice no need verify", "id", id, "status", invoice.Status)
			id++
			continue
		}

		if op.isVerified(invoice) {
			op.logger.Info("invoice has self-verified", "address", address.Hex())
			id++
			continue
		}
		return id, nil
	}

}

func (op *Operator) incomingEventsLoop() {
	op.logger.Info("starting incoming events loop")

	ticker := time.NewTicker(time.Duration(op.config.Evm.QueryInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-op.ctx.Done():
			op.logger.Info("context done")
			return
		case <-ticker.C:
			nextId, err := op.findNextIncomingIdNeedVerify()
			if err != nil {
				op.logger.Error("find next incoming invoice id error", "err", err)
				time.Sleep(1 * time.Second)
				continue
			}
			op.logger.Info("next incoming id for verify", "id", nextId)

			// Process next id
			invoice, err := op.evmVerifier.GetIncomingInvoice(nextId)
			if err != nil {
				op.logger.Error("get incoming invoice error", "err", err, "id", nextId, "err", err)
				continue
			}
			op.logger.Info("found incoming invoice", "id", nextId)

			// Verify invoice
			valid, err := op.btcVerifier.VerifyBtcDeposit(invoice.Utxo, invoice.Amount.Uint64(), invoice.Recipient.Hex())
			if err != nil {
				op.logger.Error("verify btc deposit failed", "err", err)
				continue
			}
			if !valid {
				op.logger.Info("btc deposit not vaild", "id", invoice.InvoiceId)
				// Vote no and wait
				if err := op.evmVerifier.VerifyIncomingInvoice(
					invoice.InvoiceId.Uint64(),
					invoice.Utxo,
					invoice.Amount,
					invoice.Recipient,
					false,
				); err != nil {
					op.logger.Error("verify incomming invoice error", "err", err)
					continue
				}
				continue
			}
			// Vote yes and wait
			op.logger.Info("btc deposit vaild", "id", invoice.InvoiceId)
			if err := op.evmVerifier.VerifyIncomingInvoice(
				invoice.InvoiceId.Uint64(),
				invoice.Utxo, invoice.Amount,
				invoice.Recipient, true,
			); err != nil {
				op.logger.Error("verify incomming invoice error", "err", err)
				continue
			}
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
			op.logger.Info("context done")
			return
		case <-ticker.C:
			lastId, err := op.evmVerifier.GetOutgoingTxCount()
			if err != nil {
				op.logger.Error("get last id failed", "err", err)
				continue
			}
			if lastId == nil || lastId.Cmp(big.NewInt(0)) == 0 {
				op.logger.Info("no outgoing tx")
				continue
			}
			op.logger.Info("outgoingEventsLoop", "last_id", lastId.Uint64())

			// Process next id
			txOutgoing, err := op.evmVerifier.GetOutgoingTx(lastId)
			if err != nil {
				op.logger.Error("get outgoing invoice error", "err", err, "id", lastId.Uint64())
				continue
			}

			op.logger.Info("found invoice", "id", lastId)

			if evm.InvoiceStatus(txOutgoing.Status) != evm.Pending {
				op.logger.Info("outgoing invoice no need verify", "id", lastId, "status", txOutgoing.Status)
				continue
			}

			isValidate := true
			outputs := make([]types.Utxo, 0)
			for _, invoiceId := range txOutgoing.InvoiceIds {
				invoice, err := op.evmVerifier.GetOutgoingInvoice(invoiceId.Uint64())
				if err != nil {
					op.logger.Error("get outgoing invoice error", "err", err, "id", invoiceId)
					isValidate = false
					continue
				}

				if evm.InvoiceStatus(invoice.Status) != evm.Pending {
					op.logger.Info("outgoing invoice no need verify", "id", invoiceId, "status", invoice.Status)
					continue
				}

				outputs = append(outputs, types.Utxo{
					Address: invoice.Recipient,
					Amount:  invoice.Amount.Int64(),
				})
			}
			if !isValidate {
				op.logger.Info("outgoing invoice not validate", "id", lastId)
				// submit verify failed to contract
				if err := op.evmVerifier.VerifyOutgoingTx(lastId.Uint64(), false, ""); err != nil {
					op.logger.Error("verify outgoing tx error", "err", err)
					continue
				}
			}

			// Verify and sign btc
			signature, err := op.verifyAndSignBtc(txOutgoing.TxContent, outputs)
			if err != nil {
				op.logger.Error("verify and sign btc error", "err", err)
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
	op.server.Stop()
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
		op.logger.Error("decode tx context error", "err", err)
		return nil, err
	}

	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		op.logger.Error("deserialize tx error", "err", err)
		return nil, err
	}

	// Verify
	allHasUtxo := 0
	for _, output := range outputs {
		for _, uxto := range msgTx.TxOut {
			receiver, err := op.btcVerifier.ConvertToAddress(uxto.PkScript)
			if err != nil {
				op.logger.Error("convert to address error", "err", err)
				return nil, err
			}
			if output.Address == receiver && output.Amount == uxto.Value {
				allHasUtxo++
			}
		}
	}
	if allHasUtxo != len(outputs) {
		op.logger.Error("not all outputs has utxo")
		return nil, err
	}

	// Sign
	signature, err := op.btcVerifier.Sign(&msgTx)
	if err != nil {
		op.logger.Error("sign tx error", "err", err)
		return nil, err
	}

	return signature, nil
}
