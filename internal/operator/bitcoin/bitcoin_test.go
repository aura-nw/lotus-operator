package bitcoin_test

import (
	"testing"

	"github.com/aura-nw/lotus-operator/internal/operator/bitcoin"
	"github.com/stretchr/testify/require"
)

func TestUtxo(t *testing.T) {
	utxo := bitcoin.UtxoDef{
		Height: 1000,
		TxHash: "5c1822815e8362821970adea33f9eee07692e137bfe430664ee619bef93a9304",
		Amount: 100,
		Memo:   "tiennv",
	}

	utxoStr := utxo.String()

	t.Log("uxto", utxoStr)

	reUtxo := bitcoin.UtxoFromStr(utxoStr)
	require.Equal(t, utxo.Height, reUtxo.Height)
}
