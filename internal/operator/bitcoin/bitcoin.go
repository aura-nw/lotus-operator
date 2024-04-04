package bitcoin

import (
	"encoding/hex"
	"encoding/json"
	"log/slog"

	"github.com/aura-nw/lotus-operator/config"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type Verifier interface {
	GetMultisigAddr() string

	VerifyBtcDeposit(utxo string, amount uint64, recipient string) (bool, error)
	VerifyTokenDeposit(utxo string) (bool, error)
	VerifyInscriptionDeposit(utxo string) (bool, error)
	Sign(tx *wire.MsgTx) ([]byte, error)
	ConvertToAddress(pk []byte) (string, error)
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
	logger       *slog.Logger
	info         config.BitcoinInfo
	client       *rpcclient.Client
	redeemScript []byte
	privateKey   *btcec.PrivateKey
	chainParam   *chaincfg.Params
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

// Sign implements Verifier.
func (v *verifierImpl) Sign(tx *wire.MsgTx) ([]byte, error) {
	return txscript.SignatureScript(tx, 0, v.redeemScript, txscript.SigHashAll, v.privateKey, true)
}

// ConvertToAddress implements Verifier.
func (v *verifierImpl) ConvertToAddress(pkScript []byte) (string, error) {
	pk, err := txscript.ParsePkScript(pkScript)
	if err != nil {
		v.logger.Error("ConvertToAddress: parse pkscript error", "err", err)
		return "", err
	}

	btcAddr, err := pk.Address(v.chainParam)
	if err != nil {
		v.logger.Error("ConvertToAddress: get address error", "err", err)
		return "", err
	}

	return btcAddr.EncodeAddress(), nil

}

// VerifyInscriptionDeposit implements Verifier.
func (v *verifierImpl) VerifyInscriptionDeposit(utxo string) (bool, error) {
	panic("unimplemented")
}

// VerifyTokenDeposit implements Verifier.
func (v *verifierImpl) VerifyTokenDeposit(utxo string) (bool, error) {
	panic("unimplemented")
}

func NewVerifier(logger *slog.Logger, info config.BitcoinInfo) (Verifier, error) {
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
	pk, err := btcutil.DecodeWIF(info.PrivateKey)
	if err != nil {
		return nil, err
	}

	redeemScript, err := hex.DecodeString(info.RedeemScript)
	if err != nil {
		return nil, err
	}

	chainParam := &chaincfg.MainNetParams
	if info.Network == "testnet" {
		chainParam = &chaincfg.TestNet3Params
	}

	return &verifierImpl{
		logger:       logger,
		client:       client,
		info:         info,
		privateKey:   pk.PrivKey,
		redeemScript: redeemScript,
		chainParam:   chainParam,
	}, nil
}

var _ Verifier = &verifierImpl{}
