package evm_test

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/aura-nw/btc-bridge-operator/config"
	"github.com/aura-nw/btc-bridge-operator/internal/operator/evm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var (
	evmosTestnetRpc           = "https://evmos-testnet-jsonrpc.alkadeta.com"
	evmosTestnetChainId int64 = 9000
)

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

func TestQueryGatewayCount(t *testing.T) {
	verifier, err := evm.NewVerifier(context.Background(), slog.Default(), config.EvmInfo{
		Url:              evmosTestnetRpc,
		ChainID:          evmosTestnetChainId,
		QueryInterval:    10,
		MinConfirmations: 5,
		PrivateKey:       "883d80012adf2272875981428715c56558eb388dcea4b48e030bd63ddd23c128",
		Contracts: config.EvmContract{
			WrappedBtcAddr: "0xC70b52bBFd514859FA01728FcE22DABb96cc130D",
			GatewayAddr:    "0x4F80aD4F4F398465EaED7b5a6Cb5f2Fe256f7239",
		},
		CallTimeout: 10,
	})
	require.NoError(t, err)

	count, err := verifier.GetIncomingInvoiceCount()
	require.NoError(t, err)
	t.Log("incoming invoice count: ", count)
}
