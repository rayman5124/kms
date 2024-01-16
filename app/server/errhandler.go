package server

import (
	"errors"
	"kms/wallet/app/api/model/dto"
	"kms/wallet/common/errs"
	"kms/wallet/common/logger"
	"kms/wallet/common/utils/timeutil"

	"github.com/gofiber/fiber/v2"
)

func ErrHandler(c *fiber.Ctx, err error) error {
	var (
		fiberErr  *fiber.Error
		customErr *errs.CusErr
		code      = errs.Errs["UnhandledServerErr"].Code
		errType   = errs.Errs["UnhandledServerErr"].Type
		msg       = ""
	)

	if errors.As(err, &customErr) {
		code = customErr.Code
		errType = customErr.Type
		switch code / 100 {
		case 4:
			msg = customErr.Inner.Error()
		case 5, 6:
			logger.Error().E(customErr.Inner).D("trace", customErr.Trace).D("func", customErr.Func).W(customErr.Type)
		}
	} else if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		switch {
		case code == fiber.StatusNotFound:
			errType = fiber.ErrNotFound.Message
		case code == fiber.StatusBadRequest:
			errType = fiberErr.Message
		default:
			logger.Error().E(err).W("unhandled fiber error")
		}
	} else {
		logger.Error().E(err).W(errType)
	}

	return c.Status(code).JSON(&dto.ErrRes{
		Status:    code,
		Timestamp: timeutil.FormatNow(),
		Method:    c.Method(),
		Path:      c.Path(),
		Message:   []string{errType, msg},
	})
}
