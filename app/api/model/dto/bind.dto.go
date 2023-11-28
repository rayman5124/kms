package dto

import (
	"kms/wallet/common/utils/errutil"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

var (
	validate   = validator.New()
	enInstance = en.New()
	uni        = ut.New(enInstance, enInstance)
	trans, _   = uni.GetTranslator("en")
)

func ShouldBind[T any](parser func(any) error) (*T, *errutil.ErrWrap) {
	var data T
	if err := parser(&data); err != nil {
		return nil, errutil.NewErrWrap(500, "ShouldBindDTO_fiber.ctx_parser", err)
	}

	if err := validate.Struct(&data); err != nil {
		// errs := err.(validator.ValidationErrors)
		// for _, e := range errs {
		// 	fmt.Printf("err: %+#v", e)
		// }
		return nil, errutil.NewErrWrap(400, "", err)
	}

	return &data, nil
}
