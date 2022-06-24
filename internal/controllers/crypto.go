package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type CryptoController struct {
	secretKey string
}

func NewCryptoController(secretKey string) *CryptoController {
	return &CryptoController{
		secretKey: secretKey,
	}
}

func (c *CryptoController) GetSignature(query string) string {

	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(query))
	sig := hex.EncodeToString(h.Sum(nil))

	return sig
}
