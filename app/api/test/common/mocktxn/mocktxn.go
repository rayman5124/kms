package mocktxn

import (
	"math/big"
)

type Legacy_CommonField struct {
	GasPrice *big.Int
	Gas      uint64
	Nonce    uint64
}

// func ERC20Transfer_LegcyTxn(client *backends.SimulatedBackend, fromAddr, toAddr, erc20 common.Address) ([]byte, error) {

// }
