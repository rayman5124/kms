package server

import (
	"errors"
	"kms/wallet/app/api/model/res"
	"kms/wallet/common/errwrap"
	"kms/wallet/common/utils/timeutil"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ErrHandler(c *fiber.Ctx, err error) error {
	var (
		fiberErr  *fiber.Error
		customErr *errwrap.ErrWrap
		code      = 500
		msg       = []string{"Internal Server Error"}
		level     = "[Error]"
	)

	if errors.As(err, &customErr) {
		code = customErr.Code
		if (code / 100) == 4 {
			msg = strings.Split(customErr.Message, "\r\n")
			level = "[Info]"
		}
	} else if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		switch code {
		case fiber.StatusNotFound:
			msg = []string{fiber.ErrNotFound.Message}
			level = "[Info]"
		case fiber.StatusBadRequest:
			msg = []string{fiberErr.Message}
			level = "[Info]"
		}
	}
	c.Locals("level", level) // logger 스텍으로 넘길 값

	return c.Status(code).JSON(&res.ErrRes{
		Status:    code,
		Timestamp: timeutil.FormatNow(),
		Method:    c.Method(),
		Path:      c.Path(),
		Message:   msg,
	})
}
