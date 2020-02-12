package lndwrap

import "errors"

var (
	errNotEnabled   = errors.New("RPC sub-server not enabled")
	errWalletLocked = errors.New("Wallet is locked, please unlock first")
)
