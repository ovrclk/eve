package client

import "os"

type Config struct {
	Gas           string
	GasPrices     string
	GasAdjustment float64
	ChainID       string
	Home          string
	Keyring       struct {
		Backend string
		Dir     string
	}
	// Node defines the URI of the node to connect to.
	Node string
}

// DefaultConfig is a default configuration for the client.
var DefaultConfig = Config{
	Gas:       "auto",
	GasPrices: "0.025uakt",
	ChainID:   "akashnet-2",
	Home:      os.ExpandEnv("$HOME/.akash"),
	Keyring: struct {
		Backend string
		Dir     string
	}{
		Backend: "test",
		Dir:     os.ExpandEnv("$HOME/.akash"),
	},
	Node: "tcp://localhost:26657",
}
