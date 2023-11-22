package controller

import (
	"kms/tutorial/api/model/dto"
	"kms/tutorial/api/model/res"
	srv "kms/tutorial/api/service"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
)

type txnCtrl struct {
	txnSrv *srv.TxnSrv
}

func NewTxnCtrl(txnSrv *srv.TxnSrv, router fiber.Router) *txnCtrl {
	ctrl := &txnCtrl{txnSrv}

	router.Post("/txn", ctrl.SendSerializedTxn)
	router.Get("/txn", ctrl.MakeSerializedTxn)

	return ctrl
}

// @summary Send serialized transaction.
// @produce json
// @success 201 {object} res.TxnRes
// @router  /api/txn [post]
// @param   subject body dto.TxnDTO true "subject"
func (c *txnCtrl) SendSerializedTxn(ctx *fiber.Ctx) error {
	txnDTO, errWrap := dto.ShouldBind[dto.TxnDTO](ctx.BodyParser)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	txnRes, errWrap := c.txnSrv.SendSerializedTxn(txnDTO)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	return ctx.Status(fiber.StatusCreated).JSON(txnRes)
}

// @summary Get Serialized Txn
// @produce json
// @success 200
// @router  /api/txn [get]
// @param conditions query dto.RawTxDTO true "raw tx"
func (c *txnCtrl) MakeSerializedTxn(ctx *fiber.Ctx) error {
	rawTxDTO, errWrap := dto.ShouldBind[dto.RawTxDTO](ctx.QueryParser)
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	fullTxn, errWrap := c.txnSrv.AutoFillTxn(rawTxDTO.ToLegacyTx(), common.HexToAddress(rawTxDTO.From))
	if errWrap != nil {
		return res.ProcessErrRes(errWrap, ctx)
	}

	serialized, _ := fullTxn.MarshalBinary()
	res := make(map[string]string)
	res["serialized"] = common.Bytes2Hex(serialized)
	return ctx.Status(fiber.StatusOK).JSON(res)
}
