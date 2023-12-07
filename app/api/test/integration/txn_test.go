package txn_test

// erc20 transfer 트렌젝션 서명 후 해당 트렌젝션이 성공하는지 여부까지 확인하는 테스트

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"kms/wallet/app/api/model/dto"
	"kms/wallet/app/api/model/res"
	"kms/wallet/app/api/test/common/erc20"
	"kms/wallet/app/api/test/common/http"
	"kms/wallet/app/api/test/common/testnet"
	"kms/wallet/app/server"
	"kms/wallet/common/config"
	"kms/wallet/common/utils/ethutil"
	"math/big"
	"testing"

	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/slices"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type TxnTestSuite struct {
	suite.Suite
	app     *fiber.App
	testNet *testnet.TestNet
	erc20   *erc20.ERC20
}

var (
	curEnv = flag.String("env", "local", "environment")
	log    = flag.Bool("log", false, "log")
)

// 스킵할 테스트 선정
func (t *TxnTestSuite) BeforeTest(suiteName, testName string) {
	// "Test_LegacyTxn", "Test_EIP1559Txn", "Test_AceesListTxn"

	skips := []string{}
	if slices.Contains(skips, testName) {
		t.T().Skip()
	}
}

func (t *TxnTestSuite) SetupSuite() {
	flag.Parse()

	t.NoError(config.Init("../../../../env/.env." + *curEnv))
	config.Env.Log = *log
	dto.Init()

	server := server.New()
	t.app = server.App
	t.testNet = testnet.NewTestNet()

	var err error
	t.erc20, err = erc20.NewERC20WithDeploy(t.testNet.Accounts[0].PK, t.testNet.Client)
	t.NoError(err)
}

func (t *TxnTestSuite) AfterTest(suiteName, testName string) {
	fmt.Println("")
}

// 각 테스트 실행전에 실행됨
func (t *TxnTestSuite) SetupTest() {
}

func (t *TxnTestSuite) Test_LegacyTxn() {
	var (
		fromAccount   *res.AccountRes
		toAccount     = t.testNet.Accounts[5]
		sendAmount, _ = ethutil.ParseUnit("20", 18)
	)
	fromAccount, err := t.getKmsAccount()
	t.NoError(err)

	t.NoError(t.testNet.Faucet(common.HexToAddress(fromAccount.Address), "10"))
	t.NoError(t.erc20.Faucet(common.HexToAddress(fromAccount.Address), "20"))

	pnonce, err := t.testNet.Client.PendingNonceAt(context.Background(), common.HexToAddress(fromAccount.Address))
	t.NoError(err)
	calldata, err := t.erc20.TransferCallData(toAccount.Address, sendAmount)
	t.NoError(err)
	gas, err := t.testNet.Client.EstimateGas(context.Background(), ethereum.CallMsg{From: common.HexToAddress(fromAccount.Address), To: &t.erc20.CA, Data: calldata})
	t.NoError(err)

	serializedTxn, err := types.NewTx(&types.LegacyTx{
		To:       &t.erc20.CA,
		GasPrice: t.testNet.ChainInfo.GasPrice,
		Gas:      gas,
		Nonce:    pnonce,
		Data:     calldata,
	}).MarshalBinary()
	t.NoError(err)

	beforeBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	receipt, err := t.signAndSendTxn(fromAccount.KeyID, serializedTxn)
	t.NoError(err)
	t.EqualValues(1, receipt.Status, "Txn failed")

	afterBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	balIncreased := new(big.Int).Sub(afterBal, beforeBal)
	t.EqualValuesf(sendAmount, balIncreased, "expected balance to be increased %v but %v", sendAmount, balIncreased)
}

func (t *TxnTestSuite) Test_EIP1559Txn() {
	var (
		fromAccount   *res.AccountRes
		toAccount     = t.testNet.Accounts[0]
		sendAmount, _ = ethutil.ParseUnit("20", 18)
	)
	fromAccount, err := t.getKmsAccount()
	t.NoError(err)

	t.NoError(t.testNet.Faucet(common.HexToAddress(fromAccount.Address), "10"))
	t.NoError(t.erc20.Faucet(common.HexToAddress(fromAccount.Address), "20"))

	pnonce, err := t.testNet.Client.PendingNonceAt(context.Background(), common.HexToAddress(fromAccount.Address))
	t.NoError(err)
	calldata, err := t.erc20.TransferCallData(toAccount.Address, sendAmount)
	t.NoError(err)

	gas, err := t.testNet.Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: common.HexToAddress(fromAccount.Address),
		To:   &t.erc20.CA, Data: calldata,
	})
	t.NoError(err)
	gasTipCap, err := t.testNet.Client.SuggestGasTipCap(context.Background())
	gasFeeCap := new(big.Int).Add(t.testNet.Client.Blockchain().CurrentBlock().BaseFee, gasTipCap)
	t.NoError(err)

	serializedTxn, _ := types.NewTx(&types.DynamicFeeTx{
		To:        &t.erc20.CA,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Nonce:     pnonce,
		Gas:       gas,
		Data:      calldata,
	}).MarshalBinary()

	beforeBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	receipt, err := t.signAndSendTxn(fromAccount.KeyID, serializedTxn)
	t.NoError(err)
	t.EqualValues(1, receipt.Status, "Txn failed")

	afterBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	balIncreased := new(big.Int).Sub(afterBal, beforeBal)
	t.EqualValuesf(sendAmount, balIncreased, "expected balance to be increased %v but %v", sendAmount, balIncreased)
}

