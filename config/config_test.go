package config_test

import (
	"testing"

	"github.com/aura-nw/btc-bridge-operator/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	c, err := config.LoadConfig("../operator.toml")
	require.NoError(t, err)
	t.Log("config: ", c)
}
