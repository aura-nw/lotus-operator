package evm

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/aura-nw/btc-bridge-core/clients/evm/contracts"
	"github.com/aura-nw/btc-bridge-operator/config"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Verifier interface {
	Sender
	Reader
}

type Sender interface {
	GetAddress() common.Address

	VerifyIncomingInvoice(id uint64, utxo string, amount *big.Int, recipient common.Address, isVerified bool) error

	VerifyOutgoingInvoice(id uint64, amount *big.Int, recipient common.Address, signature string) error
}

type Reader interface {
	// Incoming invoice
	GetNextIdVerifyIncomingInvoice(operator common.Address) (*big.Int, error)
	GetIncomingInvoiceCount() (*big.Int, error)
	GetIncomingInvoice(id uint64) (contracts.IGatewayIncomingInvoiceResponse, error)

	// Outgoing invoice
	GetNextIdVerifyOutgoingInvoice(operator common.Address) (*big.Int, error)
	GetOutgoingInvoiceCount() (*big.Int, error)
	GetOutgoingInvoice(id uint64) (contracts.IGatewayOutgoingInvoiceResponse, error)
}

type InvoiceStatus uint8

const (
	Waiting InvoiceStatus = iota
	Pending
	Minted
	Refunding
	Refunded
	Manual
	Paid
)

type verifierImpl struct {
	ctx             context.Context
	logger          *slog.Logger
	info            config.EvmInfo
	client          *ethclient.Client
	auth            *bind.TransactOpts
	gatewayContract *contracts.Gateway
}

func NewVerifier(ctx context.Context, logger *slog.Logger, info config.EvmInfo) (Verifier, error) {
	client, err := ethclient.Dial(info.Url)
	if err != nil {
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(info.PrivateKey)
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(info.ChainID))
	if err != nil {
		return nil, err
	}

	gatewayContract, err := contracts.NewGateway(common.HexToAddress(info.Contracts.GatewayAddr), client)
	if err != nil {
		return nil, err
	}

	return &verifierImpl{
		ctx:             ctx,
		logger:          logger,
		info:            info,
		client:          client,
		auth:            auth,
		gatewayContract: gatewayContract,
	}, nil
}

var _ Verifier = &verifierImpl{}

// GetAddress implements Verifier.
func (v *verifierImpl) GetAddress() common.Address {
	return v.auth.From
}

// GetIncomingInvoice implements Verifier.
func (v *verifierImpl) GetIncomingInvoice(id uint64) (contracts.IGatewayIncomingInvoiceResponse, error) {
	return v.gatewayContract.IncomingInvoice(&bind.CallOpts{}, fmt.Sprintf("%d", id))
}

// GetIncomingInvoiceCount implements Verifier.
func (v *verifierImpl) GetIncomingInvoiceCount() (*big.Int, error) {
	return v.gatewayContract.IncomingInvoicesCount(&bind.CallOpts{})
}

// GetNextIdVerifyIncomingInvoice implements Verifier.
func (v *verifierImpl) GetNextIdVerifyIncomingInvoice(operator common.Address) (*big.Int, error) {
	validatorInfo, err := v.gatewayContract.Validator(&bind.CallOpts{}, v.GetAddress())
	if err != nil {
		return nil, err
	}
	return validatorInfo.NextIncomingInvoice, nil
}

// GetNextIdVerifyOutgoingInvoice implements Verifier.
func (v *verifierImpl) GetNextIdVerifyOutgoingInvoice(operator common.Address) (*big.Int, error) {
	validatorInfo, err := v.gatewayContract.Validator(&bind.CallOpts{}, v.GetAddress())
	if err != nil {
		return nil, err
	}
	return validatorInfo.NextOutgoingInvoice, nil
}

// GetOutgoingInvoice implements Verifier.
func (v *verifierImpl) GetOutgoingInvoice(id uint64) (contracts.IGatewayOutgoingInvoiceResponse, error) {
	panic("unimplemented")
}

// GetOutgoingInvoiceCount implements Verifier.
func (v *verifierImpl) GetOutgoingInvoiceCount() (*big.Int, error) {
	return v.gatewayContract.OutgoingInvoicesCount(&bind.CallOpts{})
}

// VerifyIncomingInvoice implements Verifier.
func (v *verifierImpl) VerifyIncomingInvoice(id uint64, utxo string, amount *big.Int, recipient common.Address, isVerified bool) error {
	tx, err := v.gatewayContract.VerifyIncomingInvoice(v.auth, big.NewInt(int64(id)), utxo, amount, recipient, isVerified)
	if err != nil {
		v.logger.Error("call VerifyIncomingInvoice error", "err", err)
		return err
	}

	ctx, cancel := context.WithTimeout(v.ctx, time.Duration(v.info.CallTimeout)*time.Second)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, v.client, tx)
	if err != nil {
		v.logger.Error("call WaitMined error", "err", err)
		return err
	}
	v.logger.Info("call WaitMined ok", "tx_hash", receipt.TxHash.Hex())
	return nil
}

// VerifyOutgoingInvoice implements Verifier.
func (v *verifierImpl) VerifyOutgoingInvoice(id uint64, amount *big.Int, recipient common.Address, signature string) error {
	panic("unimplemented")
}
