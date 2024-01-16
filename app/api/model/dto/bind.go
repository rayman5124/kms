package dto

import (
	"errors"
	"fmt"
	"kms/wallet/common/errwrap"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	validate = validator.New()
)

func Init() {
	validate.RegisterValidation("marker", func(fl validator.FieldLevel) bool {
		re := regexp.MustCompile("^[\u0020-\u00FF]")
		return re.MatchString(fl.Field().String())
	})
}

func ShouldBind[T any](parser func(any) error) (*T, *errwrap.ErrWrap) {
	var data T
	if err := parser(&data); err != nil {
		return nil, errwrap.ClientErr(err)
	}

	if errs := validate.Struct(&data); errs != nil {
		errMsgs := []string{}
		for _, err := range errs.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: required", err.Field()))
			case "lte":
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: got '%v' should be less than %s", err.Field(), err.Value(), err.Param()))
			default:
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: got '%v' need correct %s", err.Field(), err.Value(), err.Tag()))
			}
		}
		return nil, errwrap.ClientErr(errors.New(strings.Join(errMsgs, "\r\n")))
	}

	return &data, nil
}
