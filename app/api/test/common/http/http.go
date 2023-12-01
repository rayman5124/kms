package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"
)

type resData struct {
	Status int
	Body   []byte
}

func Request(app *fiber.App, method string, path string, body []byte) (*resData, error) {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return &resData{res.StatusCode, resBody}, nil
}

func PrettyJson(v interface{}) string {
	ret, _ := json.MarshalIndent(v, "", "\t")
	return string(ret)
}
