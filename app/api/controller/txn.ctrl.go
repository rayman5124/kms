package controller

import (
	"kms/wallet/app/api/model/dto"
	srv "kms/wallet/app/api/service"

	"github.com/gofiber/fiber/v2"
)

type txnCtrl struct {
	txnSrv *srv.TxnSrv
}

func NewTxnCtrl(txnSrv *srv.TxnSrv) *txnCtrl {
	c := &txnCtrl{txnSrv}

	return c
}

func (c *txnCtrl) BootStrap(router fiber.Router) {
	router.Post("/sign/txn", c.SignSerializedTxn)
}

// @tags Transaction
// @summary Sign serialized transaction.
// @produce json
// @success 201 {object} res.SingedTxnRes
// @router  /api/sign/txn [post]
// @param   subject body dto.SerializedTxnDTO true "subject"
func (c *txnCtrl) SignSerializedTxn(ctx *fiber.Ctx) error {
	txnDTO, errWrap := dto.ShouldBind[dto.SerializedTxnDTO](ctx.BodyParser)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	signedTxnRes, errWrap := c.txnSrv.SignSerializedTxn(txnDTO)
	if errWrap != nil {
		return errWrap.CombineLayer()
	}

	return ctx.Status(fiber.StatusCreated).JSON(signedTxnRes)
}
