package res

import (
	"fmt"
	"kms/wallet/common/utils/errutil"
	"kms/wallet/common/utils/timeutil"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ClientErrRes struct {
	Status    int      `json:"status"`
	Timestamp string   `json:"timestamp"`
	Method    string   `json:"method"`
	Path      string   `json:"path"`
	Message   []string `json:"message"`
}

func ProcessErrRes(errWrap *errutil.ErrWrap, ctx *fiber.Ctx) error {
	switch errWrap.Code {
	case 400:
		return ctx.Status(fiber.StatusBadRequest).JSON(&ClientErrRes{
			Status:    fiber.StatusBadRequest,
			Timestamp: timeutil.FormatNow(),
			Method:    ctx.Method(),
			Path:      ctx.Path(),
			Message:   strings.Split(errWrap.Msg, "\n"),
		})
	case 500:
		fmt.Printf("%v\n", errWrap.Msg)
		return fiber.ErrInternalServerError
	default:
		fmt.Printf("%v\n", errWrap.Msg)
		return fiber.ErrInternalServerError
	}
}
