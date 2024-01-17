package txn_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	ctrl "kms/wallet/app/api/controller"
	"kms/wallet/app/api/model/dto"
	srv "kms/wallet/app/api/service"
	"kms/wallet/app/api/test/common/erc20"
	"kms/wallet/app/api/test/common/http"
	"kms/wallet/app/api/test/common/testnet"

	awscfg "github.com/aws/aws-sdk-go-v2/config"

	"kms/wallet/app/server"
	"kms/wallet/common/config"
	"kms/wallet/common/logger"
	"kms/wallet/common/utils/ethutil"
	"math/big"
	"testing"

	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/slices"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/ethereum/go-ethereum"
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

	config.Init("../../../../../env/.env." + *curEnv)
	config.Env.Log = *log
	dto.Init()
	logger.Init(*curEnv)

	creds := credentials.NewStaticCredentialsProvider(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, "")
	awsCfg, err := awscfg.LoadDefaultConfig(
		context.Background(),
		awscfg.WithCredentialsProvider(creds),
		awscfg.WithRegion(config.Env.AWS_REGION),
	)
	t.NoError(err)

	var kmsClient *kms.Client
	if config.Env.ENV == "local" {
		kmsClient = kms.NewFromConfig(awsCfg, func(o *kms.Options) {
			o.BaseEndpoint = aws.String("http://localhost:8080")
		})
	} else {
		kmsClient = kms.NewFromConfig(awsCfg)
	}

	chainID, ok := new(big.Int).SetString(config.Env.CHAIN_ID, 10)
	if !ok {
		t.Fail("invalid chain id")
	}

	server := server.New()
	kmsSrv := srv.NewKmsSrv(kmsClient)
	txnSrv := srv.NewTxnSrv(chainID, kmsSrv)
	ctrl.NewKmsCtrl(kmsSrv).BootStrap(server.App)
	ctrl.NewTxnCtrl(txnSrv).BootStrap(server.App)

	t.app = server.App
	t.testNet = testnet.NewTestNet()

	t.erc20, err = erc20.NewERC20WithDeploy(t.testNet.Accounts[0].PK, t.testNet.Client)
	t.NoError(err)
}

func (t *TxnTestSuite) AfterTest(suiteName, testName string) {
	fmt.Println("")
}

func (t *TxnTestSuite) Test_LegacyTxn() {
	var (
		fromAccount   *dto.AccountRes
		toAccount     = t.testNet.Accounts[5]
		sendAmount, _ = ethutil.ParseUnit("20", 18)
	)
	fromAccount, err := t.getKmsAccount()
	t.NoError(err)
	t.NoError(t.erc20.Faucet(common.HexToAddress(fromAccount.Address), "20")) // estimate gas 실패를 방지하기 위해서

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

	reqBodyDto := &dto.TxnReq{KeyID: fromAccount.KeyID, SerializedTxn: common.Bytes2Hex(serializedTxn)}
	reqBody, _ := json.Marshal(reqBodyDto)
	resData, err := http.Request(t.app, "POST", "/sign/txn", reqBody)
	t.NoError(err)

	if resData.Status == fiber.StatusCreated {
		var resolvedRes dto.SingedTxnRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}

}

func (t *TxnTestSuite) Test_EIP1559Txn() {
	var (
		fromAccount   *dto.AccountRes
		toAccount     = t.testNet.Accounts[0]
		sendAmount, _ = ethutil.ParseUnit("20", 18)
	)
	fromAccount, err := t.getKmsAccount()
	t.NoError(err)
	t.NoError(t.erc20.Faucet(common.HexToAddress(fromAccount.Address), "20")) // estimate gas 실패를 방지하기 위해서

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

	reqBodyDto := &dto.TxnReq{KeyID: fromAccount.KeyID, SerializedTxn: common.Bytes2Hex(serializedTxn)}
	reqBody, _ := json.Marshal(reqBodyDto)
	resData, err := http.Request(t.app, "POST", "/sign/txn", reqBody)
	t.NoError(err)

	if resData.Status == fiber.StatusCreated {
		var resolvedRes dto.SingedTxnRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

func (t *TxnTestSuite) Test_AceesListTxn() {
	var (
		fromAccount   *dto.AccountRes
		toAccount     = t.testNet.Accounts[0]
		sendAmount, _ = ethutil.ParseUnit("20", 18)
	)
	fromAccount, err := t.getKmsAccount()
	t.NoError(err)
	t.NoError(t.erc20.Faucet(common.HexToAddress(fromAccount.Address), "20")) // estimate gas 실패를 방지하기 위해서

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

	reqBodyDto := &dto.TxnReq{KeyID: fromAccount.KeyID, SerializedTxn: common.Bytes2Hex(serializedTxn)}
	reqBody, _ := json.Marshal(reqBodyDto)
	resData, err := http.Request(t.app, "POST", "/sign/txn", reqBody)
	t.NoError(err)

	if resData.Status == fiber.StatusCreated {
		var resolvedRes dto.SingedTxnRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

// getAccountList를 통해서 존재하는 계정을 찾은다음 없으면 새로 만들어서 리턴
func (t *TxnTestSuite) getKmsAccount() (*dto.AccountRes, error) {
	resData, err := http.Request(t.app, "GET", "/accounts?limit=3", nil)
	if err != nil {
		return nil, err
	}

	if resData.Status == fiber.StatusOK {
		var accountListRes dto.AccountListRes
		json.Unmarshal(resData.Body, &accountListRes)

		for _, account := range accountListRes.Accounts {
			if account.Address != "" {
				return &account, nil
			}
		}
	} else {
		return nil, errors.New(string(resData.Body))
	}

	resData, err = http.Request(t.app, "POST", "/create/ccount", nil)
	if err != nil {
		return nil, err
	}

	if resData.Status == fiber.StatusCreated {
		var resolvedRes dto.AccountRes
		json.Unmarshal(resData.Body, &resolvedRes)

		return &resolvedRes, nil
	} else {
		return nil, errors.New(string(resData.Body))
	}
}

func Test(t *testing.T) {
	suite.Run(t, new(TxnTestSuite))
}
