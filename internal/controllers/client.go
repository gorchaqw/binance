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
	ErrCodeOrderWouldImmediatelyTrigger = -2021
	ErrOrderWouldImmediatelyTrigger     = fmt.Errorf("%s", "Order would immediately trigger.")

	ErrCodeUnknownOrderSent = -2011
	ErrUnknownOrderSent     = fmt.Errorf("%s", "Unknown order sent.")

	ErrCodeInternalError = -1001
	ErrErrInternalError  = fmt.Errorf("%s", "Internal error; unable to process your request. Please try again.")
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

		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusServiceUnavailable {
			var errMsg ErrStruct
			if err := json.Unmarshal(respErr, &errMsg); err != nil {
				return nil, err
			}
			switch errMsg.Code {
			case ErrCodeOrderWouldImmediatelyTrigger:
				return nil, ErrOrderWouldImmediatelyTrigger
			case ErrCodeInternalError:
				return nil, ErrErrInternalError
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
