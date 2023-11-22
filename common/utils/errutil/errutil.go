package errutil

import (
	"errors"
	"fmt"
	"log"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
)

func HandleFatal[T any](val T, err error) T {
	if err != nil {
		log.Fatal(err.Error())
	}
	return val
}

type ErrWrap struct {
	IsErr bool
	Code  int
	Msg   string
}

func FilterAwsErr(err error) (int, error) {
	var (
		resErr *awshttp.ResponseError
		code   int
	)
	if errors.As(err, &resErr) {
		code = resErr.Response.StatusCode
		err = resErr.Err
	} else {
		code = 500
	}
	return code, err
}

func NewErrWrap(code int, label string, err error) *ErrWrap {
	var msg string

	switch code {
	case 400:
		msg = err.Error()
	case 500:
		msg = fmt.Sprintf("<%v>\n%v", label, err.Error())
	default:
		msg = fmt.Sprintf("<%v>\n%v", label, err.Error())
	}

	return &ErrWrap{true, code, msg}
}
