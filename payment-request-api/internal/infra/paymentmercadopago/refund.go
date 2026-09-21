package paymentmercadopago

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/refund"
)

type RefundClient struct {
	accessToken string
	store       *EncryptedFileTokenStore
}

func NewRefundClient(accessToken string, store *EncryptedFileTokenStore) *RefundClient {
	return &RefundClient{accessToken: strings.TrimSpace(accessToken), store: store}
}

func (c *RefundClient) Refund(ctx context.Context, gatewayPaymentID, sellerID string, amountCents int64) (string, error) {
	client, id, err := c.client(ctx, gatewayPaymentID, sellerID)
	if err != nil {
		return "", err
	}
	var result *refund.Response
	if amountCents > 0 {
		result, err = client.CreatePartialRefund(ctx, id, float64(amountCents)/100)
	} else {
		result, err = client.Create(ctx, id)
	}
	if err != nil {
		return "", fmt.Errorf("create Mercado Pago refund: %w", err)
	}
	if result == nil || result.ID <= 0 || !strings.EqualFold(result.Status, "approved") {
		return "", fmt.Errorf("Mercado Pago refund was not approved")
	}
	return strconv.Itoa(result.ID), nil
}

func (c *RefundClient) FindRefund(ctx context.Context, gatewayPaymentID, sellerID string, amountCents int64) (string, bool, error) {
	client, id, err := c.client(ctx, gatewayPaymentID, sellerID)
	if err != nil {
		return "", false, err
	}
	items, err := client.List(ctx, id)
	if err != nil {
		return "", false, fmt.Errorf("list Mercado Pago refunds: %w", err)
	}
	for _, item := range items {
		if item.ID > 0 && strings.EqualFold(item.Status, "approved") && int64(item.Amount*100+0.5) == amountCents {
			return strconv.Itoa(item.ID), true, nil
		}
	}
	return "", false, nil
}

func (c *RefundClient) client(ctx context.Context, gatewayPaymentID, sellerID string) (refund.Client, int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(gatewayPaymentID))
	if err != nil || id <= 0 {
		return nil, 0, fmt.Errorf("invalid Mercado Pago payment id %q", gatewayPaymentID)
	}
	accessToken := c.accessToken
	if strings.TrimSpace(sellerID) != "" {
		if c.store == nil {
			return nil, 0, fmt.Errorf("seller token store is not configured")
		}
		token, err := c.store.Load(sellerID)
		if err != nil {
			return nil, 0, err
		}
		accessToken = token.AccessToken
	}
	if accessToken == "" {
		return nil, 0, fmt.Errorf("Mercado Pago access token is not configured")
	}
	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, 0, err
	}
	return refund.NewClient(cfg), id, nil
}
