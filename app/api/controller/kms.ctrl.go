package controller

import (
	"kms/wallet/app/api/model/dto"
	srv "kms/wallet/app/api/service"

	"github.com/gofiber/fiber/v2"
)

type kmsCtrl struct {
	kmsSrv *srv.KmsSrv
}

func NewKmsCtrl(kmsSrv *srv.KmsSrv, router fiber.Router) *kmsCtrl {
	c := &kmsCtrl{kmsSrv}

	router.Post("/create/account", c.CreateAccount)
	router.Post("/import/account", c.ImportAccount)
	router.Get("/address/keyID/:keyID", c.GetAddress)
	router.Get("/account/list", c.GetAccountList)

	return c
}

// @tags Kms
// @summary Get address of target key id
// @produce json
// @success 200 {object} res.AddressRes
// @router  /api/address/keyID/{keyID} [get]
// @param   keyID path string true "kms key-id"
func (c *kmsCtrl) GetAddress(ctx *fiber.Ctx) error {
	AddressDTO, errWrap := dto.ShouldBind[dto.AddressDTO](ctx.ParamsParser)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	addressRes, errWrap := c.kmsSrv.GetAddress(AddressDTO)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	return ctx.Status(fiber.StatusOK).JSON(addressRes)
}

// @tags Kms
// @summary Get accounst list
// @produce json
// @success 200 {object} res.AccountListRes
// @router  /api/account/list [get]
// @param   keyID query dto.AccountListDTO false "account list dto"
func (c *kmsCtrl) GetAccountList(ctx *fiber.Ctx) error {
	accountListDTO, errWrap := dto.ShouldBind[dto.AccountListDTO](ctx.QueryParser)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	accountListRes, errWrap := c.kmsSrv.GetAccountList(accountListDTO.Limit, accountListDTO.Marker)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	return ctx.Status(fiber.StatusOK).JSON(accountListRes)
}

// @tags Kms
// @summary Create new account
// @produce json
// @success 201 {object} res.AccountRes
// @router  /api/create/account [post]
func (c *kmsCtrl) CreateAccount(ctx *fiber.Ctx) error {
	accountRes, errWrap := c.kmsSrv.CreateAccount()
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	return ctx.Status(fiber.StatusCreated).JSON(accountRes)
}

// @tags Kms
// @summary Import account to kms
// @produce json
// @success 201 {object} res.AccountRes
// @router  /api/import/account [post]
// @param   subject body dto.ImportAccountDTO true "subject"
func (c *kmsCtrl) ImportAccount(ctx *fiber.Ctx) error {
	importAddressDTO, errWrap := dto.ShouldBind[dto.ImportAccountDTO](ctx.BodyParser)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}
	accountRes, errWrap := c.kmsSrv.ImportAccount(importAddressDTO.PK)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	return ctx.Status(fiber.StatusCreated).JSON(accountRes)
}
