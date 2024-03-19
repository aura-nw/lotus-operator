package operator

import (
	"math/big"

	"github.com/aura-nw/btc-bridge-core/clients/evm/contracts"
	"github.com/aura-nw/btc-bridge-core/clients/evm/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type EvmVerifier interface {
	VerifyIncomingInvoice(id uint64, utxo string, amount *big.Int, recipient common.Address) (bool, error)
	GetLastIdVerifyIncomingInvoice(operator common.Address) (uint64, error)
	GetIncomingInvoiceCount() (uint64, error)
	GetIncomingInvoice(id uint64) (*contracts.IGatewayIncomingInvoiceResponse, error)

	VerifyOutgoingInvoice(id uint64, amount *big.Int, recipient common.Address, signature string) (bool, error)
	GetLastIdVerifyOutgoingInvoice(operator common.Address) (uint64, error)
	GetOutgoingInvoiceCount() (uint64, error)
	GetOutgoingInvoice(id uint64) (*contracts.IGatewayOutgoingInvoiceResponse, error)
}

type EvmSender interface {
	GetAddress() common.Address
	SendAndWait(tx txmgr.TxCandidate) (*types.Receipt, error)
}
