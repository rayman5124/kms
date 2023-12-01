package kms_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"kms/wallet/app/api/model/dto"
	"kms/wallet/app/api/model/res"
	"kms/wallet/app/api/test/common/http"
	"kms/wallet/app/server"
	"kms/wallet/common/config"

	"strings"
	"testing"

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
	// "Test_CreateAccount", "Test_GetAccountList"

	skips := []string{}
	if slices.Contains(skips, testName) {
		t.T().Skip()
	}
}

func (t *KmsTestSuite) SetupSuite() {
	flag.Parse()

	t.NoError(config.Init("../../../../../env/.env." + *curEnv))
	config.Env.Log = *log
	dto.Init()

	server := server.NewServer()
	t.app = server.App
}

func (t *KmsTestSuite) Test_CreateAccount() {
	resData, err := http.Request(t.app, "POST", "/api/create/account", nil)
	t.NoError(err)

	if resData.Status == fiber.StatusCreated {
		var resolvedRes res.AccountRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes res.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

func (t *KmsTestSuite) Test_GetAccountList() {
	t.T().Log(t.T().Name())

	queryVal := []string{""}
	path := fmt.Sprintf("/api/account/list?%s", strings.Join(queryVal, "&"))
	resData, err := http.Request(t.app, "GET", path, nil)
	t.NoError(err)

	if resData.Status == fiber.StatusOK {
		var resolvedRes res.AccountListRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.T().Log(http.PrettyJson(resolvedRes))
	} else {
		var resolvedRes res.ErrRes
		t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
		t.Fail(http.PrettyJson(resolvedRes))
	}
}

// local-kms 에서 ecc_secg_p256k1 스펙의 키를 외부에서 주입하는게 현재 불가능해서 로컬테스트 불가
// func (t *KmsTestSuite) Test_ImportAccount() {
// 	ecdsaPK, err := crypto.GenerateKey()
// 	t.NoError(err)
// 	pk := common.Bytes2Hex(crypto.FromECDSA(ecdsaPK))

// 	reqBodyDto := &dto.ImportAccountDTO{PK: pk}
// 	reqBody, err := json.Marshal(reqBodyDto)
// 	t.NoError(err)

// 	resData, err := http.Request("POST", "/api/import/account", reqBody)
// 	t.NoError(err)
// 	var resolvedRes res.AccountRes
// 	t.NoError(json.Unmarshal(resData.Body, &resolvedRes))
// 	t.T().Log(http.PrettyJson(resolvedRes))

// }

func Test(t *testing.T) {
	suite.Run(t, new(KmsTestSuite))

}
