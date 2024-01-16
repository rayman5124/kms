package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/exp/maps"
)

var (
	// logFields = []string{"ip", "status", "path", "method", "reqHeaders", "queryParams", "body", "resBody", "time", "latency", "error"}
	logFields = []string{"ip", "status", "path", "method", "queryParams", "body", "resBody", "time", "latency", "error"}
	// zlogger   = zerolog.New(os.Stdout).With().Timestamp().Logger()
	zlogger = zerolog.New(os.Stdout)
	sep     = "\r\n"
)

func formatter() string {
	formatted := make([]string, len(logFields))
	for i, field := range logFields {
		formatted[i] = fmt.Sprintf("%s:${"+field+"}", field)
	}
	return strings.Join(formatted, sep)
}

type writer struct {
}

func (w *writer) Write(elements []byte) (int, error) {
	elMap := make(map[string]interface{})
	for _, each := range bytes.Split(elements, []byte(sep)) {
		k, val := destructNprocess(each, []byte(":"))
		elMap[k] = val
	}

	var logEvent *zerolog.Event
	switch errLayer := math.Floor(elMap["status"].(float64)/100) * 100; errLayer {
	case 500:
		logEvent = zlogger.Error().Err(fmt.Errorf(elMap["error"].(string)))
	default:
		logEvent = zlogger.Info()
	}
	delete(elMap, "error")

	for _, k := range maps.Keys(elMap) {
		logEvent.Interface(k, elMap[k])
	}
	logEvent.Send()

	return len(elements), nil
}

func destructNprocess(slice []byte, sep []byte) (string, interface{}) {
	splited := bytes.SplitN(slice, sep, 2)
	var (
		k     = string(splited[0])
		byteV = splited[1]
		jsonV any
	)

	if k == "body" {
		k = "reqBody"
	}

	if bytes.Contains(byteV, []byte("=")) {
		newMapVal := make(map[string]interface{})
		for _, each := range bytes.Split(byteV, []byte("&")) {
			innerKV := bytes.SplitN(each, []byte("="), 2)
			if len(innerKV) >= 2 {
				newMapVal[string(innerKV[0])] = string(innerKV[1])
			}
		}
		return k, newMapVal

	} else if err := json.Unmarshal(byteV, &jsonV); err != nil {
		return k, strings.Trim(string(byteV), " ")

	} else {
		return k, jsonV
	}
}
