package controller

import (
	"kms/tutorial/api/model/dto"
	"kms/tutorial/api/model/res"
	srv "kms/tutorial/api/service"

	"github.com/gofiber/fiber/v2"
)

type txnCtrl struct {
	txnSrv *srv.TxnSrv
}

func NewTxnCtrl(txnSrv *srv.TxnSrv, router fiber.Router) *txnCtrl {
	ctrl := &txnCtrl{txnSrv}

	router.Post("/sign/txn", ctrl.SignSerializedTxn)

	return ctrl
}

// @summary Sign serialized transaction.
// @produce json
// @success 201 {object} res.SingedTxnRes
// @router  /api/sign/txn [post]
// @param   subject body dto.TxnDTO true "subject"
func (c *txnCtrl) SignSerializedTxn(ctx *fiber.Ctx) error {
	txnDTO, errWrap := dto.ShouldBind[dto.TxnDTO](ctx.BodyParser)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	signedTxnRes, errWrap := c.txnSrv.SignSerializedTxn(txnDTO)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	return ctx.Status(fiber.StatusCreated).JSON(signedTxnRes)
}
