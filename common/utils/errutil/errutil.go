package errutil

import (
	"log"
)

func HandleFatal[T any](val T, err error) T {
	if err != nil {
		log.Fatal(err.Error())
	}
	return val
}
