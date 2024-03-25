package main

import (
	"context"
	"log/slog"

	"github.com/aura-nw/btc-bridge-operator/config"
	"github.com/aura-nw/btc-bridge-operator/internal/operator"
)

const (
	defaultConfigPath = "./operator.toml"
)

func main() {
	config, err := config.LoadConfig(defaultConfigPath)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	operator, err := operator.NewOperator(ctx, &config, slog.Default())
	if err != nil {
		panic(err)
	}

	operator.Start()
	<-ctx.Done()
}
