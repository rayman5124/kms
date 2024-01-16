package controller

import (
	"kms/wallet/app/api/model/dto"
	srv "kms/wallet/app/api/service"

	"github.com/gofiber/fiber/v2"
)

type kmsCtrl struct {
	kmsSrv *srv.KmsSrv
}

func NewKmsCtrl(kmsSrv *srv.KmsSrv) *kmsCtrl {
	return &kmsCtrl{kmsSrv}
}

func (c *kmsCtrl) BootStrap(router fiber.Router) {
	router.Post("/create/account", c.CreateAccount)
	router.Post("/import/account", c.ImportAccount)
	router.Get("/accounts", c.GetAccountList)
	router.Get("/accounts/:keyID", c.GetAccount)
	router.Delete("/accounts/:keyID", c.DeleteAccount)
}

// @tags Kms
// @summary Get account of target key id
// @produce json
// @success 200 {object} dto.AccountRes
// @router  /api/accounts/{keyID} [get]
// @param   keyID path string true "kms key-id"
func (c *kmsCtrl) GetAccount(ctx *fiber.Ctx) error {
	keyIdReq, err := dto.ShouldBind[dto.KeyIdReq](ctx.ParamsParser)
	if err != nil {
		return err
	}

	accountRes, err := c.kmsSrv.GetAccount(keyIdReq)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(accountRes)
}

// @tags Kms
// @summary Get accounst list
// @produce json
// @success 200 {object} dto.AccountListRes
// @router  /api/accounts [get]
// @param   keyID query dto.AccountListReq false "account list dto"
func (c *kmsCtrl) GetAccountList(ctx *fiber.Ctx) error {
	accountListReq, err := dto.ShouldBind[dto.AccountListReq](ctx.QueryParser)
	if err != nil {
		return err
	}

	accountListRes, err := c.kmsSrv.GetAccountList(accountListReq)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(accountListRes)
}

// @tags Kms
// @summary Create new account
// @produce json
// @success 201 {object} dto.AccountRes
// @router  /api/create/account [post]
func (c *kmsCtrl) CreateAccount(ctx *fiber.Ctx) error {
	accountRes, err := c.kmsSrv.CreateAccount()
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(accountRes)
}

// @tags Kms
// @summary Import account to kms
// @produce json
// @success 201 {object} dto.AccountRes
// @router  /api/import/account [post]
// @param   subject body dto.PkReq true "subject"
func (c *kmsCtrl) ImportAccount(ctx *fiber.Ctx) error {
	pkReq, err := dto.ShouldBind[dto.PkReq](ctx.BodyParser)
	if err != nil {
		return err
	}

	accountRes, err := c.kmsSrv.ImportAccount(pkReq)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(accountRes)
}

// @tags Kms
// @summary delete account of target key id
// @produce json
// @success 200 {object} dto.AccountDeletionRes
// @router  /api/accounts/{keyID} [delete]
// @param   keyID path string true "kms key-id"
func (c *kmsCtrl) DeleteAccount(ctx *fiber.Ctx) error {
	keyIdReq, err := dto.ShouldBind[dto.KeyIdReq](ctx.ParamsParser)
	if err != nil {
		return err
	}

	accountDeletionRes, err := c.kmsSrv.DeleteAccount(keyIdReq)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(accountDeletionRes)
}
