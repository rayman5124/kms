package dto

import (
	"fmt"
	"kms/wallet/common/errs"
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

func ShouldBind[T any](parser func(any) error) (*T, error) {
	var data T
	if err := parser(&data); err != nil {
		return nil, errs.BadRequestErr(err)
	}

	if errors := validate.Struct(&data); errors != nil {
		errMsgs := []string{}
		for _, err := range errors.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: required", err.Field()))
			case "lte":
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: got '%v' should be less than %s", err.Field(), err.Value(), err.Param()))
			case "marker":
				errMsgs = append(errMsgs, fmt.Sprintln("field [%s]: got '%v' invalid marker"))
			default:
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: got '%v' need correct %s", err.Field(), err.Value(), err.Tag()))
			}
		}
		return nil, (errs.BadRequestErr(fmt.Errorf(strings.Join(errMsgs, "\r\n"))))
	}

	return &data, nil
}
