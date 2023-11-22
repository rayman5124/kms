package controller

import (
	"kms/tutorial/api/model/dto"
	"kms/tutorial/api/model/res"
	srv "kms/tutorial/api/service"

	"github.com/gofiber/fiber/v2"
)

type kmsCtrl struct {
	kmsSrv *srv.KmsSrv
}

func NewKmsCtrl(kmsSrv *srv.KmsSrv, router fiber.Router) *kmsCtrl {
	ctrl := &kmsCtrl{kmsSrv}

	router.Post("/account", ctrl.CreateAccount)
	router.Post("/import/account", ctrl.ImportAccount)
	router.Get("/account/keyID/:keyID", ctrl.GetAccount)
	router.Get("/account/list", ctrl.GetAccountList)

	return ctrl
}

// @summary Get account of target key id
// @produce json
// @success 200 {object} res.AccountRes
// @router  /api/account/keyID/{keyID} [get]
// @param   keyID path string true "kms key-id"
func (c *kmsCtrl) GetAccount(ctx *fiber.Ctx) error {
	accountDTO, errWrap := dto.ShouldBind[dto.AccountDTO](ctx.ParamsParser)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	accountRes, errWrap := c.kmsSrv.GetAccount(accountDTO)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	return ctx.Status(fiber.StatusOK).JSON(accountRes)
}

// @summary Get accounst list
// @produce json
// @success 200 {object} res.AccountListRes
// @router  /api/account/list [get]
// @param   keyID query dto.AccountListDTO false "account list dto"
func (c *kmsCtrl) GetAccountList(ctx *fiber.Ctx) error {
	accountListDTO, errWrap := dto.ShouldBind[dto.AccountListDTO](ctx.QueryParser)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	accountListRes, errWrap := c.kmsSrv.GetAccountList(accountListDTO.Limit, accountListDTO.Marker)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	return ctx.Status(fiber.StatusOK).JSON(accountListRes)
}

// @summary Create new account
// @produce json
// @success 201 {object} res.AccountRes
// @router  /api/account [post]
func (c *kmsCtrl) CreateAccount(ctx *fiber.Ctx) error {
	accountRes, errWrap := c.kmsSrv.CreateAccount()
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	return ctx.Status(fiber.StatusCreated).JSON(accountRes)
}

// @summary Import account to kms
// @produce json
// @success 201 {object} res.AccountRes
// @router  /api/import/account [post]
// @param   subject body dto.ImportAccountDTO true "subject"
func (c *kmsCtrl) ImportAccount(ctx *fiber.Ctx) error {
	importAccountDTO, errWrap := dto.ShouldBind[dto.ImportAccountDTO](ctx.BodyParser)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}
	accountRes, errWrap := c.kmsSrv.ImportAccount(importAccountDTO.PK)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	return ctx.Status(fiber.StatusCreated).JSON(accountRes)
}
