package operator

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type EvmVerifier interface {
	VerifyIncomingInvoice(id string, utxo string, amount *big.Int, recipient common.Address) bool
}
