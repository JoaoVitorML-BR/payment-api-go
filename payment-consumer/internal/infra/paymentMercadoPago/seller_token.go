package paymentmercadopago

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type encryptedSellerCredentials struct {
	UserID string `json:"user_id"`
	Token  struct {
		AccessToken string `json:"access_token"`
		UserID      string `json:"user_id"`
	} `json:"token"`
}

func (c *Client) sellerAccessToken(sellerID string) (string, error) {
	if strings.TrimSpace(c.tokenFile) == "" || strings.TrimSpace(c.encryptionKey) == "" {
		return "", errors.New("seller OAuth token storage is not configured")
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(c.encryptionKey))
	if err != nil || len(key) != 32 {
		return "", errors.New("seller OAuth encryption key is invalid")
	}
	ciphertext, err := os.ReadFile(c.tokenFile)
	if err != nil {
		return "", fmt.Errorf("read seller OAuth token: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(ciphertext)))
	if err != nil {
		return "", fmt.Errorf("decode seller OAuth token: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create seller OAuth cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create seller OAuth cipher mode: %w", err)
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("seller OAuth token ciphertext is invalid")
	}
	nonce, encrypted := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", errors.New("seller OAuth token could not be decrypted")
	}
	var credentials encryptedSellerCredentials
	if err := json.Unmarshal(plain, &credentials); err != nil {
		return "", fmt.Errorf("decode seller OAuth credentials: %w", err)
	}
	if credentials.UserID != sellerID || credentials.Token.UserID != sellerID || credentials.Token.AccessToken == "" {
		return "", errors.New("seller OAuth token does not match requested seller")
	}
	return credentials.Token.AccessToken, nil
}
