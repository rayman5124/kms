package erc20

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"kms/wallet/common/utils/ethutil"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type ERC20 struct {
	client  *backends.SimulatedBackend
	abi     *abi.ABI
	ownerPK *ecdsa.PrivateKey
	CA      common.Address
}

func NewERC20WithDeploy(ownerPK *ecdsa.PrivateKey, client *backends.SimulatedBackend) (*ERC20, error) {
	abi, err := abi.JSON(strings.NewReader(ERC20Abi))
	if err != nil {
		return nil, err
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(ownerPK, client.Blockchain().Config().ChainID)
	if err != nil {
		return nil, err
	}

	CA, _, _, err := bind.DeployContract(transactor, abi, common.FromHex(ERC20ByteCode), client, "TestToken", "TTK", ethutil.ParseUnit("10000000000", 18))
	if err != nil {
		return nil, err
	}
	client.Commit()

	return &ERC20{client, &abi, ownerPK, CA}, nil
}

func (e *ERC20) TransferCallData(to common.Address, weiAmount *big.Int) ([]byte, error) {
	return e.abi.Pack("transfer", to, weiAmount)
}

func (e *ERC20) BalanceOfCallData(owner common.Address) ([]byte, error) {
	return e.abi.Pack("balanceOf", owner)
}

func (e *ERC20) BalanceOf(owner common.Address, blockNumber *big.Int) (*big.Int, error) {
	calldata, err := e.BalanceOfCallData(owner)
	if err != nil {
		return nil, err
	}

	bal, err := e.client.CallContract(context.Background(), ethereum.CallMsg{To: &e.CA, Data: calldata}, blockNumber)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(bal), nil
}

func (e *ERC20) Faucet(to common.Address, ethAmount string) error {
	ownerAddr := crypto.PubkeyToAddress(e.ownerPK.PublicKey)
	nonce, err := e.client.PendingNonceAt(context.Background(), ownerAddr)
	if err != nil {
		return err
	}
	gasPrice, err := e.client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	mintCalldata, err := e.abi.Pack("mint", to, ethutil.ParseUnit(ethAmount, 18))
	if err != nil {
		return err
	}
	gas, err := e.client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: ownerAddr,
		To:   &e.CA,
		Data: mintCalldata,
	})
	if err != nil {
		return err
	}

	signedTxn, err := types.SignNewTx(e.ownerPK, types.NewCancunSigner(e.client.Blockchain().Config().ChainID), &types.LegacyTx{
		Nonce:    nonce,
		To:       &e.CA,
		GasPrice: gasPrice,
		Gas:      gas,
		Data:     mintCalldata,
	})
	if err != nil {
		return err
	}

	err = e.client.SendTransaction(context.Background(), signedTxn)
	if err != nil {
		return err
	}

	beforeBal, err := e.BalanceOf(to, nil)
	if err != nil {
		return err
	}

	e.client.Commit()

	curBal, err := e.BalanceOf(to, nil)
	if err != nil {
		return err
	}

	if increased := new(big.Int).Sub(curBal, beforeBal); increased.Cmp(ethutil.ParseUnit(ethAmount, 18)) != 0 {
		return fmt.Errorf("faucet failed, balance increased %v expected: %v", increased, ethutil.ParseUnit(ethAmount, 18))
	}

	return nil

}
