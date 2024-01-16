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
		msg       = []string{errs.Errs["UnhandledServerErr"].Type}
	)

	if errors.As(err, &customErr) {
		code = customErr.Code
		msg[0] = customErr.Type
		switch code / 100 {
		case 4:
			msg = append(msg, customErr.Inner.Error())
		case 5, 6:
			logger.Error().E(customErr.Inner).D("trace", customErr.Trace).D("func", customErr.Func).W(customErr.Type)
		}
	} else if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		switch {
		case code == fiber.StatusNotFound:
			msg[0] = fiber.ErrNotFound.Message
		case code == fiber.StatusBadRequest:
			msg[0] = fiberErr.Message
		default:
			logger.Error().E(err).W("unhandled fiber error")
		}
	} else {
		logger.Error().E(err).W(msg[0])
	}

	return c.Status(code).JSON(&dto.ErrRes{
		Status:    code,
		Timestamp: timeutil.FormatNow(),
		Method:    c.Method(),
		Path:      c.Path(),
		Message:   msg,
	})
}
