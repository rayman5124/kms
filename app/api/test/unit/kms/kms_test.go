package kms_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"kms/wallet/app/api/model/dto"
	"kms/wallet/app/api/test/common/http"
	"kms/wallet/app/server"
	"kms/wallet/common/config"
	"kms/wallet/common/logger"

	ctrl "kms/wallet/app/api/controller"
	srv "kms/wallet/app/api/service"
	"strings"
	"testing"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
)

type KmsTestSuite struct {
	suite.Suite
	app *fiber.App
}

var (
	curEnv = flag.String("env", "local", "environment")
	log    = flag.Bool("log", false, "log")
)

// 스킵할 테스트 선정
func (t *KmsTestSuite) BeforeTest(suiteName, testName string) {
	// "Test_CreateAccount", "Test_GetAccountList", "Test_DeleteAccount", "Test_GetAddress", "Test_ImportAccount"

	skips := []string{"Test_DeleteAccount", "Test_ImportAccount", "Test_DeleteAccount", "Test_GetAddress"}
	if slices.Contains(skips, testName) {
		t.T().Skip()
	}
}

func (t *KmsTestSuite) SetupSuite() {
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

	server := server.New()
	kmsSrv := srv.NewKmsSrv(kmsClient)
	ctrl.NewKmsCtrl(kmsSrv).BootStrap(server.App)

	t.app = server.App
}

func (t *KmsTestSuite) Test_CreateAccount() {
	resData, err := http.Request(t.app, "POST", "/create/account", nil)
	t.NoError(err)

	if resData.Status == fiber.StatusCreated {
		var resolvedRes dto.AccountRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

func (t *KmsTestSuite) Test_GetAccountList() {
	queryVal := []string{"limit=12"}
	path := fmt.Sprintf("/accounts?%s", strings.Join(queryVal, "&"))
	resData, err := http.Request(t.app, "GET", path, nil)
	t.NoError(err)

	if resData.Status == fiber.StatusOK {
		var resolvedRes dto.AccountListRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

func (t *KmsTestSuite) Test_DeleteAccount() {
	targetKeyID := "f50a9229-e7c7-45ba-b06c-8036b894424e"
	path := fmt.Sprintf("/accounts/%s", targetKeyID)
	resData, err := http.Request(t.app, "DELETE", path, nil)
	t.NoError(err)

	if resData.Status == fiber.StatusOK {
		var resolvedRes dto.AccountDeletionRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

func (t *KmsTestSuite) Test_GetAddress() {
	targetKeyID := "76eab66d-7bfd-49ac-827e-f9e04aed098e"
	path := fmt.Sprintf("/accounts/%s", targetKeyID)
	resData, err := http.Request(t.app, "GET", path, nil)
	t.NoError(err)

	if resData.Status == fiber.StatusOK {
		var resolvedRes dto.AddressRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

// local-kms 에서 ecc_secg_p256k1 스펙의 키를 외부에서 주입하는게 현재 불가능해서 로컬테스트 불가
func (t *KmsTestSuite) Test_ImportAccount() {
	ecdsaPK, err := crypto.GenerateKey()
	t.NoError(err)
	pk := common.Bytes2Hex(crypto.FromECDSA(ecdsaPK))

	reqBodyDto := &dto.PkReq{PK: pk}
	reqBody, err := json.Marshal(reqBodyDto)
	t.NoError(err)

	resData, err := http.Request(t.app, "POST", "/import/account", reqBody)
	t.NoError(err)

	if resData.Status == fiber.StatusCreated {
		var resolvedRes dto.AccountRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes dto.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}

}

func Test(t *testing.T) {
	suite.Run(t, new(KmsTestSuite))

}
