package test_test

import (
	"context"
	"fmt"
	"kms/wallet/app/api/model/dto"
	srv "kms/wallet/app/api/service"
	"kms/wallet/app/api/test/erc20"
	"kms/wallet/app/api/test/testnet"
	"kms/wallet/common/config"
	"kms/wallet/common/utils/errutil"
	"kms/wallet/common/utils/ethutil"
	"math/big"
	"os"
	"path"
	"testing"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"golang.org/x/crypto/sha3"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"
)

type KmsAccount struct {
	keyID   string
	address common.Address
}

type TxnTestSuite struct {
	suite.Suite
	kmsSrv      *srv.KmsSrv
	s           *srv.TxnSrv
	testNet     *testnet.TestNet
	kmsAccounts []KmsAccount
	erc20       *erc20.ERC20
}

func Test(t *testing.T) {
	suite.Run(t, new(TxnTestSuite))
}

func (t *TxnTestSuite) SetupSuite() {
	rootPath, err := os.Getwd()
	t.NoError(err)
	config.Init(path.Join(rootPath, "../../../env/.env.dev"))
	creds := credentials.NewStaticCredentialsProvider(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, "")
	cfg := errutil.HandleFatal(
		awscfg.LoadDefaultConfig(
			context.Background(),
			awscfg.WithCredentialsProvider(creds),
			awscfg.WithRegion(config.Env.AWS_REGION),
		),
	)

	t.kmsSrv = srv.NewKmsSrv(cfg)
	t.testNet = testnet.NewTestNet()
	t.s = srv.NewTxnSrv(t.testNet.ChainInfo.ChainID, t.kmsSrv)

	kmsAccountList, errWrap := t.kmsSrv.GetAccountList(nil, nil)
	if errWrap != nil {
		t.Fail(errWrap.Msg)
	}
	kmsAccounts := []KmsAccount{}
	for _, account := range kmsAccountList.Accounts {
		if account.Address != "" {
			kmsAccounts = append(kmsAccounts, KmsAccount{keyID: account.KeyID, address: common.HexToAddress(account.Address)})
		}
	}
	t.kmsAccounts = kmsAccounts

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
		fromAccount = t.kmsAccounts[0]
		toAccount   = t.testNet.Accounts[5]
		sendAmount  = ethutil.ParseUnit("20", 18)
	)
	t.NoError(t.testNet.Faucet(fromAccount.address, "10"))
	t.NoError(t.erc20.Faucet(fromAccount.address, "20"))

	pnonce, err := t.testNet.Client.PendingNonceAt(context.Background(), fromAccount.address)
	t.NoError(err)
	calldata, err := t.erc20.TransferCallData(toAccount.Address, sendAmount)
	t.NoError(err)
	gas, err := t.testNet.Client.EstimateGas(context.Background(), ethereum.CallMsg{From: fromAccount.address, To: &t.erc20.CA, Data: calldata})
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

	receipt := t.signAndSendTxn(fromAccount.keyID, serializedTxn)
	t.EqualValues(1, receipt.Status, "Txn failed")

	afterBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	balIncreased := new(big.Int).Sub(afterBal, beforeBal)
	t.EqualValuesf(sendAmount, balIncreased, "expected balance to be increased %v but %v", sendAmount, balIncreased)
}

func (t *TxnTestSuite) Test_EIP1559Txn() {
	var (
		fromAccount = t.kmsAccounts[0]
		toAccount   = t.testNet.Accounts[0]
		sendAmount  = ethutil.ParseUnit("20", 18)
	)
	t.NoError(t.testNet.Faucet(fromAccount.address, "10"))
	t.NoError(t.erc20.Faucet(fromAccount.address, "20"))

	pnonce, err := t.testNet.Client.PendingNonceAt(context.Background(), fromAccount.address)
	t.NoError(err)
	calldata, err := t.erc20.TransferCallData(toAccount.Address, sendAmount)
	t.NoError(err)

	gas, err := t.testNet.Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: fromAccount.address,
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

	receipt := t.signAndSendTxn(fromAccount.keyID, serializedTxn)
	t.EqualValues(1, receipt.Status, "Txn failed")

	afterBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	balIncreased := new(big.Int).Sub(afterBal, beforeBal)
	t.EqualValuesf(sendAmount, balIncreased, "expected balance to be increased %v but %v", sendAmount, balIncreased)
}

func (t *TxnTestSuite) Test_AceesListTxn() {
	var (
		fromAccount = t.kmsAccounts[0]
		toAccount   = t.testNet.Accounts[0]
		sendAmount  = ethutil.ParseUnit("20", 18)
	)
	t.NoError(t.testNet.Faucet(fromAccount.address, "10"))
	t.NoError(t.erc20.Faucet(fromAccount.address, "20"))

	pnonce, err := t.testNet.Client.PendingNonceAt(context.Background(), fromAccount.address)
	t.NoError(err)
	calldata, err := t.erc20.TransferCallData(toAccount.Address, sendAmount)
	t.NoError(err)

	hashForStorageOfFromAddrBal := sha3.NewLegacyKeccak256()
	hashForStorageOfFromAddrBal.Write((append(ethutil.PadLeftTo32Bytes(fromAccount.address.Bytes()), ethutil.PadLeftTo32Bytes([]byte{0})...)))
	StorageOfFromAddrBal := hashForStorageOfFromAddrBal.Sum(nil)

	hashForStorageOfToAddrBal := sha3.NewLegacyKeccak256()
	hashForStorageOfToAddrBal.Write((append(ethutil.PadLeftTo32Bytes(fromAccount.address.Bytes()), ethutil.PadLeftTo32Bytes([]byte{0})...)))
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
		From:       fromAccount.address,
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

	receipt := t.signAndSendTxn(fromAccount.keyID, serializedTxn)
	t.EqualValues(1, receipt.Status, "Txn failed")

	afterBal, err := t.erc20.BalanceOf(toAccount.Address, nil)
	t.NoError(err)

	balIncreased := new(big.Int).Sub(afterBal, beforeBal)
	t.EqualValuesf(sendAmount, balIncreased, "expected balance to be increased %v but %v", sendAmount, balIncreased)
}

func (t *TxnTestSuite) signAndSendTxn(keyID string, serializedTxn []byte) *types.Receipt {
	// sign with kms
	signedTxnRes, errWrap := t.s.SignSerializedTxn(&dto.TxnDTO{
		KeyID:         keyID,
		SerializedTxn: common.Bytes2Hex(serializedTxn),
	})
	if errWrap != nil {
		t.Fail(errWrap.Msg)
	}
	signedTxn := new(types.Transaction)
	t.NoError(signedTxn.UnmarshalBinary(common.FromHex(signedTxnRes.SignedTxn)))

	// send & mine
	t.NoError(t.testNet.Client.SendTransaction(context.Background(), signedTxn))
	t.testNet.Client.Commit()

	receipt, err := bind.WaitMined(context.Background(), t.testNet.Client, signedTxn)
	t.NoError(err)

	return receipt
}
