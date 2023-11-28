package dto

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type TxnDTO struct {
	KeyID         string `json:"keyID" validate:"required" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
	SerializedTxn string `json:"serializedTxn" validate:"required,hexadecimal" example:"0xea5685ba43b740008252089439e243a7f209932df41e1fc0a1ada51b3a04b46d018086059407ad8e8b8080"`
}

type RawTxDTO struct {
	Nonce    uint64 `json:"nonce" validate:"omitempty,numeric"`
	GasPrice string `json:"gasPrice" validate:"omitempty,numeric"`
	Gas      uint64 `json:"gas" validate:"omitempty,numeric"`
	To       string `json:"to" validate:"omitempty,eth_addr"`
	Value    string `json:"value" validate:"omitempty,numeric"`
	Data     string `json:"data" validate:"omitempty,hexadecimal"`
	From     string `json:"from" validate:"required,eth_addr"`
}

func (r *RawTxDTO) ToLegacyTx() *types.LegacyTx {
	gasPrice, _ := new(big.Int).SetString(r.GasPrice, 10)
	value, _ := new(big.Int).SetString(r.Value, 10)
	to := common.HexToAddress(r.To)

	return &types.LegacyTx{
		GasPrice: gasPrice,
		Nonce:    r.Nonce,
		Gas:      r.Gas,
		To:       &to,
		Value:    value,
		Data:     common.FromHex(r.Data),
	}
}
