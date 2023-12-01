package errwrap

import (
	"errors"
	"fmt"
	"strings"

	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type ErrWrap struct {
	inner   error
	Code    int    `json:"code"`
	Message string `json:"message"`
	Layer   string
}

func ClientErr(err error) *ErrWrap {
	return &ErrWrap{
		inner:   err,
		Code:    400,
		Message: err.Error(),
	}
}

func ServerErr(err error) *ErrWrap {
	return &ErrWrap{
		inner:   err,
		Code:    500,
		Message: err.Error(),
	}
}

func AwsErr(err error) *ErrWrap {
	var (
		awsErrRes        *awsHttp.ResponseError
		notFoundErr      *awsTypes.NotFoundException
		invalidMarkerErr *awsTypes.InvalidMarkerException
		code             = 500
	)

	if errors.As(err, &notFoundErr) {
		code = 400
		if splitedMsg := strings.Split(*notFoundErr.Message, "/"); len(splitedMsg) > 1 {
			// not found
			err = fmt.Errorf("keyId '%v not found", splitedMsg[1])
		} else {
			// invalid key
			err = fmt.Errorf(*notFoundErr.Message)
		}
	} else if errors.As(err, &invalidMarkerErr) {
		code = 400
		newMsg, _ := strings.CutSuffix(invalidMarkerErr.Error(), ": ")
		err = fmt.Errorf(newMsg)
	} else if errors.As(err, &awsErrRes) {
		// err = awsErrRes.ResponseError.Err
	}

	return &ErrWrap{
		inner:   err,
		Code:    code,
		Message: err.Error(),
	}
}

func (e *ErrWrap) Error() string {
	return e.Message
}

func (e *ErrWrap) Inner() error {
	return e.inner
}

func (e *ErrWrap) AddLayer(layers ...string) *ErrWrap {
	if e.Layer != "" {
		preLayers := strings.Split(e.Layer, " -> ")
		layers = append(layers, preLayers...)
	}
	e.Layer = strings.Join(layers, " -> ")
	return e
}

func (e *ErrWrap) ChangeCode(code int) *ErrWrap {
	e.Code = code
	return e
}

func (e *ErrWrap) CombineLayer() *ErrWrap {
	if e.Code == 500 && e.Layer != "" {
		e.Message = fmt.Sprintf("%s [%s]", e.Message, e.Layer)
	}
	return e
}
