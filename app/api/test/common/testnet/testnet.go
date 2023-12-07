package testnet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"kms/wallet/common/utils/errutil"
	"kms/wallet/common/utils/ethutil"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Account struct {
	PK      *ecdsa.PrivateKey
	Address common.Address
}

type ChainInfo struct {
	ChainID  *big.Int
	GasPrice *big.Int
}

type TestNet struct {
	Accounts      []Account
	Client        *backends.SimulatedBackend
	ChainInfo     ChainInfo
	faucetAccount *ecdsa.PrivateKey
}

var (
	genesis_accounts        = 10
	genesis_alloc_amount, _ = ethutil.ParseUnit("1000000", 18)
)

func NewTestNet() *TestNet {
	var genesisAllocInfo = map[common.Address]core.GenesisAccount{}
	var faucetAccount *ecdsa.PrivateKey

	accounts := []Account{}
	for i := 0; i < genesis_accounts+1; i++ {
		ecdsPK := errutil.HandleFatal(crypto.GenerateKey())
		// pk := common.Bytes2Hex(crypto.FromECDSA(ecdsPK))
		address := crypto.PubkeyToAddress(ecdsPK.PublicKey)

		if i < genesis_accounts {
			accounts = append(accounts, Account{
				PK:      ecdsPK,
				Address: address,
			})
		} else {
			faucetAccount = ecdsPK
		}
		genesisAllocInfo[address] = core.GenesisAccount{Balance: genesis_alloc_amount}
	}

	client := backends.NewSimulatedBackend(genesisAllocInfo, 10000000)
	gasPrice := errutil.HandleFatal(client.SuggestGasPrice(context.Background()))
	return &TestNet{
		Accounts:      accounts,
		Client:        client,
		ChainInfo:     ChainInfo{ChainID: client.Blockchain().Config().ChainID, GasPrice: gasPrice},
		faucetAccount: faucetAccount,
	}
}

func (t *TestNet) Faucet(to common.Address, ethAmount string) error {
	val, err := ethutil.ParseUnit(ethAmount, 18)
	if err != nil {
		return err
	}
	if ethAmount == "" {
		val, _ = ethutil.ParseUnit("100", 18)
	}

	gasPrice, err := t.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	pnonce, err := t.Client.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.faucetAccount.PublicKey))
	if err != nil {
		return nil
	}
	singedTxn, err := types.SignNewTx(t.faucetAccount, types.NewCancunSigner(t.ChainInfo.ChainID), &types.LegacyTx{
		To:       &to,
		Value:    val,
		GasPrice: gasPrice,
		Gas:      21000,
		Nonce:    pnonce,
	})
	if err != nil {
		return err
	}

	// send & mine
	if err := t.Client.SendTransaction(context.Background(), singedTxn); err != nil {
		return err
	}
	beforeBlock := t.Client.Blockchain().CurrentBlock().Number
	t.Client.Commit()

	beforeBal, err := t.Client.BalanceAt(context.Background(), to, beforeBlock)
	if err != nil {
		return nil
	}
	curBal, err := t.Client.BalanceAt(context.Background(), to, nil)
	if err != nil {
		return nil
	}
	if increased := new(big.Int).Sub(curBal, beforeBal); increased.Cmp(val) != 0 {
		return fmt.Errorf("faucet failed, balance increased %v expected: %v", increased, val)
	}
	return nil
}
