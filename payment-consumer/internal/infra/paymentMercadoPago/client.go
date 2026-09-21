package paymentmercadopago

import (
	"fmt"

	"github.com/mercadopago/sdk-go/pkg/config"
)

type Client struct {
	cfg           *config.Config
	tokenFile     string
	encryptionKey string
}

func NewClient(accessToken, tokenFile, encryptionKey string) (*Client, error) {
	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, fmt.Errorf("create mercado pago config: %w", err)
	}
	return &Client{cfg: cfg, tokenFile: tokenFile, encryptionKey: encryptionKey}, nil
}
