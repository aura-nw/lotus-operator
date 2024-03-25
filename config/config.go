package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server  ServerInfo  `toml:"server"`
	Evm     EvmInfo     `toml:"evm"`
	Bitcoin BitcoinInfo `toml:"bitcoin"`
}

type ServerInfo struct {
	GrpcPort string `toml:"grpc-port"`
}

type BitcoinInfo struct {
	Network          string `toml:"network"`
	Host             string `toml:"host"`
	User             string `toml:"user"`
	Pass             string `toml:"pass"`
	QueryInterval    int64  `toml:"query-interval"`
	MinConfirmations int64  `toml:"min-confirmations"`
	MultisigAddress  string `toml:"multisig-address"`
}

type EvmInfo struct {
	Url              string      `toml:"url"`
	ChainID          int64       `toml:"chain-id"`
	QueryInterval    int64       `toml:"query-interval"`
	MinConfirmations int64       `toml:"min-confirmations"`
	PrivateKey       string      `toml:"private-key"`
	Contracts        EvmContract `toml:"contracts"`
	CallTimeout      uint64      `toml:"call-timeout"`
}

type EvmContract struct {
	WrappedBtcAddr string `toml:"wrapped-btc-addr"`
	GatewayAddr    string `toml:"gateway-addr"`
}

// LoadConfig loads config from toml file to OperatorConfig
func LoadConfig(path string) (Config, error) {
	var config Config

	// Decode the TOML file into the config struct
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return config, fmt.Errorf("failed to decode TOML configuration: %w", err)
	}

	return config, nil
}
