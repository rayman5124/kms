package errs

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type CusErr struct {
	Code  int
	Type  string
	Inner error
	Trace string
	Func  string
}

func (e *CusErr) Error() string {
	return e.Type
}

type err struct {
	Code int
	Type string
}

var Errs = map[string]err{
	"BadRequestErr":    {400, "bad request error"},
	"KeyIdNotFoundErr": {402, "keyID not found"},
	"InvalidKeyErr":    {403, "key is invalid"},
	"InvalidMarkerErr": {404, "marker is invalid"},
	"InvalidTxnErr":    {405, "serialized transaction is invalid"},

	"InternalServerErr":  {500, "internal server error"},
	"UnhandledServerErr": {501, "unhandled server error"},

	"UnhandledAwsKmsErr": {600, "unhandled aws_kms error"},
}

func BadRequestErr(err error) error {
	return &CusErr{
		Code:  Errs["BadRequestErr"].Code,
		Type:  Errs["BadRequestErr"].Type,
		Inner: err,
	}
}

func KeyIdNotFoundErr(err error) error {
	return &CusErr{
		Code:  Errs["KeyIdNotFoundErr"].Code,
		Type:  Errs["KeyIdNotFoundErr"].Type,
		Inner: err,
	}
}

func InvalidKeyErr(err error) error {
	return &CusErr{
		Code:  Errs["InvalidKeyErr"].Code,
		Type:  Errs["InvalidKeyErr"].Type,
		Inner: err,
	}
}

func InvalidMarkerErr(err error) error {
	return &CusErr{
		Code:  Errs["InvalidMarkerErr"].Code,
		Type:  Errs["InvalidMarkerErr"].Type,
		Inner: err,
	}
}

func InvalidTxnErr(err error) error {
	return &CusErr{
		Code:  Errs["InvalidTxnErr"].Code,
		Type:  Errs["InvalidTxnErr"].Type,
		Inner: err,
	}
}

func InternalServerErr(err error) error {
	pc, file, line, _ := runtime.Caller(1)
	funcs := runtime.FuncForPC(pc).Name()
	trace := fmt.Sprintf("%s:%v", file, line)

	return &CusErr{
		Code:  Errs["InternalServerErr"].Code,
		Type:  Errs["InternalServerErr"].Type,
		Inner: err,
		Trace: trace,
		Func:  funcs,
	}
}

func UnhandledAwsKmsErr(err error) error {
	pc, file, line, _ := runtime.Caller(2)
	funcs := runtime.FuncForPC(pc).Name()
	trace := fmt.Sprintf("%s:%v", file, line)

	return &CusErr{
		Code:  Errs["UnhandledAwsKmsErr"].Code,
		Type:  Errs["UnhandledAwsKmsErr"].Type,
		Inner: err,
		Func:  funcs,
		Trace: trace,
	}
}

func UnhandledServerErr(err error) error {
	return &CusErr{
		Code:  Errs["UnhandledServerErr"].Code,
		Type:  Errs["UnhandledServerErr"].Type,
		Inner: err,
	}
}

func RouteAwsErr(err error) error {
	var (
		awsErrRes        *awsHttp.ResponseError
		notFoundErr      *awsTypes.NotFoundException
		invalidMarkerErr *awsTypes.InvalidMarkerException
		invalidStateErr  *awsTypes.KMSInvalidStateException
	)

	if errors.As(err, &notFoundErr) { // 없거나 잘못된 kms-keyID
		if splitedMsg := strings.Split(*notFoundErr.Message, "/"); len(splitedMsg) > 1 {
			return KeyIdNotFoundErr(fmt.Errorf("keyId '%v not found", splitedMsg[1]))
		} else {
			return KeyIdNotFoundErr(fmt.Errorf(*notFoundErr.Message))
		}

	} else if errors.As(err, &invalidMarkerErr) { // 없거나 잘못된 marker
		newMsg, _ := strings.CutSuffix(invalidMarkerErr.Error(), ": ")
		return InvalidMarkerErr(fmt.Errorf(newMsg))

	} else if errors.As(err, &invalidStateErr) { // 사용불가능한 상태의 kms-key
		if splitedMsg := strings.Split(*invalidStateErr.Message, "/"); len(splitedMsg) > 1 {
			return InvalidKeyErr(fmt.Errorf("keyId '%v", splitedMsg[1]))
		} else {
			return InvalidKeyErr(fmt.Errorf(*invalidStateErr.Message))
		}

	} else if errors.As(err, &awsErrRes) {
		return UnhandledAwsKmsErr(err)
	} else {
		return UnhandledAwsKmsErr(err)
	}
}
