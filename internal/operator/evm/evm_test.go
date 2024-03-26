package evm_test

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/aura-nw/btc-bridge-operator/config"
	"github.com/aura-nw/btc-bridge-operator/internal/operator/evm"
	"github.com/ethereum/go-ethereum/common"
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

func TestQueryGateway(t *testing.T) {
	verifier, err := evm.NewVerifier(context.Background(), slog.Default(), config.EvmInfo{
		Url:              evmosTestnetRpc,
		ChainID:          evmosTestnetChainId,
		QueryInterval:    10,
		MinConfirmations: 5,
		PrivateKey:       "883d80012adf2272875981428715c56558eb388dcea4b48e030bd63ddd23c128",
		Contracts: config.EvmContract{
			WrappedBtcAddr: "0x7fd84b9a10f13acD07B9fA95D217827dCf608140",
			GatewayAddr:    "0x6731881DE07Ffce55968a583F5f641C589d25ea7",
		},
		CallTimeout: 10,
	})
	require.NoError(t, err)

	require.Equal(t, common.HexToAddress("0xC32B94C38bbbfe65eCe90daF3493c7603dA2c19A"), verifier.GetAddress())

	count, err := verifier.GetIncomingInvoiceCount()
	require.NoError(t, err)
	t.Log("incoming invoice count: ", count)

	nextIdIncoming, err := verifier.GetNextIdVerifyIncomingInvoice(verifier.GetAddress())
	require.NoError(t, err)
	t.Log("next id incoming: ", nextIdIncoming)
}
