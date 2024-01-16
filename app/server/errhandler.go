package server

import (
	"errors"
	"kms/wallet/app/api/model/dto"
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
	)

	if errors.As(err, &customErr) {
		code = customErr.Code
		switch code / 100 {
		case 4:
			msg = strings.Split(customErr.Message, "\r\n")
		case 5:
			// err = customErr.CombineLayer()
		}
	} else if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		switch code {
		case fiber.StatusNotFound:
			msg = []string{fiber.ErrNotFound.Message}
		case fiber.StatusBadRequest:
			msg = []string{fiberErr.Message}
		}
	}

	return c.Status(code).JSON(&dto.ErrRes{
		Status:    code,
		Timestamp: timeutil.FormatNow(),
		Method:    c.Method(),
		Path:      c.Path(),
		Message:   msg,
	})
}
