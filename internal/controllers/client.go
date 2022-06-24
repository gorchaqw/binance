package controllers

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
)

type ClientController struct {
	client *http.Client

	apiKey string
}

func NewClientController(
	client *http.Client,
	apiKey string,
) *ClientController {
	return &ClientController{
		client: client,
		apiKey: apiKey,
	}
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

	defer func() {
		_ = resp.Body.Close()
	}()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return out, nil
}