func (t *TxnTestSuite) Test_AceesListTxn() {
	var (
		fromAccount   *res.AccountRes
		toAccount     = t.testNet.Accounts[0]
		sendAmount, _ = ethutil.ParseUnit("20", 18)
	)
	fromAccount, err := t.getKmsAccount()
	t.NoError(err)
	t.NoError(t.testNet.Faucet(common.HexToAddress(fromAccount.Address), "10"))
	t.NoError(t.erc20.Faucet(common.HexToAddress(fromAccount.Address), "20"))

	pnonce, err := t.testNet.Client.PendingNonceAt(context.Background(), common.HexToAddress(fromAccount.Address))
	t.NoError(err)
	calldata, err := t.erc20.TransferCallData(toAccount.Address, sendAmount)
	t.NoError(err)

	hashForStorageOfFromAddrBal := sha3.NewLegacyKeccak256()
	hashForStorageOfFromAddrBal.Write((append(ethutil.PadLeftTo32Bytes(common.HexToAddress(fromAccount.Address).Bytes()), ethutil.PadLeftTo32Bytes([]byte{0})...)))
	StorageOfFromAddrBal := hashForStorageOfFromAddrBal.Sum(nil)

	hashForStorageOfToAddrBal := sha3.NewLegacyKeccak256()
	hashForStorageOfToAddrBal.Write((append(ethutil.PadLeftTo32Bytes(common.HexToAddress(fromAccount.Address).Bytes()), ethutil.PadLeftTo32Bytes([]byte{0})...)))
	StorageOfToAddrBal := hashForStorageOfToAddrBal.Sum(nil)

	accessList := types.AccessList{}
	accessList = append(accessList, types.AccessTuple{
		Address: t.erc20.CA,
		StorageKeys: []common.Hash{
			common.BytesToHash(StorageOfFromAddrBal),
			common.BytesToHash(StorageOfToAddrBal),
		},
	})
	gas, err := t.testNet.Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:       common.HexToAddress(fromAccount.Address),
		To:         &t.erc20.CA,
		Data:       calldata,
		AccessList: accessList,
	})
	t.NoError(err)

	serializedTxn, err := types.NewTx(&types.AccessListTx{
		To:         &t.erc20.CA,
		Nonce:      pnonce,
		Gas:        gas,
		Data:       calldata,
		AccessList: accessList,
		GasPrice:   t.testNet.ChainInfo.GasPrice,
	}).MarshalBinary()
	t.NoError(err)

	beforeBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	receipt, err := t.signAndSendTxn(fromAccount.KeyID, serializedTxn)
	t.NoError(err)
	t.EqualValues(1, receipt.Status, "Txn failed")

	afterBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	balIncreased := new(big.Int).Sub(afterBal, beforeBal)
	t.EqualValuesf(sendAmount, balIncreased, "expected balance to be increased %v but %v", sendAmount, balIncreased)
}

func (t *TxnTestSuite) signAndSendTxn(keyID string, serializedTxn []byte) (*types.Receipt, error) {
	reqBodyDto := &dto.SerializedTxnDTO{KeyID: keyID, SerializedTxn: common.Bytes2Hex(serializedTxn)}
	reqBody, _ := json.Marshal(reqBodyDto)
	resData, err := http.Request(t.app, "POST", "/api/sign/txn", reqBody)
	if err != nil {
		return nil, err
	}

	var signedTxnRes *res.SingedTxnRes
	if resData.Status == fiber.StatusCreated {
		json.Unmarshal(resData.Body, &signedTxnRes)
	} else {
		return nil, errors.New(string(resData.Body))
	}

	signedTxn := new(types.Transaction)
	t.NoError(signedTxn.UnmarshalBinary(common.FromHex(signedTxnRes.SignedTxn)))

	// send & mine
	t.NoError(t.testNet.Client.SendTransaction(context.Background(), signedTxn))
	t.testNet.Client.Commit()

	receipt, err := bind.WaitMined(context.Background(), t.testNet.Client, signedTxn)
	t.NoError(err)

	return receipt, nil
}

// getAccountList를 통해서 존재하는 계정을 찾은다음 없으면 새로 만들어서 리턴
func (t *TxnTestSuite) getKmsAccount() (*res.AccountRes, error) {
	resData, err := http.Request(t.app, "GET", "/api/account/list", nil)
	if err != nil {
		return nil, err
	}

	if resData.Status == fiber.StatusOK {
		var accountListRes res.AccountListRes
		json.Unmarshal(resData.Body, &accountListRes)

		for _, account := range accountListRes.Accounts {
			if account.Address != "" {
				return &account, nil
			}
		}
	} else {
		return nil, errors.New(string(resData.Body))
	}

	resData, err = http.Request(t.app, "POST", "/api/create/account", nil)
	if err != nil {
		return nil, err
	}

	if resData.Status == fiber.StatusCreated {
		var resolvedRes res.AccountRes
		json.Unmarshal(resData.Body, &resolvedRes)

		return &resolvedRes, nil
	} else {
		return nil, errors.New(string(resData.Body))
	}
}

func Test(t *testing.T) {
	suite.Run(t, new(TxnTestSuite))
}
