package evm

import (
	"math/big"

	"github.com/aura-nw/btc-bridge-core/clients/evm/contracts"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Verifier interface {
	VerifyIncomingInvoice(id uint64, utxo string, amount *big.Int, recipient common.Address) (bool, error)
	GetLastIdVerifyIncomingInvoice(operator common.Address) (uint64, error)
	GetIncomingInvoiceCount() (uint64, error)
	GetIncomingInvoice(id uint64) (*contracts.IGatewayIncomingInvoiceResponse, error)

	VerifyOutgoingInvoice(id uint64, amount *big.Int, recipient common.Address, signature string) (bool, error)
	GetLastIdVerifyOutgoingInvoice(operator common.Address) (uint64, error)
	GetOutgoingInvoiceCount() (uint64, error)
	GetOutgoingInvoice(id uint64) (*contracts.IGatewayOutgoingInvoiceResponse, error)
}

type Sender interface {
	GetAddress() common.Address
	SendAndWait(txs ...txmgr.TxCandidate) (*types.Receipt, error)
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

type ContractOptions struct {
	GatewayContract *batching.BoundContract
}
