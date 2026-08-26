package paymentmercadopago

import (
	"fmt"

	"github.com/mercadopago/sdk-go/pkg/config"
)

type Client struct {
	cfg *config.Config
}

func NewClient(accessToken string) (*Client, error) {
	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, fmt.Errorf("create mercado pago config: %w", err)
	}
	return &Client{cfg: cfg}, nil
}
