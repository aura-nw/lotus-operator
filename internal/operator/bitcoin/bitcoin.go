package bitcoin

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/aura-nw/lotus-operator/config"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

type Verifier interface {
	GetMultisigAddr() string

	VerifyBtcDeposit(utxo string, amount uint64, recipient string) (bool, error)
	VerifyTokenDeposit(utxo string) (bool, error)
	VerifyInscriptionDeposit(utxo string) (bool, error)
}

type UtxoDef struct {
	Height   uint64 `json:"height"`
	TxHash   string `json:"tx_hash"`
	Amount   uint64 `json:"amount"`
	Receiver string `json:"receiver"`
	Memo     string `json:"memo"`
}

func (u *UtxoDef) String() string {
	if u == nil {
		return ""
	}
	bz, err := json.Marshal(u)
	if err != nil {
		return ""
	}
	return string(bz)
}
func UtxoFromStr(s string) UtxoDef {
	bz := []byte(s)
	var u UtxoDef
	err := json.Unmarshal(bz, &u)
	if err != nil {
		panic(err)
	}
	return u
}

type verifierImpl struct {
	ctx    context.Context
	logger *slog.Logger
	info   config.BitcoinInfo
	client *rpcclient.Client
}

// GetMultisigAddr implements Verifier.
func (v *verifierImpl) GetMultisigAddr() string {
	return v.info.MultisigAddress
}

// VerifyBtcDeposit implements Verifier.
func (v *verifierImpl) VerifyBtcDeposit(utxo string, amount uint64, recipient string) (bool, error) {
	utxoDef := UtxoFromStr(utxo)
	txHash, err := chainhash.NewHashFromStr(utxoDef.TxHash)
	if err != nil {
		return false, err
	}

	tx, err := v.client.GetRawTransactionVerbose(txHash)
	if err != nil {
		return false, err
	}

	for index, vout := range tx.Vout {
		_ = index
		_ = vout
	}
	return true, nil
}

// VerifyInscriptionDeposit implements Verifier.
func (v *verifierImpl) VerifyInscriptionDeposit(utxo string) (bool, error) {
	panic("unimplemented")
}

// VerifyTokenDeposit implements Verifier.
func (v *verifierImpl) VerifyTokenDeposit(utxo string) (bool, error) {
	panic("unimplemented")
}

func NewVerifier(ctx context.Context, logger *slog.Logger, info config.BitcoinInfo) (Verifier, error) {
	connCfg := rpcclient.ConnConfig{
		Host:         info.Host,
		User:         info.User,
		Pass:         info.Pass,
		DisableTLS:   true,
		HTTPPostMode: true,
	}
	client, err := rpcclient.New(&connCfg, nil)
	if err != nil {
		return nil, err
	}
	return &verifierImpl{
		ctx:    ctx,
		logger: logger,
		client: client,
		info:   info,
	}, nil
}

var _ Verifier = &verifierImpl{}
