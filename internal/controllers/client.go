package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

type ClientController struct {
	client *http.Client
	logger *logrus.Logger

	apiKey string
}

func NewClientController(
	client *http.Client,
	apiKey string,
	logger *logrus.Logger,
) *ClientController {
	return &ClientController{
		client: client,
		apiKey: apiKey,
		logger: logger,
	}
}

var (
	ErrCodeUnknownOrderSent = -2011
	ErrUnknownOrderSent     = fmt.Errorf("%s", "Unknown order sent.")
)

type ErrStruct struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (c *ClientController) Send(method string, url *url.URL, body []byte, useApiKey bool) ([]byte, error) {
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	if useApiKey {
		req.Header.Add("X-MBX-APIKEY", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respErr, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusBadRequest {
			var errMsg ErrStruct
			if err := json.Unmarshal(respErr, &errMsg); err != nil {
				return nil, err
			}
			switch errMsg.Code {
			case ErrCodeUnknownOrderSent:
				return nil, ErrUnknownOrderSent
			}

			return nil, fmt.Errorf("%s Err:%+v", "Unknown error", errMsg)
		}

		return nil, errors.New(fmt.Sprintf("statusCode %d; resp %s;", resp.StatusCode, respErr))
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return out, nil
}
