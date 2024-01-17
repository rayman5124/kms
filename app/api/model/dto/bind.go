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
		re := regexp.MustCompile("[\u0020-\u00FF]")
		fmt.Println(re.MatchString(fl.Field().String()))
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
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: got '%v' should be less than or equal to %s", err.Field(), err.Value(), err.Param()))
			case "gte":
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: got '%v' should be less than or equal to %s", err.Field(), err.Value(), err.Param()))
			case "max":
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: max length is %s", err.Field(), err.Param()))
			case "min":
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: min length is %s", err.Field(), err.Param()))
			case "marker":
				errMsgs = append(errMsgs, fmt.Sprintln("marker is invalid"))
			default:
				errMsgs = append(errMsgs, fmt.Sprintf("field [%s]: got '%v' need correct %s", err.Field(), err.Value(), err.Tag()))
			}
		}
		return nil, (errs.BadRequestErr(fmt.Errorf(strings.Join(errMsgs, "\r\n"))))
	}

	return &data, nil
}
