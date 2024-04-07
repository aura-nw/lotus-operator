package evm_test

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/aura-nw/lotus-core/types"
	"github.com/aura-nw/lotus-operator/config"
	"github.com/aura-nw/lotus-operator/internal/operator/evm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const (
	evmosTestnetRpc           = "https://jsonrpc.dev.aura.network"
	evmosTestnetChainId int64 = 1235
)

func getEvmInfo(privateKey string) config.EvmInfo {
	return config.EvmInfo{
		Url:              evmosTestnetRpc,
		ChainID:          evmosTestnetChainId,
		QueryInterval:    10,
		MinConfirmations: 5,
		// PrivateKey:       "883d80012adf2272875981428715c56558eb388dcea4b48e030bd63ddd23c128",
		PrivateKey: privateKey,
		Contracts: config.EvmContract{
			WrappedBtcAddr: "0xC70b52bBFd514859FA01728FcE22DABb96cc130D",
			GatewayAddr:    "0x4F80aD4F4F398465EaED7b5a6Cb5f2Fe256f7239",
		},
		CallTimeout: 10,
	}
}

func TestRpc(t *testing.T) {
	client, err := ethclient.Dial(evmosTestnetRpc)
	require.NoError(t, err)

	chainID, err := client.ChainID(context.Background())
	require.NoError(t, err)
	require.Equal(t, big.NewInt(evmosTestnetChainId), chainID)

	height, err := client.BlockNumber(context.Background())
	require.NoError(t, err)
	t.Log("height", height)
}

func TestQueryGateway(t *testing.T) {
	priv1 := "883d80012adf2272875981428715c56558eb388dcea4b48e030bd63ddd23c128"
	verifier, err := evm.NewVerifier(slog.Default(), getEvmInfo(priv1))
	require.NoError(t, err)

	require.Equal(t, common.HexToAddress("0xC32B94C38bbbfe65eCe90daF3493c7603dA2c19A"), verifier.GetAddress())

	count, err := verifier.GetIncomingInvoiceCount()
	require.NoError(t, err)
	t.Log("incoming invoice count: ", count)

	nextIdIncoming, err := verifier.GetNextIdVerifyIncomingInvoice(verifier.GetAddress())
	require.NoError(t, err)
	t.Log("next id incoming: ", nextIdIncoming)
}

func TestVerify(t *testing.T) {
	priv1 := "444a26796811d3b86bd1c3b85d04b9b078e4eee66203096f04081b245d6e4123"
	verifier, err := evm.NewVerifier(slog.Default(), getEvmInfo(priv1))
	require.NoError(t, err)

	t.Log("sender address: ", verifier.GetAddress().Hex())

	operators, err := verifier.GetOperators()
	require.NoError(t, err)
	t.Log("list operators: ", operators)

	testDeposit := types.BtcDeposit{
		TxId:           "12747b5c26bc02d03ab92d9ad8984539b978271941b88e781c772370b5aaf0e123",
		Height:         2574433,
		Memo:           "",
		Receiver:       "0xD02c8cebc86Bd8Cc5fE876b4B793256C0d67a887",
		Sender:         "",
		MultisigWallet: "tb1qrvjce6589p2x9zupd8p0dnkq46s8lsh3rau7v5",
		Amount:         602518,
		Idx:            0,
		UtxoStatus:     "unused",
		Status:         "new",
	}

	count, err := verifier.GetIncomingInvoiceCount()
	require.NoError(t, err)
	t.Log("count incoming: ", count)

	err = verifier.VerifyIncomingInvoice(count.Uint64(), testDeposit.TxId, big.NewInt(int64(testDeposit.Amount)), common.HexToAddress(testDeposit.Receiver), true)
	require.NoError(t, err)
}
